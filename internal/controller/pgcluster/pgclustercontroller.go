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
	"context"
	"encoding/json"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/operator"
	backrestoperator "github.com/crunchydata/postgres-operator/internal/operator/backrest"
	clusteroperator "github.com/crunchydata/postgres-operator/internal/operator/cluster"
	"github.com/crunchydata/postgres-operator/internal/operator/pvc"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	informers "github.com/crunchydata/postgres-operator/pkg/generated/informers/externalversions/crunchydata.com/v1"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Controller holds the connections for the controller
type Controller struct {
	Client               *kubeapi.Client
	Queue                workqueue.RateLimitingInterface
	Informer             informers.PgclusterInformer
	PgclusterWorkerCount int
}

// onAdd is called when a pgcluster is added
func (c *Controller) onAdd(obj interface{}) {
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
	log.Debug("pgcluster Contoller: received stop signal, worker queue told to shutdown")
}

func (c *Controller) processNextItem() bool {
	ctx := context.TODO()

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

	// get the pgcluster
	cluster, err := c.Client.CrunchydataV1().Pgclusters(keyNamespace).Get(ctx, keyResourceName, metav1.GetOptions{})
	if err != nil {
		log.Debugf("cluster add - pgcluster not found, this is invalid")
		c.Queue.Forget(key) // NB(cbandy): This should probably be a retry.
		return true
	}
	log.Debugf("[pgcluster Controller] ns %s onAdd %s", cluster.ObjectMeta.Namespace, cluster.ObjectMeta.SelfLink)

	if cluster.Spec.Status == crv1.CompletedStatus ||
		cluster.Status.State == crv1.PgclusterStateBootstrapping ||
		cluster.Status.State == crv1.PgclusterStateInitialized {
		log.Debugf("pgcluster Contoller: onAdd event received for cluster %s but "+
			"will not process because it either has a 'completed' status or is currently in an "+
			"'initialized' or 'bootstrapping' state", cluster.GetName())
		return true
	}

	addIdentifier(cluster)

	// If bootstrapping from an existing data source then attempt to create the pgBackRest repository.
	// If a repo already exists (e.g. because it is associated with a currently running cluster) then
	// proceed with bootstrapping.
	if cluster.Spec.PGDataSource.RestoreFrom != "" {
		repoCreated, err := clusteroperator.AddBootstrapRepo(c.Client, cluster)
		if err != nil {
			log.Error(err)
			c.Queue.AddRateLimited(key)
			return true
		}
		// if no errors and no repo was created, then we know that the repo is for a currently running
		// cluster and we can therefore proceed with bootstrapping.
		if !repoCreated {
			if err := clusteroperator.AddClusterBootstrap(c.Client, cluster); err != nil {
				log.Error(err)
				c.Queue.AddRateLimited(key)
				return true
			}
		}
		c.Queue.Forget(key)
		return true
	}

	patch, err := json.Marshal(map[string]interface{}{
		"status": crv1.PgclusterStatus{
			State:   crv1.PgclusterStateProcessed,
			Message: "Successfully processed Pgcluster by controller",
		},
	})
	if err == nil {
		_, err = c.Client.CrunchydataV1().Pgclusters(keyNamespace).
			Patch(ctx, cluster.Name, types.MergePatchType, patch, metav1.PatchOptions{})
	}
	if err != nil {
		log.Errorf("ERROR updating pgcluster status on add: %s", err.Error())
		c.Queue.Forget(key) // NB(cbandy): This should probably be a retry.
		return true
	}

	log.Debugf("pgcluster added: %s", cluster.ObjectMeta.Name)

	// AddClusterBase creates all deployments for the cluster (in addition to various other supporting
	// resources such as services, configMaps, secrets, etc.), but leaves them scaled to 0.  This
	// ensures all deployments exist as needed to properly orchestrate initialization of the
	// cluster, e.g. we need to ensure the primary DB deployment resource has been created before
	// bringing the repo deployment online, since that in turn will bring the primary DB online.
	clusteroperator.AddClusterBase(c.Client, cluster, cluster.ObjectMeta.Namespace)

	c.Queue.Forget(key)
	return true
}

