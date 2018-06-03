// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

/*
 Copyright 2017-2018 Crunchy Data Solutions, Inc.
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
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/util"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// FailoverBase ...
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

	strategy.Failover(clientset, client, clusterName, task, namespace, restconfig)

	//scale up the replicas to replace the failover target
	replaceReplica(client, &cluster)

}

func replaceReplica(client *rest.RESTClient, cluster *crv1.Pgcluster) {

	//generate new replica name
	uniqueName := cluster.Spec.Name + "-" + util.RandStringBytesRmndr(4)

	spec := crv1.PgreplicaSpec{}
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

	kubeapi.Createpgreplica(client, newInstance, operator.NAMESPACE)

}
