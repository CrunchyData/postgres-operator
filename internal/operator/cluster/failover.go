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
	"context"
	"encoding/json"
	"time"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	pgo "github.com/crunchydata/postgres-operator/pkg/generated/clientset/versioned"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
)

// FailoverBase ...
// gets called first on a failover
func FailoverBase(namespace string, clientset kubeapi.Interface, task *crv1.Pgtask, restconfig *rest.Config) {
	ctx := context.TODO()
	var err error

	// look up the pgcluster for this task
	// in the case, the clustername is passed as a key in the
	// parameters map
	var clusterName string
	for k := range task.Spec.Parameters {
		clusterName = k
	}

	cluster, err := clientset.CrunchydataV1().Pgclusters(namespace).Get(ctx, clusterName, metav1.GetOptions{})
	if err != nil {
		return
	}

	// create marker (clustername, namespace)
	err = PatchpgtaskFailoverStatus(clientset, task, namespace)
	if err != nil {
		log.Errorf("could not set failover started marker for task %s cluster %s", task.Spec.Name, clusterName)
		return
	}

	// get initial count of replicas --selector=pg-cluster=clusterName
	selector := config.LABEL_PG_CLUSTER + "=" + clusterName
	replicaList, err := clientset.CrunchydataV1().Pgreplicas(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		log.Error(err)
		return
	}
	log.Debugf("replica count before failover is %d", len(replicaList.Items))

	_ = Failover(cluster.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER], clientset, clusterName, task, namespace, restconfig)
}

func PatchpgtaskFailoverStatus(clientset pgo.Interface, oldCrd *crv1.Pgtask, namespace string) error {
	ctx := context.TODO()

	// change it
	oldCrd.Spec.Parameters[config.LABEL_FAILOVER_STARTED] = time.Now().Format(time.RFC3339)

	// create the patch
	patchBytes, err := json.Marshal(map[string]interface{}{
		"spec": map[string]interface{}{
			"parameters": oldCrd.Spec.Parameters,
		},
	})
	if err != nil {
		return err
	}

	// apply patch
	_, err6 := clientset.CrunchydataV1().Pgtasks(namespace).
		Patch(ctx, oldCrd.Name, types.MergePatchType, patchBytes, metav1.PatchOptions{})

	return err6
}
