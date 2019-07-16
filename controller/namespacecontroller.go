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
	//crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	//"github.com/crunchydata/postgres-operator/config"
	//"github.com/crunchydata/postgres-operator/events"
	//"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"
	//"k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// NamespaceController holds the connections for the controller
type NamespaceController struct {
	NamespaceClient    *rest.RESTClient
	NamespaceClientset *kubernetes.Clientset
}

// Run starts an pod resource controller
func (c *NamespaceController) Run(ctx context.Context) error {

	err := c.watchNamespaces(ctx)
	if err != nil {
		log.Errorf("Failed to register watch for namespace resource: %v", err)
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

// watchNamespaces is the event loop for namespace resources
func (c *NamespaceController) watchNamespaces(ctx context.Context) error {
	log.Info("starting namespace controller")

	//watch all namespaces
	ns := ""

	source := cache.NewListWatchFromClient(
		c.NamespaceClientset.CoreV1().RESTClient(),
		"namespaces",
		ns,
		fields.Everything())

	_, controller := cache.NewInformer(
		source,

		// The object type.
		&v1.Namespace{},

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

	return nil
}

func (c *NamespaceController) onAdd(obj interface{}) {
	newNs := obj.(*v1.Namespace)

	/**
	labels := newNs.GetObjectMeta().GetLabels()
	if labels[config.LABEL_VENDOR] != "crunchydata" {
		log.Debugf("NamespaceController: onAdd skipping pod that is not crunchydata %s", newpod.ObjectMeta.SelfLink)
		return
	}
	*/

	log.Debugf("[NamespaceController] OnAdd ns=%s", newNs.ObjectMeta.SelfLink)

}

// onUpdate is called when a pgcluster is updated
func (c *NamespaceController) onUpdate(oldObj, newObj interface{}) {
	//oldNs := oldObj.(*v1.Namespace)
	newNs := newObj.(*v1.Namespace)

	/**
	labels := newpod.GetObjectMeta().GetLabels()
	if labels[config.LABEL_VENDOR] != "crunchydata" {
		log.Debugf("NamespaceController: onUpdate skipping pod that is not crunchydata %s", newpod.ObjectMeta.SelfLink)
		return
	}
	*/

	log.Debugf("[NamespaceController] onUpdate ns=%s", newNs.ObjectMeta.SelfLink)

}

func (c *NamespaceController) onDelete(obj interface{}) {
	ns := obj.(*v1.Namespace)

	/**
	labels := pod.GetObjectMeta().GetLabels()
	if labels[config.LABEL_VENDOR] != "crunchydata" {
		log.Debugf("NamespaceController: onDelete skipping pod that is not crunchydata %s", pod.ObjectMeta.SelfLink)
		return
	}
	*/

	log.Debugf("[NamespaceController] onDelete ns=%s", ns.ObjectMeta.SelfLink)
}
