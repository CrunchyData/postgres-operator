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
package cmd

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/kraken/apis/cr/v1"

	"github.com/spf13/viper"
	//"k8s.io/api/core/v1"
	"k8s.io/client-go/pkg/api/v1"
	//"k8s.io/api/extensions/v1beta1"

	"k8s.io/client-go/pkg/apis/extensions/v1beta1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"strconv"
	"strings"
	"time"
)

func showCluster(args []string) {
	var err error
	//get a list of all clusters
	clusterList := crv1.PgclusterList{}
	myselector := labels.Everything()
	log.Debug("selector is " + Labelselector)
	if Labelselector != "" {
		args = make([]string, 1)
		args[0] = "all"
		myselector, err = labels.Parse(Labelselector)
		if err != nil {
			log.Error("could not parse --selector value " + err.Error())
			return
		}
	}

	log.Debugf("label selector is [%v]\n", myselector)
	err = RestClient.Get().
		Resource(crv1.PgclusterResourcePlural).
		Namespace(v1.NamespaceDefault).
		LabelsSelectorParam(myselector).
		Do().
		Into(&clusterList)
	if err != nil {
		log.Error("error getting list of clusters" + err.Error())
		return
	}

	if len(clusterList.Items) == 0 {
		fmt.Println("no clusters found")
		return
	}

	itemFound := false

	//each arg represents a cluster name or the special 'all' value
	for _, arg := range args {
		for _, cluster := range clusterList.Items {
			//fmt.Println("")
			if arg == "all" || cluster.Spec.Name == arg {
				itemFound = true
				if PostgresVersion == "" || (PostgresVersion != "" && cluster.Spec.POSTGRES_FULL_VERSION == PostgresVersion) {
					fmt.Println("cluster : " + cluster.Spec.Name + " (" + cluster.Spec.POSTGRES_FULL_VERSION + ")")
					log.Debug("listing cluster " + arg)
					log.Debugf("last password update %v\n", cluster.Spec.PSW_LAST_UPDATE)
					//list the deployments
					listDeployments(cluster.Spec.Name)
					//list the replicasets
					listReplicaSets(cluster.Spec.Name)
					//list the pods
					listPods(cluster.Spec.Name)
					//list the services
					listServices(cluster.Spec.Name)
					if ShowSecrets {
						PrintSecrets(cluster.Spec.Name)
					}
					fmt.Println("")
				}
			}
		}
		if !itemFound {
			fmt.Println(arg + " was not found")
		}
		itemFound = false
	}

}

func listReplicaSets(name string) {
	lo := meta_v1.ListOptions{LabelSelector: "pg-cluster=" + name}
	reps, err := Clientset.ReplicaSets(v1.NamespaceDefault).List(lo)
	if err != nil {
		log.Error("error getting list of replicasets" + err.Error())
		return
	}
	for _, r := range reps.Items {
		fmt.Println(TREE_BRANCH + "replicaset : " + r.ObjectMeta.Name)
	}

}
func listDeployments(name string) {
	lo := meta_v1.ListOptions{LabelSelector: "pg-cluster=" + name}
	deployments, err := Clientset.ExtensionsV1beta1().Deployments(v1.NamespaceDefault).List(lo)
	if err != nil {
		log.Error("error getting list of deployments" + err.Error())
		return
	}

	for _, d := range deployments.Items {
		fmt.Println(TREE_BRANCH + "deployment : " + d.ObjectMeta.Name)
	}
	if len(deployments.Items) > 0 {
		printPolicies(&deployments.Items[0])
	}

}

func printPolicies(d *v1beta1.Deployment) {
	labels := d.ObjectMeta.Labels
	for k, v := range labels {
		if v == "pgpolicy" {
			fmt.Printf("%spolicy: %s\n", TREE_BRANCH, k)
		}
	}
}

