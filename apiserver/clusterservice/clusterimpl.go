package clusterservice

/*
Copyright 2017 Crunchy Data Solutions, Inc.
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
	"k8s.io/api/core/v1"

	"fmt"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"strconv"
	"strings"
	"time"
)

// DeleteCluster ...
func DeleteCluster(namespace, name, selector string, deleteData, deleteBackups bool) msgs.DeleteClusterResponse {
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
		Namespace(namespace).
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
			createDeleteDataTasks(namespace, cluster.Spec.Name, cluster.Spec.PrimaryStorage, deleteBackups)
		}

		err := apiserver.RESTClient.Delete().
			Resource(crv1.PgclusterResourcePlural).
			Namespace(namespace).
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
func ShowCluster(namespace, name, selector string) msgs.ShowClusterResponse {
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
		Namespace(namespace).
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
		detail.Deployments, err = getDeployments(&c, namespace)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		detail.Pods, err = getPods(&c, namespace)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		detail.Services, err = getServices(&c, namespace)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		detail.Secrets, err = getSecrets(&c, namespace)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		response.Results = append(response.Results, detail)
	}

	return response

}

func getDeployments(cluster *crv1.Pgcluster, namespace string) ([]msgs.ShowClusterDeployment, error) {
	output := make([]msgs.ShowClusterDeployment, 0)

	lo := meta_v1.ListOptions{LabelSelector: "pg-cluster=" + cluster.Spec.Name}
	deployments, err := apiserver.Clientset.ExtensionsV1beta1().Deployments(namespace).List(lo)
	if err != nil {
		log.Error("error getting list of deployments" + err.Error())
		return output, err
	}

	for _, dep := range deployments.Items {
		d := msgs.ShowClusterDeployment{}
		d.Name = dep.Name
		d.PolicyLabels = make([]string, 0)

		labels := dep.ObjectMeta.Labels
		for k, v := range labels {
			if v == "pgpolicy" {
				d.PolicyLabels = append(d.PolicyLabels, k)
			}
		}
		output = append(output, d)

	}

	return output, err
}
func getPods(cluster *crv1.Pgcluster, namespace string) ([]msgs.ShowClusterPod, error) {

	output := make([]msgs.ShowClusterPod, 0)
	lo := meta_v1.ListOptions{LabelSelector: "pg-cluster=" + cluster.Spec.Name}
	pods, err := apiserver.Clientset.CoreV1().Pods(namespace).List(lo)
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
func getServices(cluster *crv1.Pgcluster, namespace string) ([]msgs.ShowClusterService, error) {

	output := make([]msgs.ShowClusterService, 0)
	lo := meta_v1.ListOptions{LabelSelector: "pg-cluster=" + cluster.Spec.Name}
	services, err := apiserver.Clientset.CoreV1().Services(namespace).List(lo)
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

func getSecrets(cluster *crv1.Pgcluster, namespace string) ([]msgs.ShowClusterSecret, error) {

	output := make([]msgs.ShowClusterSecret, 0)
	lo := meta_v1.ListOptions{LabelSelector: "pg-database=" + cluster.Spec.Name}
	secrets, err := apiserver.Clientset.Core().Secrets(namespace).List(lo)
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

func TestCluster(namespace, name string) msgs.ClusterTestResponse {
	var err error

	response := msgs.ClusterTestResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	cluster := crv1.Pgcluster{}
	err = apiserver.RESTClient.Get().
		Resource(crv1.PgclusterResourcePlural).
		Namespace(namespace).
		Name(name).
		Do().Into(&cluster)

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

	lo := meta_v1.ListOptions{LabelSelector: "pg-cluster=" + cluster.Spec.Name}
	services, err := apiserver.Clientset.CoreV1().Services(namespace).List(lo)
	if err != nil {
		log.Error("error getting list of services" + err.Error())
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	lo = meta_v1.ListOptions{LabelSelector: "pg-database=" + cluster.Spec.Name}
	secrets, err := apiserver.Clientset.Core().Secrets(namespace).List(lo)
	if err != nil {
		log.Error("error getting list of secrets" + err.Error())
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	response.Items = make([]msgs.ClusterTestDetail, 0)

	for _, service := range services.Items {
		for _, s := range secrets.Items {
			item := msgs.ClusterTestDetail{}
			username := string(s.Data["username"][:])
			password := string(s.Data["password"][:])
			database := "postgres"
			if username == cluster.Spec.User {
				database = cluster.Spec.Database
			}
			item.PsqlString = "psql -p " + cluster.Spec.Port + " -h " + service.Spec.ClusterIP + " -U " + username + " " + database
			log.Debug(item.PsqlString)
			status := query(username, service.Spec.ClusterIP, cluster.Spec.Port, database, password)
			item.Working = false
			if status {
				item.Working = true
			}
			response.Items = append(response.Items, item)
		}
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

	for i := 0; i < request.Series; i++ {
		if request.Series > 1 {
			clusterName = request.Name + strconv.Itoa(i)
		}
		log.Debug("create cluster called for " + clusterName)
		result := crv1.Pgcluster{}

		// error if it already exists
		err = apiserver.RESTClient.Get().
			Resource(crv1.PgclusterResourcePlural).
			Namespace(request.Namespace).
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
		if request.SecretFrom != "" {
			err = validateSecretFrom(request.SecretFrom, request.Namespace)
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
		validateConfigPolicies(request.Policies, request.Namespace)

		t := time.Now()
		newInstance.Spec.PswLastUpdate = t.Format(time.RFC3339)
		err = apiserver.RESTClient.Post().
			Resource(crv1.PgclusterResourcePlural).
			Namespace(request.Namespace).
			Body(newInstance).
			Do().Into(&result)
		if err != nil {
			log.Error(" in creating Pgcluster instance" + err.Error())
		}
		resp.Results = append(resp.Results, "created Pgcluster "+clusterName)
	}

	return resp

}

func validateConfigPolicies(PoliciesFlag, namespace string) error {
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
			Namespace(namespace).
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
	primaryStorageSpec := crv1.PgStorageSpec{}
	spec.PrimaryStorage = primaryStorageSpec
	replicaStorageSpec := crv1.PgStorageSpec{}
	spec.ReplicaStorage = replicaStorageSpec
	spec.CCPImageTag = viper.GetString("Cluster.CCPImageTag")
	if request.CCPImageTag != "" {
		spec.CCPImageTag = request.CCPImageTag
		log.Debug("using CCPImageTag from command line " + request.CCPImageTag)
	}

	spec.PrimaryStorage.Name = viper.GetString("PrimaryStorage.Name")
	spec.PrimaryStorage.StorageClass = viper.GetString("PrimaryStorage.StorageClass")
	spec.PrimaryStorage.AccessMode = viper.GetString("PrimaryStorage.AccessMode")
	spec.PrimaryStorage.Size = viper.GetString("PrimaryStorage.Size")
	spec.PrimaryStorage.StorageType = viper.GetString("PrimaryStorage.StorageType")
	spec.PrimaryStorage.Fsgroup = viper.GetString("PrimaryStorage.Fsgroup")
	spec.PrimaryStorage.SupplementalGroups = viper.GetString("PrimaryStorage.SupplementalGroups")

	spec.ReplicaStorage.Name = viper.GetString("ReplicaStorage.Name")
	spec.ReplicaStorage.StorageClass = viper.GetString("ReplicaStorage.StorageClass")
	spec.ReplicaStorage.AccessMode = viper.GetString("ReplicaStorage.AccessMode")
	spec.ReplicaStorage.Size = viper.GetString("ReplicaStorage.Size")
	spec.ReplicaStorage.StorageType = viper.GetString("ReplicaStorage.StorageType")
	spec.ReplicaStorage.Fsgroup = viper.GetString("ReplicaStorage.Fsgroup")
	spec.ReplicaStorage.SupplementalGroups = viper.GetString("ReplicaStorage.SupplementalGroups")

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
	spec.PrimaryPassword = viper.GetString("Cluster.PrimaryPassword")
	spec.User = "testuser"
	spec.Password = viper.GetString("Cluster.Password")
	spec.Database = "userdb"
	spec.RootPassword = viper.GetString("Cluster.RootPassword")
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

func validateSecretFrom(secretname, namespace string) error {
	var err error
	lo := meta_v1.ListOptions{LabelSelector: "pg-database=" + secretname}
	secrets, err := apiserver.Clientset.Core().Secrets(namespace).List(lo)
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

func createDeleteDataTasks(namespace, clusterName string, storageSpec crv1.PgStorageSpec, deleteBackups bool) {

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
	pods, err = getPods(cluster, namespace)
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
			Namespace(namespace).
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
		_, err = pvcservice.ShowPVC(namespace, backupPVCName, "")
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
			Namespace(namespace).
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
