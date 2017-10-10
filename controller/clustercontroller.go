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
	clusteroperator "github.com/crunchydata/kraken/operator/cluster"
)

// Watcher is an cluster of watching on resource create/update/delete events
type PgclusterController struct {
	PgclusterClient    *rest.RESTClient
	PgclusterScheme    *runtime.Scheme
	PgclusterClientset *kubernetes.Clientset
}

// Run starts an Example resource controller
func (c *PgclusterController) Run(ctx context.Context) error {
	fmt.Print("Watch Pgcluster objects\n")

	// Watch Example objects
	_, err := c.watchPgclusters(ctx)
	if err != nil {
		fmt.Printf("Failed to register watch for Pgcluster resource: %v\n", err)
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

func (c *PgclusterController) watchPgclusters(ctx context.Context) (cache.Controller, error) {
	source := cache.NewListWatchFromClient(
		c.PgclusterClient,
		crv1.PgclusterResourcePlural,
		apiv1.NamespaceAll,
		fields.Everything())

	_, controller := cache.NewInformer(
		source,

		// The object type.
		&crv1.Pgcluster{},

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

func (c *PgclusterController) onAdd(obj interface{}) {
	cluster := obj.(*crv1.Pgcluster)
	fmt.Printf("[PgclusterCONTROLLER] OnAdd %s\n", cluster.ObjectMeta.SelfLink)
	if cluster.Status.State == crv1.PgclusterStateProcessed {
		log.Info("pgcluster " + cluster.ObjectMeta.Name + " already processed")
		return
	}

	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use clusterScheme.Copy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	copyObj, err := c.PgclusterScheme.Copy(cluster)
	if err != nil {
		fmt.Printf("ERROR creating a deep copy of cluster object: %v\n", err)
		return
	}

	clusterCopy := copyObj.(*crv1.Pgcluster)
	clusterCopy.Status = crv1.PgclusterStatus{
		State:   crv1.PgclusterStateProcessed,
		Message: "Successfully processed Pgcluster by controller",
	}

	err = c.PgclusterClient.Put().
		Name(cluster.ObjectMeta.Name).
		Namespace(cluster.ObjectMeta.Namespace).
		Resource(crv1.PgclusterResourcePlural).
		Body(clusterCopy).
		Do().
		Error()

	if err != nil {
		fmt.Printf("ERROR updating status: %v\n", err)
	} else {
		fmt.Printf("UPDATED status: %#v\n", clusterCopy)
	}

	clusteroperator.AddClusterBase(c.PgclusterClientset, c.PgclusterClient, clusterCopy, cluster.ObjectMeta.Namespace)
}

func (c *PgclusterController) onUpdate(oldObj, newObj interface{}) {
	oldExample := oldObj.(*crv1.Pgcluster)
	newExample := newObj.(*crv1.Pgcluster)
	//fmt.Printf("[PgclusterCONTROLLER] OnUpdate oldObj: %s\n", oldExample.ObjectMeta.SelfLink)
	//fmt.Printf("[PgclusterCONTROLLER] OnUpdate newObj: %s\n", newExample.ObjectMeta.SelfLink)

	//look for scale commands
	clusteroperator.ScaleCluster(c.PgclusterClientset, c.PgclusterClient, newExample, oldExample, oldExample.ObjectMeta.Namespace)
}

func (c *PgclusterController) onDelete(obj interface{}) {
	cluster := obj.(*crv1.Pgcluster)
	fmt.Printf("[PgclusterCONTROLLER] OnDelete %s\n", cluster.ObjectMeta.SelfLink)
	clusteroperator.DeleteClusterBase(c.PgclusterClientset, c.PgclusterClient, cluster, cluster.ObjectMeta.Namespace)
}
