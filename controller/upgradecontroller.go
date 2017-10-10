package controller

import (
	"context"
	"fmt"
	log "github.com/Sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	crv1 "github.com/crunchydata/kraken/apis/cr/v1"
	upgradeoperator "github.com/crunchydata/kraken/operator/upgrade"
)

// Watcher is an upgrade of watching on resource create/update/delete events
type PgupgradeController struct {
	PgupgradeClient    *rest.RESTClient
	PgupgradeClientset *kubernetes.Clientset
	PgupgradeScheme    *runtime.Scheme
}

// Run starts an Example resource controller
func (c *PgupgradeController) Run(ctx context.Context) error {
	fmt.Print("Watch Pgupgrade objects\n")

	// Watch Example objects
	_, err := c.watchPgupgrades(ctx)
	if err != nil {
		fmt.Printf("Failed to register watch for Pgupgrade resource: %v\n", err)
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

func (c *PgupgradeController) watchPgupgrades(ctx context.Context) (cache.Controller, error) {
	source := cache.NewListWatchFromClient(
		c.PgupgradeClient,
		crv1.PgupgradeResourcePlural,
		apiv1.NamespaceAll,
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

func (c *PgupgradeController) onAdd(obj interface{}) {
	upgrade := obj.(*crv1.Pgupgrade)
	fmt.Printf("[PgupgradeCONTROLLER] OnAdd %s\n", upgrade.ObjectMeta.SelfLink)

	if upgrade.Status.State == crv1.PgupgradeStateProcessed {
		log.Info("pgupgrade " + upgrade.ObjectMeta.Name + " already processed")
		return
	}

	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use upgradeScheme.Copy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	copyObj, err := c.PgupgradeScheme.Copy(upgrade)
	if err != nil {
		fmt.Printf("ERROR creating a deep copy of upgrade object: %v\n", err)
		return
	}

	upgradeCopy := copyObj.(*crv1.Pgupgrade)
	upgradeCopy.Status = crv1.PgupgradeStatus{
		State:   crv1.PgupgradeStateProcessed,
		Message: "Successfully processed Pgupgrade by controller",
	}

	err = c.PgupgradeClient.Put().
		Name(upgrade.ObjectMeta.Name).
		Namespace(upgrade.ObjectMeta.Namespace).
		Resource(crv1.PgupgradeResourcePlural).
		Body(upgradeCopy).
		Do().
		Error()

	if err != nil {
		fmt.Printf("ERROR updating status: %v\n", err)
	} else {
		fmt.Printf("UPDATED status: %#v\n", upgradeCopy)
	}

	upgradeoperator.AddUpgrade(c.PgupgradeClientset, c.PgupgradeClient, upgradeCopy, upgrade.ObjectMeta.Namespace)
}

func (c *PgupgradeController) onUpdate(oldObj, newObj interface{}) {
	//oldExample := oldObj.(*crv1.Pgupgrade)
	//newExample := newObj.(*crv1.Pgupgrade)
	//fmt.Printf("[PgupgradeCONTROLLER] OnUpdate oldObj: %s\n", oldExample.ObjectMeta.SelfLink)
	//fmt.Printf("[PgupgradeCONTROLLER] OnUpdate newObj: %s\n", newExample.ObjectMeta.SelfLink)
}

func (c *PgupgradeController) onDelete(obj interface{}) {
	upgrade := obj.(*crv1.Pgupgrade)
	fmt.Printf("[PgupgradeCONTROLLER] OnDelete %s\n", upgrade.ObjectMeta.SelfLink)
	upgradeoperator.DeleteUpgrade(c.PgupgradeClientset, c.PgupgradeClient, upgrade, upgrade.ObjectMeta.Namespace)
}
