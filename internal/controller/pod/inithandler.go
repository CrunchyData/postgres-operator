package pod

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
	"fmt"
	"time"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/controller"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/operator"
	backrestoperator "github.com/crunchydata/postgres-operator/internal/operator/backrest"
	clusteroperator "github.com/crunchydata/postgres-operator/internal/operator/cluster"
	taskoperator "github.com/crunchydata/postgres-operator/internal/operator/task"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

// handleClusterInit is responsible for proceeding with initialization of the PG cluster once the
// primary PG pod for a new or restored PG cluster reaches a ready status
func (c *Controller) handleClusterInit(newPod *apiv1.Pod, cluster *crv1.Pgcluster) error {
	clusterName := cluster.GetName()

	// first check to see if the update is a repo pod.  If so, then call repo init handler and
	// return since the other handlers are only applicable to PG pods
	if isBackRestRepoPod(newPod) {
		log.Debugf("Pod Controller: calling pgBackRest repo init for cluster %s", clusterName)
		if err := c.handleBackRestRepoInit(newPod, cluster); err != nil {
			log.Error(err)
			return err
		}
		return nil
	}

	// handle common tasks for initializing a cluster, whether due to bootstap or reinitialization
	// following a restore, or if a regular or standby cluster
	if err := c.handleCommonInit(cluster); err != nil {
		log.Error(err)
		return err
	}

	// call the standby init logic if a standby cluster
	if cluster.Spec.Standby {
		log.Debugf("Pod Controller: standby cluster detected during cluster %s init, calling "+
			"standby handler", clusterName)
		return c.handleStandbyInit(cluster)
	}

	// call bootstrap init for all other cluster initialization
	log.Debugf("Pod Controller: calling bootstrap init for cluster %s", clusterName)
	return c.handleBootstrapInit(newPod, cluster)
}

// handleBackRestRepoInit handles cluster initialization tasks that must be executed once
// as a result of an update to a cluster's pgBackRest repository pod
func (c *Controller) handleBackRestRepoInit(newPod *apiv1.Pod, cluster *crv1.Pgcluster) error {
	// if the repo pod is for a cluster bootstrap, the kick of the bootstrap job and return
	if _, ok := newPod.GetLabels()[config.LABEL_PGHA_BOOTSTRAP]; ok {
		if err := clusteroperator.AddClusterBootstrap(c.Client, cluster); err != nil {
			log.Error(err)
			return err
		}
		return nil
	}

	// first: a sanity check that there exists a primary deployment to scale. this
	// is to attempt to avoid any silent failures in the deployment scaling
	// function.
	//
	// If we do encounter an error, we will proceed in case the deployment becomes
	// available after.
	if err := c.waitForPrimaryDeployment(cluster); err != nil {
		log.Warn(err)
	}

	clusterInfo, err := clusteroperator.ScaleClusterDeployments(c.Client, *cluster, 1,
		true, false, false, false)
	if err != nil {
		log.Error(err)
		return err
	}

	log.Debugf("Pod Controller: scaled primary deployment %s to 1 to proceed with initializing "+
		"cluster %s", clusterInfo.PrimaryDeployment, cluster.Name)

	return nil
}

// handleCommonInit is resposible for handling common initilization tasks for a PG cluster
// regardless of the specific type of cluster (e.g. regualar or standby) or the reason the
// cluster is being initialized (initial bootstrap or restore)
func (c *Controller) handleCommonInit(cluster *crv1.Pgcluster) error {
	// Disable autofailover in the cluster that is now "Ready" if autofilover
	// is disabled for the cluster
	if cluster.Spec.DisableAutofail {
		// accepts the inverse
		if err := util.ToggleAutoFailover(c.Client, !cluster.Spec.DisableAutofail,
			cluster.Name, cluster.Namespace); err != nil {
			log.Error(err)
		}
	}

	if err := operator.UpdatePGHAConfigInitFlag(c.Client, false, cluster.Name,
		cluster.Namespace); err != nil {
		log.Error(err)
	}

	return nil
}