// onUpdate is called when a pgcluster is updated
func (c *Controller) onUpdate(oldObj, newObj interface{}) {
	oldcluster := oldObj.(*crv1.Pgcluster)
	newcluster := newObj.(*crv1.Pgcluster)
	// initialize a slice that may contain functions that need to be executed
	// as part of a rolling update
	rollingUpdateFuncs := [](func(kubeapi.Interface, *crv1.Pgcluster, *appsv1.Deployment) error){}

	log.Debugf("pgcluster onUpdate for cluster %s (namespace %s)", newcluster.ObjectMeta.Namespace,
		newcluster.ObjectMeta.Name)

	// if the status of the pgcluster shows that it has been bootstrapped, then proceed with
	// creating the cluster (i.e. the cluster deployment, services, etc.)
	if newcluster.Spec.Status != crv1.CompletedStatus &&
		newcluster.Status.State == crv1.PgclusterStateBootstrapped {
		clusteroperator.AddClusterBase(c.Client, newcluster, newcluster.GetNamespace())
		return
	}

	// if the 'shutdown' parameter in the pgcluster update shows that the cluster should be either
	// shutdown or started but its current status does not properly reflect that it is, then
	// proceed with the logic needed to either shutdown or start the cluster
	if newcluster.Spec.Shutdown && newcluster.Status.State != crv1.PgclusterStateShutdown {
		_ = clusteroperator.ShutdownCluster(c.Client, *newcluster)
	} else if !newcluster.Spec.Shutdown &&
		newcluster.Status.State == crv1.PgclusterStateShutdown {
		_ = clusteroperator.StartupCluster(c.Client, *newcluster)
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
			_ = util.ToggleAutoFailover(c.Client, autofailEnabledNew,
				newcluster.ObjectMeta.Labels[config.LABEL_PGHA_SCOPE],
				newcluster.ObjectMeta.Namespace)
		}

	}

	// handle standby being enabled and disabled for the cluster
	if oldcluster.Spec.Standby && !newcluster.Spec.Standby {
		if err := clusteroperator.DisableStandby(c.Client, *newcluster); err != nil {
			log.Error(err)
			return
		}
	} else if !oldcluster.Spec.Standby && newcluster.Spec.Standby {
		if err := clusteroperator.EnableStandby(c.Client, *newcluster); err != nil {
			log.Error(err)
			return
		}
	}

	// see if we are adding / removing the metrics collection sidecar
	if oldcluster.Spec.Exporter != newcluster.Spec.Exporter {
		var err error

		// determine if the sidecar is being enabled/disabled and take the precursor
		// actions before the deployment template is modified
		if newcluster.Spec.Exporter {
			err = clusteroperator.AddExporter(c.Client, c.Client.Config, newcluster)
		} else {
			err = clusteroperator.RemoveExporter(c.Client, c.Client.Config, newcluster)
		}

		if err == nil {
			rollingUpdateFuncs = append(rollingUpdateFuncs, clusteroperator.UpdateExporterSidecar)
		} else {
			log.Errorf("could not update metrics collection sidecar: %q", err.Error())
		}
	}

	// see if any of the resource values have changed for the database or exporter container,
	// if so, update them
	if !reflect.DeepEqual(oldcluster.Spec.Resources, newcluster.Spec.Resources) ||
		!reflect.DeepEqual(oldcluster.Spec.Limits, newcluster.Spec.Limits) ||
		!reflect.DeepEqual(oldcluster.Spec.ExporterResources, newcluster.Spec.ExporterResources) ||
		!reflect.DeepEqual(oldcluster.Spec.ExporterLimits, newcluster.Spec.ExporterLimits) {
		rollingUpdateFuncs = append(rollingUpdateFuncs, clusteroperator.UpdateResources)
	}

	// see if any of the pgBackRest repository resource values have changed, and
	// if so, update them
	if !reflect.DeepEqual(oldcluster.Spec.BackrestResources, newcluster.Spec.BackrestResources) ||
		!reflect.DeepEqual(oldcluster.Spec.BackrestLimits, newcluster.Spec.BackrestLimits) {
		if err := backrestoperator.UpdateResources(c.Client, newcluster); err != nil {
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
		rollingUpdateFuncs = append(rollingUpdateFuncs, clusteroperator.UpdateTablespaces)
	}

	// check to see if any of the annotations have been modified, in particular,
	// the non-system annotations
	if !reflect.DeepEqual(oldcluster.Spec.Annotations, newcluster.Spec.Annotations) {
		if changed, err := updateAnnotations(c, oldcluster, newcluster); err != nil {
			log.Error(err)
			return
		} else if changed {
			// append the PostgreSQL specific functions as part of a rolling update
			rollingUpdateFuncs = append(rollingUpdateFuncs, clusteroperator.UpdateAnnotations)
		}
	}

	// check to see if any tolerations have been modified
	if !reflect.DeepEqual(oldcluster.Spec.Tolerations, newcluster.Spec.Tolerations) {
		rollingUpdateFuncs = append(rollingUpdateFuncs, clusteroperator.UpdateTolerations)
	}

	// if there is no need to perform a rolling update, exit here
	if len(rollingUpdateFuncs) == 0 {
		return
	}

	// otherwise, create an anonymous function that executes each of the rolling
	// update functions as part of the rolling update
	if err := clusteroperator.RollingUpdate(c.Client, c.Client.Config, newcluster,
		func(clientset kubeapi.Interface, cluster *crv1.Pgcluster, deployment *appsv1.Deployment) error {
			for _, fn := range rollingUpdateFuncs {
				if err := fn(clientset, cluster, deployment); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
		log.Error(err)
		return
	}
}

// onDelete is called when a pgcluster is deleted
func (c *Controller) onDelete(obj interface{}) {
	// cluster := obj.(*crv1.Pgcluster)
	//	log.Debugf("[Controller] ns=%s onDelete %s", cluster.ObjectMeta.Namespace, cluster.ObjectMeta.SelfLink)

	// handle pgcluster cleanup
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

// updateAnnotations updates any custom annitations that may be on the managed
// deployments, which includes:
//
// - globally applied annotations
// - pgBackRest instance specific annotations
// - pgBouncer instance specific annotations
//
// The Postgres specific annotations need to be handled by the caller function,
// due to the fact they need to be applied in a rolling update manner that can
// be controlled. We indicate this to the calling function by returning "true"
func updateAnnotations(c *Controller, oldCluster *crv1.Pgcluster, newCluster *crv1.Pgcluster) (bool, error) {
	// so we have a two-tier problem we need to solve:
	// 1. Which of the deployment types are being modified (or in the case of
	//    global, all of them)?
	// 2. Which annotations are being added/modified/removed? Kubernetes actually
	//    has a convenient function for updating the annotations, so we do no
	//    need to do too much works
	annotationsPostgres := map[string]string{}
	annotationsBackrest := map[string]string{}
	annotationsPgBouncer := map[string]string{}

	// check the individual deployment groups. If the annotations differ in either the specific group or
	// in the global group, set them in their respective map
	if !reflect.DeepEqual(oldCluster.Spec.Annotations.Postgres, newCluster.Spec.Annotations.Postgres) ||
		!reflect.DeepEqual(oldCluster.Spec.Annotations.Global, newCluster.Spec.Annotations.Global) {
		// store the global annotations first
		for k, v := range newCluster.Spec.Annotations.Global {
			annotationsPostgres[k] = v
		}

		// then store the postgres specific annotations
		for k, v := range newCluster.Spec.Annotations.Postgres {
			annotationsPostgres[k] = v
		}
	}

	if !reflect.DeepEqual(oldCluster.Spec.Annotations.Backrest, newCluster.Spec.Annotations.Backrest) ||
		!reflect.DeepEqual(oldCluster.Spec.Annotations.Global, newCluster.Spec.Annotations.Global) {
		// store the global annotations first
		for k, v := range newCluster.Spec.Annotations.Global {
			annotationsBackrest[k] = v
		}

		// then store the pgbackrest specific annotations
		for k, v := range newCluster.Spec.Annotations.Backrest {
			annotationsBackrest[k] = v
		}
	}

	if !reflect.DeepEqual(oldCluster.Spec.Annotations.PgBouncer, newCluster.Spec.Annotations.PgBouncer) ||
		!reflect.DeepEqual(oldCluster.Spec.Annotations.Global, newCluster.Spec.Annotations.Global) {
		// store the global annotations first
		for k, v := range newCluster.Spec.Annotations.Global {
			annotationsPgBouncer[k] = v
		}

		// then store the pgbouncer specific annotations
		for k, v := range newCluster.Spec.Annotations.PgBouncer {
			annotationsPgBouncer[k] = v
		}
	}

	// so if there are changes, we can apply them to the various deployments,
	// but only do so if we have to
	if len(annotationsBackrest) != 0 {
		if err := backrestoperator.UpdateAnnotations(c.Client, newCluster, annotationsBackrest); err != nil {
			return false, err
		}
	}

	if len(annotationsPgBouncer) != 0 {
		if err := clusteroperator.UpdatePgBouncerAnnotations(c.Client, newCluster, annotationsPgBouncer); err != nil && !kerrors.IsNotFound(err) {
			return false, err
		}
	}

	return len(annotationsPostgres) != 0, nil
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
			return clusteroperator.AddPgbouncer(c.Client, c.Client.Config, newCluster)
		}

		// if we're not enabled, we're disabled
		return clusteroperator.DeletePgbouncer(c.Client, c.Client.Config, newCluster)
	}

	// otherwise, this is an update
	return clusteroperator.UpdatePgbouncer(c.Client, oldCluster, newCluster)
}

// updateTablespaces updates the PostgreSQL instance Deployments to reflect the
// new PostgreSQL tablespaces that should be added
func updateTablespaces(c *Controller, oldCluster *crv1.Pgcluster, newCluster *crv1.Pgcluster) error {
	// first, get a list of all of the instance deployments for the cluster
	deployments, err := operator.GetInstanceDeployments(c.Client, newCluster)
	if err != nil {
		return err
	}

	// iterate through the the tablespace mount map that is present in and create
	// any new PVCs
	for tablespaceName, storageSpec := range newCluster.Spec.TablespaceMounts {
		// if the tablespace does not exist in the old version of the cluster,
		// then add it in!
		if _, ok := oldCluster.Spec.TablespaceMounts[tablespaceName]; ok {
			continue
		}

		log.Debugf("new tablespace found: [%s]", tablespaceName)

		// This is a new tablespace, great. Create the new PVCs.
		// The PVCs are created for each **instance** in the cluster, as every
		// instance needs to have a distinct PVC for each tablespace
		// get the name of the tablespace PVC for that instance.
		for _, deployment := range deployments.Items {
			tablespacePVCName := operator.GetTablespacePVCName(deployment.Name, tablespaceName)

			log.Debugf("creating tablespace PVC [%s] for [%s]", tablespacePVCName, deployment.Name)

			// Now create it! If it errors, we just need to return, which
			// potentially leaves things in an inconsistent state, but at this point
			// only PVC objects have been created
			if _, err := pvc.CreateIfNotExists(c.Client, storageSpec, tablespacePVCName,
				newCluster.Name, newCluster.Namespace); err != nil {
				return err
			}
		}
	}

	return nil
}

// WorkerCount returns the worker count for the controller
func (c *Controller) WorkerCount() int {
	return c.PgclusterWorkerCount
}
