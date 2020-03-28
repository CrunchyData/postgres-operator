package pgcluster

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
	"fmt"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"

	crv1 "github.com/crunchydata/postgres-operator/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/util"

	clusteroperator "github.com/crunchydata/postgres-operator/operator/cluster"
	informers "github.com/crunchydata/postgres-operator/pkg/generated/informers/externalversions/crunchydata.com/v1"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Controller holds the connections for the controller
type Controller struct {
	PgclusterClient    *rest.RESTClient
	PgclusterClientset *kubernetes.Clientset
	Queue              workqueue.RateLimitingInterface
	Informer           informers.PgclusterInformer
}

// onAdd is called when a pgcluster is added
func (c *Controller) onAdd(obj interface{}) {
	cluster := obj.(*crv1.Pgcluster)
	log.Debugf("[pgcluster Controller] ns %s onAdd %s", cluster.ObjectMeta.Namespace, cluster.ObjectMeta.SelfLink)

	//handle the case when the operator restarts and don't
	//process already processed pgclusters
	if cluster.Status.State == crv1.PgclusterStateProcessed {
		log.Debug("pgcluster " + cluster.ObjectMeta.Name + " already processed")
		return
	}

	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err == nil {
		log.Debugf("cluster putting key in queue %s", key)
		c.Queue.Add(key)
	}

}

func (c *Controller) RunWorker() {

	//process the 'add' work queue forever
	for c.processNextItem() {
	}
}

func (c *Controller) processNextItem() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.Queue.Get()
	if quit {
		return false
	}

	log.Debugf("working on %s", key.(string))
	keyParts := strings.Split(key.(string), "/")
	keyNamespace := keyParts[0]
	keyResourceName := keyParts[1]

	log.Debugf("cluster add queue got key ns=[%s] resource=[%s]", keyNamespace, keyResourceName)

	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two pods with the same key are never processed in
	// parallel.
	defer c.Queue.Done(key)

	// Invoke the method containing the business logic
	// in this case, the de-dupe logic is to test whether a cluster
	// deployment exists , if so, then we don't create another
	_, found, err := kubeapi.GetDeployment(c.PgclusterClientset, keyResourceName, keyNamespace)

	if found {
		log.Debugf("cluster add - dep already found, not creating again")
		return true
	}

	//get the pgcluster
	cluster := crv1.Pgcluster{}
	found, err = kubeapi.Getpgcluster(c.PgclusterClient, &cluster, keyResourceName, keyNamespace)
	if !found {
		log.Debugf("cluster add - pgcluster not found, this is invalid")
		return false
	}

	addIdentifier(&cluster)

	state := crv1.PgclusterStateProcessed
	message := "Successfully processed Pgcluster by controller"
	err = kubeapi.PatchpgclusterStatus(c.PgclusterClient, state, message, &cluster, keyNamespace)
	if err != nil {
		log.Errorf("ERROR updating pgcluster status on add: %s", err.Error())
		return false
	}

	log.Debugf("pgcluster added: %s", cluster.ObjectMeta.Name)

	clusteroperator.AddClusterBase(c.PgclusterClientset, c.PgclusterClient, &cluster, cluster.ObjectMeta.Namespace)

	return true
}

// onUpdate is called when a pgcluster is updated
func (c *Controller) onUpdate(oldObj, newObj interface{}) {
	oldcluster := oldObj.(*crv1.Pgcluster)
	newcluster := newObj.(*crv1.Pgcluster)
	//	log.Debugf("pgcluster ns=%s %s onUpdate", newcluster.ObjectMeta.Namespace, newcluster.ObjectMeta.Name)

	// if the 'shutdown' parameter in the pgcluster update shows that the cluster should be either
	// shutdown or started but its current status does not properly reflect that it is, then
	// proceed with the logic needed to either shutdown or start the cluster
	if newcluster.Spec.Shutdown && newcluster.Status.State != crv1.PgclusterStateShutdown {
		clusteroperator.ShutdownCluster(c.PgclusterClientset, c.PgclusterClient, *newcluster)
	} else if !newcluster.Spec.Shutdown &&
		newcluster.Status.State != crv1.PgclusterStateInitialized {
		clusteroperator.StartupCluster(c.PgclusterClientset, *newcluster)
	}

	// check to see if the "autofail" label on the pgcluster CR has been changed from either true to false, or from
	// false to true.  If it has been changed to false, autofail will then be disabled in the pg cluster.  If has
	// been changed to true, autofail will then be enabled in the pg cluster
	if newcluster.ObjectMeta.Labels[config.LABEL_AUTOFAIL] != "" {
		autofailEnabledOld, err := strconv.ParseBool(oldcluster.ObjectMeta.Labels[config.LABEL_AUTOFAIL])
		if err != nil {
			log.Error(err)
			return
		}
		autofailEnabledNew, err := strconv.ParseBool(newcluster.ObjectMeta.Labels[config.LABEL_AUTOFAIL])
		if err != nil {
			log.Error(err)
			return
		}
		if autofailEnabledNew != autofailEnabledOld {
			util.ToggleAutoFailover(c.PgclusterClientset, autofailEnabledNew,
				newcluster.ObjectMeta.Labels[config.LABEL_PGHA_SCOPE],
				newcluster.ObjectMeta.Namespace)
		}

	}

	// handle standby being enabled and disabled for the cluster
	if oldcluster.Spec.Standby && !newcluster.Spec.Standby {
		if err := clusteroperator.DisableStandby(c.PgclusterClientset, *newcluster); err != nil {
			log.Error(err)
			return
		}
	} else if !oldcluster.Spec.Standby && newcluster.Spec.Standby {
		if err := clusteroperator.EnableStandby(c.PgclusterClientset, *newcluster); err != nil {
			log.Error(err)
			return
		}
	}

	// see if any of the resource values have changed, and if so, upate them
	if oldcluster.Spec.ContainerResources.RequestsCPU != newcluster.Spec.ContainerResources.RequestsCPU ||
		oldcluster.Spec.ContainerResources.RequestsMemory != newcluster.Spec.ContainerResources.RequestsMemory {
		if err := updateResources(c, newcluster); err != nil {
			log.Error(err)
			return
		}
	}

	// if we are not in a standby state, check to see if the tablespaces have
	// differed, and if so, add the additional volumes to the primary and replicas
	if !reflect.DeepEqual(oldcluster.Spec.TablespaceMounts, newcluster.Spec.TablespaceMounts) {
		if err := updateTablespaces(c, oldcluster, newcluster); err != nil {
			log.Error(err)
			return
		}
	}
}

