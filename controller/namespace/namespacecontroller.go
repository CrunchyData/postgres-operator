package namespace

/*
Copyright 2019 - 2020 Crunchy Data Solutions, Inc.
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
	"github.com/crunchydata/postgres-operator/controller"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
)

// Controller holds the connections for the controller
type Controller struct {
	ControllerManager controller.Manager
	Informer          coreinformers.NamespaceInformer
}

// NewNamespaceController creates a new namespace controller that will watch for namespace events
// as responds accordingly.  This adding and removing controller groups as namespaces watched by the
// PostgreSQL Operator are added and deleted.
func NewNamespaceController(controllerManager controller.Manager,
	informer coreinformers.NamespaceInformer) (*Controller, error) {

	controller := &Controller{
		ControllerManager: controllerManager,
		Informer:          informer,
	}

	return controller, nil
}

// AddNamespaceEventHandler adds the pod event handler to the namespace informer
func (c *Controller) AddNamespaceEventHandler() {

	c.Informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	})

	log.Debugf("Namespace Controller: added event handler to informer")
}

// onUpdate is called when a namespace is added
func (c *Controller) onAdd(obj interface{}) {

	newNs := obj.(*v1.Namespace)

	log.Debugf("namespace Controller: onAdd will now add a controller "+
		"group for namespace %s", newNs.Name)
	if err := c.ControllerManager.AddAndRunGroup(newNs.Name); err != nil {
		log.Error(err)
	}
}

// onUpdate is called when a namespace is updated
func (c *Controller) onUpdate(oldObj, newObj interface{}) {

	newNs := newObj.(*v1.Namespace)

	log.Debugf("namespace Controller: onUpdate will now attempt to add and run a controller "+
		"group for namespace %s", newNs.Name)
	// Add and run the controller group if namespace is part of the current installation.
	// AddAndRunGroup can be called over and over again, and the controller group will only
	// be created and/or run if not already created and/or running
	if err := c.ControllerManager.AddAndRunGroup(newNs.Name); err != nil {
		log.Error(err)
	}
}

func (c *Controller) onDelete(obj interface{}) {

	ns := obj.(*v1.Namespace)

	log.Debugf("namespace Controller: onDelete will now remove the controller "+
		"group for namespace %s if it exists", ns.Name)
	c.ControllerManager.RemoveGroup(ns.Name)
}

// isNamespaceInForegroundDeletion determines if a namespace is currently being deleted using
// foreground cascading deletion, as indicated by the presence of value “foregroundDeletion” in
// the namespace's metadata.finalizers.
func isNamespaceInForegroundDeletion(namespace *v1.Namespace) bool {
	for _, finalizer := range namespace.Finalizers {
		if finalizer == meta_v1.FinalizerDeleteDependents {
			return true
		}
	}
	return false
}
