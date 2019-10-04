// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

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
	"encoding/json"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/events"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/operator/backrest"
	"github.com/crunchydata/postgres-operator/util"
	jsonpatch "github.com/evanphx/json-patch"
	log "github.com/sirupsen/logrus"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"time"
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

	//get initial count of replicas --selector=pg-cluster=clusterName
	replicaList := crv1.PgreplicaList{}
	selector := config.LABEL_PG_CLUSTER + "=" + clusterName
	err = kubeapi.GetpgreplicasBySelector(client, &replicaList, selector, namespace)
	if err != nil {
		log.Error(err)
		return
	}
	log.Debug("replica count before failover is %d", len(replicaList.Items))

	//publish event for failover
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventFailoverClusterFormat{
		EventHeader: events.EventHeader{
			Namespace: namespace,
			Username:  task.ObjectMeta.Labels[config.LABEL_PGOUSER],
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventFailoverCluster,
		},
		Clustername:       clusterName,
		Target:            task.ObjectMeta.Labels[config.LABEL_TARGET],
	}

	err = events.Publish(f)
	if err != nil {
		log.Error(err)
	}

	Failover(cluster.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER], clientset, client, clusterName, task, namespace, restconfig)
	//remove the pgreplica CRD for the promoted replica
	kubeapi.Deletepgreplica(client, task.ObjectMeta.Labels[config.LABEL_TARGET], namespace)

	//optionally, scale up the replicas to replace the failover target
	replaced := false
	userSelection := task.ObjectMeta.Labels[config.LABEL_AUTOFAIL_REPLACE_REPLICA]
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
	if cluster.ObjectMeta.Labels[config.LABEL_BACKREST] == "true" {
		err = backrest.UpdateDBPath(clientset, &cluster, task.ObjectMeta.Labels[config.LABEL_TARGET], namespace)
		if err != nil {
			log.Error(err)
			log.Error("error bouncing backrest-repo during failover")
			return
		}
	}

	//publish event for failover completed
	topics = make([]string, 1)
	topics[0] = events.EventTopicCluster

	g := events.EventFailoverClusterCompletedFormat{
		EventHeader: events.EventHeader{
			Namespace: namespace,
			Username:  task.ObjectMeta.Labels[config.LABEL_PGOUSER],
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventFailoverClusterCompleted,
		},
		Clustername: clusterName,
		Target:      task.ObjectMeta.Labels[config.LABEL_TARGET],
	}

	err = events.Publish(g)
	if err != nil {
		log.Error(err)
	}

	//remove marker

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

	spec.UserLabels[config.LABEL_PG_CLUSTER] = cluster.Spec.Name

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

	newInstance.ObjectMeta.Labels[config.LABEL_NAME] = uniqueName

	kubeapi.Createpgreplica(client, newInstance, ns)

}

func PatchpgtaskFailoverStatus(restclient *rest.RESTClient, oldCrd *crv1.Pgtask, namespace string) error {

	oldData, err := json.Marshal(oldCrd)
	if err != nil {
		return err
	}

	//change it
	oldCrd.Spec.Parameters[config.LABEL_FAILOVER_STARTED] = time.Now().Format("2006-01-02.15.04.05")

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
