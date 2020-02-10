// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

/*
 Copyright 2018 - 2020 Crunchy Data Solutions, Inc.
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
	jsonpatch "github.com/evanphx/json-patch"
	log "github.com/sirupsen/logrus"
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
		Clustername: clusterName,
		Target:      task.ObjectMeta.Labels[config.LABEL_TARGET],
	}

	err = events.Publish(f)
	if err != nil {
		log.Error(err)
	}

	Failover(cluster.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER], clientset, client, clusterName, task, namespace, restconfig)

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

func PatchpgtaskFailoverStatus(restclient *rest.RESTClient, oldCrd *crv1.Pgtask, namespace string) error {

	oldData, err := json.Marshal(oldCrd)
	if err != nil {
		return err
	}

	//change it
	oldCrd.Spec.Parameters[config.LABEL_FAILOVER_STARTED] = time.Now().Format(time.RFC3339)

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
