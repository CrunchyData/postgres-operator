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
	//kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/pkg/api/v1"
	//"k8s.io/client-go/pkg/apis/extensions/v1beta1"

	"bytes"
	"encoding/json"
	//"fmt"
	"github.com/crunchydata/postgres-operator/clusterservice"
	"net/http"
)

func showCluster(args []string) {
	var err error

	url := "http://localhost:8080/clusters/somename?showsecrets=true&other=thing"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal("NewRequest: ", err)
		return
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Do: ", err)
		return
	}

	defer resp.Body.Close()

	var response clusterservice.ShowClusterResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Println(err)
	}

	fmt.Println("Name = ", response.Items[0].Name)

}

func createCluster(args []string) {
	//var err error
	url := "http://localhost:8080/clusters"

	for _, arg := range args {
		fmt.Println(arg)
		cl := new(clusterservice.CreateClusterRequest)
		cl.Name = "newcluster"
		jsonValue, _ := json.Marshal(cl)
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
		if err != nil {
			log.Fatal("NewRequest: ", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}

		resp, err := client.Do(req)
		if err != nil {
			log.Fatal("Do: ", err)
			return
		}
		fmt.Printf("%v\n", resp)
	}

}

func getClusterParams(name string) *tpr.PgCluster {

	spec := tpr.PgClusterSpec{}
	masterStorageSpec := tpr.PgStorageSpec{}
	spec.MasterStorage = masterStorageSpec
	replicaStorageSpec := tpr.PgStorageSpec{}
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

	newInstance := &tpr.PgCluster{
		Metadata: meta_v1.ObjectMeta{
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
		//fmt.Println("")
		//fmt.Println("secret : " + s.ObjectMeta.Name)
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
