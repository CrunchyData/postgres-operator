// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

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
	"encoding/json"
	"time"
	log "github.com/sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/operator/backrest"
	"github.com/crunchydata/postgres-operator/util"
	jsonpatch "github.com/evanphx/json-patch"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// FailoverBase ...
// gets called first on a failover
func FailoverBase(namespace string, clientset *kubernetes.Clientset, client *rest.RESTClient, task *crv1.Pgtask, restconfig *rest.Config) {
	var err error

	//look up the pgcluster for this task
	//in the case, the clustername is passed as a key in the
	//parameters map
	var clusterName string
	for k, _ := range task.Spec.Parameters {
		clusterName = k
	}

	cluster := crv1.Pgcluster{}
	_, err = kubeapi.Getpgcluster(client, &cluster,
		clusterName, namespace)
	if err != nil {
		return
	}

	//create marker (clustername, namespace)
	err = PatchpgtaskFailoverStatus(client, task, namespace)
	if err != nil {
		log.Error("could not set failover started marker for task %s cluster %s", task.Spec.Name, clusterName)
		return
	}

	if cluster.Spec.Strategy == "" {
		cluster.Spec.Strategy = "1"
		log.Info("using default strategy")
	}

	strategy, ok := strategyMap[cluster.Spec.Strategy]
	if ok {
		log.Info("strategy found")
	} else {
		log.Error("invalid Strategy requested for cluster failover " + cluster.Spec.Strategy)
		return
	}

	//get initial count of replicas --selector=pg-cluster=clusterName
	replicaList := crv1.PgreplicaList{}
	selector := util.LABEL_PG_CLUSTER + "=" + clusterName
	err = kubeapi.GetpgreplicasBySelector(client, &replicaList, selector, namespace)
	if err != nil {
		log.Error(err)
		return
	}
	log.Debug("replica count before failover is %d", len(replicaList.Items))

	strategy.Failover(clientset, client, clusterName, task, namespace, restconfig)
	//remove the pgreplica CRD for the promoted replica
	kubeapi.Deletepgreplica(client, task.ObjectMeta.Labels[util.LABEL_TARGET], namespace)

	//optionally, scale up the replicas to replace the failover target
	replaced := false
	userSelection := task.ObjectMeta.Labels[util.LABEL_AUTOFAIL_REPLACE_REPLICA]
	if userSelection == "true" {
		log.Debug("replacing replica based on user selection")
		replaceReplica(client, &cluster, namespace)
		replaced = true
	} else if userSelection == "false" {
		log.Debug("not replacing replica based on user selection")
	} else if operator.Pgo.Cluster.AutofailReplaceReplica {
		log.Debug("replacing replica based on pgo.yaml setting")
		replaceReplica(client, &cluster, namespace)
		replaced = true
	} else {
		log.Debug("not replacing replica")
	}

	//see if the replica service needs to be removed
	if !replaced {
		if len(replicaList.Items) == 1 {
			log.Debug("removing replica service since last replica was removed by the failover %s", clusterName)
			err = kubeapi.DeleteService(clientset, clusterName+"-replica", namespace)
			if err != nil {
				log.Error("could not delete replica service as part of failover")
				return
			}
		}
	}

	//if a backrest-repo exists, bounce it with the new
	//DB_PATH set to the new primary deployment name
	if cluster.ObjectMeta.Labels[util.LABEL_BACKREST] == "true" {
		err = backrest.UpdateDBPath(clientset, &cluster, task.ObjectMeta.Labels[util.LABEL_TARGET], namespace)
		if err != nil {
			log.Error(err)
			log.Error("error bouncing backrest-repo during failover")
			return
		}
	}

}

func replaceReplica(client *rest.RESTClient, cluster *crv1.Pgcluster, ns string) {

	//generate new replica name
	uniqueName := cluster.Spec.Name + "-" + util.RandStringBytesRmndr(4)

	spec := crv1.PgreplicaSpec{}
	spec.Namespace = ns
	spec.Name = uniqueName
	spec.ClusterName = cluster.Spec.Name
	spec.ReplicaStorage = cluster.Spec.ReplicaStorage
	spec.ContainerResources = cluster.Spec.ContainerResources
	spec.UserLabels = make(map[string]string)

	for k, v := range cluster.Spec.UserLabels {
		spec.UserLabels[k] = v
	}
	spec.UserLabels[util.LABEL_PRIMARY] = "false"
	spec.UserLabels[util.LABEL_PG_CLUSTER] = cluster.Spec.Name

	newInstance := &crv1.Pgreplica{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: uniqueName,
		},
		Spec: spec,
		Status: crv1.PgreplicaStatus{
			State:   crv1.PgreplicaStateCreated,
			Message: "Created, not processed yet",
		},
	}

	newInstance.ObjectMeta.Labels = make(map[string]string)
	for x, y := range cluster.ObjectMeta.Labels {
		newInstance.ObjectMeta.Labels[x] = y
	}
	newInstance.ObjectMeta.Labels[util.LABEL_PRIMARY] = "false"
	newInstance.ObjectMeta.Labels[util.LABEL_NAME] = uniqueName

	kubeapi.Createpgreplica(client, newInstance, ns)

}

// PatchpgtaskFailoverStatus - patch the pgtask with failover status
func PatchpgtaskFailoverStatus(restclient *rest.RESTClient, oldCrd *crv1.Pgtask, namespace string) error {

	oldData, err := json.Marshal(oldCrd)
   if err != nil {
	   return err
   }

	//change it
   oldCrd.Spec.Parameters[util.LABEL_FAILOVER_STARTED] = time.Now().Format("2006-01-02.15.04.05")

	//create the patch
   var newData, patchBytes []byte
   newData, err = json.Marshal(oldCrd)
   if err != nil {
	   return err
   }
   patchBytes, err = jsonpatch.CreateMergePatch(oldData, newData)
   if err != nil {
	   return err
   }
   log.Debug(string(patchBytes))

	//apply patch
   _, err6 := restclient.Patch(types.MergePatchType).
	   Namespace(namespace).
	   Resource(crv1.PgtaskResourcePlural).
	   Name(oldCrd.Spec.Name).
	   Body(patchBytes).
	   Do().
	   Get()

	return err6

}