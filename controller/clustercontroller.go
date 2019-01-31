package controller

/*
Copyright 2017 Crunchy Data Solutions, Inc.
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
	clusteroperator "github.com/crunchydata/postgres-operator/operator/cluster"
	"github.com/crunchydata/postgres-operator/util"
)

// PgclusterController holds the connections for the controller
type PgclusterController struct {
	PgclusterClient    *rest.RESTClient
	PgclusterScheme    *runtime.Scheme
	PgclusterClientset *kubernetes.Clientset
	Namespace          string
}

// Run starts an pgcluster resource controller
func (c *PgclusterController) Run(ctx context.Context) error {
	log.Debug("Watch Pgcluster objects")

	_, err := c.watchPgclusters(ctx)
	if err != nil {
		log.Errorf("Failed to register watch for Pgcluster resource: %v", err)
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

// watchPgclusters is the event loop for pgcluster resources
func (c *PgclusterController) watchPgclusters(ctx context.Context) (cache.Controller, error) {
	source := cache.NewListWatchFromClient(
		c.PgclusterClient,
		crv1.PgclusterResourcePlural,
		c.Namespace,
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

// onAdd is called when a pgcluster is added
func (c *PgclusterController) onAdd(obj interface{}) {
	cluster := obj.(*crv1.Pgcluster)
	log.Debugf("[PgclusterController] ns %s onAdd %s", cluster.ObjectMeta.Namespace, cluster.ObjectMeta.SelfLink)

	//handle the case when the operator restarts and don't
	//process already processed pgclusters
	if cluster.Status.State == crv1.PgclusterStateProcessed {
		log.Debug("pgcluster " + cluster.ObjectMeta.Name + " already processed")
		return
	}

	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use clusterScheme.Copy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	copyObj := cluster.DeepCopyObject()
	clusterCopy := copyObj.(*crv1.Pgcluster)

	clusterCopy.Status = crv1.PgclusterStatus{
		State:   crv1.PgclusterStateProcessed,
		Message: "Successfully processed Pgcluster by controller",
	}

	err := c.PgclusterClient.Put().
		Name(cluster.ObjectMeta.Name).
		Namespace(cluster.ObjectMeta.Namespace).
		Resource(crv1.PgclusterResourcePlural).
		Body(clusterCopy).
		Do().
		Error()

	if err != nil {
		log.Errorf("ERROR updating pgcluster status on add: %s", err.Error())
	}

	log.Debugf("pgcluster added: %s", cluster.ObjectMeta.Name)

	clusteroperator.AddClusterBase(c.PgclusterClientset, c.PgclusterClient, clusterCopy, cluster.ObjectMeta.Namespace)
}

// onUpdate is called when a pgcluster is updated
func (c *PgclusterController) onUpdate(oldObj, newObj interface{}) {
	oldcluster := oldObj.(*crv1.Pgcluster)
	newcluster := newObj.(*crv1.Pgcluster)
	log.Debugf("pgcluster ns=%s %s onUpdate", newcluster.ObjectMeta.Namespace, newcluster.ObjectMeta.Name)

	//handle the case for when the autofail lable is updated
	if oldcluster.ObjectMeta.Labels[util.LABEL_AUTOFAIL] != "" {
		log.Debugf("pgcluster update %s autofail old %s new %s", oldcluster.Name, oldcluster.ObjectMeta.Labels[util.LABEL_AUTOFAIL], newcluster.ObjectMeta.Labels[util.LABEL_AUTOFAIL])
	}

}

// onDelete is called when a pgcluster is deleted
func (c *PgclusterController) onDelete(obj interface{}) {
	cluster := obj.(*crv1.Pgcluster)
	log.Debugf("[PgclusterController] ns=%s onDelete %s", cluster.ObjectMeta.Namespace, cluster.ObjectMeta.SelfLink)

	//handle pgcluster cleanup
	clusteroperator.DeleteClusterBase(c.PgclusterClientset, c.PgclusterClient, cluster, cluster.ObjectMeta.Namespace)
}
