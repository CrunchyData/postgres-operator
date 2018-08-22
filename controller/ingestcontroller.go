package controller

/*
Copyright 2017-2018 Crunchy Data Solutions, Inc.
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
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	ingestoperator "github.com/crunchydata/postgres-operator/operator/ingest"
)

// PgingestController holds the connections for the controller
type PgingestController struct {
	PgingestClient    *rest.RESTClient
	PgingestScheme    *runtime.Scheme
	PgingestClientset *kubernetes.Clientset
	Namespace         string
}

// Run starts an pgcluster resource controller
func (c *PgingestController) Run(ctx context.Context) error {

	_, err := c.watchPgingests(ctx)
	if err != nil {
		log.Errorf("Failed to register watch for Pgingest resource: %v", err)
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

// watchPgingests is the event loop for pgingest resources
func (c *PgingestController) watchPgingests(ctx context.Context) (cache.Controller, error) {
	source := cache.NewListWatchFromClient(
		c.PgingestClient,
		crv1.PgingestResourcePlural,
		c.Namespace,
		fields.Everything())

	_, controller := cache.NewInformer(
		source,

		// The object type.
		&crv1.Pgingest{},

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

// onAdd is called when a pgingest is added
func (c *PgingestController) onAdd(obj interface{}) {
	ingest := obj.(*crv1.Pgingest)
	log.Infof("[PgingestCONTROLLER] OnAdd %s", ingest.ObjectMeta.SelfLink)
	if ingest.Status.State == crv1.PgingestStateProcessed {
		log.Info("pgingest " + ingest.ObjectMeta.Name + " already processed")
		return
	}

	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use clusterScheme.Copy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	copyObj := ingest.DeepCopyObject()
	ingestCopy := copyObj.(*crv1.Pgingest)

	ingestCopy.Status = crv1.PgingestStatus{
		State:   crv1.PgingestStateProcessed,
		Message: "Successfully processed Pgingest by controller",
	}

	err := c.PgingestClient.Put().
		Name(ingest.ObjectMeta.Name).
		Namespace(ingest.ObjectMeta.Namespace).
		Resource(crv1.PgingestResourcePlural).
		Body(ingestCopy).
		Do().
		Error()

	if err != nil {
		log.Errorf("ERROR updating ingest status: %v", err)
	}

	ingestoperator.CreateIngest(ingest.ObjectMeta.Namespace, c.PgingestClientset, c.PgingestClient, ingestCopy)
}

// onUpdate is called when a pgingest is updated
func (c *PgingestController) onUpdate(oldObj, newObj interface{}) {
	log.Debug("onUpdate pgingest CRD called")

}

// onDelete is called when a pgingest is deleted
func (c *PgingestController) onDelete(obj interface{}) {
	ingest := obj.(*crv1.Pgingest)
	log.Infof("[PgingestCONTROLLER] OnDelete %s", ingest.ObjectMeta.SelfLink)
	ingestoperator.Delete(c.PgingestClientset, ingest.Spec.Name, ingest.ObjectMeta.Namespace)
}
