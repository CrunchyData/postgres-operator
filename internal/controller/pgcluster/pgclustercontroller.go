package pgcluster

/*
Copyright 2017 - 2021 Crunchy Data Solutions, Inc.
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
	"fmt"
	"reflect"
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
	"k8s.io/apimachinery/pkg/fields"
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
	// set "rescale" to true if we are adding a rolling update function that
	// requires the Deployment to be scaled down in order for it to work
	rescale := false

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
	//
	// we do need to check if the status has info in it. There have been cases
	// where the entire status has been removed that could be external to the
	// operator itself. In the case of checking that the state is in a shutdown
	// phase, we also want to check if the status is completely empty. If it is,
	// we will proceed with the shutdown.
	if newcluster.Spec.Shutdown && newcluster.Status.State != crv1.PgclusterStateShutdown {
		if err := clusteroperator.ShutdownCluster(c.Client, *newcluster); err != nil {
			log.Error(err)
		}
	} else if !newcluster.Spec.Shutdown &&
		(newcluster.Status.State == crv1.PgclusterStateShutdown || newcluster.Status.State == "") {
		if err := clusteroperator.StartupCluster(c.Client, *newcluster); err != nil {
			log.Error(err)
		}
	}

	// check to see if autofail setting has been changed. If set to "true", it
	// will be disabled, otherwise it will be enabled. Simple.
	if oldcluster.Spec.DisableAutofail != newcluster.Spec.DisableAutofail {
		// take the inverse, as this func checks for autofail being enabled
		// if we can't toggle autofailover, log the error but continue on
		if err := util.ToggleAutoFailover(c.Client, !newcluster.Spec.DisableAutofail,
			newcluster.Name, newcluster.Namespace); err != nil {
			log.Error(err)
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

	// if the service type has changed, update the service type. Log an error if
	// it fails, but continue on
	if oldcluster.Spec.ServiceType != newcluster.Spec.ServiceType {
		updateServices(c.Client, newcluster)
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

	// see if we are adding / removing the pgBadger sidecar
	if oldcluster.Spec.PGBadger != newcluster.Spec.PGBadger {
		var err error

		// determine if the sidecar is being enabled/disabled and take the precursor
		// actions before the deployment template is modified
		if newcluster.Spec.PGBadger {
			err = clusteroperator.AddPGBadger(c.Client, c.Client.Config, newcluster)
		} else {
			err = clusteroperator.RemovePGBadger(c.Client, c.Client.Config, newcluster)
		}

		if err == nil {
			rollingUpdateFuncs = append(rollingUpdateFuncs, clusteroperator.UpdatePGBadgerSidecar)
		} else {
			log.Errorf("could not update pgbadger sidecar: %q", err.Error())
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

	// see if the pgBackRest PVC size value changed.
	if oldcluster.Spec.BackrestStorage.Size != newcluster.Spec.BackrestStorage.Size {
		// validate that this resize can occur
		if err := util.ValidatePVCResize(oldcluster.Spec.BackrestStorage.Size, newcluster.Spec.BackrestStorage.Size); err != nil {
			log.Error(err)
		} else {
			if err := backrestoperator.ResizePVC(c.Client, newcluster); err != nil {
				log.Error(err)
			}
		}
	}

	// see if the pgAdmin PVC size valued changed.
	if oldcluster.Spec.PGAdminStorage.Size != newcluster.Spec.PGAdminStorage.Size {
		// validate that this resize can occur
		if err := util.ValidatePVCResize(oldcluster.Spec.PGAdminStorage.Size, newcluster.Spec.PGAdminStorage.Size); err != nil {
			log.Error(err)
		} else {
			if err := clusteroperator.ResizePGAdminPVC(c.Client, newcluster); err != nil {
				log.Error(err)
			}
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

	// check to see if any of the custom labels have been modified
	if !reflect.DeepEqual(util.GetCustomLabels(oldcluster), util.GetCustomLabels(newcluster)) {
		// update the custom labels on all of the managed objects at are not the
		// Postgres cluster deployments
		if err := updateLabels(c, oldcluster, newcluster); err != nil {
			log.Error(err)
			return
		}

		// append the PostgreSQL specific functions as part of a rolling update
		rollingUpdateFuncs = append(rollingUpdateFuncs, clusteroperator.UpdateLabels)
	}

	// check to see if any tolerations have been modified
	if !reflect.DeepEqual(oldcluster.Spec.Tolerations, newcluster.Spec.Tolerations) {
		rollingUpdateFuncs = append(rollingUpdateFuncs, clusteroperator.UpdateTolerations)
	}

	// check to see if there are any modifications to TLS
	if !reflect.DeepEqual(oldcluster.Spec.TLS, newcluster.Spec.TLS) ||
		oldcluster.Spec.TLSOnly != newcluster.Spec.TLSOnly {
		rollingUpdateFuncs = append(rollingUpdateFuncs, clusteroperator.UpdateTLS)

		// if need be, toggle the TLS settings
		if !reflect.DeepEqual(oldcluster.Spec.TLS, newcluster.Spec.TLS) {
			if err := clusteroperator.ToggleTLS(c.Client, newcluster); err != nil {
				log.Error(err)
				return
			}
		}
	}

	// check to see if the S3 bucket name has changed. If it has, this requires
	// both updating the Postgres + pgBackRest Deployments AND reruning the stanza
	// create Job
	if oldcluster.Spec.BackrestS3Bucket != newcluster.Spec.BackrestS3Bucket {
		// first, update the pgBackRest repository
		if err := updateBackrestS3(c, newcluster); err != nil {
			log.Errorf("not updating pgBackrest S3 settings: %s", err.Error())
		} else {
			// if that is successful, add updating the pgBackRest S3 settings to the
			// rolling update changes
			rollingUpdateFuncs = append(rollingUpdateFuncs, clusteroperator.UpdateBackrestS3)
		}
	}

	// check to see if the size of the primary PVC has changed
	if oldcluster.Spec.PrimaryStorage.Size != newcluster.Spec.PrimaryStorage.Size {
		// validate that this resize can occur
		if err := util.ValidatePVCResize(oldcluster.Spec.PrimaryStorage.Size, newcluster.Spec.PrimaryStorage.Size); err != nil {
			log.Error(err)
		} else {
			rescale = true
			rollingUpdateFuncs = append(rollingUpdateFuncs, clusteroperator.ResizeClusterPVC)
		}
	}

	// check to see if the size of the WAL PVC has changed
	if oldcluster.Spec.WALStorage.Size != newcluster.Spec.WALStorage.Size {
		// validate that this resize can occur
		if err := util.ValidatePVCResize(oldcluster.Spec.WALStorage.Size, newcluster.Spec.WALStorage.Size); err != nil {
			log.Error(err)
		} else {
			rescale = true
			rollingUpdateFuncs = append(rollingUpdateFuncs, clusteroperator.ResizeWALPVC)
		}
	}

	// if there is no need to perform a rolling update, exit here
	if len(rollingUpdateFuncs) == 0 {
		return
	}

	// otherwise, create an anonymous function that executes each of the rolling
	// update functions as part of the rolling update
	if err := clusteroperator.RollingUpdate(c.Client, c.Client.Config, newcluster, rescale,
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

	// one follow-up post rolling update: if the S3 bucket changed, issue a
	// "create stanza" job
	if oldcluster.Spec.BackrestS3Bucket != newcluster.Spec.BackrestS3Bucket {
		backrestoperator.StanzaCreate(newcluster.Namespace, newcluster.Name, c.Client)
	}
}

// onDelete is called when a pgcluster is deleted
func (c *Controller) onDelete(obj interface{}) {
	ctx := context.TODO()
	cluster := obj.(*crv1.Pgcluster)

	log.Debugf("pgcluster onDelete for cluster %s (namespace %s)", cluster.Name, cluster.Namespace)

	// guard: if an upgrade is in progress, do not do any of the rest
	if _, ok := cluster.ObjectMeta.GetAnnotations()[config.ANNOTATION_UPGRADE_IN_PROGRESS]; ok {
		log.Debug("upgrade in progress, not proceeding with additional cleanups")
		return
	}

	// guard: see if the "rmdata Job" is running.
	options := metav1.ListOptions{
		LabelSelector: fields.AndSelectors(
			fields.OneTermEqualSelector(config.LABEL_PG_CLUSTER, cluster.Name),
			fields.OneTermEqualSelector(config.LABEL_RMDATA, config.LABEL_TRUE),
		).String(),
	}

	jobs, err := c.Client.BatchV1().Jobs(cluster.Namespace).List(ctx, options)

	if err != nil {
		log.Error(err)
		return
	}

	// iterate through the list of Jobs and see if any are currently active or
	// succeeded.
	// a succeeded Job could be a remnaint of an old Job for the cluser of a
	// same name, in which case, we can continue with deleting the cluster
	for _, job := range jobs.Items {
		// we will return for one of two reasons:
		// 1. if the Job is currently active
		// 2. if the Job is not active but never has completed and is below the
		// backoff limit -- this could be  evidence that the Job is retrying
		if job.Status.Active > 0 {
			return
		} else if job.Status.Succeeded < 1 && job.Status.Failed < *job.Spec.BackoffLimit {
			return
		}
	}

	// we need to create a special pgtask that will create the Job (I know). So
	// let's attempt to do that here. First, clear out any other pgtask with this
	// existing name. If it errors because it's not found, we're OK
	taskName := cluster.Name + "-rmdata"
	if err := c.Client.CrunchydataV1().Pgtasks(cluster.Namespace).Delete(
		ctx, taskName, metav1.DeleteOptions{}); err != nil && !kerrors.IsNotFound(err) {
		log.Error(err)
		return
	}

	// determine if the data directory or backups should be kept
	_, keepBackups := cluster.ObjectMeta.GetAnnotations()[config.ANNOTATION_CLUSTER_KEEP_BACKUPS]
	_, keepData := cluster.ObjectMeta.GetAnnotations()[config.ANNOTATION_CLUSTER_KEEP_DATA]

	// create the deletion job. this will delete any data and backups for this
	// cluster
	if err := util.CreateRMDataTask(c.Client, cluster, "", !keepBackups, !keepData, false, false); err != nil {
		log.Error(err)
	}
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

// updateBackrestS3 makes updates to the pgBackRest repo Deployment if any of
// the S3 specific settings have changed. Presently, this is just the S3 bucket
// name
func updateBackrestS3(c *Controller, cluster *crv1.Pgcluster) error {
	ctx := context.TODO()

	// get the pgBackRest deployment
	backrestDeploymentName := fmt.Sprintf(util.BackrestRepoDeploymentName, cluster.Name)
	backrestDeployment, err := c.Client.AppsV1().Deployments(cluster.Namespace).Get(ctx,
		backrestDeploymentName, metav1.GetOptions{})

	if err != nil {
		return err
	}

	// update the environmental variable(s) in the container that is aptly(?)
	// named database
	for i, container := range backrestDeployment.Spec.Template.Spec.Containers {
		if container.Name != "database" {
			continue
		}

		for j, envVar := range backrestDeployment.Spec.Template.Spec.Containers[i].Env {
			if envVar.Name == "PGBACKREST_REPO1_S3_BUCKET" {
				backrestDeployment.Spec.Template.Spec.Containers[i].Env[j].Value = cluster.Spec.BackrestS3Bucket
			}
		}
	}

	if _, err := c.Client.AppsV1().Deployments(cluster.Namespace).Update(ctx,
		backrestDeployment, metav1.UpdateOptions{}); err != nil {
		return err
	}

	// update the annotation on the pgBackRest Secret too
	secretName := fmt.Sprintf(util.BackrestRepoSecretName, cluster.Name)
	patch, _ := kubeapi.NewMergePatch().Add("metadata", "annotations")(map[string]string{
		config.ANNOTATION_S3_BUCKET: cluster.Spec.BackrestS3Bucket,
	}).Bytes()

	if _, err := c.Client.CoreV1().Secrets(cluster.Namespace).Patch(ctx,
		secretName, types.MergePatchType, patch, metav1.PatchOptions{}); err != nil {
		return err
	}

	return nil
}

// updateLabels updates the custom labels on all of the managed objects *except*
// the Postgres instances themselves, i.e. the deployment templates
func updateLabels(c *Controller, oldCluster *crv1.Pgcluster, newCluster *crv1.Pgcluster) error {
	// we need to figure out which labels need to be removed from the list
	labelsToRemove := make([]string, 0)
	labels := util.GetCustomLabels(newCluster)

	for old := range util.GetCustomLabels(oldCluster) {
		if _, ok := labels[old]; !ok {
			labelsToRemove = append(labelsToRemove, old)
		}
	}

	// go through each object group and update the labels.
	if err := updateLabelsForDeployments(c, newCluster, labels, labelsToRemove); err != nil {
		return err
	}

	if err := updateLabelsForPVCs(c, newCluster, labels, labelsToRemove); err != nil {
		return err
	}

	if err := updateLabelsForConfigMaps(c, newCluster, labels, labelsToRemove); err != nil {
		return err
	}

	if err := updateLabelsForSecrets(c, newCluster, labels, labelsToRemove); err != nil {
		return err
	}

	return updateLabelsForServices(c, newCluster, labels, labelsToRemove)
}

// updateLabelsForConfigMaps updates the custom labels for ConfigMaps
func updateLabelsForConfigMaps(c *Controller, cluster *crv1.Pgcluster, labels map[string]string, labelsToRemove []string) error {
	ctx := context.TODO()

	options := metav1.ListOptions{
		LabelSelector: fields.AndSelectors(
			fields.OneTermEqualSelector(config.LABEL_PG_CLUSTER, cluster.Name),
			fields.OneTermEqualSelector(config.LABEL_VENDOR, config.LABEL_CRUNCHY),
		).String(),
	}

	items, err := c.Client.CoreV1().ConfigMaps(cluster.Namespace).List(ctx, options)

	if err != nil {
		return err
	}

	for i := range items.Items {
		item := &items.Items[i]

		for j := range labelsToRemove {
			delete(item.ObjectMeta.Labels, labelsToRemove[j])
		}

		for k, v := range labels {
			item.ObjectMeta.Labels[k] = v
		}

		if _, err := c.Client.CoreV1().ConfigMaps(cluster.Namespace).Update(ctx,
			item, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

// updateLabelsForDeployments updates the custom labels for Deployments, except
// for the **templates** on the Postgres instances
func updateLabelsForDeployments(c *Controller, cluster *crv1.Pgcluster, labels map[string]string, labelsToRemove []string) error {
	ctx := context.TODO()

	options := metav1.ListOptions{
		LabelSelector: fields.AndSelectors(
			fields.OneTermEqualSelector(config.LABEL_PG_CLUSTER, cluster.Name),
			fields.OneTermEqualSelector(config.LABEL_VENDOR, config.LABEL_CRUNCHY),
		).String(),
	}

	items, err := c.Client.AppsV1().Deployments(cluster.Namespace).List(ctx, options)

	if err != nil {
		return err
	}

	for i := range items.Items {
		item := &items.Items[i]

		for j := range labelsToRemove {
			delete(item.ObjectMeta.Labels, labelsToRemove[j])

			// only remove the labels on the template if this is not a Postgres
			// instance
			if _, ok := item.ObjectMeta.Labels[config.LABEL_PG_DATABASE]; !ok {
				delete(item.Spec.Template.ObjectMeta.Labels, labelsToRemove[j])
			}
		}

		for k, v := range labels {
			item.ObjectMeta.Labels[k] = v

			// only update the labels on the template if this is not a Postgres
			// instance
			if _, ok := item.ObjectMeta.Labels[config.LABEL_PG_DATABASE]; !ok {
				item.Spec.Template.ObjectMeta.Labels[k] = v
			}
		}

		if _, err := c.Client.AppsV1().Deployments(cluster.Namespace).Update(ctx,
			item, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

// updateLabelsForPVCs updates the custom labels for PVCs
func updateLabelsForPVCs(c *Controller, cluster *crv1.Pgcluster, labels map[string]string, labelsToRemove []string) error {
	ctx := context.TODO()

	options := metav1.ListOptions{
		LabelSelector: fields.AndSelectors(
			fields.OneTermEqualSelector(config.LABEL_PG_CLUSTER, cluster.Name),
			fields.OneTermEqualSelector(config.LABEL_VENDOR, config.LABEL_CRUNCHY),
		).String(),
	}

	items, err := c.Client.CoreV1().PersistentVolumeClaims(cluster.Namespace).List(ctx, options)

	if err != nil {
		return err
	}

	for i := range items.Items {
		item := &items.Items[i]

		for j := range labelsToRemove {
			delete(item.ObjectMeta.Labels, labelsToRemove[j])
		}

		for k, v := range labels {
			item.ObjectMeta.Labels[k] = v
		}

		if _, err := c.Client.CoreV1().PersistentVolumeClaims(cluster.Namespace).Update(ctx,
			item, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

// updateLabelsForSecrets updates the custom labels for Secrets
func updateLabelsForSecrets(c *Controller, cluster *crv1.Pgcluster, labels map[string]string, labelsToRemove []string) error {
	ctx := context.TODO()

	options := metav1.ListOptions{
		LabelSelector: fields.AndSelectors(
			fields.OneTermEqualSelector(config.LABEL_PG_CLUSTER, cluster.Name),
			fields.OneTermEqualSelector(config.LABEL_VENDOR, config.LABEL_CRUNCHY),
		).String(),
	}

	items, err := c.Client.CoreV1().Secrets(cluster.Namespace).List(ctx, options)

	if err != nil {
		return err
	}

	for i := range items.Items {
		item := &items.Items[i]

		for j := range labelsToRemove {
			delete(item.ObjectMeta.Labels, labelsToRemove[j])
		}

		for k, v := range labels {
			item.ObjectMeta.Labels[k] = v
		}

		if _, err := c.Client.CoreV1().Secrets(cluster.Namespace).Update(ctx,
			item, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

// updateLabelsForServices updates the custom labels for Services
func updateLabelsForServices(c *Controller, cluster *crv1.Pgcluster, labels map[string]string, labelsToRemove []string) error {
	ctx := context.TODO()

	options := metav1.ListOptions{
		LabelSelector: fields.AndSelectors(
			fields.OneTermEqualSelector(config.LABEL_PG_CLUSTER, cluster.Name),
			fields.OneTermEqualSelector(config.LABEL_VENDOR, config.LABEL_CRUNCHY),
		).String(),
	}

	items, err := c.Client.CoreV1().Services(cluster.Namespace).List(ctx, options)

	if err != nil {
		return err
	}

	for i := range items.Items {
		item := &items.Items[i]

		for j := range labelsToRemove {
			delete(item.ObjectMeta.Labels, labelsToRemove[j])
		}

		for k, v := range labels {
			item.ObjectMeta.Labels[k] = v
		}

		if _, err := c.Client.CoreV1().Services(cluster.Namespace).Update(ctx,
			item, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
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

// updateServices handles any updates to the Service objects. Given how legacy
// replica services are handled (really, replica service singular), the update
// around replica services is a bit grotty, but it is what it is.
//
// If there are errors on the updates, this logs them but will continue on
// unless otherwise noted.
func updateServices(clientset kubeapi.Interface, cluster *crv1.Pgcluster) {
	ctx := context.TODO()

	// handle the primary instance
	if err := clusteroperator.UpdateClusterService(clientset, cluster); err != nil {
		log.Error(err)
	}

	// if there is a pgBouncer and the pgBouncer service type value is empty,
	// update the pgBouncer Service
	if cluster.Spec.PgBouncer.Enabled() && cluster.Spec.PgBouncer.ServiceType == "" {
		if err := clusteroperator.UpdatePgBouncerService(clientset, cluster); err != nil {
			log.Error(err)
		}
	}

	// handle the replica instances. Ish. This is kind of "broken" due to the
	// fact that we have a single service for all of the replicas. so, we'll
	// loop through all of the replicas and try to see if any of them have
	// any specialized service types. If so, we'll pluck that one out and use
	// it to apply
	options := metav1.ListOptions{
		LabelSelector: fields.OneTermEqualSelector(config.LABEL_PG_CLUSTER, cluster.Name).String(),
	}
	replicas, err := clientset.CrunchydataV1().Pgreplicas(cluster.Namespace).List(ctx, options)

	// well, if there is an error here, log it and abort
	if err != nil {
		log.Error(err)
		return
	}

	// if there are no replicas, also return
	if len(replicas.Items) == 0 {
		return
	}

	// ok, we're guaranteed at least one replica, so there should be a Service
	var replica *crv1.Pgreplica
	for i := range replicas.Items {
		// store the replica no matter what, for later comparison
		replica = &replicas.Items[i]
		// however, if the servicetype is customized, break out. Yup.
		if replica.Spec.ServiceType != "" {
			break
		}
	}

	if err := clusteroperator.UpdateReplicaService(clientset, cluster, replica); err != nil {
		log.Error(err)
	}
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
				newCluster.Name, newCluster.Namespace, util.GetCustomLabels(newCluster)); err != nil {
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
