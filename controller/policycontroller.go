package controller

/*
Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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
	"sync"
	"time"

	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/ns"
	"github.com/crunchydata/postgres-operator/operator"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/events"
)

// PgpolicyController holds connections for the controller
type PgpolicyController struct {
	PgpolicyClient     *rest.RESTClient
	PgpolicyScheme     *runtime.Scheme
	PgpolicyClientset  *kubernetes.Clientset
	Ctx                context.Context
	informerNsMutex    sync.Mutex
	InformerNamespaces map[string]struct{}
}

// Run starts an pgpolicy resource controller
func (c *PgpolicyController) Run() error {

	// Watch Example objects
	err := c.watchPgpolicys(c.Ctx)
	if err != nil {
		log.Errorf("Failed to register watch for Pgpolicy resource: %v", err)
		return err
	}

	<-c.Ctx.Done()
	return c.Ctx.Err()
}

// watchPgpolicys watches the pgpolicy resource catching events
func (c *PgpolicyController) watchPgpolicys(ctx context.Context) error {
	nsList := ns.GetNamespaces(c.PgpolicyClientset, operator.InstallationName)

	for i := 0; i < len(nsList); i++ {
		log.Infof("starting pgpolicy controller on ns [%s]", nsList[i])

		c.SetupWatch(nsList[i])
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

	state := crv1.PgpolicyStateProcessed
	message := "Successfully processed Pgpolicy by controller"
	err := kubeapi.PatchpgpolicyStatus(c.PgpolicyClient, state, message, policyCopy, policy.ObjectMeta.Namespace)
	if err != nil {
		log.Errorf("ERROR updating pgpolicy status: %s", err.Error())
	}

	//publish event
	topics := make([]string, 1)
	topics[0] = events.EventTopicPolicy

	f := events.EventCreatePolicyFormat{
		EventHeader: events.EventHeader{
			Namespace: policy.ObjectMeta.Namespace,
			Username:  policy.ObjectMeta.Labels[config.LABEL_PGOUSER],
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventCreatePolicy,
		},
		Policyname: policy.ObjectMeta.Name,
	}

	err = events.Publish(f)
	if err != nil {
		log.Error(err.Error())
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

	//publish event
	topics := make([]string, 1)
	topics[0] = events.EventTopicPolicy

	f := events.EventDeletePolicyFormat{
		EventHeader: events.EventHeader{
			Namespace: policy.ObjectMeta.Namespace,
			Username:  policy.ObjectMeta.Labels[config.LABEL_PGOUSER],
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventDeletePolicy,
		},
		Policyname: policy.ObjectMeta.Name,
	}

	err := events.Publish(f)
	if err != nil {
		log.Error(err.Error())
	}

}
func (c *PgpolicyController) SetupWatch(ns string) {

	// don't create informer for namespace if one has already been created
	c.informerNsMutex.Lock()
	defer c.informerNsMutex.Unlock()
	if _, ok := c.InformerNamespaces[ns]; ok {
		return
	}
	c.InformerNamespaces[ns] = struct{}{}

	source := cache.NewListWatchFromClient(
		c.PgpolicyClient,
		crv1.PgpolicyResourcePlural,
		ns,
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

	go controller.Run(c.Ctx.Done())
	log.Debugf("PgpolicyController: created informer for namespace %s", ns)
}
