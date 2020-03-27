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
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/controller"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// Controller holds the connections for the controller
type Controller struct {
	NamespaceClient    *rest.RESTClient
	NamespaceClientset *kubernetes.Clientset
	ControllerManager  controller.ManagerInterface
	Informer           coreinformers.NamespaceInformer
}

// NewNamespaceController creates a new namespace controller that will watch for namespace events
// as responds accordingly.  This adding and removing controller groups as namespaces watched by the
// PostgreSQL Operator are added and deleted.
func NewNamespaceController(clients *kubeapi.ControllerClients,
	controllerManager controller.ManagerInterface,
	informer coreinformers.NamespaceInformer) (*Controller, error) {

	controller := &Controller{
		NamespaceClient:    clients.PGORestclient,
		NamespaceClientset: clients.Kubeclientset,
		ControllerManager:  controllerManager,
		Informer:           informer,
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

	log.Debugf("[namespace Controller] OnAdd ns=%s", newNs.ObjectMeta.SelfLink)
	labels := newNs.GetObjectMeta().GetLabels()
	if labels[config.LABEL_VENDOR] != config.LABEL_CRUNCHY || labels[config.LABEL_PGO_INSTALLATION_NAME] != operator.InstallationName {
		log.Debugf("namespace Controller: onAdd skipping namespace that is not crunchydata or not belonging to this Operator installation %s", newNs.ObjectMeta.SelfLink)
		return
	}

	log.Debugf("namespace Controller: onAdd crunchy namespace %s created", newNs.ObjectMeta.SelfLink)
	c.ControllerManager.AddAndRunControllerGroup(newNs.Name)
}

// onUpdate is called when a namespace is updated
func (c *Controller) onUpdate(oldObj, newObj interface{}) {

	newNs := newObj.(*v1.Namespace)

	log.Debugf("[namespace Controller] onUpdate ns=%s", newNs.ObjectMeta.SelfLink)

	labels := newNs.GetObjectMeta().GetLabels()
	if labels[config.LABEL_VENDOR] != config.LABEL_CRUNCHY || labels[config.LABEL_PGO_INSTALLATION_NAME] != operator.InstallationName {
		log.Debugf("namespace Controller: onUpdate skipping namespace that is not crunchydata %s", newNs.ObjectMeta.SelfLink)
		return
	}

	log.Debugf("namespace Controller: onUpdate crunchy namespace updated %s", newNs.ObjectMeta.SelfLink)
	c.ControllerManager.AddAndRunControllerGroup(newNs.Name)
}

func (c *Controller) onDelete(obj interface{}) {

	ns := obj.(*v1.Namespace)

	log.Debugf("[namespace Controller] onDelete ns=%s", ns.ObjectMeta.SelfLink)
	labels := ns.GetObjectMeta().GetLabels()
	if labels[config.LABEL_VENDOR] != config.LABEL_CRUNCHY {
		log.Debugf("namespace Controller: onDelete skipping namespace that is not crunchydata %s", ns.ObjectMeta.SelfLink)
		return
	}

	log.Debugf("namespace Controller: onDelete crunchy operator namespace %s is deleted", ns.ObjectMeta.SelfLink)
	c.ControllerManager.RemoveGroup(ns.Name)
	log.Debugf("namespace Controller: instance removed for ns %s", ns.Name)
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
