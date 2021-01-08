package configmap

/*
Copyright 2021 Crunchy Data Solutions, Inc.
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
	"github.com/crunchydata/postgres-operator/internal/config"
	pgoinformers "github.com/crunchydata/postgres-operator/pkg/generated/informers/externalversions/crunchydata.com/v1"
	pgolisters "github.com/crunchydata/postgres-operator/pkg/generated/listers/crunchydata.com/v1"

	log "github.com/sirupsen/logrus"

	apiv1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Controller holds connections and other resources for the ConfigMap controller
type Controller struct {
	cmRESTConfig    *rest.Config
	kubeclientset   kubernetes.Interface
	cmLister        corelisters.ConfigMapLister
	cmSynced        cache.InformerSynced
	pgclusterLister pgolisters.PgclusterLister
	pgclusterSynced cache.InformerSynced
	workqueue       workqueue.RateLimitingInterface
	workerCount     int
}

// NewConfigMapController is responsible for creating a new ConfigMap controller
func NewConfigMapController(restConfig *rest.Config,
	clientset kubernetes.Interface, coreInformer coreinformers.ConfigMapInformer,
	pgoInformer pgoinformers.PgclusterInformer, workerCount int) (*Controller, error) {
	controller := &Controller{
		cmRESTConfig:    restConfig,
		kubeclientset:   clientset,
		cmLister:        coreInformer.Lister(),
		cmSynced:        coreInformer.Informer().HasSynced,
		pgclusterLister: pgoInformer.Lister(),
		pgclusterSynced: pgoInformer.Informer().HasSynced,
		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(),
			"ConfigMaps"),
		workerCount: workerCount,
	}

	coreInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			controller.enqueueConfigMap(obj)
		},
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueConfigMap(new)
		},
	})

	return controller, nil
}

// RunWorker is a long-running function that will continually call the processNextWorkItem
// function in order to read and process a message on the worker queue.  Once the worker queue
// is instructed to shutdown, a message is written to the done channel.
func (c *Controller) RunWorker(stopCh <-chan struct{}, doneCh chan<- struct{}) {
	go c.waitForShutdown(stopCh)

	for c.processNextWorkItem() {
	}

	log.Debug("ConfigMap Contoller: worker queue has been shutdown, writing to the done channel")

	doneCh <- struct{}{}
}

// waitForShutdown waits for a message on the stop channel and then shuts down the work queue
func (c *Controller) waitForShutdown(stopCh <-chan struct{}) {
	<-stopCh
	c.workqueue.ShutDown()
	log.Debug("ConfigMap Contoller: received stop signal, worker queue told to shutdown")
}

// ShutdownWorker shuts down the work queue
func (c *Controller) ShutdownWorker() {
	c.workqueue.ShutDown()
	log.Debug("ConfigMap Contoller: worker queue told to shutdown")
}

// enqueueConfigMap inspects a configMap to determine if it should be added to the work queue.  If
// so, the configMap resource is converted into a namespace/name string and is then added to the
// work queue
func (c *Controller) enqueueConfigMap(obj interface{}) {
	configMap := obj.(*apiv1.ConfigMap)
	labels := configMap.GetObjectMeta().GetLabels()

	// Right now we only care about updates to the PGHA configMap, which is the configMap created
	// for each cluster with label 'pgha-config'.  Therefore, simply return if the configMap
	// does not have this label, and don't add the resource to the queue.
	if _, ok := labels[config.LABEL_PGHA_CONFIGMAP]; !ok {
		return
	}

	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// processNextWorkItem will read a single work item off the work queue and processes it via
// the ConfigMap sync handler
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
		log.Errorf("ConfigMap Controller: expected string in workqueue but got %#v", obj)
		return true
	}

	// Run handleConfigMapSync, passing it the namespace/name key of the configMap that
	// needs to be synced
	if err := c.handleConfigMapSync(key); err != nil {
		// Put the item back on the workqueue to handle any transient errors
		c.workqueue.AddRateLimited(key)
		log.Errorf("ConfigMap Controller: error syncing ConfigMap '%s', will now requeue: %v",
			key, err)
		return true
	}

	// Finally if no error has occurred forget this item
	c.workqueue.Forget(obj)

	return true
}

// WorkerCount returns the worker count for the controller
func (c *Controller) WorkerCount() int {
	return c.workerCount
}
