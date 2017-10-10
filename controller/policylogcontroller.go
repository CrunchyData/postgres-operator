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
	policylogoperator "github.com/crunchydata/kraken/operator/cluster"
)

// Watcher is an policylog of watching on resource create/update/delete events
type PgpolicylogController struct {
	PgpolicylogClientset *kubernetes.Clientset
	PgpolicylogClient    *rest.RESTClient
	PgpolicylogScheme    *runtime.Scheme
}

// Run starts an Example resource controller
func (c *PgpolicylogController) Run(ctx context.Context) error {
	fmt.Print("Watch Pgpolicylog objects\n")

	// Watch Example objects
	_, err := c.watchPgpolicylogs(ctx)
	if err != nil {
		fmt.Printf("Failed to register watch for Pgpolicylog resource: %v\n", err)
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

func (c *PgpolicylogController) watchPgpolicylogs(ctx context.Context) (cache.Controller, error) {
	source := cache.NewListWatchFromClient(
		c.PgpolicylogClient,
		crv1.PgpolicylogResourcePlural,
		apiv1.NamespaceAll,
		fields.Everything())

	_, controller := cache.NewInformer(
		source,

		// The object type.
		&crv1.Pgpolicylog{},

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

func (c *PgpolicylogController) onAdd(obj interface{}) {
	policylog := obj.(*crv1.Pgpolicylog)
	fmt.Printf("[PgpolicylogCONTROLLER] OnAdd %s\n", policylog.ObjectMeta.SelfLink)
	if policylog.Status.State == crv1.PgpolicylogStateProcessed {
		log.Infoln("pgpolicylog " + policylog.ObjectMeta.Name + " already processed")
		return
	}

	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use policylogScheme.Copy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	copyObj, err := c.PgpolicylogScheme.Copy(policylog)
	if err != nil {
		fmt.Printf("ERROR creating a deep copy of policylog object: %v\n", err)
		return
	}

	policylogCopy := copyObj.(*crv1.Pgpolicylog)
	policylogCopy.Status = crv1.PgpolicylogStatus{
		State:   crv1.PgpolicylogStateProcessed,
		Message: "Successfully processed Pgpolicylog by controller",
	}

	err = c.PgpolicylogClient.Put().
		Name(policylog.ObjectMeta.Name).
		Namespace(policylog.ObjectMeta.Namespace).
		Resource(crv1.PgpolicylogResourcePlural).
		Body(policylogCopy).
		Do().
		Error()

	if err != nil {
		fmt.Printf("ERROR updating status: %v\n", err)
	} else {
		fmt.Printf("UPDATED status: %#v\n", policylogCopy)
	}

	policylogoperator.AddPolicylog(c.PgpolicylogClientset, c.PgpolicylogClient, policylogCopy, policylog.ObjectMeta.Namespace)
}

func (c *PgpolicylogController) onUpdate(oldObj, newObj interface{}) {
	//oldExample := oldObj.(*crv1.Pgpolicylog)
	//newExample := newObj.(*crv1.Pgpolicylog)
	//fmt.Printf("[PgpolicylogCONTROLLER] OnUpdate oldObj: %s\n", oldExample.ObjectMeta.SelfLink)
	//fmt.Printf("[PgpolicylogCONTROLLER] OnUpdate newObj: %s\n", newExample.ObjectMeta.SelfLink)
}

func (c *PgpolicylogController) onDelete(obj interface{}) {
	policylog := obj.(*crv1.Pgpolicylog)
	fmt.Printf("[PgpolicylogCONTROLLER] OnDelete %s\n", policylog.ObjectMeta.SelfLink)
}
