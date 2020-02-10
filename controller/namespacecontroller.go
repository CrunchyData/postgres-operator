package controller

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
	"context"

	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/operator"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// NamespaceController holds the connections for the controller
type NamespaceController struct {
	NamespaceClient        *rest.RESTClient
	NamespaceClientset     *kubernetes.Clientset
	Ctx                    context.Context
	ThePodController       *PodController
	TheJobController       *JobController
	ThePgpolicyController  *PgpolicyController
	ThePgbackupController  *PgbackupController
	ThePgreplicaController *PgreplicaController
	ThePgclusterController *PgclusterController
	ThePgtaskController    *PgtaskController
}

// Run starts a namespace resource controller
func (c *NamespaceController) Run() error {

	err := c.watchNamespaces(c.Ctx)
	if err != nil {
		log.Errorf("Failed to register watch for namespace resource: %v", err)
		return err
	}

	<-c.Ctx.Done()
	return c.Ctx.Err()
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

	log.Debugf("[NamespaceController] OnAdd ns=%s", newNs.ObjectMeta.SelfLink)
	labels := newNs.GetObjectMeta().GetLabels()
	if labels[config.LABEL_VENDOR] != config.LABEL_CRUNCHY || labels[config.LABEL_PGO_INSTALLATION_NAME] != operator.InstallationName {
		log.Debugf("NamespaceController: onAdd skipping namespace that is not crunchydata or not belonging to this Operator installation %s", newNs.ObjectMeta.SelfLink)
		return
	} else {
		log.Debugf("NamespaceController: onAdd crunchy namespace %s created", newNs.ObjectMeta.SelfLink)
		c.ThePodController.SetupWatch(newNs.Name)
		c.TheJobController.SetupWatch(newNs.Name)
		c.ThePgpolicyController.SetupWatch(newNs.Name)
		c.ThePgbackupController.SetupWatch(newNs.Name)
		c.ThePgreplicaController.SetupWatch(newNs.Name)
		c.ThePgclusterController.SetupWatch(newNs.Name)
		c.ThePgtaskController.SetupWatch(newNs.Name)
	}

}

// onUpdate is called when a pgcluster is updated
func (c *NamespaceController) onUpdate(oldObj, newObj interface{}) {
	//oldNs := oldObj.(*v1.Namespace)
	newNs := newObj.(*v1.Namespace)
	log.Debugf("[NamespaceController] onUpdate ns=%s", newNs.ObjectMeta.SelfLink)

	labels := newNs.GetObjectMeta().GetLabels()
	if labels[config.LABEL_VENDOR] != config.LABEL_CRUNCHY || labels[config.LABEL_PGO_INSTALLATION_NAME] != operator.InstallationName {
		log.Debugf("NamespaceController: onUpdate skipping namespace that is not crunchydata %s", newNs.ObjectMeta.SelfLink)
		return
	} else {
		log.Debugf("NamespaceController: onUpdate crunchy namespace updated %s", newNs.ObjectMeta.SelfLink)
		c.ThePodController.SetupWatch(newNs.Name)
		c.TheJobController.SetupWatch(newNs.Name)
		c.ThePgpolicyController.SetupWatch(newNs.Name)
		c.ThePgbackupController.SetupWatch(newNs.Name)
		c.ThePgreplicaController.SetupWatch(newNs.Name)
		c.ThePgclusterController.SetupWatch(newNs.Name)
		c.ThePgtaskController.SetupWatch(newNs.Name)
	}

}

func (c *NamespaceController) onDelete(obj interface{}) {
	ns := obj.(*v1.Namespace)

	log.Debugf("[NamespaceController] onDelete ns=%s", ns.ObjectMeta.SelfLink)
	labels := ns.GetObjectMeta().GetLabels()
	if labels[config.LABEL_VENDOR] != config.LABEL_CRUNCHY {
		log.Debugf("NamespaceController: onDelete skipping namespace that is not crunchydata %s", ns.ObjectMeta.SelfLink)
		return
	} else {
		log.Debugf("NamespaceController: onDelete crunchy operator namespace %s is deleted", ns.ObjectMeta.SelfLink)
	}

}
