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
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"

	crv1 "github.com/crunchydata/postgres-operator/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	backrestoperator "github.com/crunchydata/postgres-operator/operator/backrest"
	clusteroperator "github.com/crunchydata/postgres-operator/operator/cluster"
	informers "github.com/crunchydata/postgres-operator/pkg/generated/informers/externalversions/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/util"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Controller holds the connections for the controller
type Controller struct {
	PgclusterClient    *rest.RESTClient
	PgclusterClientset *kubernetes.Clientset
	PgclusterConfig    *rest.Config
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

// RunWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) RunWorker(stopCh <-chan struct{}, doneCh chan<- struct{}) {

	go c.waitForShutdown(stopCh)

	for c.processNextItem() {
	}

	log.Debug("pgcluster Contoller: worker queue has been shutdown, writing to the done channel")
	doneCh <- struct{}{}
}

// waitForShutdown waits for a message on the stop channel and then shuts down the work queue
func (c *Controller) waitForShutdown(stopCh <-chan struct{}) {
	<-stopCh
	c.Queue.ShutDown()
	log.Debug("pgcluster Contoller: recieved stop signal, worker queue told to shutdown")
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

	// see if any of the resource values have changed, and if so, update them
	if oldcluster.Spec.Resources[v1.ResourceCPU] != newcluster.Spec.Resources[v1.ResourceCPU] ||
		oldcluster.Spec.Resources[v1.ResourceMemory] != newcluster.Spec.Resources[v1.ResourceMemory] {
		if err := clusteroperator.UpdateResources(c.PgclusterClientset, c.PgclusterConfig, newcluster); err != nil {
			log.Error(err)
			return
		}
	}

	// see if any of the pgBackRest repository resource values have changed, and
	// if so, update them
	if oldcluster.Spec.BackrestResources[v1.ResourceCPU] != newcluster.Spec.BackrestResources[v1.ResourceCPU] ||
		oldcluster.Spec.BackrestResources[v1.ResourceMemory] != newcluster.Spec.BackrestResources[v1.ResourceMemory] {
		if err := backrestoperator.UpdateResources(c.PgclusterClientset, c.PgclusterConfig, newcluster); err != nil {
			log.Error(err)
			return
		}
	}

	// see if any of the pgBouncer values have changed, and if so, update the
	// pgBouncer deployment
	if !reflect.DeepEqual(oldcluster.Spec.PgBouncer, newcluster.Spec.PgBouncer) {
		if err := updatePgBouncer(c, oldcluster, newcluster); err != nil {
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

// updatePgBouncer updates the pgBouncer Deployment to reflect any changes that
// may be made, which include:
// - enabling a pgBouncer Deployment :)
// - disabling a pgBouncer Deployment :(
// - any changes to the resizing, etc.
func updatePgBouncer(c *Controller, oldCluster *crv1.Pgcluster, newCluster *crv1.Pgcluster) error {
	log.Debugf("update pgbouncer for cluster %s", newCluster.Name)

	// first, handle the easy ones, i.e. determine if we are enabling or disabling
	if oldCluster.Spec.PgBouncer.Enabled() != newCluster.Spec.PgBouncer.Enabled() {
		log.Debugf("pgbouncer enabled: %t", newCluster.Spec.PgBouncer.Enabled())

		// if this is being enabled, it's a simple step where we can return here
		if newCluster.Spec.PgBouncer.Enabled() {
			return clusteroperator.AddPgbouncer(c.PgclusterClientset, c.PgclusterClient, c.PgclusterConfig, newCluster)
		}

		// if we're not enabled, we're disabled
		return clusteroperator.DeletePgbouncer(c.PgclusterClientset, c.PgclusterClient, c.PgclusterConfig, newCluster)
	}

	// otherwise, this is an update
	return clusteroperator.UpdatePgbouncer(c.PgclusterClientset, c.PgclusterClient, oldCluster, newCluster)
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
	if err := clusteroperator.UpdateTablespaces(c.PgclusterClientset, c.PgclusterConfig, newCluster, newTablespaces); err != nil {
		return err
	}

	return nil
}
