package clusterservice

/*
Copyright 2017-2018 Crunchy Data Solutions, Inc.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import (
	"database/sql"
	"errors"
	"fmt"

	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	"strconv"
	"strings"
	"time"

	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	_ "github.com/lib/pq"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

// DeleteCluster ...
func DeleteCluster(name, selector string, deleteData, deleteBackups, deleteConfigs bool) msgs.DeleteClusterResponse {
	var err error

	response := msgs.DeleteClusterResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	response.Results = make([]string, 0)

	if name != "all" {
		if selector == "" {
			selector = "name=" + name
		}
	}
	log.Debugf("delete-data is [%v]\n", deleteData)
	log.Debugf("delete-backups is [%v]\n", deleteBackups)
	log.Debugf("delete-configs is [%v]\n", deleteConfigs)

	clusterList := crv1.PgclusterList{}

	//get the clusters list
	err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient,
		&clusterList, selector,
		apiserver.Namespace)
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	if len(clusterList.Items) == 0 {
		response.Status.Code = msgs.Error
		response.Status.Msg = "no clusters found"
		return response
	}

	for _, cluster := range clusterList.Items {

		if deleteData {
			err := deleteDatabaseSecrets(cluster.Spec.Name)
			if err != nil {
				response.Status.Code = msgs.Error
				response.Status.Msg = err.Error()
				return response
			}
			err = createDeleteDataTasks(cluster.Spec.Name, cluster.Spec.PrimaryStorage, deleteBackups)
			if err != nil {
				response.Status.Code = msgs.Error
				response.Status.Msg = err.Error()
				return response
			}
		}

		if deleteConfigs {
			if err := deleteConfigMaps(cluster.Spec.Name); err != nil {
				response.Status.Code = msgs.Error
				response.Status.Msg = err.Error()
				return response
			}
		}

		err := kubeapi.Deletepgcluster(apiserver.RESTClient,
			cluster.Spec.Name, apiserver.Namespace)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		} else {
			response.Results = append(response.Results, "deleted pgcluster "+cluster.Spec.Name)
		}
	}

	return response

}

// ShowCluster ...
func ShowCluster(name, selector, ccpimagetag string) msgs.ShowClusterResponse {
	var err error

	response := msgs.ShowClusterResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	response.Results = make([]msgs.ShowClusterDetail, 0)

	if selector == "" && name == "all" {
	} else {
		if selector == "" {
			selector = "name=" + name
		}
	}

	clusterList := crv1.PgclusterList{}

	//get a list of all clusters
	err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient,
		&clusterList, selector, apiserver.Namespace)
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	log.Debugf("clusters found len is %d\n", len(clusterList.Items))

	for _, c := range clusterList.Items {
		detail := msgs.ShowClusterDetail{}
		detail.Cluster = c
		detail.Deployments, err = getDeployments(&c)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		detail.Pods, err = GetPods(&c)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		detail.Services, err = getServices(&c)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		detail.Replicas, err = getReplicas(&c)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		if ccpimagetag == "" {
			response.Results = append(response.Results, detail)
		} else if ccpimagetag == c.Spec.CCPImageTag {
			response.Results = append(response.Results, detail)
		}
	}

	return response

}

func getDeployments(cluster *crv1.Pgcluster) ([]msgs.ShowClusterDeployment, error) {
	output := make([]msgs.ShowClusterDeployment, 0)

	selector := util.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name
	deployments, err := kubeapi.GetDeployments(apiserver.Clientset, selector, apiserver.Namespace)
	if err != nil {
		return output, err
	}

	for _, dep := range deployments.Items {
		d := msgs.ShowClusterDeployment{}
		d.Name = dep.Name
		d.PolicyLabels = make([]string, 0)

		for k, v := range dep.ObjectMeta.Labels {
			if v == "pgpolicy" {
				d.PolicyLabels = append(d.PolicyLabels, k)
			}
		}
		output = append(output, d)

	}

	return output, err
}

func GetPods(cluster *crv1.Pgcluster) ([]msgs.ShowClusterPod, error) {

	output := make([]msgs.ShowClusterPod, 0)

	//get pods, but exclude pgpool and backup pods
	selector := util.LABEL_PGBACKUP + "!=true," + util.LABEL_PGBACKUP + "!=false," + util.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name

	pods, err := kubeapi.GetPods(apiserver.Clientset, selector, apiserver.Namespace)
	if err != nil {
		return output, err
	}
	for _, p := range pods.Items {
		d := msgs.ShowClusterPod{}
		d.Name = p.Name
		d.Phase = string(p.Status.Phase)
		d.NodeName = p.Spec.NodeName
		d.ReadyStatus, d.Ready = getReadyStatus(&p)
		d.PVCName = apiserver.GetPVCName(&p)
		log.Infof("after getPVCName call")

		d.Primary = false
		d.Type = getType(&p)
		if d.Type == msgs.PodTypePrimary {
			d.Primary = true
		}
		output = append(output, d)

	}

	return output, err

}

func getServices(cluster *crv1.Pgcluster) ([]msgs.ShowClusterService, error) {

	output := make([]msgs.ShowClusterService, 0)
	selector := util.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name

	services, err := kubeapi.GetServices(apiserver.Clientset, selector, apiserver.Namespace)
	if err != nil {
		return output, err
	}

	log.Debugf("got %d services for %s\n", len(services.Items), cluster.Spec.Name)
	for _, p := range services.Items {
		d := msgs.ShowClusterService{}
		d.Name = p.Name
		if strings.Contains(p.Name, "-pgbouncer") {
			d.Pgbouncer = true
			d.ClusterName = cluster.Name
		}
		d.ClusterIP = p.Spec.ClusterIP
		if len(p.Spec.ExternalIPs) > 0 {
			d.ExternalIP = p.Spec.ExternalIPs[0]
		}
		if len(p.Status.LoadBalancer.Ingress) > 0 {
			d.ExternalIP = p.Status.LoadBalancer.Ingress[0].IP
		}

		output = append(output, d)

	}

	return output, err
}

func TestCluster(name, selector string) msgs.ClusterTestResponse {
	var err error

	response := msgs.ClusterTestResponse{}
	response.Results = make([]msgs.ClusterTestResult, 0)
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	log.Debugf("selector is %s", selector)
	if selector == "" && name == "all" {
		log.Debug("selector is empty and name is all")
	} else {
		if selector == "" {
			selector = "name=" + name
		}
	}

	//get a list of matching clusters
	clusterList := crv1.PgclusterList{}
	err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient, &clusterList, selector, apiserver.Namespace)

	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	//loop thru each cluster

	log.Debugf("clusters found len is %d\n", len(clusterList.Items))

	for _, c := range clusterList.Items {
		result := msgs.ClusterTestResult{}
		result.ClusterName = c.Name

		detail := msgs.ShowClusterDetail{}
		detail.Cluster = c

		pods, err := GetPods(&c)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		//loop thru the pods, make sure they are all ready
		primaryReady := true
		replicaReady := true
		for _, pod := range pods {
			if pod.Type == msgs.PodTypePrimary {
				if !pod.Ready {
					primaryReady = false
				}
			} else if pod.Type == msgs.PodTypeReplica {
				if !pod.Ready {
					replicaReady = false
				}
			}

		}
		if !primaryReady {
			response.Status.Code = msgs.Error
			response.Status.Msg = "cluster not ready yet, try later"
			return response
		}

		//get the services for this cluster
		detail.Services, err = getServices(&c)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		//get the secrets for this cluster
		secrets, err := apiserver.GetSecrets(&c)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		result.Items = make([]msgs.ClusterTestDetail, 0)

		//for each service run a test and add results to output
		for _, service := range detail.Services {

			databases := make([]string, 0)
			if service.Pgbouncer {
				databases = append(databases, service.ClusterName)
				databases = append(databases, service.ClusterName+"-replica")
			} else {
				databases = append(databases, "postgres")
				databases = append(databases, c.Spec.Database)
			}
			for _, s := range secrets {
				for _, db := range databases {
					item := msgs.ClusterTestDetail{}
					username := s.Username
					password := s.Password
					database := db
					item.PsqlString = "psql -p " + c.Spec.Port + " -h " + service.ClusterIP + " -U " + username + " " + database
					log.Debug(item.PsqlString)
					if (service.Name != c.ObjectMeta.Name) && replicaReady == false {
						item.Working = false
					} else {
						status := query(username, service.ClusterIP, c.Spec.Port, database, password)
						item.Working = false
						if status {
							item.Working = true
						}
					}
					result.Items = append(result.Items, item)
				}
			}
		}
		response.Results = append(response.Results, result)
	}

	return response
}

func query(dbUser, dbHost, dbPort, database, dbPassword string) bool {
	var conn *sql.DB
	var err error

	connString := "sslmode=disable user=" + dbUser + " host=" + dbHost + " port=" + dbPort + " dbname=" + database + " password=" + dbPassword
	//log.Debugf("connString=%s", connString)

	conn, err = sql.Open("postgres", connString)
	if err != nil {
		log.Debug(err.Error())
		return false
	}

	var ts string
	var rows *sql.Rows

	rows, err = conn.Query("select now()::text")
	if err != nil {
		log.Debug(err.Error())
		return false
	}

	defer func() {
		if conn != nil {
			conn.Close()
		}
		if rows != nil {
			rows.Close()
		}
	}()
	for rows.Next() {
		if err = rows.Scan(&ts); err != nil {
			log.Debug(err.Error())
			return false
		}
		log.Debugf("returned %s", ts)
		return true
	}
	return false

}

// CreateCluster ...
// pgo create cluster mycluster
func CreateCluster(request *msgs.CreateClusterRequest) msgs.CreateClusterResponse {
	resp := msgs.CreateClusterResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)
	clusterName := request.Name

	if clusterName == "all" {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "invalid cluster name 'all' is not allowed as a cluster name"
		return resp
	}

	if request.ReplicaCount < 0 {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "invalid replica-count , should be greater than or equal to 0"
		return resp
	}

	errs := validation.IsDNS1035Label(clusterName)
	if len(errs) > 0 {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "invalid cluster name format " + errs[0]
		return resp
	}

	for i := 0; i < request.Series; i++ {
		if request.Series > 1 {
			clusterName = request.Name + strconv.Itoa(i)
		}
		log.Debugf("create cluster called for %s", clusterName)
		result := crv1.Pgcluster{}

		// error if it already exists
		found, err := kubeapi.Getpgcluster(apiserver.RESTClient, &result, clusterName, apiserver.Namespace)
		if err == nil {
			log.Debugf("pgcluster %s was found so we will not create it", clusterName)
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "pgcluster " + clusterName + " was found so we will not create it"
			return resp
		} else if !found {
			log.Debugf("pgcluster %s not found so we will create it", clusterName)
		} else {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "error getting pgcluster " + clusterName + err.Error()
			return resp
		}

		userLabelsMap := make(map[string]string)
		if request.UserLabels != "" {
			labels := strings.Split(request.UserLabels, ",")

			for _, v := range labels {
				p := strings.Split(v, "=")
				if len(p) < 2 {
					resp.Status.Code = msgs.Error
					resp.Status.Msg = "invalid labels format"
					return resp
				}
				userLabelsMap[p[0]] = p[1]
			}
		}

		if request.CustomConfig != "" {
			found := validateCustomConfig(request.CustomConfig)
			if !found {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = request.CustomConfig + " configmap was not found "
				return resp
			}
			//add a label for the custom config
			userLabelsMap[util.LABEL_CUSTOM_CONFIG] = request.CustomConfig
		}
		//set the metrics flag with the global setting first
		userLabelsMap[util.LABEL_COLLECT] = strconv.FormatBool(apiserver.MetricsFlag)
		if err != nil {
			log.Error(err)
		}
		//set the badger flag with the global setting first
		userLabelsMap[util.LABEL_BADGER] = strconv.FormatBool(apiserver.BadgerFlag)
		if err != nil {
			log.Error(err)
		}

		//if metrics is chosen on the pgo command, stick it into the user labels
		if request.MetricsFlag {
			userLabelsMap[util.LABEL_COLLECT] = "true"
		}
		if request.BadgerFlag {
			userLabelsMap[util.LABEL_BADGER] = "true"
		}
		if request.AutofailFlag || apiserver.Pgo.Cluster.Autofail {
			userLabelsMap[util.LABEL_AUTOFAIL] = "true"
		}
		if request.ServiceType != "" {
			if request.ServiceType != config.DEFAULT_SERVICE_TYPE && request.ServiceType != config.LOAD_BALANCER_SERVICE_TYPE && request.ServiceType != config.NODEPORT_SERVICE_TYPE {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = "error ServiceType should be either ClusterIP or LoadBalancer "

				return resp
			}
			userLabelsMap[util.LABEL_SERVICE_TYPE] = request.ServiceType
		}

		if request.ArchiveFlag {
			userLabelsMap[util.LABEL_ARCHIVE] = "true"
			log.Debug("archive set to true in user labels")
		} else {
			log.Debug("using ArchiveMode from pgo.yaml")
			userLabelsMap[util.LABEL_ARCHIVE] = apiserver.Pgo.Cluster.ArchiveMode
		}
		if request.BackrestFlag {
			userLabelsMap[util.LABEL_BACKREST] = "true"
			log.Debug("backrest set to true in user labels")
		} else {
			log.Debug("using Backrest from pgo.yaml")
			userLabelsMap[util.LABEL_BACKREST] = strconv.FormatBool(apiserver.Pgo.Cluster.Backrest)
		}

		err = validateBackrestConfig(userLabelsMap)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		//add archive if backrest is requested
		if userLabelsMap[util.LABEL_BACKREST] == "true" {
			userLabelsMap[util.LABEL_ARCHIVE] = "true"
		}

		userLabelsMap[util.LABEL_ARCHIVE_TIMEOUT] = apiserver.Pgo.Cluster.ArchiveTimeout

		if request.PgpoolFlag {
			userLabelsMap[util.LABEL_PGPOOL] = "true"
			userLabelsMap[util.LABEL_PGPOOL_SECRET] = request.PgpoolSecret
			log.Debug("userLabelsMap")
			log.Debugf("%v", userLabelsMap)
		}

		if request.PgbouncerFlag {
			userLabelsMap[util.LABEL_PGBOUNCER] = "true"
			userLabelsMap[util.LABEL_PGBOUNCER_SECRET] = request.PgbouncerSecret
			log.Debug("userLabelsMap")
			log.Debugf("%v", userLabelsMap)
		}

		if existsGlobalConfig() {
			userLabelsMap[util.LABEL_CUSTOM_CONFIG] = util.GLOBAL_CUSTOM_CONFIGMAP
		}

		if request.StorageConfig != "" {
			if apiserver.IsValidStorageName(request.StorageConfig) == false {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = request.StorageConfig + " Storage config was not found "
				return resp
			}
		}

		if apiserver.Pgo.Cluster.PrimaryNodeLabel != "" {
			//already should be validate at apiserver startup
			parts := strings.Split(apiserver.Pgo.Cluster.PrimaryNodeLabel, "=")
			userLabelsMap[util.LABEL_NODE_LABEL_KEY] = parts[0]
			userLabelsMap[util.LABEL_NODE_LABEL_VALUE] = parts[1]
			log.Debug("primary node labels used from pgo.yaml")
		}

		if request.NodeLabel != "" {
			parts := strings.Split(request.NodeLabel, "=")
			if len(parts) != 2 {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = request.NodeLabel + " node label does not follow key=value format"
				return resp
			}

			keyValid, valueValid, err := apiserver.IsValidNodeLabel(parts[0], parts[1])
			if err != nil {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = err.Error()
				return resp
			}

			if !keyValid {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = request.NodeLabel + " key was not valid .. check node labels for correct values to specify"
				return resp
			}
			if !valueValid {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = request.NodeLabel + " node label value was not valid .. check node labels for correct values to specify"
				return resp
			}
			log.Debug("primary node labels used from user entered flag")
			userLabelsMap[util.LABEL_NODE_LABEL_KEY] = parts[0]
			userLabelsMap[util.LABEL_NODE_LABEL_VALUE] = parts[1]
		}

		if request.ReplicaStorageConfig != "" {
			if apiserver.IsValidStorageName(request.ReplicaStorageConfig) == false {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = request.ReplicaStorageConfig + " Storage config was not found "
				return resp
			}
		}

		if request.ContainerResources != "" {
			if apiserver.IsValidContainerResource(request.ContainerResources) == false {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = request.ContainerResources + " ContainerResource config was not found "
				return resp
			}
		}

		if request.SecretFrom != "" {
			err = validateSecretFrom(request.SecretFrom)
			if err != nil {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = request.SecretFrom + " secret was not found "
				return resp
			}
		}

		// Create an instance of our CRD
		newInstance := getClusterParams(request, clusterName, userLabelsMap)
		validateConfigPolicies(clusterName, request.Policies)

		t := time.Now()
		newInstance.Spec.PswLastUpdate = t.Format(time.RFC3339)

		err = kubeapi.Createpgcluster(apiserver.RESTClient,
			newInstance, apiserver.Namespace)
		if err != nil {
			resp.Results = append(resp.Results, err.Error())
		} else {
			resp.Results = append(resp.Results, "created Pgcluster "+clusterName)
		}
	}

	return resp

}

func validateConfigPolicies(clusterName, PoliciesFlag string) error {
	var err error
	var configPolicies string

	if PoliciesFlag == "" {
		log.Debugf("%s is Pgo.Cluster.Policies", apiserver.Pgo.Cluster.Policies)
		configPolicies = apiserver.Pgo.Cluster.Policies
	} else {
		configPolicies = PoliciesFlag
	}

	if configPolicies == "" {
		log.Debug("no policies are specified in either pgo.yaml or from user")
		return err
	}

	policies := strings.Split(configPolicies, ",")

	for _, v := range policies {
		result := crv1.Pgpolicy{}

		// error if it already exists
		found, err := kubeapi.Getpgpolicy(apiserver.RESTClient,
			&result, v, apiserver.Namespace)
		if !found {
			log.Error("policy " + v + " specified in configuration was not found")
			return err
		}

		if err != nil {
			log.Error("error getting pgpolicy " + v + err.Error())
			return err
		}
		//create a pgtask to add the policy after the db is ready
	}

	spec := crv1.PgtaskSpec{}
	spec.StorageSpec = crv1.PgStorageSpec{}
	spec.TaskType = crv1.PgtaskAddPolicies
	spec.Status = "requested"
	spec.Parameters = make(map[string]string)
	for _, v := range policies {
		spec.Parameters[v] = v
	}
	spec.Name = clusterName + "-policies"
	labels := make(map[string]string)
	labels[util.LABEL_PG_CLUSTER] = clusterName

	newInstance := &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:   spec.Name,
			Labels: labels,
		},
		Spec: spec,
	}

	kubeapi.Createpgtask(apiserver.RESTClient, newInstance, apiserver.Namespace)

	return err
}

func getClusterParams(request *msgs.CreateClusterRequest, name string, userLabelsMap map[string]string) *crv1.Pgcluster {

	spec := crv1.PgclusterSpec{}

	if request.ContainerResources != "" {
		spec.ContainerResources, _ = apiserver.Pgo.GetContainerResource(request.ContainerResources)
	} else {
		log.Debugf("Pgo.DefaultContainerResources is %s", apiserver.Pgo.DefaultContainerResources)
		defaultContainerResource := apiserver.Pgo.DefaultContainerResources
		if defaultContainerResource != "" {
			spec.ContainerResources, _ = apiserver.Pgo.GetContainerResource(defaultContainerResource)
		}
	}

	if request.StorageConfig != "" {
		spec.PrimaryStorage, _ = apiserver.Pgo.GetStorageSpec(request.StorageConfig)
	} else {
		log.Debugf("%v", apiserver.Pgo.PrimaryStorage)
		spec.PrimaryStorage, _ = apiserver.Pgo.GetStorageSpec(apiserver.Pgo.PrimaryStorage)
	}

	if request.ReplicaStorageConfig != "" {
		spec.ReplicaStorage, _ = apiserver.Pgo.GetStorageSpec(request.ReplicaStorageConfig)
	} else {
		spec.ReplicaStorage, _ = apiserver.Pgo.GetStorageSpec(apiserver.Pgo.ReplicaStorage)
		log.Debugf("%v", apiserver.Pgo.ReplicaStorage)
	}

	spec.CCPImageTag = apiserver.Pgo.Cluster.CCPImageTag
	log.Debugf("Pgo.Cluster.CCPImageTag %s", apiserver.Pgo.Cluster.CCPImageTag)
	if request.CCPImageTag != "" {
		spec.CCPImageTag = request.CCPImageTag
		log.Debugf("using CCPImageTag from command line %s", request.CCPImageTag)
	}

	spec.Name = name
	spec.ClusterName = name
	spec.Port = "5432"
	spec.SecretFrom = ""
	spec.BackupPath = ""
	spec.BackupPVCName = ""
	spec.PrimaryHost = name
	if request.Policies == "" {
		spec.Policies = apiserver.Pgo.Cluster.Policies
		log.Debugf("Pgo.Cluster.Policies %s", apiserver.Pgo.Cluster.Policies)
	} else {
		spec.Policies = request.Policies
	}

	spec.PrimaryPassword = request.Password
	spec.User = "testuser"
	spec.Password = request.Password
	spec.Database = "userdb"
	spec.RootPassword = request.Password
	spec.Replicas = "0"
	str := apiserver.Pgo.Cluster.Replicas
	log.Debugf("[%s] is Pgo.Cluster.Replicas", str)
	if str != "" {
		spec.Replicas = str
	}
	log.Debugf("replica count is %d", request.ReplicaCount)
	if request.ReplicaCount > 0 {
		spec.Replicas = strconv.Itoa(request.ReplicaCount)
		log.Debugf("replicas is  %s", spec.Replicas)
	}
	spec.Strategy = "1"
	spec.UserLabels = userLabelsMap
	spec.UserLabels[util.LABEL_PGO_VERSION] = msgs.PGO_VERSION

	//override any values from config file
	str = apiserver.Pgo.Cluster.Port
	log.Debugf("%d", apiserver.Pgo.Cluster.Port)
	if str != "" {
		spec.Port = str
	}
	str = apiserver.Pgo.Cluster.User
	log.Debugf("Pgo.Cluster.User is %s", apiserver.Pgo.Cluster.User)
	if str != "" {
		spec.User = str
	}
	str = apiserver.Pgo.Cluster.Database
	log.Debugf("Pgo.Cluster.Database is %s", apiserver.Pgo.Cluster.Database)
	if str != "" {
		spec.Database = str
	}
	str = apiserver.Pgo.Cluster.Strategy
	log.Debugf("%d", apiserver.Pgo.Cluster.Strategy)
	if str != "" {
		spec.Strategy = str
	}
	//pass along command line flags for a restore
	if request.SecretFrom != "" {
		spec.SecretFrom = request.SecretFrom
	}

	spec.BackupPath = request.BackupPath
	if request.BackupPVC != "" {
		spec.BackupPVCName = request.BackupPVC
	}

	spec.CustomConfig = request.CustomConfig

	labels := make(map[string]string)
	labels["name"] = name

	newInstance := &crv1.Pgcluster{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: spec,
		Status: crv1.PgclusterStatus{
			State:   crv1.PgclusterStateCreated,
			Message: "Created, not processed yet",
		},
	}
	return newInstance
}

func validateSecretFrom(secretname string) error {
	var err error
	selector := util.LABEL_PG_DATABASE + "=" + secretname
	secrets, err := kubeapi.GetSecrets(apiserver.Clientset, selector, apiserver.Namespace)
	if err != nil {
		return err
	}

	log.Debugf("secrets for %s", secretname)
	pgprimaryFound := false
	pgrootFound := false
	pguserFound := false

	for _, s := range secrets.Items {
		if s.ObjectMeta.Name == secretname+crv1.PrimarySecretSuffix {
			pgprimaryFound = true
		} else if s.ObjectMeta.Name == secretname+crv1.RootSecretSuffix {
			pgrootFound = true
		} else if s.ObjectMeta.Name == secretname+crv1.UserSecretSuffix {
			pguserFound = true
		}
	}
	if !pgprimaryFound {
		return errors.New(secretname + crv1.PrimarySecretSuffix + " not found")
	}
	if !pgrootFound {
		return errors.New(secretname + crv1.RootSecretSuffix + " not found")
	}
	if !pguserFound {
		return errors.New(secretname + crv1.UserSecretSuffix + " not found")
	}

	return err
}

func getReadyStatus(pod *v1.Pod) (string, bool) {
	equal := false
	readyCount := 0
	containerCount := 0
	for _, stat := range pod.Status.ContainerStatuses {
		containerCount++
		if stat.Ready {
			readyCount++
		}
	}
	if readyCount == containerCount {
		equal = true
	}
	return fmt.Sprintf("%d/%d", readyCount, containerCount), equal

}

// removes data and or backup volumes for all pods in a cluster
func createDeleteDataTasks(clusterName string, storageSpec crv1.PgStorageSpec, deleteBackups bool) error {

	var err error

	selector := util.LABEL_PG_CLUSTER + "=" + clusterName + "," + util.LABEL_PGBACKUP + "!=true"
	log.Debugf("selector for delete is %s", selector)
	pods, err := kubeapi.GetPods(apiserver.Clientset, selector, apiserver.Namespace)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Debugf("got %d cluster pods for %s\n", len(pods.Items), clusterName)

	//a flag for the case when a cluster has performed an autofailover
	//we need to go back and remove the original primary's pgdata PVC
	originalClusterPrimaryDeleted := false

	for _, pod := range pods.Items {
		deploymentName := pod.ObjectMeta.Labels[util.LABEL_PG_CLUSTER]
		if pod.ObjectMeta.Labels[util.LABEL_REPLICA_NAME] != "" {
			deploymentName = pod.ObjectMeta.Labels[util.LABEL_REPLICA_NAME]
		}

		//get the volumes for this pod
		for _, v := range pod.Spec.Volumes {

			log.Debugf("volume name in delete logic is %s", v.Name)
			dataRoots := make([]string, 0)
			if v.Name == "pgdata" {
				dataRoots = append(dataRoots, deploymentName)
			} else if v.Name == "backrestrepo-volume" {
				dataRoots = append(dataRoots, deploymentName+"{-backups,-spool}")
			} else if v.Name == "backup" {
				dataRoots = append(dataRoots, deploymentName+"-backups")
			} else if v.Name == "pgwal-volume" {
				dataRoots = append(dataRoots, deploymentName+"-wal")
			}

			if v.VolumeSource.PersistentVolumeClaim != nil {
				log.Debugf("volume [%s] pvc [%s] dataroots [%v]\n", v.Name, v.VolumeSource.PersistentVolumeClaim.ClaimName, dataRoots)
				if clusterName == v.VolumeSource.PersistentVolumeClaim.ClaimName {
					originalClusterPrimaryDeleted = true
				}
				err := apiserver.CreateRMDataTask(storageSpec, clusterName, v.VolumeSource.PersistentVolumeClaim.ClaimName, dataRoots)
				if err != nil {
					return err
				}
			}
		}
	}

	if originalClusterPrimaryDeleted == false {
		log.Debugf("for autofailover case, removing orignal primary PVC %s", clusterName)
		dataRoots := make([]string, 0)
		dataRoots = append(dataRoots, clusterName)
		err := apiserver.CreateRMDataTask(storageSpec, clusterName, clusterName, dataRoots)
		if err != nil {
			return err
		}
	}

	if deleteBackups {
		log.Debug("check for backup PVC to delete")
		//get the deployment names for this cluster
		//by convention if basebackups are run, a pvc named
		//deploymentName-backup will be created to hold backups
		deps, err := kubeapi.GetDeployments(apiserver.Clientset, selector, apiserver.Namespace)
		if err != nil {
			log.Error(err)
			return err
		}

		for _, dep := range deps.Items {
			pvcName := dep.Name + "-backup"
			log.Debugf("checking dep %s for backup pvc %s\n", dep.Name, pvcName)
			_, found, err := kubeapi.GetPVC(apiserver.Clientset, pvcName, apiserver.Namespace)
			if !found {
				log.Debugf("%s pvc was not found when looking for backups to delete\n", pvcName)
			} else {
				if err != nil {
					log.Error(err)
					return err
				}
				//by convention, the root directory name
				//created by the backup job is depName-backups
				dataRoots := []string{dep.Name + "-backups"}
				err = apiserver.CreateRMDataTask(storageSpec, clusterName, pvcName, dataRoots)
				if err != nil {
					log.Error(err)
					return err
				}
			}

		}
	}
	return err
}

/**
func createTask(storageSpec crv1.PgStorageSpec, clusterName, pvcName string, dataRoots []string) {
	//create a pgtask for each root at this volume/pvc
	for i := 0; i < len(dataRoots); i++ {

		//create pgtask CRD
		spec := crv1.PgtaskSpec{}
		spec.Name = pvcName
		spec.TaskType = crv1.PgtaskDeleteData
		spec.StorageSpec = storageSpec

		spec.Parameters = make(map[string]string)
		spec.Parameters[util.LABEL_PVC_NAME] = pvcName
		spec.Parameters[util.LABEL_DATA_ROOT] = dataRoots[i]
		spec.Parameters[util.LABEL_PG_CLUSTER] = clusterName

		newInstance := &crv1.Pgtask{
			ObjectMeta: meta_v1.ObjectMeta{
				Name: pvcName,
			},
			Spec: spec,
		}
		newInstance.ObjectMeta.Labels = make(map[string]string)
		newInstance.ObjectMeta.Labels[util.LABEL_PG_CLUSTER] = clusterName
		//newInstance.ObjectMeta.Labels[util.LABEL_DATA_ROOT] = dataRoots[i]
		newInstance.ObjectMeta.Labels[util.LABEL_RMDATA] = "true"

		err := kubeapi.Createpgtask(apiserver.RESTClient,
			newInstance, apiserver.Namespace)
		if err != nil {
			log.Error(err)
		}
	}

}
*/

