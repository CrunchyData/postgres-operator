package cmd

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
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"

	"github.com/spf13/viper"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"strconv"
	"strings"
	"time"
)

// showCluster ...
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
				if PostgresVersion == "" || (PostgresVersion != "" && cluster.Spec.CCPImageTag == PostgresVersion) {
					fmt.Println("cluster : " + cluster.Spec.Name + " (" + cluster.Spec.CCPImageTag + ")")
					log.Debug("listing cluster " + arg)
					log.Debugf("last password update %v\n", cluster.Spec.PswLastUpdate)
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

// listReplicaSets ....
func listReplicaSets(name string) {
	lo := meta_v1.ListOptions{LabelSelector: "pg-cluster=" + name}
	reps, err := Clientset.ReplicaSets(v1.NamespaceDefault).List(lo)
	if err != nil {
		log.Error("error getting list of replicasets" + err.Error())
		return
	}
	for _, r := range reps.Items {
		fmt.Println(TreeBranch + "replicaset : " + r.ObjectMeta.Name)
	}

}

// listDeployments ...
func listDeployments(name string) {
	lo := meta_v1.ListOptions{LabelSelector: "pg-cluster=" + name}
	deployments, err := Clientset.ExtensionsV1beta1().Deployments(v1.NamespaceDefault).List(lo)
	if err != nil {
		log.Error("error getting list of deployments" + err.Error())
		return
	}

	for _, d := range deployments.Items {
		fmt.Println(TreeBranch + "deployment : " + d.ObjectMeta.Name)
	}
	if len(deployments.Items) > 0 {
		printPolicies(&deployments.Items[0])
	}

}

// printPolicies ...
func printPolicies(d *v1beta1.Deployment) {
	labels := d.ObjectMeta.Labels
	for k, v := range labels {
		if v == "pgpolicy" {
			fmt.Printf("%spolicy: %s\n", TreeBranch, k)
		}
	}
}

// listPods ...
func listPods(name string) {
	lo := meta_v1.ListOptions{LabelSelector: "pg-cluster=" + name}
	pods, err := Clientset.CoreV1().Pods(v1.NamespaceDefault).List(lo)
	if err != nil {
		log.Error("error getting list of pods" + err.Error())
		return
	}
	for _, pod := range pods.Items {
		fmt.Println(TreeBranch + "pod : " + pod.ObjectMeta.Name + " (" + string(pod.Status.Phase) + " on " + pod.Spec.NodeName + ") (" + getReadyStatus(&pod) + ")")
		//fmt.Println(TreeTrunk + " phase : " + pod.Status.Phase)
	}

}

// listServices ...
func listServices(name string) {
	lo := meta_v1.ListOptions{LabelSelector: "pg-cluster=" + name}
	services, err := Clientset.CoreV1().Services(v1.NamespaceDefault).List(lo)
	if err != nil {
		log.Error("error getting list of services" + err.Error())
		return
	}
	for i, service := range services.Items {
		if i == len(services.Items)-1 {
			fmt.Println(TreeTrunk + "service : " + service.ObjectMeta.Name + " (" + service.Spec.ClusterIP + ")")
		} else {
			fmt.Println(TreeBranch + "service : " + service.ObjectMeta.Name + " (" + service.Spec.ClusterIP + ")")
		}
	}
}

// createCluster ...
func createCluster(args []string) {
	var err error

	//validate configuration
	if viper.GetString("PrimaryStorage.StorageTYPE") == "existing" {
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

			// Create an instance of our CRD
			newInstance := getClusterParams(clusterName)
			validateConfigPolicies()

			t := time.Now()
			newInstance.Spec.PswLastUpdate = t.Format(time.RFC3339)

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

// getClusterParams ...
func getClusterParams(name string) *crv1.Pgcluster {

	spec := crv1.PgclusterSpec{}
	primaryStorageSpec := crv1.PgStorageSpec{}
	spec.PrimaryStorage = primaryStorageSpec
	replicaStorageSpec := crv1.PgStorageSpec{}
	spec.ReplicaStorage = replicaStorageSpec
	spec.CCPImageTag = viper.GetString("Cluster.CCPImageTag")
	if CCPImageTag != "" {
		spec.CCPImageTag = CCPImageTag
		log.Debug("using ccp-image-tag from command line " + CCPImageTag)
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
	if PoliciesFlag == "" {
		spec.Policies = viper.GetString("Cluster.Policies")
	} else {
		spec.Policies = PoliciesFlag
	}
	spec.PrimaryPassword = viper.GetString("Cluster.PrimaryPassword")
	spec.User = "testuser"
	spec.Password = viper.GetString("Cluster.Password")
	spec.Database = "userdb"
	spec.RootPassword = viper.GetString("Cluster.RootPassword")
	spec.Replicas = "0"
	spec.Strategy = "1"
	spec.NodeName = NodeName
	spec.UserLabels = UserLabelsMap

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
	if SecretFrom != "" {
		spec.SecretFrom = SecretFrom
	}

	spec.BackupPath = BackupPath
	if BackupPVC != "" {
		spec.BackupPVCName = BackupPVC
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

// deleteCluster ....
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

// getReadyStatus ...
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

// getValidNodeName ...
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

// validateUserLabels ...
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

// validateNodeName ...
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

// validateSecretFrom ...
func validateSecretFrom(secretname string) error {
	var err error
	lo := meta_v1.ListOptions{LabelSelector: "pg-database=" + secretname}
	secrets, err := Clientset.Core().Secrets(v1.NamespaceDefault).List(lo)
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
