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
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	upgradeoperator "github.com/crunchydata/postgres-operator/operator/cluster"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// PgupgradeController holds the connections for the controller
type PgupgradeController struct {
	PgupgradeClient    *rest.RESTClient
	PgupgradeClientset *kubernetes.Clientset
	PgupgradeScheme    *runtime.Scheme
	Namespace          string
}

// Run starts an pgupgrade resource controller
func (c *PgupgradeController) Run(ctx context.Context) error {

	_, err := c.watchPgupgrades(ctx)
	if err != nil {
		log.Errorf("Failed to register watch for Pgupgrade resource: %v", err)
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

// watchPgupgrades is the event loop for pgupgrade resources
func (c *PgupgradeController) watchPgupgrades(ctx context.Context) (cache.Controller, error) {
	source := cache.NewListWatchFromClient(
		c.PgupgradeClient,
		crv1.PgupgradeResourcePlural,
		c.Namespace,
		fields.Everything())

	_, controller := cache.NewInformer(
		source,

		// The object type.
		&crv1.Pgupgrade{},

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

// onAdd is called when pgupgrades are added
func (c *PgupgradeController) onAdd(obj interface{}) {
	upgrade := obj.(*crv1.Pgupgrade)
	log.Debugf("[PgupgradeCONTROLLER] OnAdd %s", upgrade.ObjectMeta.SelfLink)

	if upgrade.Status.State == crv1.PgupgradeStateProcessed {
		log.Info("pgupgrade " + upgrade.ObjectMeta.Name + " already processed")
		return
	}

	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use upgradeScheme.Copy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	copyObj := upgrade.DeepCopyObject()
	upgradeCopy := copyObj.(*crv1.Pgupgrade)

	upgradeCopy.Status = crv1.PgupgradeStatus{
		State:   crv1.PgupgradeStateProcessed,
		Message: "Successfully processed Pgupgrade by controller",
	}

	err := c.PgupgradeClient.Put().
		Name(upgrade.ObjectMeta.Name).
		Namespace(upgrade.ObjectMeta.Namespace).
		Resource(crv1.PgupgradeResourcePlural).
		Body(upgradeCopy).
		Do().
		Error()

	if err != nil {
		log.Errorf("ERROR updating status: %v", err)
	} else {
		log.Debugf("UPDATED status: %#v", upgradeCopy)
	}

	upgradeoperator.AddUpgrade(c.PgupgradeClientset, c.PgupgradeClient, upgradeCopy, upgrade.ObjectMeta.Namespace)
}

// onUpdate is called when a pgupgrade is updated
func (c *PgupgradeController) onUpdate(oldObj, newObj interface{}) {
}

// onDelete is called when a pgupgrade is deleted
func (c *PgupgradeController) onDelete(obj interface{}) {
	upgrade := obj.(*crv1.Pgupgrade)
	log.Debugf("[PgupgradeCONTROLLER] OnDelete %s", upgrade.ObjectMeta.SelfLink)
	upgradeoperator.DeleteUpgrade(c.PgupgradeClientset, c.PgupgradeClient, upgrade, upgrade.ObjectMeta.Namespace)
}
