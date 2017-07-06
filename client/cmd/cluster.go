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
	"github.com/crunchydata/postgres-operator/tpr"
	"github.com/spf13/viper"
	"k8s.io/client-go/pkg/api"
	kerrors "k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/labels"

	"k8s.io/client-go/pkg/api/v1"
)

func showCluster(args []string) {
	var err error
	//get a list of all clusters
	clusterList := tpr.PgClusterList{}
	myselector := labels.Everything()
	log.Info("selector is " + Labelselector)
	if Labelselector != "" {
		myselector, err = labels.Parse(Labelselector)
		if err != nil {
			log.Error("could not parse --selector value " + err.Error())
			return
		}
	}
	err = Tprclient.Get().
		Resource(tpr.CLUSTER_RESOURCE).
		Namespace(Namespace).
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
			fmt.Println("")
			if arg == "all" || cluster.Spec.Name == arg {
				itemFound = true
				if PostgresVersion == "" || (PostgresVersion != "" && cluster.Spec.POSTGRES_FULL_VERSION == PostgresVersion) {
					fmt.Println("cluster : " + cluster.Spec.Name + " (" + cluster.Spec.POSTGRES_FULL_VERSION + ")")
					log.Debug("listing cluster " + arg)
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
	lo := v1.ListOptions{LabelSelector: "pg-cluster=" + name}
	reps, err := Clientset.ReplicaSets(Namespace).List(lo)
	if err != nil {
		log.Error("error getting list of replicasets" + err.Error())
		return
	}
	for _, r := range reps.Items {
		fmt.Println(TREE_BRANCH + "replicaset : " + r.ObjectMeta.Name)
	}

}
func listDeployments(name string) {
	lo := v1.ListOptions{LabelSelector: "pg-cluster=" + name}
	deployments, err := Clientset.Deployments(Namespace).List(lo)
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
	lo := v1.ListOptions{LabelSelector: "pg-cluster=" + name}
	pods, err := Clientset.Core().Pods(Namespace).List(lo)
	if err != nil {
		log.Error("error getting list of pods" + err.Error())
		return
	}
	for _, pod := range pods.Items {
		fmt.Println(TREE_BRANCH + "pod : " + pod.ObjectMeta.Name + " (" + string(pod.Status.Phase) + ") (" + getReadyStatus(&pod) + ")")
		//fmt.Println(TREE_TRUNK + " phase : " + pod.Status.Phase)
	}

}
func listServices(name string) {
	lo := v1.ListOptions{LabelSelector: "pg-cluster=" + name}
	services, err := Clientset.Core().Services(Namespace).List(lo)
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

	for _, arg := range args {
		log.Debug("create cluster called for " + arg)
		result := tpr.PgCluster{}

		// error if it already exists
		err = Tprclient.Get().
			Resource(tpr.CLUSTER_RESOURCE).
			Namespace(Namespace).
			Name(arg).
			Do().
			Into(&result)
		if err == nil {
			log.Debug("pgcluster " + arg + " was found so we will not create it")
			break
		} else if kerrors.IsNotFound(err) {
			log.Debug("pgcluster " + arg + " not found so we will create it")
		} else {
			log.Error("error getting pgcluster " + arg + err.Error())
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
		newInstance := getClusterParams(arg)

		err = Tprclient.Post().
			Resource(tpr.CLUSTER_RESOURCE).
			Namespace(Namespace).
			Body(newInstance).
			Do().Into(&result)
		if err != nil {
			log.Error(" in creating PgCluster instance" + err.Error())
		}
		fmt.Println("created PgCluster " + arg)

	}
}

func getClusterParams(name string) *tpr.PgCluster {

	spec := tpr.PgClusterSpec{}
	spec.CCP_IMAGE_TAG = viper.GetString("CLUSTER.CCP_IMAGE_TAG")
	if CCP_IMAGE_TAG != "" {
		spec.CCP_IMAGE_TAG = CCP_IMAGE_TAG
		log.Debug("using CCP_IMAGE_TAG from command line " + CCP_IMAGE_TAG)
	}
	spec.Name = name
	spec.ClusterName = name
	spec.Port = "5432"
	spec.PVC_NAME = viper.GetString("CLUSTER.PVC_NAME")
	spec.PVC_SIZE = "100M"
	spec.PVC_ACCESS_MODE = "ReadWriteMany"
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
	spec.FS_GROUP = ""
	spec.SUPPLEMENTAL_GROUPS = ""
	spec.STRATEGY = "1"

	//override any values from config file
	str := viper.GetString("CLUSTER.PORT")
	if str != "" {
		spec.Port = str
	}
	str = viper.GetString("CLUSTER.PVC_SIZE")
	if str != "" {
		spec.PVC_SIZE = str
	}
	str = viper.GetString("CLUSTER.PVC_ACCESS_MODE")
	if str != "" {
		spec.PVC_ACCESS_MODE = str
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
	str = viper.GetString("CLUSTER.FSGROUP")
	if str != "" {
		spec.FS_GROUP = str
	}
	str = viper.GetString("CLUSTER.SUPPLEMENTALGROUPS")
	if str != "" {
		spec.SUPPLEMENTAL_GROUPS = str
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

	newInstance := &tpr.PgCluster{
		Metadata: api.ObjectMeta{
			Name: name,
		},
		Spec: spec,
	}
	return newInstance
}

func deleteCluster(args []string) {
	// Fetch a list of our cluster TPRs
	clusterList := tpr.PgClusterList{}
	err := Tprclient.Get().Resource(tpr.CLUSTER_RESOURCE).Do().Into(&clusterList)
	if err != nil {
		log.Error("error getting cluster list" + err.Error())
		return
	}

	//to remove a cluster, you just have to remove
	//the pgcluster object, the operator will do the actual deletes
	for _, arg := range args {
		clusterFound := false
		log.Debug("deleting cluster " + arg)
		for _, cluster := range clusterList.Items {
			if arg == "all" || arg == cluster.Spec.Name {
				clusterFound = true
				err = Tprclient.Delete().
					Resource(tpr.CLUSTER_RESOURCE).
					Namespace(Namespace).
					Name(cluster.Spec.Name).
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

func validateSecretFrom(secretname string) error {
	var err error
	lo := v1.ListOptions{LabelSelector: "pg-database=" + secretname}
	secrets, err := Clientset.Secrets(Namespace).List(lo)
	if err != nil {
		log.Error("error getting list of secrets" + err.Error())
		return err
	}

	log.Debug("secrets for " + secretname)
	pgmasterFound := false
	pgrootFound := false
	pguserFound := false

	for _, s := range secrets.Items {
		fmt.Println("")
		fmt.Println("secret : " + s.ObjectMeta.Name)
		if s.ObjectMeta.Name == secretname+tpr.PGMASTER_SECRET_SUFFIX {
			pgmasterFound = true
		} else if s.ObjectMeta.Name == secretname+tpr.PGROOT_SECRET_SUFFIX {
			pgrootFound = true
		} else if s.ObjectMeta.Name == secretname+tpr.PGUSER_SECRET_SUFFIX {
			pguserFound = true
		}
	}
	if !pgmasterFound {
		return errors.New(secretname + tpr.PGMASTER_SECRET_SUFFIX + " not found")
	}
	if !pgrootFound {
		return errors.New(secretname + tpr.PGROOT_SECRET_SUFFIX + " not found")
	}
	if !pguserFound {
		return errors.New(secretname + tpr.PGUSER_SECRET_SUFFIX + " not found")
	}

	return err
}