// onDelete is called when a pgcluster is deleted
func (c *Controller) onDelete(obj interface{}) {
	//cluster := obj.(*crv1.Pgcluster)
	//	log.Debugf("[Controller] ns=%s onDelete %s", cluster.ObjectMeta.Namespace, cluster.ObjectMeta.SelfLink)

	//handle pgcluster cleanup
	//	clusteroperator.DeleteClusterBase(c.PgclusterClientset, c.PgclusterClient, cluster, cluster.ObjectMeta.Namespace)
}

// AddPGClusterEventHandler adds the pgcluster event handler to the pgcluster informer
func (c *Controller) AddPGClusterEventHandler() {

	c.Informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	})

	log.Debugf("pgcluster Controller: added event handler to informer")
}

func addIdentifier(clusterCopy *crv1.Pgcluster) {
	u, err := ioutil.ReadFile("/proc/sys/kernel/random/uuid")
	if err != nil {
		log.Error(err)
	}

	clusterCopy.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER] = string(u[:len(u)-1])
}

// updateResources updates the PostgreSQL instance Deployments to reflect the
// update resources (i.e. CPU, memory)
func updateResources(c *Controller, cluster *crv1.Pgcluster) error {
	// put the resources in their proper format for updating the cluster
	// the "remove*" bit lets us know that we will have the request be "unbounded"
	cpu, err := resource.ParseQuantity(cluster.Spec.ContainerResources.RequestsCPU)
	removeCPU := err != nil

	memory, err := resource.ParseQuantity(cluster.Spec.ContainerResources.RequestsMemory)
	removeMemory := err != nil

	// get a list of all of the instance deployments for the cluster
	deployments, err := operator.GetInstanceDeployments(c.PgclusterClientset, cluster)

	if err != nil {
		return err
	}

	// iterate through each PostgreSQL instnace deployment and update the
	// resources values for the database container
	//
	// NOTE: a future version (near future) will first try to detect the primary
	// so that all the replicas are updated first, and then the primary gets the
	// update
	for _, deployment := range deployments.Items {
		// NOTE: this works as the "database" container is always first
		// first handle the CPU update
		if removeCPU {
			delete(deployment.Spec.Template.Spec.Containers[0].Resources.Requests, v1.ResourceCPU)
		} else {
			deployment.Spec.Template.Spec.Containers[0].Resources.Requests[v1.ResourceCPU] = cpu
		}

		// regardless, ensure the limit is gone
		delete(deployment.Spec.Template.Spec.Containers[0].Resources.Limits, v1.ResourceCPU)

		// handle the memory update
		if removeMemory {
			delete(deployment.Spec.Template.Spec.Containers[0].Resources.Requests, v1.ResourceMemory)
		} else {
			deployment.Spec.Template.Spec.Containers[0].Resources.Requests[v1.ResourceMemory] = memory
		}

		// regardless, ensure the limit is gone
		delete(deployment.Spec.Template.Spec.Containers[0].Resources.Limits, v1.ResourceMemory)

		// update the deployment with the new values
		if err := kubeapi.UpdateDeployment(c.PgclusterClientset, &deployment); err != nil {
			return err
		}
	}

	return nil
}

// updateTablespaces updates the PostgreSQL instance Deployments to reflect the
// new PostgreSQL tablespaces that should be added
func updateTablespaces(c *Controller, oldCluster *crv1.Pgcluster, newCluster *crv1.Pgcluster) error {
	// to help the Operator function do less work, we will get a list of new
	// tablespaces. Though these are already present in the CRD, this will isolate
	// exactly which PVCs need to be created
	//
	// To do this, iterate through the the tablespace mount map that is present in
	// the new cluster.
	newTablespaces := map[string]crv1.PgStorageSpec{}

	for tablespaceName, storageSpec := range newCluster.Spec.TablespaceMounts {
		// if the tablespace does not exist in the old version of the cluster,
		// then add it in!
		if _, ok := oldCluster.Spec.TablespaceMounts[tablespaceName]; !ok {
			log.Debugf("new tablespace found: [%s]", tablespaceName)

			newTablespaces[tablespaceName] = storageSpec
		}
	}

	// alright, update the tablespace entries for this cluster!
	// if it returns an error, pass the error back up to the caller
	if err := clusteroperator.UpdateTablespaces(c.PgclusterClientset, newCluster, newTablespaces); err != nil {
		return err
	}

	return nil
}
