package controller

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
	"context"
	log "github.com/Sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// PodController holds the connections for the controller
type PodController struct {
	PodClient    *rest.RESTClient
	PodClientset *kubernetes.Clientset
	Namespace    string
}

// Run starts an pod resource controller
func (c *PodController) Run(ctx context.Context) error {
	log.Info("Watch pod objects")

	_, err := c.watchPods(ctx)
	if err != nil {
		log.Errorf("Failed to register watch for pod resource: %v\n", err)
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

// watchPods is the event loop for pod resources
func (c *PodController) watchPods(ctx context.Context) (cache.Controller, error) {
	source := cache.NewListWatchFromClient(
		c.PodClientset.CoreV1().RESTClient(),
		"pods",
		//apiv1.NamespaceAll,
		c.Namespace,
		fields.Everything())

	_, controller := cache.NewInformer(
		source,

		// The object type.
		&apiv1.Pod{},

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

	go controller.Run(ctx.Done())
	return controller, nil
}

// onAdd is called when a pgcluster is added
func (c *PodController) onAdd(obj interface{}) {
}

// onUpdate is called when a pgcluster is updated
func (c *PodController) onUpdate(oldObj, newObj interface{}) {
}

// onDelete is called when a pgcluster is deleted
func (c *PodController) onDelete(obj interface{}) {
	pod := obj.(*apiv1.Pod)
	log.Infof("[PodCONTROLLER] OnDelete %s\n", pod.ObjectMeta.SelfLink)
}
