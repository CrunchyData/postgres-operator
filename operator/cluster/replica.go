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
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
)

// DeletePgreplicas
func DeletePgreplicas(restclient *rest.RESTClient, clusterName, namespace string) {

	replicaList := crv1.PgreplicaList{}

	//get a list of pgreplicas for this cluster
	err := kubeapi.GetpgreplicasBySelector(restclient,
		&replicaList, config.LABEL_PG_CLUSTER+"="+clusterName,
		namespace)
	if err != nil {
		return
	}

	log.Debug("pgreplicas found len is %d\n", len(replicaList.Items))

	for _, r := range replicaList.Items {
		err = kubeapi.Deletepgreplica(restclient, r.Spec.Name, namespace)
	}

}
