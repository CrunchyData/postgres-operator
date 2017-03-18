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
	"github.com/crunchydata/crunchy-operator/tpr"
	"github.com/spf13/viper"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/v1"
)

func showCluster(args []string) {
	//get a list of all clusters
	clusterList := tpr.CrunchyClusterList{}
	err := Tprclient.Get().Resource("crunchyclusters").Do().Into(&clusterList)
	if err != nil {
		fmt.Println("error getting list of clusters")
		fmt.Println(err.Error())
		return
	}

	//each arg represents a cluster name or the special 'all' value
	for _, arg := range args {
		for _, cluster := range clusterList.Items {
			fmt.Println("")
			fmt.Println("cluster : " + cluster.Spec.Name)
			if arg == "all" || cluster.Spec.Name == arg {
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
	lo := v1.ListOptions{LabelSelector: "crunchy-cluster=" + name}
	reps, err := Clientset.ReplicaSets(api.NamespaceDefault).List(lo)
	if err != nil {
		fmt.Println("error getting list of replicasets")
		fmt.Println(err.Error())
		return
	}
	for _, r := range reps.Items {
		fmt.Println(TREE_BRANCH + "replicaset : " + r.ObjectMeta.Name)
	}

}
func listDeployments(name string) {
	lo := v1.ListOptions{LabelSelector: "crunchy-cluster=" + name}
	deployments, err := Clientset.Deployments(api.NamespaceDefault).List(lo)
	if err != nil {
		fmt.Println("error getting list of deployments")
		fmt.Println(err.Error())
		return
	}
	for _, d := range deployments.Items {
		fmt.Println(TREE_BRANCH + "deployment : " + d.ObjectMeta.Name)
	}

}
func listPods(name string) {
	lo := v1.ListOptions{LabelSelector: "crunchy-cluster=" + name}
	pods, err := Clientset.Core().Pods(api.NamespaceDefault).List(lo)
	if err != nil {
		fmt.Println("error getting list of pods")
		fmt.Println(err.Error())
		return
	}
	for _, pod := range pods.Items {
		fmt.Println(TREE_BRANCH + "pod : " + pod.ObjectMeta.Name)
		//fmt.Println(TREE_TRUNK + " phase : " + pod.Status.Phase)
	}

}
func listServices(name string) {
	lo := v1.ListOptions{LabelSelector: "crunchy-cluster=" + name}
	services, err := Clientset.Core().Services(api.NamespaceDefault).List(lo)
	if err != nil {
		fmt.Println("error getting list of services")
		fmt.Println(err.Error())
		return
	}
	for i, service := range services.Items {
		if i == len(services.Items)-1 {
			fmt.Println(TREE_TRUNK + "service : " + service.ObjectMeta.Name)
		} else {
			fmt.Println(TREE_BRANCH + "service : " + service.ObjectMeta.Name)
		}
	}
}

func createCluster(args []string) {
	var err error

	for _, arg := range args {
		fmt.Println("create cluster called for " + arg)
		result := tpr.CrunchyCluster{}

		// error if it already exists
		err = Tprclient.Get().
			Resource("crunchyclusters").
			Namespace(api.NamespaceDefault).
			Name(arg).
			Do().
			Into(&result)
		if err == nil {
			fmt.Println("crunchycluster " + arg + " was found so we will not create it")
			break
		} else if errors.IsNotFound(err) {
			fmt.Println("crunchycluster " + arg + " not found so we will create it")
		} else {
			fmt.Println("error getting crunchycluster " + arg)
			fmt.Println(err.Error())
			break
		}

		// Create an instance of our TPR
		newInstance := getClusterParams(arg)

		err = Tprclient.Post().
			Resource("crunchyclusters").
			Namespace(api.NamespaceDefault).
			Body(newInstance).
			Do().Into(&result)
		if err != nil {
			fmt.Println("error in creating CrunchyCluster instance")
			fmt.Println(err.Error())
		}
		fmt.Println("created CrunchyCluster " + arg)

	}
}

func getClusterParams(name string) *tpr.CrunchyCluster {

	//set to internal defaults
	spec := tpr.CrunchyClusterSpec{
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

	newInstance := &tpr.CrunchyCluster{
		Metadata: api.ObjectMeta{
			Name: name,
		},
		Spec: spec,
	}
	return newInstance
}

func deleteCluster(args []string) {
	// Fetch a list of our cluster TPRs
	clusterList := tpr.CrunchyClusterList{}
	err := Tprclient.Get().Resource("crunchyclusters").Do().Into(&clusterList)
	if err != nil {
		fmt.Println("error getting cluster list")
		fmt.Println(err.Error())
		return
	}

	//to remove a cluster, you just have to remove
	//the crunchycluster object, the operator will do the actual deletes
	for _, arg := range args {
		fmt.Println("deleting cluster " + arg)
		for _, cluster := range clusterList.Items {
			if arg == "all" || arg == cluster.Spec.Name {
				err = Tprclient.Delete().
					Resource("crunchyclusters").
					Namespace(api.NamespaceDefault).
					Name(cluster.Spec.Name).
					Do().
					Error()
				if err != nil {
					fmt.Println("error deleting crunchycluster " + arg)
					fmt.Println(err.Error())
				} else {
					fmt.Println("deleted crunchycluster " + cluster.Spec.Name)
				}

			}
		}
	}
}
