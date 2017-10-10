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
	//policycontroller "github.com/crunchydata/kraken/operator/policy"
)

// Watcher is an policy of watching on resource create/update/delete events
type PgpolicyController struct {
	PgpolicyClient    *rest.RESTClient
	PgpolicyScheme    *runtime.Scheme
	PgpolicyClientset *kubernetes.Clientset
}

// Run starts an Example resource controller
func (c *PgpolicyController) Run(ctx context.Context) error {
	fmt.Print("Watch Pgpolicy objects\n")

	// Watch Example objects
	_, err := c.watchPgpolicys(ctx)
	if err != nil {
		fmt.Printf("Failed to register watch for Pgpolicy resource: %v\n", err)
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

func (c *PgpolicyController) watchPgpolicys(ctx context.Context) (cache.Controller, error) {
	source := cache.NewListWatchFromClient(
		c.PgpolicyClient,
		crv1.PgpolicyResourcePlural,
		apiv1.NamespaceAll,
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
	return controller, nil
}

func (c *PgpolicyController) onAdd(obj interface{}) {
	policy := obj.(*crv1.Pgpolicy)
	fmt.Printf("[PgpolicyCONTROLLER] OnAdd %s\n", policy.ObjectMeta.SelfLink)
	if policy.Status.State == crv1.PgpolicyStateProcessed {
		log.Info("pgpolicy " + policy.ObjectMeta.Name + " already processed")
		return
	}

	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use policyScheme.Copy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	copyObj, err := c.PgpolicyScheme.Copy(policy)
	if err != nil {
		fmt.Printf("ERROR creating a deep copy of policy object: %v\n", err)
		return
	}

	policyCopy := copyObj.(*crv1.Pgpolicy)
	policyCopy.Status = crv1.PgpolicyStatus{
		State:   crv1.PgpolicyStateProcessed,
		Message: "Successfully processed Pgpolicy by controller",
	}

	err = c.PgpolicyClient.Put().
		Name(policy.ObjectMeta.Name).
		Namespace(policy.ObjectMeta.Namespace).
		Resource(crv1.PgpolicyResourcePlural).
		Body(policyCopy).
		Do().
		Error()

	if err != nil {
		fmt.Printf("ERROR updating status: %v\n", err)
	} else {
		fmt.Printf("UPDATED status: %#v\n", policyCopy)
	}
	//policyoperator.AddPolicyBase(c.PgpolicyClientset, c.PgpolicyClient, policyCopy, policy.ObjectMeta.Namespace)
}

func (c *PgpolicyController) onUpdate(oldObj, newObj interface{}) {
	//oldExample := oldObj.(*crv1.Pgpolicy)
	//newExample := newObj.(*crv1.Pgpolicy)
	//fmt.Printf("[PgpolicyCONTROLLER] OnUpdate oldObj: %s\n", oldExample.ObjectMeta.SelfLink)
	//fmt.Printf("[PgpolicyCONTROLLER] OnUpdate newObj: %s\n", newExample.ObjectMeta.SelfLink)
}

func (c *PgpolicyController) onDelete(obj interface{}) {
	policy := obj.(*crv1.Pgpolicy)
	fmt.Printf("[PgpolicyCONTROLLER] OnDelete %s\n", policy.ObjectMeta.SelfLink)
	err := c.PgpolicyClient.Delete().
		Resource(crv1.PgpolicyResourcePlural).
		Namespace(policy.ObjectMeta.Namespace).
		Name(policy.ObjectMeta.Name).
		Do().
		Error()

	if err != nil {
		fmt.Printf("ERROR deleting pgpolicy: %v\n", err)
	} else {
		fmt.Println("DELETED pgpolicy " + policy.ObjectMeta.Name)
	}
}
