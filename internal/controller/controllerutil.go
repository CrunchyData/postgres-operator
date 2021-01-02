package controller

/*
Copyright 2020 - 2021 Crunchy Data Solutions, Inc.
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
	"errors"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
)

// ErrControllerGroupExists is the error that is thrown when a controller group for a specific
// namespace already exists
var ErrControllerGroupExists = errors.New("A controller group for the namespace specified already" +
	"exists")

// WorkerRunner is an interface for controllers the have worker queues that need to be run
type WorkerRunner interface {
	RunWorker(stopCh <-chan struct{}, doneCh chan<- struct{})
	WorkerCount() int
}

// Manager defines the interface for a controller manager
type Manager interface {
	AddGroup(namespace string) error
	AddAndRunGroup(namespace string) error
	RemoveAll()
	RemoveGroup(namespace string)
	RunAll() error
	RunGroup(namespace string) error
}

// InitializeReplicaCreation initializes the creation of replicas for a cluster.  For a regular
// (i.e. non-standby) cluster this is called following the creation of the initial cluster backup,
// which is needed to bootstrap replicas.  However, for a standby cluster this is called as
// soon as the primary PG pod reports ready and the cluster is marked as initialized.
func InitializeReplicaCreation(restclient *rest.RESTClient, clusterName,
	namespace string) error {

	pgreplicaList := &crv1.PgreplicaList{}
	selector := config.LABEL_PG_CLUSTER + "=" + clusterName
	err := kubeapi.GetpgreplicasBySelector(restclient, pgreplicaList, selector, namespace)
	if err != nil {
		log.Error(err)
		return err
	}
	for _, pgreplica := range pgreplicaList.Items {

		if pgreplica.Annotations == nil {
			pgreplica.Annotations = make(map[string]string)
		}

		pgreplica.Annotations[config.ANNOTATION_PGHA_BOOTSTRAP_REPLICA] = "true"

		if err = kubeapi.Updatepgreplica(restclient, &pgreplica, pgreplica.Name,
			namespace); err != nil {

			log.Error(err)
			return err
		}
	}
	return nil
}

// SetClusterInitializedStatus sets the status of a pgcluster CR to indicate that it has been
// initialized.  This is specifically done by patching the status of the pgcluster CR with the
// proper initialization status.
func SetClusterInitializedStatus(restclient *rest.RESTClient, clusterName,
	namespace string) error {

	cluster := crv1.Pgcluster{}
	if _, err := kubeapi.Getpgcluster(restclient, &cluster, clusterName,
		namespace); err != nil {
		log.Error(err)
		return err
	}
	message := "Cluster has been initialized"
	if err := kubeapi.PatchpgclusterStatus(restclient, crv1.PgclusterStateInitialized, message,
		&cluster, namespace); err != nil {
		log.Error(err)
		return err
	}

	return nil
}