func getType(pod *v1.Pod) string {

	log.Infof("%v\n", pod.ObjectMeta.Labels)
	//map[string]string
	if pod.ObjectMeta.Labels[util.LABEL_PGBACKUP] == "true" {
		log.Infoln("this is a backup pod")
		return msgs.PodTypeBackup
	} else if pod.ObjectMeta.Labels[util.LABEL_PRIMARY] == "true" {
		log.Infoln("this is a primary pod")
		return msgs.PodTypePrimary
	} else {
		log.Infoln("this is a replica pod")
		return msgs.PodTypeReplica
	}
	return msgs.PodTypeUnknown

}

func validateCustomConfig(configmapname string) bool {
	_, found := kubeapi.GetConfigMap(apiserver.Clientset, configmapname, apiserver.Namespace)
	return found
}

func existsGlobalConfig() bool {
	_, found := kubeapi.GetConfigMap(apiserver.Clientset, util.GLOBAL_CUSTOM_CONFIGMAP, apiserver.Namespace)
	return found
}

func getReplicas(cluster *crv1.Pgcluster) ([]msgs.ShowClusterReplica, error) {

	output := make([]msgs.ShowClusterReplica, 0)
	replicaList := crv1.PgreplicaList{}

	selector := util.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name

	err := kubeapi.GetpgreplicasBySelector(apiserver.RESTClient,
		&replicaList, selector, apiserver.Namespace)
	if err != nil {
		return output, err
	}

	if len(replicaList.Items) == 0 {
		log.Debug("no replicas found")
		return output, err
	}

	for _, replica := range replicaList.Items {
		d := msgs.ShowClusterReplica{}
		d.Name = replica.Spec.Name
		output = append(output, d)
	}

	return output, err
}