// handleBootstrapInit is resposible for handling cluster initilization (e.g. initiating pgBackRest
// stanza creation) when a the database container within the primary PG Pod for a new PG cluster
// enters a ready status
func (c *Controller) handleBootstrapInit(newPod *apiv1.Pod, cluster *crv1.Pgcluster) error {
	ctx := context.TODO()
	clusterName := cluster.Name
	namespace := cluster.Namespace

	// determine if restore, and if delete the restore label since it is no longer needed
	if _, restore := cluster.GetAnnotations()[config.ANNOTATION_BACKREST_RESTORE]; restore {
		patch, err := kubeapi.NewJSONPatch().
			Remove("metadata", "annotations", config.LABEL_BACKREST_RESTORE).Bytes()
		if err == nil {
			log.Debugf("patching cluster %s: %s", cluster.GetName(), patch)
			_, err = c.Client.CrunchydataV1().Pgclusters(namespace).
				Patch(ctx, cluster.GetName(), types.JSONPatchType, patch, metav1.PatchOptions{})
		}
		if err != nil {
			log.Errorf("Pod Controller unable to remove backrest restore annotation from "+
				"pgcluster %s: %s", cluster.GetName(), err.Error())
		}
	} else {
		log.Debugf("%s went to Ready from Not Ready, apply policies...", clusterName)
		taskoperator.ApplyPolicies(clusterName, c.Client, c.Client.Config, namespace)
	}

	taskoperator.CompleteCreateClusterWorkflow(clusterName, c.Client, namespace)

	// publish event for cluster complete
	_ = publishClusterComplete(clusterName, namespace, cluster)

	// first clean any stanza create resources from a previous stanza-create, e.g. during a
	// restore when these resources may already exist from initial creation of the cluster
	if err := backrestoperator.CleanStanzaCreateResources(namespace, clusterName,
		c.Client); err != nil {
		log.Error(err)
		return err
	}

	// create the pgBackRest stanza
	backrestoperator.StanzaCreate(newPod.ObjectMeta.Namespace, clusterName, c.Client)

	// if this is a pgbouncer enabled cluster, add a pgbouncer
	// Note: we only warn if we cannot create the pgBouncer, so eecution can
	// continue
	if cluster.Spec.PgBouncer.Enabled() {
		if err := clusteroperator.AddPgbouncer(c.Client, c.Client.Config, cluster); err != nil {
			log.Warn(err)
		}
	}

	return nil
}

// handleStandbyInit is resposible for handling standby cluster initilization when the database
// container within the primary PG Pod for a new standby cluster enters a ready status
func (c *Controller) handleStandbyInit(cluster *crv1.Pgcluster) error {
	ctx := context.TODO()
	clusterName := cluster.Name
	namespace := cluster.Namespace

	taskoperator.CompleteCreateClusterWorkflow(clusterName, c.Client, namespace)

	// publish event for cluster complete
	_ = publishClusterComplete(clusterName, namespace, cluster)

	// now scale any replicas deployments to 1
	_, _ = clusteroperator.ScaleClusterDeployments(c.Client, *cluster, 1, false, true, false, false)

	// Proceed with stanza-creation of this is not a standby cluster, or if its
	// a standby cluster that does not have s3/gcs storage only enabled.
	// If this is a standby cluster and the pgBackRest storage type is set
	// to "s3" or "gcs" storage only, set the cluster to an initialized status.
	if !(len(cluster.Spec.BackrestStorageTypes) == 1 &&
		(cluster.Spec.BackrestStorageTypes[0] == crv1.BackrestStorageTypeS3 ||
			cluster.Spec.BackrestStorageTypes[0] == crv1.BackrestStorageTypeGCS)) {
		// first try to delete any existing stanza create task and/or job
		if err := c.Client.CrunchydataV1().Pgtasks(namespace).
			Delete(ctx, fmt.Sprintf("%s-%s", clusterName, crv1.PgtaskBackrestStanzaCreate),
				metav1.DeleteOptions{}); err != nil && !kerrors.IsNotFound(err) {
			return err
		}
		deletePropagation := metav1.DeletePropagationForeground
		if err := c.Client.
			BatchV1().Jobs(namespace).
			Delete(ctx, fmt.Sprintf("%s-%s", clusterName, crv1.PgtaskBackrestStanzaCreate),
				metav1.DeleteOptions{PropagationPolicy: &deletePropagation}); err != nil && !kerrors.IsNotFound(err) {
			return err
		}
		backrestoperator.StanzaCreate(namespace, clusterName, c.Client)
	} else {
		if err := controller.SetClusterInitializedStatus(c.Client, clusterName,
			namespace); err != nil {
			log.Error(err)
		}

		// If a standby cluster with s3/gcs only initialize the creation of any replicas.  Replicas
		// can be initialized right away, i.e. there is no dependency on
		// stanza-creation and/or the creation of any backups, since the replicas
		// will be generated from the pgBackRest repository of an external PostgreSQL
		// database (which should already exist).
		if err := controller.InitializeReplicaCreation(c.Client, clusterName,
			namespace); err != nil {
			log.Error(err)
		}
	}

	// if this is a pgbouncer enabled cluster, add a pgbouncer
	// Note: we only warn if we cannot create the pgBouncer, so eecution can
	// continue
	if cluster.Spec.PgBouncer.Enabled() {
		if err := clusteroperator.AddPgbouncer(c.Client, c.Client.Config, cluster); err != nil {
			log.Warn(err)
		}
	}

	return nil
}

