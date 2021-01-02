package namespace

/*
Copyright 2019 - 2021 Crunchy Data Solutions, Inc.
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
	"github.com/crunchydata/postgres-operator/internal/controller"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	coreinformers "k8s.io/client-go/informers/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Controller holds the connections for the controller
type Controller struct {
	ControllerManager controller.Manager
	Informer          coreinformers.NamespaceInformer
	namespaceLister   corelisters.NamespaceLister
	workqueue         workqueue.RateLimitingInterface
	workerCount       int
}

// NewNamespaceController creates a new namespace controller that will watch for namespace events
// as responds accordingly.  This adding and removing controller groups as namespaces watched by the
// PostgreSQL Operator are added and deleted.
func NewNamespaceController(controllerManager controller.Manager,
	informer coreinformers.NamespaceInformer, workerCount int) (*Controller, error) {

	controller := &Controller{
		ControllerManager: controllerManager,
		Informer:          informer,
		namespaceLister:   informer.Lister(),
		workerCount:       workerCount,
		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(),
			"Namespaces"),
	}

	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			controller.enqueueNamespace(obj)
		},
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueNamespace(new)
		},
		DeleteFunc: func(obj interface{}) {
			controller.enqueueNamespace(obj)
		},
	})

	return controller, nil
}

// RunWorker is a long-running function that will continually call the processNextWorkItem
// function in order to read and process a message on the worker queue.  Once the worker queue
// is instructed to shutdown, a message is written to the done channel.
func (c *Controller) RunWorker(stopCh <-chan struct{}) {

	go c.waitForShutdown(stopCh)

	for c.processNextWorkItem() {
	}
}

// waitForShutdown waits for a message on the stop channel and then shuts down the work queue
func (c *Controller) waitForShutdown(stopCh <-chan struct{}) {
	<-stopCh
	c.workqueue.ShutDown()
	log.Debug("Namespace Contoller: received stop signal, worker queue told to shutdown")
}

// ShutdownWorker shuts down the work queue
func (c *Controller) ShutdownWorker() {
	c.workqueue.ShutDown()
	log.Debug("Namespace Contoller: worker queue told to shutdown")
}

// enqueueNamespace inspects a namespace to determine if it should be added to the work queue.  If
// so, the namespace resource is converted into a namespace/name string and is then added to the
// work queue
func (c *Controller) enqueueNamespace(obj interface{}) {

	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// processNextWorkItem will read a single work item off the work queue and processes it via
// the Namespace sync handler
func (c *Controller) processNextWorkItem() bool {

	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We call Done here so the workqueue knows we have finished processing this item
	defer c.workqueue.Done(obj)

	var key string
	var ok bool
	// We expect strings to come off the workqueue in the form namespace/name
	if key, ok = obj.(string); !ok {
		c.workqueue.Forget(obj)
		log.Errorf("Namespace Controller: expected string in workqueue but got %#v", obj)
		return true
	}

	_, namespace, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		c.workqueue.Forget(obj)
		log.Error(err)
		return true
	}

	// remove the controller group for the namespace if the namespace no longer exists or is
	// termininating
	ns, err := c.namespaceLister.Get(namespace)
	if (err == nil && ns.Status.Phase == corev1.NamespaceTerminating) ||
		(err != nil && kerrors.IsNotFound(err)) {
		c.ControllerManager.RemoveGroup(namespace)
		c.workqueue.Forget(obj)
		return true
	} else if err != nil {
		log.Errorf("Namespace Controller: error getting namespace %s from namespaceLister, will "+
			"now requeue: %v", key, err)
		c.workqueue.AddRateLimited(key)
		return true
	}

	// Run AddAndRunGroup, passing it the namespace that needs to be synced
	if err := c.ControllerManager.AddAndRunGroup(namespace); err != nil {
		log.Errorf("Namespace Controller: error syncing Namespace '%s': %s",
			key, err.Error())
	}

	// Finally if no error has occurred forget this item
	c.workqueue.Forget(obj)

	return true
}

// WorkerCount returns the worker count for the controller
func (c *Controller) WorkerCount() int {
	return c.workerCount
}
