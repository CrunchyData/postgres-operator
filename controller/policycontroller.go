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
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
)

// PgpolicyController holds connections for the controller
type PgpolicyController struct {
	PgpolicyClient    *rest.RESTClient
	PgpolicyScheme    *runtime.Scheme
	PgpolicyClientset *kubernetes.Clientset
	Namespace         []string
}

// Run starts an pgpolicy resource controller
func (c *PgpolicyController) Run(ctx context.Context) error {

	// Watch Example objects
	err := c.watchPgpolicys(ctx)
	if err != nil {
		log.Errorf("Failed to register watch for Pgpolicy resource: %v", err)
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

// watchPgpolicys watches the pgpolicy resource catching events
func (c *PgpolicyController) watchPgpolicys(ctx context.Context) error {
	for i := 0; i < len(c.Namespace); i++ {
		log.Infof("starting pgpolicy controller on ns [%s]", c.Namespace[i])

		source := cache.NewListWatchFromClient(
			c.PgpolicyClient,
			crv1.PgpolicyResourcePlural,
			c.Namespace[i],
			fields.Everything())

		_, controller := cache.NewInformer(
			source,

			// The object type.
			&crv1.Pgpolicy{},

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
	}
	return nil
}

// onAdd is called when a pgpolicy is added
func (c *PgpolicyController) onAdd(obj interface{}) {
	policy := obj.(*crv1.Pgpolicy)
	log.Debugf("[PgpolicyController] onAdd ns=%s %s", policy.ObjectMeta.Namespace, policy.ObjectMeta.SelfLink)

	//handle the case of when a pgpolicy is already processed, which
	//is the case when the operator restarts
	if policy.Status.State == crv1.PgpolicyStateProcessed {
		log.Debug("pgpolicy " + policy.ObjectMeta.Name + " already processed")
		return
	}

	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use policyScheme.Copy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	copyObj := policy.DeepCopyObject()
	policyCopy := copyObj.(*crv1.Pgpolicy)

	policyCopy.Status = crv1.PgpolicyStatus{
		State:   crv1.PgpolicyStateProcessed,
		Message: "Successfully processed Pgpolicy by controller",
	}

	err := c.PgpolicyClient.Put().
		Name(policy.ObjectMeta.Name).
		Namespace(policy.ObjectMeta.Namespace).
		Resource(crv1.PgpolicyResourcePlural).
		Body(policyCopy).
		Do().
		Error()

	if err != nil {
		log.Errorf("ERROR updating pgpolicy status: %s", err.Error())
	}

}

// onUpdate is called when a pgpolicy is updated
func (c *PgpolicyController) onUpdate(oldObj, newObj interface{}) {
}

// onDelete is called when a pgpolicy is deleted
func (c *PgpolicyController) onDelete(obj interface{}) {
	policy := obj.(*crv1.Pgpolicy)
	log.Debugf("[PgpolicyController] onDelete ns=%s %s", policy.ObjectMeta.Namespace, policy.ObjectMeta.SelfLink)

	log.Debugf("DELETED pgpolicy %s", policy.ObjectMeta.Name)
}
