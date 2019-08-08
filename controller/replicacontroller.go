package controller

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
	"context"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/ns"
	"github.com/crunchydata/postgres-operator/operator"
	clusteroperator "github.com/crunchydata/postgres-operator/operator/cluster"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"strings"
)

// PgreplicaController holds the connections for the controller
type PgreplicaController struct {
	PgreplicaClient    *rest.RESTClient
	PgreplicaScheme    *runtime.Scheme
	PgreplicaClientset *kubernetes.Clientset
	Queue              workqueue.RateLimitingInterface

	Ctx context.Context
}

// Run starts an pgreplica resource controller
func (c *PgreplicaController) Run() error {

	defer c.Queue.ShutDown()

	err := c.watchPgreplicas(c.Ctx)
	if err != nil {
		log.Errorf("Failed to register watch for Pgreplica resource: %v", err)
		return err
	}

	<-c.Ctx.Done()
	return c.Ctx.Err()
}

// watchPgreplicas is the event loop for pgreplica resources
func (c *PgreplicaController) watchPgreplicas(ctx context.Context) error {
	nsList := ns.GetNamespaces(c.PgreplicaClientset, operator.InstallationName)

	for i := 0; i < len(nsList); i++ {

		log.Infof("starting pgreplica controller on ns [%s]", nsList[i])
		c.SetupWatch(nsList[i])

	}
	return nil
}

func (c *PgreplicaController) RunWorker() {

	//process the 'add' work queue forever
	for c.processNextItem() {
	}
}

func (c *PgreplicaController) processNextItem() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.Queue.Get()
	if quit {
		return false
	}

	log.Debugf("working on %s", key.(string))
	keyParts := strings.Split(key.(string), "/")
	keyNamespace := keyParts[0]
	keyResourceName := keyParts[1]

	log.Debugf("pgreplica queue got key ns=[%s] resource=[%s]", keyNamespace, keyResourceName)

	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two pods with the same key are never processed in
	// parallel.
	defer c.Queue.Done(key)
	// Invoke the method containing the business logic
	// for pgbackups, the convention is the CRD name is always
	// the same as the pg-cluster label value

	// in this case, the de-dupe logic is to test whether a replica
	// deployment exists already , if so, then we don't create another
	// backup job
	_, found, _ := kubeapi.GetDeployment(c.PgreplicaClientset, keyResourceName, keyNamespace)

	depRunning := false
	if found {
		depRunning = true
	}

	if depRunning {
		log.Debugf("working...found replica already, would do nothing")
	} else {
		log.Debugf("working...no replica found, means we process")

		//handle the case of when a pgreplica is added which is
		//scaling up a cluster
		replica := crv1.Pgreplica{}
		found, err := kubeapi.Getpgreplica(c.PgreplicaClient, &replica, keyResourceName, keyNamespace)
		if !found {
			log.Error(err)
			return false
		}
		clusteroperator.ScaleBase(c.PgreplicaClientset, c.PgreplicaClient, &replica, replica.ObjectMeta.Namespace)

		state := crv1.PgreplicaStateProcessed
		message := "Successfully processed Pgreplica by controller"
		err = kubeapi.PatchpgreplicaStatus(c.PgreplicaClient, state, message, &replica, replica.ObjectMeta.Namespace)
		if err != nil {
			log.Errorf("ERROR updating pgreplica status: %s", err.Error())
		}

		//no error, tell the queue to stop tracking history
		c.Queue.Forget(key)
	}
	return true
}

// onAdd is called when a pgreplica is added
func (c *PgreplicaController) onAdd(obj interface{}) {
	replica := obj.(*crv1.Pgreplica)
	log.Debugf("[PgreplicaController] OnAdd ns=%s %s", replica.ObjectMeta.Namespace, replica.ObjectMeta.SelfLink)

	//handle the case of pgreplicas being processed already and
	//when the operator restarts
	if replica.Status.State == crv1.PgreplicaStateProcessed {
		log.Debug("pgreplica " + replica.ObjectMeta.Name + " already processed")
		return
	}

	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err == nil {
		log.Debugf("onAdd putting key in queue %s", key)
		c.Queue.Add(key)
	}

}

// onUpdate is called when a pgreplica is updated
func (c *PgreplicaController) onUpdate(oldObj, newObj interface{}) {
	newExample := newObj.(*crv1.Pgreplica)
	log.Debugf("[PgreplicaController] ns=%s %s ", newExample.ObjectMeta.Namespace, newExample.ObjectMeta.Name)

}

// onDelete is called when a pgreplica is deleted
func (c *PgreplicaController) onDelete(obj interface{}) {
	replica := obj.(*crv1.Pgreplica)
	log.Debugf("[PgreplicaController] OnDelete ns=%s %s", replica.ObjectMeta.Namespace, replica.ObjectMeta.SelfLink)

	clusteroperator.ScaleDownBase(c.PgreplicaClientset, c.PgreplicaClient, replica, replica.ObjectMeta.Namespace)

}

func (c *PgreplicaController) SetupWatch(ns string) {
	source := cache.NewListWatchFromClient(
		c.PgreplicaClient,
		crv1.PgreplicaResourcePlural,
		ns,
		fields.Everything())

	_, controller := cache.NewInformer(
		source,

		// The object type.
		&crv1.Pgreplica{},

		// resyncPeriod
		// Every resyncPeriod, all resources in the cache will retrigger events.
		// Set to 0 to disable the resync.
		0,

		// Your custom resource event handlers.
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.onAdd,
			UpdateFunc: c.onUpdate,
			DeleteFunc: c.onDelete,
		})

	go controller.Run(c.Ctx.Done())
}
