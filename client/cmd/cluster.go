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
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/crunchydata/postgres-operator/tpr"
	"github.com/spf13/viper"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/v1"
)

func showCluster(args []string) {
	//get a list of all clusters
	clusterList := tpr.PgClusterList{}
	err := Tprclient.Get().Resource("pgclusters").Do().Into(&clusterList)
	if err != nil {
		log.Error("error getting list of clusters" + err.Error())
		return
	}

	//each arg represents a cluster name or the special 'all' value
	for _, arg := range args {
		for _, cluster := range clusterList.Items {
			fmt.Println("")
			fmt.Println("cluster : " + cluster.Spec.Name)
			if arg == "all" || cluster.Spec.Name == arg {
				log.Debug("listing cluster " + arg)
				//list the deployments
				listDeployments(cluster.Spec.Name)
				//list the replicasets
				listReplicaSets(cluster.Spec.Name)
				//list the pods
				listPods(cluster.Spec.Name)
				//list the services
				listServices(cluster.Spec.Name)
			}
		}
	}

}

func listReplicaSets(name string) {
	lo := v1.ListOptions{LabelSelector: "pg-cluster=" + name}
	reps, err := Clientset.ReplicaSets(api.NamespaceDefault).List(lo)
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
	deployments, err := Clientset.Deployments(api.NamespaceDefault).List(lo)
	if err != nil {
		log.Error("error getting list of deployments" + err.Error())
		return
	}
	for _, d := range deployments.Items {
		fmt.Println(TREE_BRANCH + "deployment : " + d.ObjectMeta.Name)
	}

}
func listPods(name string) {
	lo := v1.ListOptions{LabelSelector: "pg-cluster=" + name}
	pods, err := Clientset.Core().Pods(api.NamespaceDefault).List(lo)
	if err != nil {
		log.Error("error getting list of pods" + err.Error())
		return
	}
	for _, pod := range pods.Items {
		fmt.Println(TREE_BRANCH + "pod : " + pod.ObjectMeta.Name)
		//fmt.Println(TREE_TRUNK + " phase : " + pod.Status.Phase)
	}

}
func listServices(name string) {
	lo := v1.ListOptions{LabelSelector: "pg-cluster=" + name}
	services, err := Clientset.Core().Services(api.NamespaceDefault).List(lo)
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
			Resource("pgclusters").
			Namespace(api.NamespaceDefault).
			Name(arg).
			Do().
			Into(&result)
		if err == nil {
			log.Debug("pgcluster " + arg + " was found so we will not create it")
			break
		} else if errors.IsNotFound(err) {
			log.Debug("pgcluster " + arg + " not found so we will create it")
		} else {
			log.Error("error getting pgcluster " + arg + err.Error())
			break
		}

		// Create an instance of our TPR
		newInstance := getClusterParams(arg)

		err = Tprclient.Post().
			Resource("pgclusters").
			Namespace(api.NamespaceDefault).
			Body(newInstance).
			Do().Into(&result)
		if err != nil {
			log.Error(" in creating PgCluster instance" + err.Error())
		}
		fmt.Println("created PgCluster " + arg)

	}
}

func getClusterParams(name string) *tpr.PgCluster {

	//set to internal defaults
	spec := tpr.PgClusterSpec{
		Name:               name,
		ClusterName:        name,
		CCP_IMAGE_TAG:      "centos7-9.5-1.2.8",
		Port:               "5432",
		PVC_NAME:           "crunchy-pvc",
		PG_MASTER_HOST:     name,
		PG_MASTER_USER:     "master",
		PG_MASTER_PASSWORD: "password",
		PG_USER:            "testuser",
		PG_PASSWORD:        "password",
		PG_DATABASE:        "userdb",
		PG_ROOT_PASSWORD:   "password",
		REPLICAS:           "2",
	}

	//override any values from config file
	str := viper.GetString("cluster.CCP_IMAGE_TAG")
	if str != "" {
		spec.CCP_IMAGE_TAG = str
	}
	str = viper.GetString("cluster.Port")
	if str != "" {
		spec.Port = str
	}
	str = viper.GetString("cluster.PVC_NAME")
	if str != "" {
		spec.PVC_NAME = str
	}
	str = viper.GetString("cluster.PG_MASTER_USER")
	if str != "" {
		spec.PG_MASTER_USER = str
	}
	str = viper.GetString("cluster.PG_MASTER_PASSWORD")
	if str != "" {
		spec.PG_MASTER_PASSWORD = str
	}
	str = viper.GetString("cluster.PG_USER")
	if str != "" {
		spec.PG_USER = str
	}
	str = viper.GetString("cluster.PG_PASSWORD")
	if str != "" {
		spec.PG_PASSWORD = str
	}
	str = viper.GetString("cluster.PG_DATABASE")
	if str != "" {
		spec.PG_DATABASE = str
	}
	str = viper.GetString("cluster.PG_ROOT_PASSWORD")
	if str != "" {
		spec.PG_ROOT_PASSWORD = str
	}
	str = viper.GetString("cluster.REPLICAS")
	if str != "" {
		spec.REPLICAS = str
	}

	//override from command line

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
	err := Tprclient.Get().Resource("pgclusters").Do().Into(&clusterList)
	if err != nil {
		log.Error("error getting cluster list" + err.Error())
		return
	}

	//to remove a cluster, you just have to remove
	//the pgcluster object, the operator will do the actual deletes
	for _, arg := range args {
		fmt.Println("deleting cluster " + arg)
		for _, cluster := range clusterList.Items {
			if arg == "all" || arg == cluster.Spec.Name {
				err = Tprclient.Delete().
					Resource("pgclusters").
					Namespace(api.NamespaceDefault).
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
	}
}
