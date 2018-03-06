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
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	"github.com/crunchydata/postgres-operator/apiserver/pvcservice"
	"github.com/crunchydata/postgres-operator/util"
	"k8s.io/api/core/v1"

	"fmt"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation"
	"strconv"
	"strings"
	"time"
)

// DeleteCluster ...
func DeleteCluster(name, selector string, deleteData, deleteBackups bool) msgs.DeleteClusterResponse {
	var err error

	response := msgs.DeleteClusterResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	response.Results = make([]string, 0)

	myselector := labels.Everything()

	if name != "all" {
		if selector == "" {
			selector = "name=" + name
		}
		myselector, err = labels.Parse(selector)
		if err != nil {
			log.Error("could not parse selector value of " + selector + " " + err.Error())
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
	}
	log.Debugf("label selector is [%s]\n", myselector.String())
	log.Debugf("delete-data is [%v]\n", deleteData)
	log.Debugf("delete-backups is [%v]\n", deleteBackups)

	clusterList := crv1.PgclusterList{}

	//get the clusters list
	err = apiserver.RESTClient.Get().
		Resource(crv1.PgclusterResourcePlural).
		Namespace(apiserver.Namespace).
		Param("labelSelector", myselector.String()).
		//LabelsSelectorParam(myselector).
		Do().
		Into(&clusterList)
	if err != nil {
		log.Error("error getting cluster list" + err.Error())
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	if len(clusterList.Items) == 0 {
		log.Debug("no clusters found")
		response.Status.Code = msgs.Error
		response.Status.Msg = "no clusters found"
		return response
	}

	for _, cluster := range clusterList.Items {

		if deleteData {
			createDeleteDataTasks(cluster.Spec.Name, cluster.Spec.PrimaryStorage, deleteBackups)
		}

		err := apiserver.RESTClient.Delete().
			Resource(crv1.PgclusterResourcePlural).
			Namespace(apiserver.Namespace).
			Name(cluster.Spec.Name).
			Do().
			Error()

		if err != nil {
			log.Error("error deleting pgcluster " + cluster.Spec.Name + err.Error())
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		} else {
			log.Debug("deleted pgcluster " + cluster.Spec.Name)
			response.Results = append(response.Results, "deleted pgcluster "+cluster.Spec.Name)
		}
	}

	return response

}

// ShowCluster ...
func ShowCluster(name, selector string) msgs.ShowClusterResponse {
	var err error

	response := msgs.ShowClusterResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	response.Results = make([]msgs.ShowClusterDetail, 0)

	myselector := labels.Everything()
	log.Debug("selector is " + selector)
	if selector == "" && name == "all" {
	} else {
		if selector == "" {
			selector = "name=" + name
			myselector, err = labels.Parse(selector)
		} else {
			myselector, err = labels.Parse(selector)
		}
		if err != nil {
			log.Error("could not parse --selector value " + err.Error())
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
	}

	log.Debugf("label selector is [%s]\n", myselector.String())

	clusterList := crv1.PgclusterList{}

	//get a list of all clusters
	err = apiserver.RESTClient.Get().
		Resource(crv1.PgclusterResourcePlural).
		Namespace(apiserver.Namespace).
		Param("labelSelector", myselector.String()).
		//LabelsSelectorParam(myselector).
		Do().Into(&clusterList)
	if err != nil {
		log.Error("error getting list of clusters" + err.Error())
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	log.Debug("clusters found len is %d\n", len(clusterList.Items))

	for _, c := range clusterList.Items {
		detail := msgs.ShowClusterDetail{}
		detail.Cluster = c
		detail.Deployments, err = getDeployments(&c)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		detail.Pods, err = getPods(&c)
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
		detail.Secrets, err = getSecrets(&c)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		response.Results = append(response.Results, detail)
	}

	return response

}

func getDeployments(cluster *crv1.Pgcluster) ([]msgs.ShowClusterDeployment, error) {
	output := make([]msgs.ShowClusterDeployment, 0)

	lo := meta_v1.ListOptions{LabelSelector: "pg-cluster=" + cluster.Spec.Name}
	deployments, err := apiserver.Clientset.ExtensionsV1beta1().Deployments(apiserver.Namespace).List(lo)
	if err != nil {
		log.Error("error getting list of deployments" + err.Error())
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
func getPods(cluster *crv1.Pgcluster) ([]msgs.ShowClusterPod, error) {

	output := make([]msgs.ShowClusterPod, 0)
	lo := meta_v1.ListOptions{LabelSelector: "pg-cluster=" + cluster.Spec.Name}
	pods, err := apiserver.Clientset.CoreV1().Pods(apiserver.Namespace).List(lo)
	if err != nil {
		log.Error("error getting list of pods" + err.Error())
		return output, err
	}
	for _, p := range pods.Items {
		d := msgs.ShowClusterPod{}
		d.Name = p.Name
		d.Phase = string(p.Status.Phase)
		d.NodeName = p.Spec.NodeName
		d.ReadyStatus = getReadyStatus(&p)
		//log.Infof("pod details are %v\n", p)
		d.PVCName = getPVCName(&p)
		d.Primary = isPrimary(&p)
		output = append(output, d)

	}

	return output, err

}
func getServices(cluster *crv1.Pgcluster) ([]msgs.ShowClusterService, error) {

	output := make([]msgs.ShowClusterService, 0)
	lo := meta_v1.ListOptions{LabelSelector: "pg-cluster=" + cluster.Spec.Name}
	services, err := apiserver.Clientset.CoreV1().Services(apiserver.Namespace).List(lo)
	if err != nil {
		log.Error("error getting list of services" + err.Error())
		return output, err
	}

	for _, p := range services.Items {
		d := msgs.ShowClusterService{}
		d.Name = p.Name
		d.ClusterIP = p.Spec.ClusterIP
		output = append(output, d)

	}

	return output, err
}

func getSecrets(cluster *crv1.Pgcluster) ([]msgs.ShowClusterSecret, error) {

	output := make([]msgs.ShowClusterSecret, 0)
	lo := meta_v1.ListOptions{LabelSelector: "pg-database=" + cluster.Spec.Name}
	secrets, err := apiserver.Clientset.Core().Secrets(apiserver.Namespace).List(lo)
	if err != nil {
		log.Error("error getting list of secrets" + err.Error())
		return output, err
	}

	for _, s := range secrets.Items {
		d := msgs.ShowClusterSecret{}
		d.Name = s.Name
		d.Username = string(s.Data["username"][:])
		d.Password = string(s.Data["password"][:])
		output = append(output, d)

	}

	return output, err
}

func TestCluster(name, selector string) msgs.ClusterTestResponse {
	var err error

	response := msgs.ClusterTestResponse{}
	response.Results = make([]msgs.ClusterTestResult, 0)
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	myselector := labels.Everything()
	log.Debug("selector is " + selector)
	if selector == "" && name == "all" {
		log.Debug("selector is empty and name is all")
	} else {
		if selector == "" {
			selector = "name=" + name
			myselector, err = labels.Parse(selector)
		} else {
			myselector, err = labels.Parse(selector)
		}
		if err != nil {
			log.Error("could not parse --selector value " + err.Error())
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
	}

	log.Debugf("label selector is [%s]\n", myselector.String())

	//get a list of matching clusters
	clusterList := crv1.PgclusterList{}
	err = apiserver.RESTClient.Get().
		Resource(crv1.PgclusterResourcePlural).
		Namespace(apiserver.Namespace).
		Param("labelSelector", myselector.String()).
		Do().Into(&clusterList)

	if kerrors.IsNotFound(err) {
		log.Error("no clusters found")
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	if err != nil {
		log.Error("error getting cluster" + err.Error())
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	//loop thru each cluster

	log.Debug("clusters found len is %d\n", len(clusterList.Items))

	for _, c := range clusterList.Items {
		result := msgs.ClusterTestResult{}
		result.ClusterName = c.Name

		detail := msgs.ShowClusterDetail{}
		detail.Cluster = c

		//get the services for this cluster
		detail.Services, err = getServices(&c)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		//get the secrets for this cluster
		detail.Secrets, err = getSecrets(&c)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		result.Items = make([]msgs.ClusterTestDetail, 0)

		//for each service run a test and add results to output
		for _, service := range detail.Services {
			for _, s := range detail.Secrets {
				item := msgs.ClusterTestDetail{}
				username := s.Username
				password := s.Password
				database := "postgres"
				if username == c.Spec.User {
					database = c.Spec.Database
				}
				item.PsqlString = "psql -p " + c.Spec.Port + " -h " + service.ClusterIP + " -U " + username + " " + database
				log.Debug(item.PsqlString)
				status := query(username, service.ClusterIP, c.Spec.Port, database, password)
				item.Working = false
				if status {
					item.Working = true
				}
				result.Items = append(result.Items, item)
			}
		}
		response.Results = append(response.Results, result)
	}

	return response
}

func query(dbUser, dbHost, dbPort, database, dbPassword string) bool {
	var conn *sql.DB
	var err error

	conn, err = sql.Open("postgres", "sslmode=disable user="+dbUser+" host="+dbHost+" port="+dbPort+" dbname="+database+" password="+dbPassword)
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
		log.Debug("returned " + ts)
		return true
	}
	return false

}

// CreateCluster ...
// pgo create cluster mycluster
func CreateCluster(request *msgs.CreateClusterRequest) msgs.CreateClusterResponse {
	var err error
	resp := msgs.CreateClusterResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)
	clusterName := request.Name

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
		log.Debug("create cluster called for " + clusterName)
		result := crv1.Pgcluster{}

		// error if it already exists
		err = apiserver.RESTClient.Get().
			Resource(crv1.PgclusterResourcePlural).
			Namespace(apiserver.Namespace).
			Name(clusterName).
			Do().
			Into(&result)
		if err == nil {
			log.Debug("pgcluster " + clusterName + " was found so we will not create it")
			resp.Status.Msg = "pgcluster " + clusterName + " was found so we will not create it"
			return resp
		} else if kerrors.IsNotFound(err) {
			log.Debug("pgcluster " + clusterName + " not found so we will create it")
		} else {
			log.Error("error getting pgcluster " + clusterName + err.Error())
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "error getting pgcluster " + clusterName + err.Error()
			return resp
		}

		userLabelsMap := make(map[string]string)
		if request.UserLabels != "" {
			labels := strings.Split(request.UserLabels, ",")

			for _, v := range labels {
				//fmt.Printf("%s\n", v)
				p := strings.Split(v, "=")
				if len(p) < 2 {
					resp.Status.Code = msgs.Error
					resp.Status.Msg = "invalid labels format"
					return resp
				}
				userLabelsMap[p[0]] = p[1]
			}
		}

		//set the metrics flag with the global setting first
		userLabelsMap["crunchy-collect"] = strconv.FormatBool(apiserver.MetricsFlag)
		if err != nil {
			log.Error(err)
		}

		//if metrics is chosen on the pgo command, stick it into the user labels
		if request.MetricsFlag {
			userLabelsMap["crunchy-collect"] = "true"
		}

		if existsGlobalConfig() {
			userLabelsMap["custom-config"] = util.GLOBAL_CUSTOM_CONFIGMAP
		}

		if request.StorageConfig != "" {
			if apiserver.IsValidStorageName(request.StorageConfig) == false {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = request.StorageConfig + " Storage config was not found "
				return resp
			}
		}

		if request.NodeName != "" {
			valid, reason, allNodes := apiserver.IsValidNodeName(request.NodeName)
			if !valid {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = request.NodeName + " NodeName was not valid, " + reason + " valid nodes are " + allNodes
				return resp
			}
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

		if request.CustomConfig != "" {
			err = validateCustomConfig(request.CustomConfig)
			if err != nil {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = request.CustomConfig + " configmap was not found "
				return resp
			}
			//add a label for the custom config
			userLabelsMap["custom-config"] = request.CustomConfig
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

		if request.NodeName != "" {
			err = validateNodeName(request.NodeName)
			if err != nil {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = err.Error()
				return resp
			}
		}

		newInstance := getClusterParams(request, clusterName, userLabelsMap)
		validateConfigPolicies(request.Policies)

		t := time.Now()
		newInstance.Spec.PswLastUpdate = t.Format(time.RFC3339)
		err = apiserver.RESTClient.Post().
			Resource(crv1.PgclusterResourcePlural).
			Namespace(apiserver.Namespace).
			Body(newInstance).
			Do().Into(&result)
		if err != nil {
			log.Error(" in creating Pgcluster instance" + err.Error())
		}
		resp.Results = append(resp.Results, "created Pgcluster "+clusterName)
	}

	return resp

}

func validateConfigPolicies(PoliciesFlag string) error {
	var err error
	var configPolicies string
	if PoliciesFlag == "" {
		configPolicies = viper.GetString("Cluster.Policies")
	} else {
		configPolicies = PoliciesFlag
	}
	if configPolicies == "" {
		log.Debug("no policies are specified")
		err = errors.New("no policies are specified")
		return err
	}

	policies := strings.Split(configPolicies, ",")

	for _, v := range policies {
		result := crv1.Pgpolicy{}

		// error if it already exists
		err = apiserver.RESTClient.Get().
			Resource(crv1.PgpolicyResourcePlural).
			Namespace(apiserver.Namespace).
			Name(v).
			Do().
			Into(&result)
		if err == nil {
			log.Debug("policy " + v + " was found in catalog")
		} else if kerrors.IsNotFound(err) {
			log.Error("policy " + v + " specified in configuration was not found")
			return err
		} else {
			log.Error("error getting pgpolicy " + v + err.Error())
			return err
		}
	}

	return err
}

func getClusterParams(request *msgs.CreateClusterRequest, name string, userLabelsMap map[string]string) *crv1.Pgcluster {

	spec := crv1.PgclusterSpec{}

	if request.ContainerResources != "" {
		spec.ContainerResources = util.GetContainerResources(viper.Sub("ContainerResources." + request.ContainerResources))
	} else {
		defaultContainerResource := viper.GetString("DefaultContainerResource")
		if defaultContainerResource != "" {
			spec.ContainerResources = util.GetContainerResources(viper.Sub("ContainerResources." + defaultContainerResource))
		}
	}

	if request.StorageConfig != "" {
		spec.PrimaryStorage = util.GetStorageSpec(viper.Sub("Storage." + request.StorageConfig))
	} else {
		spec.PrimaryStorage = util.GetStorageSpec(viper.Sub("Storage." + viper.GetString("PrimaryStorage")))
	}

	if request.ReplicaStorageConfig != "" {
		spec.ReplicaStorage = util.GetStorageSpec(viper.Sub("Storage." + request.ReplicaStorageConfig))
	} else {
		spec.ReplicaStorage = util.GetStorageSpec(viper.Sub("Storage." + viper.GetString("ReplicaStorage")))
	}

	spec.CCPImageTag = viper.GetString("Cluster.CCPImageTag")
	if request.CCPImageTag != "" {
		spec.CCPImageTag = request.CCPImageTag
		log.Debug("using CCPImageTag from command line " + request.CCPImageTag)
	}

	spec.Name = name
	spec.ClusterName = name
	spec.Port = "5432"
	spec.SecretFrom = ""
	spec.BackupPath = ""
	spec.BackupPVCName = ""
	spec.PrimaryHost = name
	if request.Policies == "" {
		spec.Policies = viper.GetString("Cluster.Policies")
	} else {
		spec.Policies = request.Policies
	}

	spec.PrimaryPassword = ""
	spec.User = "testuser"
	spec.Password = ""
	spec.Database = "userdb"
	spec.RootPassword = ""
	spec.Replicas = "0"
	spec.Strategy = "1"
	spec.NodeName = request.NodeName
	spec.UserLabels = userLabelsMap

	//override any values from config file
	str := viper.GetString("Cluster.Port")
	if str != "" {
		spec.Port = str
	}
	str = viper.GetString("Cluster.User")
	if str != "" {
		spec.User = str
	}
	str = viper.GetString("Cluster.Database")
	if str != "" {
		spec.Database = str
	}
	str = viper.GetString("Cluster.Strategy")
	if str != "" {
		spec.Strategy = str
	}
	str = viper.GetString("Cluster.Replicas")
	if str != "" {
		spec.Replicas = str
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
	lo := meta_v1.ListOptions{LabelSelector: "pg-database=" + secretname}
	secrets, err := apiserver.Clientset.Core().Secrets(apiserver.Namespace).List(lo)
	if err != nil {
		log.Error("error getting list of secrets" + err.Error())
		return err
	}

	log.Debug("secrets for " + secretname)
	pgprimaryFound := false
	pgrootFound := false
	pguserFound := false

	for _, s := range secrets.Items {
		//fmt.Println("")
		//fmt.Println("secret : " + s.ObjectMeta.Name)
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

func validateNodeName(nodeName string) error {
	var err error
	lo := meta_v1.ListOptions{}
	nodes, err := apiserver.Clientset.CoreV1().Nodes().List(lo)
	if err != nil {
		panic(err.Error())
	}

	found := false
	allNodes := ""

	for _, node := range nodes.Items {
		if node.Name == nodeName {
			found = true
		}
		allNodes += node.Name + " "
	}

	if found == false {
		return errors.New("node name was not found...valid nodes include " + allNodes)
	}

	return err

}

func getReadyStatus(pod *v1.Pod) string {
	readyCount := 0
	containerCount := 0
	for _, stat := range pod.Status.ContainerStatuses {
		containerCount++
		if stat.Ready {
			readyCount++
		}
	}
	return fmt.Sprintf("%d/%d", readyCount, containerCount)

}

func createDeleteDataTasks(clusterName string, storageSpec crv1.PgStorageSpec, deleteBackups bool) {

	var err error

	log.Info("inside createDeleteDataTasks")

	//get the pods for this cluster
	var pods []msgs.ShowClusterPod
	spec := crv1.PgclusterSpec{}
	cluster := &crv1.Pgcluster{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: clusterName,
		},
		Spec: spec,
	}
	cluster.Spec.Name = clusterName
	pods, err = getPods(cluster)
	if err != nil {
		log.Error(err)
		return
	}
	log.Debug("got the cluster...")

	for _, element := range pods {
		//log.Debugf("the pod details ... %v\n", element)
		//get the pgdata pvc for each pod

		//create pgtask CRD
		spec := crv1.PgtaskSpec{}
		if element.Primary {
			spec.Name = clusterName
		} else {
			spec.Name = element.Name
		}
		spec.TaskType = crv1.PgtaskDeleteData
		spec.StorageSpec = storageSpec
		//spec.Status = crv1.PgtaskStateCreated
		spec.Parameters = element.PVCName

		newInstance := &crv1.Pgtask{
			ObjectMeta: meta_v1.ObjectMeta{
				Name: element.Name,
			},
			Spec: spec,
		}

		result := crv1.Pgtask{}
		err = apiserver.RESTClient.Post().
			Resource(crv1.PgtaskResourcePlural).
			Namespace(apiserver.Namespace).
			Body(newInstance).
			Do().Into(&result)

		if err != nil {
			log.Error(" in creating Pgtask instance" + err.Error())
			return
		}
		log.Debug("created Pgtask " + clusterName)
	}
	if deleteBackups {

		backupPVCName := clusterName + "-backup-pvc"
		//verify backup pvc exists
		_, err = pvcservice.ShowPVC(backupPVCName, "")
		if err != nil {
			log.Debug("not running rmdata for backups, " + backupPVCName + " not found")
			return
		}

		//proceed with backups removal
		spec := crv1.PgtaskSpec{}
		spec.Name = clusterName + "-backups"
		spec.TaskType = crv1.PgtaskDeleteData
		spec.StorageSpec = storageSpec

		spec.Parameters = backupPVCName

		newInstance := &crv1.Pgtask{
			ObjectMeta: meta_v1.ObjectMeta{
				Name: spec.Name,
			},
			Spec: spec,
		}
		log.Debug("deleting backups at " + backupPVCName)
		result := crv1.Pgtask{}
		err = apiserver.RESTClient.Post().
			Resource(crv1.PgtaskResourcePlural).
			Namespace(apiserver.Namespace).
			Body(newInstance).
			Do().Into(&result)

		if err != nil {
			log.Error(" in creating Pgtask instance" + err.Error())
			return
		}
		log.Debug("created Pgtask " + clusterName)
	}

}

func getPVCName(pod *v1.Pod) string {
	pvcName := "unknown"

	for _, v := range pod.Spec.Volumes {
		if v.Name == "pgdata" {
			pvcName = v.VolumeSource.PersistentVolumeClaim.ClaimName
			log.Infof("pod.Name %s pgdata %s\n", pod.Name, pvcName)
		}
	}

	return pvcName

}

func isPrimary(pod *v1.Pod) bool {

	log.Infof("%v\n", pod.ObjectMeta.Labels)
	//map[string]string
	if pod.ObjectMeta.Labels["primary"] == "true" {
		log.Infoln("this is a primary pod")
		return true
	}
	return false

}

func validateCustomConfig(configmapname string) error {
	var err error

	_, err = apiserver.Clientset.CoreV1().ConfigMaps(apiserver.Namespace).Get(configmapname, meta_v1.GetOptions{})
	if kerrors.IsNotFound(err) {
		return err
	}
	return err
}

func existsGlobalConfig() bool {
	_, err := apiserver.Clientset.CoreV1().ConfigMaps(apiserver.Namespace).Get(util.GLOBAL_CUSTOM_CONFIGMAP, meta_v1.GetOptions{})
	if kerrors.IsNotFound(err) {
		return false
	}
	return true
}