func listPods(name string) {
	lo := meta_v1.ListOptions{LabelSelector: "pg-cluster=" + name}
	pods, err := Clientset.CoreV1().Pods(v1.NamespaceDefault).List(lo)
	if err != nil {
		log.Error("error getting list of pods" + err.Error())
		return
	}
	for _, pod := range pods.Items {
		fmt.Println(TREE_BRANCH + "pod : " + pod.ObjectMeta.Name + " (" + string(pod.Status.Phase) + " on " + pod.Spec.NodeName + ") (" + getReadyStatus(&pod) + ")")
		//fmt.Println(TREE_TRUNK + " phase : " + pod.Status.Phase)
	}

}
func listServices(name string) {
	lo := meta_v1.ListOptions{LabelSelector: "pg-cluster=" + name}
	services, err := Clientset.CoreV1().Services(v1.NamespaceDefault).List(lo)
	if err != nil {
		log.Error("error getting list of services" + err.Error())
		return
	}
	for i, service := range services.Items {
		if i == len(services.Items)-1 {
			fmt.Println(TREE_TRUNK + "service : " + service.ObjectMeta.Name + " (" + service.Spec.ClusterIP + ")")
		} else {
			fmt.Println(TREE_BRANCH + "service : " + service.ObjectMeta.Name + " (" + service.Spec.ClusterIP + ")")
		}
	}
}

func createCluster(args []string) {
	var err error

	//validate configuration
	if viper.GetString("MASTER_STORAGE.STORAGE_TYPE") == "existing" {
		if BackupPVC != "" {
			log.Error("storage type of existing not allowed when doing a restore")
			return
		}
	}

	for _, arg := range args {
		clusterName := arg
		for i := 0; i < Series; i++ {
			if Series > 1 {
				clusterName = arg + strconv.Itoa(i)
			}
			log.Debug("create cluster called for " + clusterName)
			result := crv1.Pgcluster{}

			// error if it already exists
			err = RestClient.Get().
				Resource(crv1.PgclusterResourcePlural).
				Namespace(v1.NamespaceDefault).
				Name(clusterName).
				Do().
				Into(&result)
			if err == nil {
				log.Debug("pgcluster " + clusterName + " was found so we will not create it")
				break
			} else if kerrors.IsNotFound(err) {
				log.Debug("pgcluster " + clusterName + " not found so we will create it")
			} else {
				log.Error("error getting pgcluster " + clusterName + err.Error())
				break
			}

			if SecretFrom != "" {
				err = validateSecretFrom(SecretFrom)
				if err != nil {
					log.Error(SecretFrom + " secret was not found ")
					return
				}
			}

			// Create an instance of our TPR
			newInstance := getClusterParams(clusterName)
			validateConfigPolicies()

			t := time.Now()
			newInstance.Spec.PSW_LAST_UPDATE = t.Format(time.RFC3339)

			err = RestClient.Post().
				Resource(crv1.PgclusterResourcePlural).
				Namespace(v1.NamespaceDefault).
				Body(newInstance).
				Do().Into(&result)
			if err != nil {
				log.Error(" in creating Pgcluster instance" + err.Error())
			}
			fmt.Println("created Pgcluster " + clusterName)
		}

	}
}