func validateBackrestConfig(labels map[string]string) error {
	var err error

	if labels[util.LABEL_BACKREST] == "true" {
		if labels[util.LABEL_CUSTOM_CONFIG] != "" {
			//TODO could check the contents of that config map here tomake sure it include a pgbackrest.conf key
			return err
		} else {
			//check the global configmap here
			_, found := kubeapi.GetConfigMap(apiserver.Clientset, util.GLOBAL_CUSTOM_CONFIGMAP, apiserver.Namespace)
			if !found {
				log.Debugf("%s was not found", util.GLOBAL_CUSTOM_CONFIGMAP)
				return errors.New(util.GLOBAL_CUSTOM_CONFIGMAP + " global configmap or --custom-config flag not set, one of these is required for enabling pgbackrest")
			}
		}
	}
	return err
}

// deleteDatabaseSecrets delete secrets that match pg-database=somecluster
func deleteDatabaseSecrets(db string) error {
	var err error
	//get all that match pg-database=db
	selector := util.LABEL_PG_DATABASE + "=" + db
	secrets, err := kubeapi.GetSecrets(apiserver.Clientset, selector, apiserver.Namespace)
	if err != nil {
		log.Error(err)
		return err
	}

	for _, s := range secrets.Items {
		err := kubeapi.DeleteSecret(apiserver.Clientset, s.ObjectMeta.Name, apiserver.Namespace)
		if err != nil {
			log.Error(err)
			return err
		}
	}
	return err
}

func deleteConfigMaps(clusterName string) error {
	label := fmt.Sprintf("pg-cluster=%s", clusterName)
	list, ok := kubeapi.ListConfigMap(apiserver.Clientset, label, apiserver.Namespace)
	if !ok {
		return fmt.Errorf("No configMaps found for selector: %s", label)
	}

	for _, configmap := range list.Items {
		err := kubeapi.DeleteConfigMap(apiserver.Clientset, configmap.Name, apiserver.Namespace)
		if err != nil {
			return err
		}
	}
	return nil
}
