package clusterservice

/*
Copyright 2019 Crunchy Data Solutions, Inc.
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

	"io/ioutil"
	"strconv"
	"strings"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	log "github.com/sirupsen/logrus"

	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/sshutil"
	"github.com/crunchydata/postgres-operator/util"

	_ "github.com/lib/pq"
	kerrors "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

// DeleteCluster ...
func DeleteCluster(name, selector string, deleteData, deleteBackups bool, ns string) msgs.DeleteClusterResponse {
	var err error

	response := msgs.DeleteClusterResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	response.Results = make([]string, 0)

	if name != "all" {
		if selector == "" {
			selector = "name=" + name
		}
	}

	log.Debugf("delete-data is [%t]", deleteData)
	log.Debugf("delete-backups is [%t]", deleteBackups)

	clusterList := crv1.PgclusterList{}

	//get the clusters list
	err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient,
		&clusterList, selector, ns)
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
			log.Debugf("deleting data on cluster %s", cluster.Spec.Name)
			err := deleteDatabaseSecrets(cluster.Spec.Name, ns)
			if err != nil {
				log.Debugf("error on deleting secrets %s", err.Error())
				response.Status.Code = msgs.Error
				response.Status.Msg = err.Error()
				return response
			}

			err = createDeleteDataTasks(cluster.Spec.Name, cluster.Spec.PrimaryStorage, deleteBackups, ns)
		}

		err = kubeapi.Deletepgcluster(apiserver.RESTClient,
			cluster.Spec.Name, ns)
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
func ShowCluster(name, selector, ccpimagetag, ns string, allflag bool) msgs.ShowClusterResponse {
	var err error

	response := msgs.ShowClusterResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	response.Results = make([]msgs.ShowClusterDetail, 0)

	if selector == "" && allflag {
		log.Debugf("allflags set to true")
	} else {
		if selector == "" {
			selector = "name=" + name
		}
	}

	log.Debugf("selector on showCluster is %s", selector)

	clusterList := crv1.PgclusterList{}

	//get a list of all clusters
	err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient,
		&clusterList, selector, ns)
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	log.Debugf("clusters found len is %d", len(clusterList.Items))

	for _, c := range clusterList.Items {
		detail := msgs.ShowClusterDetail{}
		detail.Cluster = c
		detail.Deployments, err = getDeployments(&c, ns)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		detail.Pods, err = GetPods(&c, ns)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		detail.Services, err = getServices(&c, ns)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		detail.Replicas, err = getReplicas(&c, ns)
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

func getDeployments(cluster *crv1.Pgcluster, ns string) ([]msgs.ShowClusterDeployment, error) {
	output := make([]msgs.ShowClusterDeployment, 0)

	selector := config.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name
	deployments, err := kubeapi.GetDeployments(apiserver.Clientset, selector, ns)
	if err != nil {
		return output, err
	}

	for _, dep := range deployments.Items {
		d := msgs.ShowClusterDeployment{}
		d.Name = dep.Name
		d.PolicyLabels = make([]string, 0)

		for k, v := range cluster.ObjectMeta.Labels {
			if v == "pgpolicy" {
				d.PolicyLabels = append(d.PolicyLabels, k)
			}
		}
		output = append(output, d)

	}

	return output, err
}

func GetPods(cluster *crv1.Pgcluster, ns string) ([]msgs.ShowClusterPod, error) {

	output := make([]msgs.ShowClusterPod, 0)

	//get pods, but exclude pgpool and backup pods and backrest repo
	selector := config.LABEL_BACKREST_JOB + "!=true," + config.LABEL_BACKREST_RESTORE + "!=true," + config.LABEL_PGO_BACKREST_REPO + "!=true," + config.LABEL_NAME + "!=lspvc," + config.LABEL_PGBACKUP + "!=true," + config.LABEL_PGBACKUP + "!=false," + config.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name
	log.Debugf("selector for GetPods is %s", selector)

	pods, err := kubeapi.GetPods(apiserver.Clientset, selector, ns)
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

		d.Primary = false
		d.Type = getType(&p, cluster.Spec.Name)
		if d.Type == msgs.PodTypePrimary {
			d.Primary = true
		}
		output = append(output, d)

	}

	return output, err

}

func getServices(cluster *crv1.Pgcluster, ns string) ([]msgs.ShowClusterService, error) {

	output := make([]msgs.ShowClusterService, 0)
	selector := config.LABEL_PGO_BACKREST_REPO + "!=true," + config.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name

	services, err := kubeapi.GetServices(apiserver.Clientset, selector, ns)
	if err != nil {
		return output, err
	}

	log.Debugf("got %d services for %s", len(services.Items), cluster.Spec.Name)
	for _, p := range services.Items {
		d := msgs.ShowClusterService{}
		d.Name = p.Name
		if strings.Contains(p.Name, "-backrest-repo") {
			d.BackrestRepo = true
			d.ClusterName = cluster.Name
		} else if strings.Contains(p.Name, "-pgbouncer") {
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

func TestCluster(name, selector, ns string, allFlag bool) msgs.ClusterTestResponse {
	var err error

	response := msgs.ClusterTestResponse{}
	response.Results = make([]msgs.ClusterTestResult, 0)
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	log.Debugf("selector is %s", selector)
	if selector == "" && allFlag {
		log.Debug("selector is empty and --all is specified")
	} else {
		if selector == "" {
			selector = "name=" + name
		}
	}

	//get a list of matching clusters
	clusterList := crv1.PgclusterList{}
	err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient, &clusterList, selector, ns)

	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	//loop thru each cluster

	log.Debugf("clusters found len is %d", len(clusterList.Items))

	for _, c := range clusterList.Items {
		result := msgs.ClusterTestResult{}
		result.ClusterName = c.Name

		detail := msgs.ShowClusterDetail{}
		detail.Cluster = c

		pods, err := GetPrimaryAndReplicaPods(&c, ns)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		log.Debugf("pods found %d", len(pods))
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
		detail.Services, err = getServices(&c, ns)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		//get the secrets for this cluster
		secrets, err := apiserver.GetSecrets(&c, ns)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		result.Items = make([]msgs.ClusterTestDetail, 0)

		//for each service run a test and add results to output
		for _, service := range detail.Services {

			databases := make([]string, 0)
			if service.BackrestRepo {
				//dont include backrest repo service
			} else if service.Pgbouncer {
				databases = append(databases, service.ClusterName)
			} else {
				databases = append(databases, "postgres")
				databases = append(databases, c.Spec.Database)
			}
			for _, s := range secrets {
				for _, db := range databases {

					// skip postgres user for pgbouncer testing
					if s.Username == "postgres" && service.Pgbouncer {
						continue
					}

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
func CreateCluster(request *msgs.CreateClusterRequest, ns string) msgs.CreateClusterResponse {
	var id string
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
		found, err := kubeapi.Getpgcluster(apiserver.RESTClient, &result, clusterName, ns)
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
			found, err := validateCustomConfig(request.CustomConfig, ns)
			if !found {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = request.CustomConfig + " configmap was not found "
				return resp
			}
			if err != nil {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = err.Error()
				return resp
			}
			//add a label for the custom config
			userLabelsMap[config.LABEL_CUSTOM_CONFIG] = request.CustomConfig
		}
		//set the metrics flag with the global setting first
		userLabelsMap[config.LABEL_COLLECT] = strconv.FormatBool(apiserver.MetricsFlag)
		if err != nil {
			log.Error(err)
		}

		//if metrics is chosen on the pgo command, stick it into the user labels
		if request.MetricsFlag {
			userLabelsMap[config.LABEL_COLLECT] = "true"
		}
		if request.ServiceType != "" {
			if request.ServiceType != config.DEFAULT_SERVICE_TYPE && request.ServiceType != config.LOAD_BALANCER_SERVICE_TYPE && request.ServiceType != config.NODEPORT_SERVICE_TYPE {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = "error ServiceType should be either ClusterIP or LoadBalancer "

				return resp
			}
			userLabelsMap[config.LABEL_SERVICE_TYPE] = request.ServiceType
		}

		err = validateBackrestStorageType(request.BackrestStorageType, request.BackrestFlag)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		if request.BackrestStorageType != "" {
			log.Debug("using backrest storage type provided by user")
			userLabelsMap[config.LABEL_BACKREST_STORAGE_TYPE] = request.BackrestStorageType
		}

		//validate --pgpool-secret
		if request.PgpoolSecret != "" {
			var found bool
			_, found, err = kubeapi.GetSecret(apiserver.Clientset, request.PgpoolSecret, ns)
			if !found {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = "--pgpool-secret specified secret " + request.PgpoolSecret + " not found"
				return resp
			}
		}

		if request.PgpoolFlag {
			userLabelsMap[config.LABEL_PGPOOL] = "true"
			userLabelsMap[config.LABEL_PGPOOL_SECRET] = request.PgpoolSecret
			log.Debug("userLabelsMap")
			log.Debugf("%v", userLabelsMap)
			if request.ReplicaCount < 1 {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = "--pgpool requires replica-count be >= 1"
				return resp
			}
		}

		if request.PgbouncerFlag {
			// set flag at cluster level later
			// userLabelsMap[config.LABEL_PGBOUNCER] = "true"

			// need to create password to be added to postgres container and pgbouncer credential...
			if !(len(request.PgbouncerPass) > 0) {
				userLabelsMap[config.LABEL_PGBOUNCER_PASS] = util.GeneratePassword(10)
			} else {
				userLabelsMap[config.LABEL_PGBOUNCER_PASS] = request.PgbouncerPass
			}

			// default pgbouncer user to "pgbouncer" - request should be empty until configurable user is implemented.
			if !(len(request.PgbouncerUser) > 0) {
				userLabelsMap[config.LABEL_PGBOUNCER_USER] = "pgbouncer"
			} else {

				userLabelsMap[config.LABEL_PGBOUNCER_USER] = request.PgbouncerUser
			}

			userLabelsMap[config.LABEL_PGBOUNCER_SECRET] = request.PgbouncerSecret

		}

		log.Debug("userLabelsMap")
		log.Debugf("%v", userLabelsMap)

		if existsGlobalConfig(ns) {
			userLabelsMap[config.LABEL_CUSTOM_CONFIG] = config.GLOBAL_CUSTOM_CONFIGMAP
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
			userLabelsMap[config.LABEL_NODE_LABEL_KEY] = parts[0]
			userLabelsMap[config.LABEL_NODE_LABEL_VALUE] = parts[1]
			log.Debug("primary node labels used from pgo.yaml")
		}

		// validate & parse nodeLabel if exists
		if request.NodeLabel != "" {

			if err = apiserver.ValidateNodeLabel(request.NodeLabel); err != nil {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = err.Error()
				return resp
			}

			parts := strings.Split(request.NodeLabel, "=")
			userLabelsMap[config.LABEL_NODE_LABEL_KEY] = parts[0]
			userLabelsMap[config.LABEL_NODE_LABEL_VALUE] = parts[1]

			log.Debug("primary node labels used from user entered flag")
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
			err = validateSecretFrom(request.SecretFrom, ns)
			if err != nil {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = request.SecretFrom + " secret was not found "
				return resp
			}
		}

		// Create an instance of our CRD
		newInstance := getClusterParams(request, clusterName, userLabelsMap, ns)

		//verify that for autofail clusters we have a replica requested
		if newInstance.Spec.Replicas == "0" {
			if request.AutofailFlag || apiserver.Pgo.Cluster.Autofail {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = "replica-count of 1 or more is required when autofail is specified"
				return resp
			}
		}

		validateConfigPolicies(clusterName, request.Policies, ns)

		t := time.Now()
		newInstance.Spec.PswLastUpdate = t.Format(time.RFC3339)

		//create secrets
		err, newInstance.Spec.RootSecretName, newInstance.Spec.PrimarySecretName, newInstance.Spec.UserSecretName = createSecrets(request, clusterName, ns)
		if err != nil {
			resp.Results = append(resp.Results, err.Error())
			return resp
		}

		// Create Backrest secret for S3/SSH Keys:
		// We make this regardless if backrest is enabled or not because
		// the deployment template always tries to mount /sshd volume
		secretName := fmt.Sprintf("%s-%s", clusterName, config.LABEL_BACKREST_REPO_SECRET)
		_, _, err = kubeapi.GetSecret(apiserver.Clientset, secretName, request.Namespace)
		if kerrors.IsNotFound(err) {
			err := createBackrestRepoSecrets(clusterName, request.Namespace)
			if err != nil {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = fmt.Sprintf("could not create backrest repo secret: %s", err)
				return resp
			}
		} else if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = fmt.Sprintf("could not query if backrest repo secret exits: %s", err)
			return resp
		}

		//create CRD for new cluster
		err = kubeapi.Createpgcluster(apiserver.RESTClient,
			newInstance, ns)
		if err != nil {
			resp.Results = append(resp.Results, err.Error())
		} else {
			resp.Results = append(resp.Results, "created Pgcluster "+clusterName)
		}
		id, err = createWorkflowTask(clusterName, ns)
		if err != nil {
			log.Error(err)
			resp.Results = append(resp.Results, err.Error())
			return resp
		}
		resp.Results = append(resp.Results, "workflow id "+id)
	}

	return resp

}

func validateConfigPolicies(clusterName, PoliciesFlag, ns string) error {
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
			&result, v, ns)
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
	spec.Namespace = ns
	labels := make(map[string]string)
	labels[config.LABEL_PG_CLUSTER] = clusterName

	newInstance := &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:   spec.Name,
			Labels: labels,
		},
		Spec: spec,
	}

	kubeapi.Createpgtask(apiserver.RESTClient, newInstance, ns)

	return err
}

func getClusterParams(request *msgs.CreateClusterRequest, name string, userLabelsMap map[string]string, ns string) *crv1.Pgcluster {

	spec := crv1.PgclusterSpec{}

	if userLabelsMap[config.LABEL_CUSTOM_CONFIG] != "" {
		spec.CustomConfig = userLabelsMap[config.LABEL_CUSTOM_CONFIG]
	}

	if request.ContainerResources != "" {
		spec.ContainerResources, _ = apiserver.Pgo.GetContainerResource(request.ContainerResources)
	} else {
		log.Debugf("Pgo.DefaultContainerResources is %s", apiserver.Pgo.DefaultContainerResources)
		defaultContainerResource := apiserver.Pgo.DefaultContainerResources
		if defaultContainerResource != "" {
			spec.ContainerResources, _ = apiserver.Pgo.GetContainerResource(defaultContainerResource)
		}
	}

	spec.PrimaryStorage, _ = apiserver.Pgo.GetStorageSpec(apiserver.Pgo.PrimaryStorage)
	if request.StorageConfig != "" {
		spec.PrimaryStorage, _ = apiserver.Pgo.GetStorageSpec(request.StorageConfig)
	}

	spec.ReplicaStorage, _ = apiserver.Pgo.GetStorageSpec(apiserver.Pgo.ReplicaStorage)
	if request.ReplicaStorageConfig != "" {
		spec.ReplicaStorage, _ = apiserver.Pgo.GetStorageSpec(request.ReplicaStorageConfig)
	}

	spec.BackrestStorage, _ = apiserver.Pgo.GetStorageSpec(apiserver.Pgo.BackrestStorage)

	spec.CCPImageTag = apiserver.Pgo.Cluster.CCPImageTag
	if request.CCPImageTag != "" {
		spec.CCPImageTag = request.CCPImageTag
		log.Debugf("using CCPImageTag from command line %s", request.CCPImageTag)
	}

	if request.CCPImage != "" {
		spec.CCPImage = request.CCPImage
		log.Debugf("user is overriding CCPImage from command line %s", request.CCPImage)
	} else {
		spec.CCPImage = "crunchy-postgres"
	}
	spec.Namespace = ns
	spec.Name = name
	spec.ClusterName = name
	spec.Port = apiserver.Pgo.Cluster.Port
	spec.SecretFrom = ""
	spec.PrimaryHost = name
	if request.Policies == "" {
		spec.Policies = apiserver.Pgo.Cluster.Policies
		log.Debugf("Pgo.Cluster.Policies %s", apiserver.Pgo.Cluster.Policies)
	} else {
		spec.Policies = request.Policies
	}

	spec.User = "testuser"
	spec.Database = "userdb"
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
	spec.UserLabels[config.LABEL_PGO_VERSION] = msgs.PGO_VERSION

	//override any values from config file
	str = apiserver.Pgo.Cluster.Port
	log.Debugf("%s", apiserver.Pgo.Cluster.Port)
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
	log.Debugf("%s", apiserver.Pgo.Cluster.Strategy)
	if str != "" {
		spec.Strategy = str
	}
	//pass along command line flags for a restore
	if request.SecretFrom != "" {
		spec.SecretFrom = request.SecretFrom
	}

	spec.CustomConfig = request.CustomConfig

	labels := make(map[string]string)
	labels[config.LABEL_NAME] = name
	if request.AutofailFlag || apiserver.Pgo.Cluster.Autofail {
		labels[config.LABEL_AUTOFAIL] = "true"
	}

	//pgbadger - set with global flag first then check for a user flag
	labels[config.LABEL_BADGER] = strconv.FormatBool(apiserver.BadgerFlag)
	if request.BadgerFlag {
		labels[config.LABEL_BADGER] = "true"
	}

	//pgbackrest - set with user request first or look at global flag is not set
	// Note: validateBackrestStorageType called earlier in CreateCluster will generate 
	// and return error to user if BackrestFlag is not true or false
	if request.BackrestFlag == "true" || request.BackrestFlag == "false" {
		labels[config.LABEL_BACKREST] = request.BackrestFlag
		log.Debug("backrest set in user labels")
	} else {
		log.Debug("using Backrest from pgo.yaml")
		labels[config.LABEL_BACKREST] = strconv.FormatBool(apiserver.Pgo.Cluster.Backrest)
	}

	// pgbouncer
	if request.PgbouncerFlag {
		labels[config.LABEL_PGBOUNCER] = "true"
	}

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

func validateSecretFrom(secretname, ns string) error {
	var err error
	selector := config.LABEL_PG_CLUSTER + "=" + secretname
	secrets, err := kubeapi.GetSecrets(apiserver.Clientset, selector, ns)
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
func createDeleteDataTasks(clusterName string, storageSpec crv1.PgStorageSpec, deleteBackups bool, ns string) error {

	var err error

	//dont include pgpool, pgbouncer, or pgbackrest_repo deployments
	selector := config.LABEL_PG_CLUSTER + "=" + clusterName + "," + config.LABEL_PGBACKUP + "!=true," + config.LABEL_PGPOOL_POD + "!=true," + config.LABEL_PGO_BACKREST_REPO + "!=true," + config.LABEL_PGBOUNCER + "!=true"
	log.Debugf("selector for delete is %s", selector)
	deployments, err := kubeapi.GetDeployments(apiserver.Clientset, selector, ns)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Debugf("creatingDeleteDataTasks %d deployments for pg-cluster=%s\n", len(deployments.Items), clusterName)

	if deleteBackups {
		log.Debug("deleteBackups is called")
		log.Debug("check for backrest-shared-repo PVC to delete")
		pvcName := clusterName + "-pgbr-repo"
		_, found, err := kubeapi.GetPVC(apiserver.Clientset, pvcName, ns)
		if found {
			log.Debugf("creating rmdata job for %s", pvcName)
			dataRoots := []string{pvcName}
			err = apiserver.CreateRMDataTask(storageSpec, clusterName, pvcName, dataRoots, clusterName+"-rmdata-backrest-shared-repo", ns)
			if err != nil {
				log.Error(err)
				return err
			}
		}
	}

	for _, dep := range deployments.Items {

		dataRoots := make([]string, 0)
		dataRoots = append(dataRoots, dep.Name)

		claimName := dep.Name
		taskName := dep.Name + "-rmdata-pgdata"
		log.Debugf("creating taskName %s", taskName)

		err := apiserver.CreateRMDataTask(storageSpec, clusterName, claimName, dataRoots, taskName, ns)
		if err != nil {
			log.Error(err)
			return err
		}
	}

	if deleteBackups {
		log.Debug("check for backup PVC to delete")
		//get the deployment names for this cluster
		//by convention if basebackups are run, a pvc named
		//deploymentName-backup will be created to hold backups
		deps, err := kubeapi.GetDeployments(apiserver.Clientset, selector, ns)
		if err != nil {
			log.Error(err)
			return err
		}

		for _, dep := range deps.Items {
			pvcName := dep.Name + "-backup"
			log.Debugf("checking dep %s for backup pvc %s\n", dep.Name, pvcName)
			_, found, err := kubeapi.GetPVC(apiserver.Clientset, pvcName, ns)
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
				err = apiserver.CreateRMDataTask(storageSpec, clusterName, pvcName, dataRoots, dep.Name+"-rmdata-backups", ns)
				if err != nil {
					log.Error(err)
					return err
				}
			}

			// check for pgdump backups
			pgdumpPvcName := "backup-" + dep.Name + "-pgdump-pvc"

			log.Debugf("checking dep %s for pgdump backup pvc %s\n", dep.Name, pgdumpPvcName)
			_, found, err = kubeapi.GetPVC(apiserver.Clientset, pgdumpPvcName, ns)
			if !found {
				log.Debugf("%s pvc was not found when looking for pgdump backups to delete\n", pgdumpPvcName)
			} else {
				if err != nil {
					log.Error(err)
					return err
				}
				//by convention, the root directory name
				//created by the backup job is depName-backups
				dataRoots := []string{dep.Name + "-backups"}
				err = apiserver.CreateRMDataTask(storageSpec, clusterName, pgdumpPvcName, dataRoots, dep.Name+"-rmdata-pgdump", ns)
				if err != nil {
					log.Error(err)
					return err
				}
			}

		}
	}
	return err
}

func createWorkflowTask(clusterName, ns string) (string, error) {

	//create pgtask CRD
	spec := crv1.PgtaskSpec{}
	spec.Namespace = ns
	spec.Name = clusterName + "-" + crv1.PgtaskWorkflowCreateClusterType
	spec.TaskType = crv1.PgtaskWorkflow

	spec.Parameters = make(map[string]string)
	spec.Parameters[crv1.PgtaskWorkflowSubmittedStatus] = time.Now().Format("2006-01-02.15.04.05")
	spec.Parameters[config.LABEL_PG_CLUSTER] = clusterName

	u, err := ioutil.ReadFile("/proc/sys/kernel/random/uuid")
	if err != nil {
		log.Error(err)
		return "", err
	}
	spec.Parameters[crv1.PgtaskWorkflowID] = string(u[:len(u)-1])

	newInstance := &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: spec.Name,
		},
		Spec: spec,
	}
	newInstance.ObjectMeta.Labels = make(map[string]string)
	newInstance.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] = clusterName
	newInstance.ObjectMeta.Labels[crv1.PgtaskWorkflowID] = spec.Parameters[crv1.PgtaskWorkflowID]

	err = kubeapi.Createpgtask(apiserver.RESTClient, newInstance, ns)
	if err != nil {
		log.Error(err)
		return "", err
	}
	return spec.Parameters[crv1.PgtaskWorkflowID], err
}

func getType(pod *v1.Pod, clusterName string) string {

	//log.Debugf("%v\n", pod.ObjectMeta.Labels)
	if pod.ObjectMeta.Labels[config.LABEL_PGO_BACKREST_REPO] != "" {
		return msgs.PodTypePgbackrest
	} else if pod.ObjectMeta.Labels[config.LABEL_PGBOUNCER] != "" {
		return msgs.PodTypePgbouncer
	} else if pod.ObjectMeta.Labels[config.LABEL_PGPOOL_POD] != "" {
		return msgs.PodTypePgpool
	} else if pod.ObjectMeta.Labels[config.LABEL_PGBACKUP] == "true" {
		return msgs.PodTypeBackup
	} else if pod.ObjectMeta.Labels[config.LABEL_SERVICE_NAME] == clusterName {
		return msgs.PodTypePrimary
	} else {
		return msgs.PodTypeReplica
	}
	return msgs.PodTypeUnknown

}

func validateCustomConfig(configmapname, ns string) (bool, error) {
	var err error
	_, found := kubeapi.GetConfigMap(apiserver.Clientset, configmapname, ns)
	if !found {
		return found, err
	}

	return found, err
}

func existsGlobalConfig(ns string) bool {
	_, found := kubeapi.GetConfigMap(apiserver.Clientset, config.GLOBAL_CUSTOM_CONFIGMAP, ns)
	return found
}

func getReplicas(cluster *crv1.Pgcluster, ns string) ([]msgs.ShowClusterReplica, error) {

	output := make([]msgs.ShowClusterReplica, 0)
	replicaList := crv1.PgreplicaList{}

	selector := config.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name

	err := kubeapi.GetpgreplicasBySelector(apiserver.RESTClient,
		&replicaList, selector, ns)
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

// deleteDatabaseSecrets delete secrets that match pg-cluster=somecluster
func deleteDatabaseSecrets(db, ns string) error {
	var err error
	//get all that match pg-cluster=db
	selector := config.LABEL_PG_CLUSTER + "=" + db
	secrets, err := kubeapi.GetSecrets(apiserver.Clientset, selector, ns)
	if err != nil {
		log.Error(err)
		return err
	}

	for _, s := range secrets.Items {
		err := kubeapi.DeleteSecret(apiserver.Clientset, s.ObjectMeta.Name, ns)
		if err != nil {
			log.Error(err)
			return err
		}
	}
	return err
}

func deleteConfigMaps(clusterName, ns string) error {
	label := fmt.Sprintf("pg-cluster=%s", clusterName)
	list, ok := kubeapi.ListConfigMap(apiserver.Clientset, label, ns)
	if !ok {
		return fmt.Errorf("No configMaps found for selector: %s", label)
	}

	for _, configmap := range list.Items {
		err := kubeapi.DeleteConfigMap(apiserver.Clientset, configmap.Name, ns)
		if err != nil {
			return err
		}
	}
	return nil
}

func createSecrets(request *msgs.CreateClusterRequest, clusterName, ns string) (error, string, string, string) {

	var err error
	var RootPassword, Password, PrimaryPassword string
	var RootSecretName, PrimarySecretName, UserSecretName string

	//allows user to override with their own passwords
	if request.Password != "" {
		log.Debug("user has set a password, will use that instead of generated ones or the secret-from settings")
		RootPassword = request.Password
		Password = request.Password
		PrimaryPassword = request.Password
	}

	if request.SecretFrom != "" {
		log.Debugf("secret-from is specified! using %s", request.SecretFrom)
		_, RootPassword, err = util.GetPasswordFromSecret(apiserver.Clientset, ns, request.SecretFrom+crv1.RootSecretSuffix)
		_, Password, err = util.GetPasswordFromSecret(apiserver.Clientset, ns, request.SecretFrom+crv1.UserSecretSuffix)
		_, PrimaryPassword, err = util.GetPasswordFromSecret(apiserver.Clientset, ns, request.SecretFrom+crv1.PrimarySecretSuffix)
		if err != nil {
			log.Error("error getting secrets using SecretFrom " + request.SecretFrom)
			return err, RootSecretName, PrimarySecretName, UserSecretName
		}
	}

	RootSecretName = clusterName + crv1.RootSecretSuffix
	pgPassword := util.GeneratePassword(10)
	if RootPassword != "" {
		log.Debugf("using user specified password for secret %s", RootSecretName)
		pgPassword = RootPassword
	}

	PrimarySecretName = clusterName + crv1.PrimarySecretSuffix
	primaryPassword := util.GeneratePassword(10)
	if PrimaryPassword != "" {
		log.Debugf("using user specified password for secret %s", PrimarySecretName)
		primaryPassword = PrimaryPassword
	}

	UserSecretName = clusterName + crv1.UserSecretSuffix
	testPassword := util.GeneratePassword(10)
	if Password != "" {
		log.Debugf("using user specified password for secret %s", UserSecretName)
		testPassword = Password
	}

	var found bool
	_, found, err = kubeapi.GetSecret(apiserver.Clientset, RootSecretName, ns)
	if found {
		log.Debugf("not creating secrets %s since it already exists", RootSecretName)
		return err, RootSecretName, PrimarySecretName, UserSecretName

	}

	username := "postgres"
	err = util.CreateSecret(apiserver.Clientset, clusterName, RootSecretName, username, pgPassword, ns)
	if err != nil {
		log.Errorf("error creating secret %s %s", RootSecretName, err.Error())
		return err, RootSecretName, PrimarySecretName, UserSecretName
	}

	username = "primaryuser"
	err = util.CreateSecret(apiserver.Clientset, clusterName, PrimarySecretName, username, primaryPassword, ns)
	if err != nil {
		log.Errorf("error creating secret %s %s", PrimarySecretName, err.Error())
		return err, RootSecretName, PrimarySecretName, UserSecretName
	}

	username = "testuser"
	err = util.CreateSecret(apiserver.Clientset, clusterName, UserSecretName, username, testPassword, ns)
	if err != nil {
		log.Errorf("error creating secret %s %s", UserSecretName, err.Error())
		return err, RootSecretName, PrimarySecretName, UserSecretName
	}

	return err, RootSecretName, PrimarySecretName, UserSecretName
}

// UpdateCluster ...
func UpdateCluster(name, selector, autofail, ns string) msgs.UpdateClusterResponse {
	var err error

	response := msgs.UpdateClusterResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	response.Results = make([]string, 0)

	if name != "all" {
		if selector == "" {
			selector = "name=" + name
		}
	}
	log.Debugf("autofail is [%v]\n", autofail)

	clusterList := crv1.PgclusterList{}

	//get the clusters list
	err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient,
		&clusterList, selector, ns)
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

		//set autofail=true or false on each pgcluster CRD
		cluster.ObjectMeta.Labels[config.LABEL_AUTOFAIL] = autofail

		err = kubeapi.Updatepgcluster(apiserver.RESTClient,
			&cluster, cluster.Spec.Name, ns)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		} else {
			response.Results = append(response.Results, "updated pgcluster "+cluster.Spec.Name)
		}
	}

	return response

}

func GetPrimaryAndReplicaPods(cluster *crv1.Pgcluster, ns string) ([]msgs.ShowClusterPod, error) {

	output := make([]msgs.ShowClusterPod, 0)

	selector := config.LABEL_SERVICE_NAME + "=" + cluster.Spec.Name + "," + config.LABEL_DEPLOYMENT_NAME
	log.Debugf("selector for GetPrimaryAndReplicaPods is %s", selector)

	pods, err := kubeapi.GetPods(apiserver.Clientset, selector, ns)
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

		d.Primary = false
		d.Type = getType(&p, cluster.Spec.Name)
		if d.Type == msgs.PodTypePrimary {
			d.Primary = true
		}
		output = append(output, d)

	}
	selector = config.LABEL_SERVICE_NAME + "=" + cluster.Spec.Name + "-replica" + "," + config.LABEL_DEPLOYMENT_NAME
	log.Debugf("selector for GetPrimaryAndReplicaPods is %s", selector)

	pods, err = kubeapi.GetPods(apiserver.Clientset, selector, ns)
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

		d.Primary = false
		d.Type = getType(&p, cluster.Spec.Name)
		if d.Type == msgs.PodTypePrimary {
			d.Primary = true
		}
		output = append(output, d)

	}

	return output, err

}

func validateBackrestStorageType(requestBackRestStorageType string, backrestEnabledFlag string) error {

	var backrestEnabled bool
	var parseError error

	// backrestEnabled looks for override from command line, otherwise use pgo.yaml value
	if backrestEnabledFlag != "" {
		backrestEnabled, parseError = strconv.ParseBool(backrestEnabledFlag)
	} else {
		backrestEnabled = apiserver.Pgo.Cluster.Backrest
	}

	if parseError != nil {
		return errors.New("valid values for --pgbackrest are true or false")
	}

	if requestBackRestStorageType != "" && !backrestEnabled {
		return errors.New("pgBackRest storage type is only applicable if pgBackRest is enabled")
	} else if requestBackRestStorageType != "" && !apiserver.IsValidBackrestStorageType(requestBackRestStorageType) {
		return fmt.Errorf("Invalid value provided for pgBackRest storage type. The following values are allowed: %s",
			"\""+strings.Join(apiserver.GetBackrestStorageTypes(), "\", \"")+"\"")
	} else if strings.Contains(requestBackRestStorageType, "s3") && (apiserver.Pgo.Cluster.BackrestS3Bucket == "" ||
		apiserver.Pgo.Cluster.BackrestS3Endpoint == "" || apiserver.Pgo.Cluster.BackrestS3Region == "") {
		return errors.New("A configuration setting for AWS S3 storage is missing from pgo.yaml. Values must be provided for " +
			"'BackrestS3Bucket', 'BackrestS3Endpoint' and 'BackrestS3Region' in order to use the 's3' storage type with pgBackRest.")
	}

	return nil
}

func createBackrestRepoSecrets(clusterName, namespace string) error {
	keys, err := sshutil.NewPrivatePublicKeyPair(config.DEFAULT_BACKREST_SSH_KEY_BITS)
	if err != nil {
		return err
	}

	// Retrieve the S3/SSHD configuration files from secret
	configs, _, err := kubeapi.GetSecret(apiserver.Clientset, "pgo-backrest-repo-config", apiserver.PgoNamespace)
	if kerrors.IsNotFound(err) || err != nil {
		return err
	}

	secret := v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", clusterName, config.LABEL_BACKREST_REPO_SECRET),
			Labels: map[string]string{
				config.LABEL_PG_CLUSTER:        clusterName,
				config.LABEL_PGO_BACKREST_REPO: "true",
			},
		},
		Data: map[string][]byte{
			"authorized_keys":         keys.Public,
			"aws-s3-ca.crt":           configs.Data["aws-s3-ca.crt"],
			"aws-s3-credentials.yaml": configs.Data["aws-s3-credentials.yaml"],
			"config":                  configs.Data["config"],
			"id_rsa":                  keys.Private,
			"sshd_config":             configs.Data["sshd_config"],
			"ssh_host_rsa_key":        keys.Private,
		},
	}
	return kubeapi.CreateSecret(apiserver.Clientset, &secret, namespace)
}