// labelPostgresPodAndDeployment
// see if this is a primary or replica being created
// update service-name label on the pod for each case
// to match the correct Service selector for the PG cluster
func (c *Controller) labelPostgresPodAndDeployment(newpod *apiv1.Pod) {
	ctx := context.TODO()
	depName := newpod.ObjectMeta.Labels[config.LABEL_DEPLOYMENT_NAME]
	ns := newpod.Namespace

	_, err := c.Client.CrunchydataV1().Pgreplicas(ns).Get(ctx, depName, metav1.GetOptions{})
	replica := err == nil
	log.Debugf("checkPostgresPods --- dep %s replica %t", depName, replica)

	dep, err := c.Client.AppsV1().Deployments(ns).Get(ctx, depName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("could not get Deployment on pod Add %s", newpod.Name)
		return
	}

	serviceName := ""

	if dep.ObjectMeta.Labels[config.LABEL_SERVICE_NAME] != "" {
		log.Debug("this means the deployment was already labeled")
		log.Debug("which means its pod was restarted for some reason")
		log.Debug("we will use the service name on the deployment")
		serviceName = dep.ObjectMeta.Labels[config.LABEL_SERVICE_NAME]
	} else if !replica {
		log.Debugf("primary pod ADDED %s service-name=%s", newpod.Name, newpod.ObjectMeta.Labels[config.LABEL_PG_CLUSTER])
		// add label onto pod "service-name=clustername"
		serviceName = newpod.ObjectMeta.Labels[config.LABEL_PG_CLUSTER]
	} else if replica {
		log.Debugf("replica pod ADDED %s service-name=%s", newpod.Name, newpod.ObjectMeta.Labels[config.LABEL_PG_CLUSTER]+"-replica")
		// add label onto pod "service-name=clustername-replica"
		serviceName = newpod.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] + "-replica"
	}

	patch, err := kubeapi.NewMergePatch().Add("metadata", "labels", config.LABEL_SERVICE_NAME)(serviceName).Bytes()
	if err == nil {
		log.Debugf("patching pod %s: %s", newpod.Name, patch)
		_, err = c.Client.CoreV1().Pods(ns).Patch(ctx, newpod.Name, types.MergePatchType, patch, metav1.PatchOptions{})
	}
	if err != nil {
		log.Error(err)
		log.Errorf(" could not add pod label for pod %s and label %s ...", newpod.Name, serviceName)
		return
	}

	// add the service name label to the Deployment
	log.Debugf("patching deployment %s: %s", dep.Name, patch)
	_, err = c.Client.AppsV1().Deployments(ns).Patch(ctx, dep.Name, types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		log.Error("could not add label to deployment on pod add")
		return
	}
}

// waitForPrimaryDeployment checks to see that a primary deployment is
// available. It does not check readiness, only that the deployment exists. This
// used before scaling to ensure scaling does not fail silently
func (c *Controller) waitForPrimaryDeployment(cluster *crv1.Pgcluster) error {
	ctx := context.TODO()
	primaryDeploymentName := cluster.Annotations[config.ANNOTATION_CURRENT_PRIMARY]
	options := metav1.ListOptions{
		LabelSelector: fields.AndSelectors(
			fields.OneTermEqualSelector(config.LABEL_PG_CLUSTER, cluster.Name),
			fields.OneTermEqualSelector(config.LABEL_PG_DATABASE, config.LABEL_TRUE),
			fields.OneTermEqualSelector(config.LABEL_DEPLOYMENT_NAME, primaryDeploymentName),
		).String(),
	}

	// start polling to see if the primary deployment is created
	if err := wait.PollImmediate(5*time.Second, 60*time.Second, func() (bool, error) {
		// check to see if the deployment exists
		d, err := c.Client.AppsV1().Deployments(cluster.Namespace).List(ctx, options)

		if err != nil {
			log.Warnf("could not find primary deployment for scaling: %s", err)
		}

		return err == nil && len(d.Items) > 0, nil
	}); err != nil {
		return fmt.Errorf("primary deployment lookup timeout reached for %q", primaryDeploymentName)
	}

	return nil
}
