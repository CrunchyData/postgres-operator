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
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	upgradeoperator "github.com/crunchydata/postgres-operator/operator/cluster"
	log "github.com/sirupsen/logrus"
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
	Namespace          []string
}

// Run starts an pgupgrade resource controller
func (c *PgupgradeController) Run(ctx context.Context) error {

	err := c.watchPgupgrades(ctx)
	if err != nil {
		log.Errorf("Failed to register watch for Pgupgrade resource: %v", err)
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

// watchPgupgrades is the event loop for pgupgrade resources
func (c *PgupgradeController) watchPgupgrades(ctx context.Context) error {
	for i := 0; i < len(c.Namespace); i++ {
		log.Infof("starting pgtask controller on ns [%s]", c.Namespace[i])

		source := cache.NewListWatchFromClient(
			c.PgupgradeClient,
			crv1.PgupgradeResourcePlural,
			c.Namespace[i],
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
	}
	return nil
}

// onAdd is called when pgupgrades are added
func (c *PgupgradeController) onAdd(obj interface{}) {
	upgrade := obj.(*crv1.Pgupgrade)
	log.Debugf("[PgupgradeCONTROLLER] OnAdd ns=%s %s", upgrade.ObjectMeta.Namespace, upgrade.ObjectMeta.SelfLink)

	//handle the case of an operator restart and to avoid processing
	//pgupgrades already processed
	if upgrade.Status.State == crv1.PgupgradeStateProcessed {
		log.Debug("pgupgrade " + upgrade.ObjectMeta.Name + " already processed")
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
		log.Errorf("ERROR updating pgupgrade status: %s", err.Error())
	}

	//handle the case of adding a pgupgrade
	upgradeoperator.AddUpgrade(c.PgupgradeClientset, c.PgupgradeClient, upgradeCopy, upgrade.ObjectMeta.Namespace)
}

// onUpdate is called when a pgupgrade is updated
func (c *PgupgradeController) onUpdate(oldObj, newObj interface{}) {
}

// onDelete is called when a pgupgrade is deleted
func (c *PgupgradeController) onDelete(obj interface{}) {
	upgrade := obj.(*crv1.Pgupgrade)
	log.Debugf("[PgupgradeController] onDelete ns=%s %s", upgrade.ObjectMeta.Namespace, upgrade.ObjectMeta.SelfLink)

	//handle the case of when a pgupgrade is removed
	upgradeoperator.DeleteUpgrade(c.PgupgradeClientset, c.PgupgradeClient, upgrade, upgrade.ObjectMeta.Namespace)
}