func getClusterParams(name string) *crv1.Pgcluster {

	spec := crv1.PgclusterSpec{}
	masterStorageSpec := crv1.PgStorageSpec{}
	spec.MasterStorage = masterStorageSpec
	replicaStorageSpec := crv1.PgStorageSpec{}
	spec.ReplicaStorage = replicaStorageSpec
	spec.CCP_IMAGE_TAG = viper.GetString("CLUSTER.CCP_IMAGE_TAG")
	if CCP_IMAGE_TAG != "" {
		spec.CCP_IMAGE_TAG = CCP_IMAGE_TAG
		log.Debug("using CCP_IMAGE_TAG from command line " + CCP_IMAGE_TAG)
	}

	spec.MasterStorage.PvcName = viper.GetString("MASTER_STORAGE.PVC_NAME")
	spec.MasterStorage.StorageClass = viper.GetString("MASTER_STORAGE.STORAGE_CLASS")
	spec.MasterStorage.PvcAccessMode = viper.GetString("MASTER_STORAGE.PVC_ACCESS_MODE")
	spec.MasterStorage.PvcSize = viper.GetString("MASTER_STORAGE.PVC_SIZE")
	spec.MasterStorage.StorageType = viper.GetString("MASTER_STORAGE.STORAGE_TYPE")
	spec.MasterStorage.FSGROUP = viper.GetString("MASTER_STORAGE.FSGROUP")
	spec.MasterStorage.SUPPLEMENTAL_GROUPS = viper.GetString("MASTER_STORAGE.SUPPLEMENTAL_GROUPS")

	spec.ReplicaStorage.PvcName = viper.GetString("REPLICA_STORAGE.PVC_NAME")
	spec.ReplicaStorage.StorageClass = viper.GetString("REPLICA_STORAGE.STORAGE_CLASS")
	spec.ReplicaStorage.PvcAccessMode = viper.GetString("REPLICA_STORAGE.PVC_ACCESS_MODE")
	spec.ReplicaStorage.PvcSize = viper.GetString("REPLICA_STORAGE.PVC_SIZE")
	spec.ReplicaStorage.StorageType = viper.GetString("REPLICA_STORAGE.STORAGE_TYPE")
	spec.ReplicaStorage.FSGROUP = viper.GetString("REPLICA_STORAGE.FSGROUP")
	spec.ReplicaStorage.SUPPLEMENTAL_GROUPS = viper.GetString("REPLICA_STORAGE.SUPPLEMENTAL_GROUPS")

	spec.Name = name
	spec.ClusterName = name
	spec.Port = "5432"
	spec.SECRET_FROM = ""
	spec.BACKUP_PATH = ""
	spec.BACKUP_PVC_NAME = ""
	spec.PG_MASTER_HOST = name
	spec.PG_MASTER_USER = "master"
	if PoliciesFlag == "" {
		spec.Policies = viper.GetString("CLUSTER.POLICIES")
	} else {
		spec.Policies = PoliciesFlag
	}
	spec.PG_MASTER_PASSWORD = viper.GetString("CLUSTER.PG_MASTER_PASSWORD")
	spec.PG_USER = "testuser"
	spec.PG_PASSWORD = viper.GetString("CLUSTER.PG_PASSWORD")
	spec.PG_DATABASE = "userdb"
	spec.PG_ROOT_PASSWORD = viper.GetString("CLUSTER.PG_ROOT_PASSWORD")
	spec.REPLICAS = "0"
	spec.STRATEGY = "1"
	spec.NodeName = NodeName
	spec.UserLabels = UserLabelsMap

	//override any values from config file
	str := viper.GetString("CLUSTER.PORT")
	if str != "" {
		spec.Port = str
	}
	str = viper.GetString("CLUSTER.PG_MASTER_USER")
	if str != "" {
		spec.PG_MASTER_USER = str
	}
	str = viper.GetString("CLUSTER.PG_USER")
	if str != "" {
		spec.PG_USER = str
	}
	str = viper.GetString("CLUSTER.PG_DATABASE")
	if str != "" {
		spec.PG_DATABASE = str
	}
	str = viper.GetString("CLUSTER.STRATEGY")
	if str != "" {
		spec.STRATEGY = str
	}
	str = viper.GetString("CLUSTER.REPLICAS")
	if str != "" {
		spec.REPLICAS = str
	}

	//pass along command line flags for a restore
	if SecretFrom != "" {
		spec.SECRET_FROM = SecretFrom
	}

	spec.BACKUP_PATH = BackupPath
	if BackupPVC != "" {
		spec.BACKUP_PVC_NAME = BackupPVC
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

func deleteCluster(args []string) {

	var err error

	// Fetch a list of our cluster TPRs
	clusterList := crv1.PgclusterList{}
	myselector := labels.Everything()

	if Selector != "" {
		//use the selector instead of an argument list to filter on
		myselector, err = labels.Parse(Selector)
		if err != nil {
			log.Error("could not parse selector flag")
			return
		}
	}

	//get the clusters list
	err = RestClient.Get().
		Resource(crv1.PgclusterResourcePlural).
		Namespace(v1.NamespaceDefault).
		LabelsSelectorParam(myselector).
		Do().
		Into(&clusterList)
	if err != nil {
		log.Error("error getting cluster list" + err.Error())
		return
	}

	if len(clusterList.Items) == 0 {
		log.Debug("no clusters found")
	} else {
		if Selector != "" {
			newargs := make([]string, 0)
			for _, cluster := range clusterList.Items {
				newargs = append(newargs, cluster.Spec.Name)
			}
			args = newargs
		}
	}

	//to remove a cluster, you just have to remove
	//the pgcluster object, the operator will do the actual deletes
	for _, arg := range args {
		clusterFound := false
		log.Debug("deleting cluster " + arg)
		for _, cluster := range clusterList.Items {
			if arg == "all" || arg == cluster.Spec.Name {
				clusterFound = true
				err := RestClient.Delete().
					Resource(crv1.PgclusterResourcePlural).
					Namespace(v1.NamespaceDefault).
					Name(arg).
					Do().
					Error()
				if err != nil {
					log.Error("error deleting pgcluster " + arg + err.Error())
				} else {
					fmt.Println("deleted pgcluster " + cluster.Spec.Name)
				}

			}
		}
		if !clusterFound {
			fmt.Println("cluster " + arg + " not found")
		}
	}
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

func getValidNodeName() string {
	var err error
	nodes, err := Clientset.CoreV1().Nodes().List(meta_v1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	//TODO randomize the node selection
	for _, node := range nodes.Items {
		return node.Name
	}

	panic("no nodes found")

	return "error here"

}
func validateUserLabels() error {

	var err error
	labels := strings.Split(UserLabels, ",")

	for _, v := range labels {
		fmt.Printf("%s\n", v)
		p := strings.Split(v, "=")
		if len(p) < 2 {
			return errors.New("invalid labels format")
		} else {
			UserLabelsMap[p[0]] = p[1]
		}
	}
	return err

}

func validateNodeName(nodeName string) error {
	var err error
	lo := meta_v1.ListOptions{}
	nodes, err := Clientset.CoreV1().Nodes().List(lo)
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

func validateSecretFrom(secretname string) error {
	var err error
	lo := meta_v1.ListOptions{LabelSelector: "pg-database=" + secretname}
	secrets, err := Clientset.Core().Secrets(v1.NamespaceDefault).List(lo)
	if err != nil {
		log.Error("error getting list of secrets" + err.Error())
		return err
	}

	log.Debug("secrets for " + secretname)
	pgmasterFound := false
	pgrootFound := false
	pguserFound := false

	for _, s := range secrets.Items {
		//fmt.Println("")
		//fmt.Println("secret : " + s.ObjectMeta.Name)
		if s.ObjectMeta.Name == secretname+crv1.PGMASTER_SECRET_SUFFIX {
			pgmasterFound = true
		} else if s.ObjectMeta.Name == secretname+crv1.PGROOT_SECRET_SUFFIX {
			pgrootFound = true
		} else if s.ObjectMeta.Name == secretname+crv1.PGUSER_SECRET_SUFFIX {
			pguserFound = true
		}
	}
	if !pgmasterFound {
		return errors.New(secretname + crv1.PGMASTER_SECRET_SUFFIX + " not found")
	}
	if !pgrootFound {
		return errors.New(secretname + crv1.PGROOT_SECRET_SUFFIX + " not found")
	}
	if !pguserFound {
		return errors.New(secretname + crv1.PGUSER_SECRET_SUFFIX + " not found")
	}

	return err
}
