// Copyright 2021 - 2023 Crunchy Data Solutions, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package managedpostgrescluster

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/crunchydata/postgres-operator/internal/bridge"
	pgoRuntime "github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

const finalizer = "managedpostgrescluster.postgres-operator.crunchydata.com/finalizer"

// ManagedPostgresClusterReconciler reconciles a ManagedPostgrescluster object
type ManagedPostgresClusterReconciler struct {
	client.Client

	Owner  client.FieldOwner
	Scheme *runtime.Scheme

	// For this iteration, we will only be setting conditions rather than
	// setting conditions and emitting events. That may change in the future,
	// so we're leaving this EventRecorder here for now.
	// record.EventRecorder

	// NewClient is called each time a new Client is needed.
	NewClient func() *bridge.Client
}

//+kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="managedpostgresclusters",verbs={list,watch}
//+kubebuilder:rbac:groups="",resources="secrets",verbs={list,watch}

// SetupWithManager sets up the controller with the Manager.
func (r *ManagedPostgresClusterReconciler) SetupWithManager(
	mgr ctrl.Manager,
) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.ManagedPostgresCluster{}).
		// Wake periodically to check Bridge API for all ManagedPostgresClusters.
		// Potentially replace with different requeue times, remove the Watch function
		// Smarter: retry after a certain time for each cluster: https://gist.github.com/cbandy/a5a604e3026630c5b08cfbcdfffd2a13
		Watches(
			pgoRuntime.NewTickerImmediate(5*time.Minute, event.GenericEvent{}),
			r.Watch(),
		).
		// Watch secrets and filter for secrets mentioned by ManagedPostgresClusters
		Watches(
			&source.Kind{Type: &corev1.Secret{}},
			r.watchForRelatedSecret(),
		).
		Complete(r)
}

// watchForRelatedSecret handles create/update/delete events for secrets,
// passing the Secret ObjectKey to findManagedPostgresClustersForSecret
func (r *ManagedPostgresClusterReconciler) watchForRelatedSecret() handler.EventHandler {
	handle := func(secret client.Object, q workqueue.RateLimitingInterface) {
		ctx := context.Background()
		key := client.ObjectKeyFromObject(secret)

		for _, cluster := range r.findManagedPostgresClustersForSecret(ctx, key) {
			q.Add(ctrl.Request{
				NamespacedName: client.ObjectKeyFromObject(cluster),
			})
		}
	}

	return handler.Funcs{
		CreateFunc: func(e event.CreateEvent, q workqueue.RateLimitingInterface) {
			handle(e.Object, q)
		},
		UpdateFunc: func(e event.UpdateEvent, q workqueue.RateLimitingInterface) {
			handle(e.ObjectNew, q)
		},
		// If the secret is deleted, we want to reconcile
		// in order to emit an event/status about this problem.
		// We will also emit a matching event/status about this problem
		// when we reconcile the cluster and can't find the secret.
		// That way, users will get two alerts: one when the secret is deleted
		// and another when the cluster is being reconciled.
		DeleteFunc: func(e event.DeleteEvent, q workqueue.RateLimitingInterface) {
			handle(e.Object, q)
		},
	}
}

//+kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="managedpostgresclusters",verbs={list}

// findManagedPostgresClustersForSecret returns ManagedPostgresClusters
// that are connected to the Secret
func (r *ManagedPostgresClusterReconciler) findManagedPostgresClustersForSecret(
	ctx context.Context, secret client.ObjectKey,
) []*v1beta1.ManagedPostgresCluster {
	var matching []*v1beta1.ManagedPostgresCluster
	var clusters v1beta1.ManagedPostgresClusterList

	// NOTE: If this becomes slow due to a large number of ManagedPostgresClusters in a single
	// namespace, we can configure the [ctrl.Manager] field indexer and pass a
	// [fields.Selector] here.
	// - https://book.kubebuilder.io/reference/watching-resources/externally-managed.html
	if r.List(ctx, &clusters, &client.ListOptions{
		Namespace: secret.Namespace,
	}) == nil {
		for i := range clusters.Items {
			if clusters.Items[i].Spec.Secret == secret.Name {
				matching = append(matching, &clusters.Items[i])
			}
		}
	}
	return matching
}

//+kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="managedpostgresclusters",verbs={list}

// Watch enqueues all existing ManagedPostgresClusters for reconciles.
func (r *ManagedPostgresClusterReconciler) Watch() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(client.Object) []reconcile.Request {
		ctx := context.Background()

		managedPostgresClusterList := &v1beta1.ManagedPostgresClusterList{}
		_ = r.List(ctx, managedPostgresClusterList)

		reconcileRequests := []reconcile.Request{}
		for index := range managedPostgresClusterList.Items {
			reconcileRequests = append(reconcileRequests,
				reconcile.Request{
					NamespacedName: client.ObjectKeyFromObject(
						&managedPostgresClusterList.Items[index],
					),
				},
			)
		}

		return reconcileRequests
	})
}

//+kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="managedpostgresclusters",verbs={get,patch,update}
//+kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="managedpostgresclusters/status",verbs={patch,update}
//+kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="managedpostgresclusters/finalizers",verbs={patch,update}
//+kubebuilder:rbac:groups="",resources="secrets",verbs={get}

// Reconcile does the work to move the current state of the world toward the
// desired state described in a [v1beta1.ManagedPostgresCluster] identified by req.
func (r *ManagedPostgresClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Retrieve the managedpostgrescluster from the client cache, if it exists. A deferred
	// function below will send any changes to its Status field.
	//
	// NOTE: No DeepCopy is necessary here because controller-runtime makes a
	// copy before returning from its cache.
	// - https://github.com/kubernetes-sigs/controller-runtime/issues/1235
	managedpostgrescluster := &v1beta1.ManagedPostgresCluster{}
	err := r.Get(ctx, req.NamespacedName, managedpostgrescluster)

	if err == nil {
		// Write any changes to the managedpostgrescluster status on the way out.
		before := managedpostgrescluster.DeepCopy()
		defer func() {
			if !equality.Semantic.DeepEqual(before.Status, managedpostgrescluster.Status) {
				status := r.Status().Patch(ctx, managedpostgrescluster, client.MergeFrom(before), r.Owner)

				if err == nil && status != nil {
					err = status
				} else if status != nil {
					log.Error(status, "Patching ManagedPostgresCluster status")
				}
			}
		}()
	} else {
		// NotFound cannot be fixed by requeuing so ignore it. During background
		// deletion, we receive delete events from managedpostgrescluster's dependents after
		// managedpostgrescluster is deleted.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// START SECRET HANDLING -- SPIN OFF INTO ITS OWN FUNC?

	// Get and validate secret for req
	key, team, err := r.GetSecretKeys(ctx, managedpostgrescluster)
	if err != nil {
		log.Error(err, "whoops, secret issue")

		meta.SetStatusCondition(&managedpostgrescluster.Status.Conditions, metav1.Condition{
			ObservedGeneration: managedpostgrescluster.GetGeneration(),
			Type:               v1beta1.ConditionCreating,
			Status:             metav1.ConditionFalse,
			Reason:             "SecretInvalid",
			Message: fmt.Sprintf(
				"Cannot create with bad secret: %v", "TODO(managedpostgrescluster)"),
		})

		// Don't automatically requeue Secret issues
		// We are watching for related secrets,
		// so will requeue when a related secret is touched
		// lint:ignore nilerr Return err as status, no requeue needed
		return ctrl.Result{}, nil
	}

	// Remove SecretInvalid condition if found
	invalid := meta.FindStatusCondition(managedpostgrescluster.Status.Conditions,
		v1beta1.ConditionCreating)
	if invalid != nil && invalid.Status == metav1.ConditionFalse && invalid.Reason == "SecretInvalid" {
		meta.RemoveStatusCondition(&managedpostgrescluster.Status.Conditions,
			v1beta1.ConditionCreating)
	}

	// END SECRET HANDLING

	// If the ManagedPostgresCluster isn't being deleted, add the finalizer
	if managedpostgrescluster.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(managedpostgrescluster, finalizer) {
			controllerutil.AddFinalizer(managedpostgrescluster, finalizer)
			if err := r.Update(ctx, managedpostgrescluster); err != nil {
				return ctrl.Result{}, err
			}
		}
		// If the ManagedPostgresCluster is being deleted,
		// handle the deletion, and remove the finalizer
	} else {
		if controllerutil.ContainsFinalizer(managedpostgrescluster, finalizer) {
			log.Info("deleting cluster", "clusterName", managedpostgrescluster.Spec.ClusterName)

			// TODO(managedpostgrescluster): If is_protected is true, maybe skip this call, but allow the deletion of the K8s object?
			_, deletedAlready, err := r.NewClient().DeleteCluster(ctx, key, managedpostgrescluster.Status.ID)
			// Requeue if error
			if err != nil {
				return ctrl.Result{}, err
			}

			if !deletedAlready {
				return ctrl.Result{RequeueAfter: 1 * time.Second}, err
			}

			// Remove finalizer if deleted already
			if deletedAlready {
				log.Info("cluster deleted", "clusterName", managedpostgrescluster.Spec.ClusterName)

				controllerutil.RemoveFinalizer(managedpostgrescluster, finalizer)
				if err := r.Update(ctx, managedpostgrescluster); err != nil {
					return ctrl.Result{}, err
				}
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	// Wonder if there's a better way to handle adding/checking/removing statuses
	// We did something in the upgrade controller
	// Exit early if we can't create from this K8s object
	// unless this K8s object has been changed (compare ObservedGeneration)
	invalid = meta.FindStatusCondition(managedpostgrescluster.Status.Conditions,
		v1beta1.ConditionCreating)
	if invalid != nil &&
		invalid.Status == metav1.ConditionFalse &&
		invalid.Message == "ClusterInvalid" &&
		invalid.ObservedGeneration == managedpostgrescluster.GetGeneration() {
		return ctrl.Result{}, nil
	}

	// Remove cluster invalid status if found
	if invalid != nil &&
		invalid.Status == metav1.ConditionFalse &&
		invalid.Reason == "ClusterInvalid" {
		meta.RemoveStatusCondition(&managedpostgrescluster.Status.Conditions,
			v1beta1.ConditionCreating)
	}

	storageVal, err := handleStorage(managedpostgrescluster.Spec.Storage)
	if err != nil {
		log.Error(err, "whoops, storage issue")
		// TODO(managedpostgrescluster)
		// lint:ignore nilerr no requeue needed
		return ctrl.Result{}, nil
	}

	// We should only be missing the ID if no create has been issued
	// or the create was interrupted and we haven't received the ID.
	if managedpostgrescluster.Status.ID == "" {
		// START FIND

		// TODO(managedpostgrescluster) If the CreateCluster response was interrupted, we won't have the ID
		// so we can get by name
		// BUT if we do that, there's a chance for the K8s object to grab a pre-existing Bridge cluster
		// which means there's a chance to delete a Bridge cluster through K8s actions
		// even though that cluster didn't originate from K8s.

		// Check if the cluster exists
		clusters, err := r.NewClient().ListClusters(ctx, key, team)
		if err != nil {
			log.Error(err, "whoops, cluster listing issue")
			return ctrl.Result{}, err
		}

		for _, cluster := range clusters {
			if managedpostgrescluster.Name == cluster.Name {
				managedpostgrescluster.Status.ID = cluster.ID
				// Requeue now that we have a cluster ID assigned
				return ctrl.Result{Requeue: true}, nil
			}
		}

		// END FIND

		// if we've gotten here then no cluster exists with that name and we're missing the ID, ergo, create cluster

		// TODO(managedpostgrescluster) Can almost just use the managed.Spec... except for the team, which we don't want
		// users to set on the spec. Do we?
		clusterReq := &v1beta1.ClusterDetails{
			IsHA:            managedpostgrescluster.Spec.IsHA,
			Name:            managedpostgrescluster.Spec.ClusterName,
			Plan:            managedpostgrescluster.Spec.Plan,
			PostgresVersion: intstr.FromInt(managedpostgrescluster.Spec.PostgresVersion),
			Provider:        managedpostgrescluster.Spec.Provider,
			Region:          managedpostgrescluster.Spec.Region,
			Storage:         storageVal,
			Team:            team,
		}
		cluster, err := r.NewClient().CreateCluster(ctx, key, clusterReq)
		if err != nil {
			log.Error(err, "whoops, cluster creating issue")
			// TODO(managedpostgrescluster): probably shouldn't set this condition unless response from Bridge
			// indicates the payload is wrong
			// Otherwise want a different condition
			meta.SetStatusCondition(&managedpostgrescluster.Status.Conditions, metav1.Condition{
				ObservedGeneration: managedpostgrescluster.GetGeneration(),
				Type:               v1beta1.ConditionCreating,
				Status:             metav1.ConditionFalse,
				Reason:             "ClusterInvalid",
				Message: fmt.Sprintf(
					"Cannot create from spec for some reason: %v", "TODO(managedpostgrescluster)"),
			})

			// TODO(managedpostgrescluster): If the payload is wrong, we don't want to requeue, so pass nil error
			// If the transmission hit a transient problem, we do want to requeue
			return ctrl.Result{}, nil
		}
		managedpostgrescluster.Status.ID = cluster.ID
		return ctrl.Result{RequeueAfter: 3 * time.Minute}, nil
	}

	// If we reach this point, our ManagedPostgresCluster object has an ID
	// so we want to fill in the details for the cluster and cluster upgrades from the Bridge API
	// Consider cluster details as a separate func.
	clusterDetails, err := r.NewClient().GetCluster(ctx, key, managedpostgrescluster.Status.ID)
	if err != nil {
		log.Error(err, "whoops, cluster getting issue")
		return ctrl.Result{}, err
	}
	managedpostgrescluster.Status.Cluster = clusterDetails

	clusterUpgradeDetails, err := r.NewClient().GetClusterUpgrade(ctx, key, managedpostgrescluster.Status.ID)
	if err != nil {
		log.Error(err, "whoops, cluster upgrade getting issue")
		return ctrl.Result{}, err
	}
	managedpostgrescluster.Status.ClusterUpgrade = clusterUpgradeDetails

	// For now, we skip updating until the upgrade status is cleared.
	// For the future, we may want to update in-progress upgrades,
	// and for that we will need a way tell that an upgrade in progress
	// is the one we want to update.
	// Consider: Perhaps add `generation` field to upgrade status?
	// Checking this here also means that if an upgrade is requested through the GUI/API
	// then we will requeue and wait for it to be done.
	// TODO(managedpostgrescluster): Do we want the operator to interrupt
	// upgrades created through the GUI/API?
	if len(managedpostgrescluster.Status.ClusterUpgrade.Operations) != 0 {
		return ctrl.Result{RequeueAfter: 3 * time.Minute}, nil
	}

	// Check if there's an upgrade difference for the three upgradeable fields that hit the upgrade endpoint
	// Why PostgresVersion and MajorVersion? Because MajorVersion in the Status is sure to be
	// an int of the major version, whereas Status.Cluster.PostgresVersion might be the ID
	if (storageVal != managedpostgrescluster.Status.Cluster.Storage) ||
		managedpostgrescluster.Spec.Plan != managedpostgrescluster.Status.Cluster.Plan ||
		managedpostgrescluster.Spec.PostgresVersion != managedpostgrescluster.Status.Cluster.MajorVersion {
		return r.handleUpgrade(ctx, key, managedpostgrescluster, storageVal)
	}

	// Are there diffs between the cluster response from the Bridge API and the spec?
	// HA diffs are sent to /clusters/{cluster_id}/actions/[enable|disable]-ha
	// so have to know (a) to send and (b) which to send to
	if managedpostgrescluster.Spec.IsHA != managedpostgrescluster.Status.Cluster.IsHA {
		return r.handleUpgradeHA(ctx, key, managedpostgrescluster)
	}

	// Check if there's a difference in is_protected, name, maintenance_window_start, etc.
	// see https://docs.crunchybridge.com/api/cluster#update-cluster
	// updates to these fields that hit the PATCH `clusters/<id>` endpoint
	// TODO(managedpostgrescluster)

	log.Info("Reconciled")
	// TODO(managedpostgrescluster): do we always want to requeue? Does the Watch mean we
	// don't need this, or do we want both?
	return ctrl.Result{RequeueAfter: 3 * time.Minute}, nil
}

// handleStorage returns a usable int in G (rounded up if the original storage was in Gi).
// Returns an error if the int is outside the range for Bridge min (10) or max (65535).
func handleStorage(storageSpec resource.Quantity) (int64, error) {
	scaledValue := storageSpec.ScaledValue(resource.Giga)

	if scaledValue < 10 || scaledValue > 65535 {
		return 0, fmt.Errorf("storage value must be between 10 and 65535")
	}

	return scaledValue, nil
}

// handleUpgrade handles upgrades that hit the "POST /clusters/<id>/upgrade" endpoint
func (r *ManagedPostgresClusterReconciler) handleUpgrade(ctx context.Context,
	apiKey string,
	managedpostgrescluster *v1beta1.ManagedPostgresCluster,
	storageVal int64,
) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Handling upgrade request")

	upgradeRequest := &v1beta1.ClusterDetails{
		Plan:            managedpostgrescluster.Spec.Plan,
		PostgresVersion: intstr.FromInt(managedpostgrescluster.Spec.PostgresVersion),
		Storage:         storageVal,
	}

	clusterUpgrade, err := r.NewClient().UpgradeCluster(ctx, apiKey,
		managedpostgrescluster.Status.ID, upgradeRequest)
	if err != nil {
		// TODO(managedpostgrescluster): consider what errors we might get
		// and what different results/requeue times we want to return.
		// Currently: don't requeue and wait for user to change spec.
		log.Error(err, "Error while attempting cluster upgrade")
		return ctrl.Result{}, nil
	}
	managedpostgrescluster.Status.ClusterUpgrade = clusterUpgrade
	return ctrl.Result{RequeueAfter: 3 * time.Minute}, nil
}

// handleUpgradeHA handles upgrades that hit the
// "PUT /clusters/<id>/actions/[enable|disable]-ha" endpoint
func (r *ManagedPostgresClusterReconciler) handleUpgradeHA(ctx context.Context,
	apiKey string,
	managedpostgrescluster *v1beta1.ManagedPostgresCluster,
) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Handling HA change request")

	action := "enable-ha"
	if !managedpostgrescluster.Spec.IsHA {
		action = "disable-ha"
	}

	clusterUpgrade, err := r.NewClient().UpgradeClusterHA(ctx, apiKey, managedpostgrescluster.Status.ID, action)
	if err != nil {
		// TODO(managedpostgrescluster): consider what errors we might get
		// and what different results/requeue times we want to return.
		// Currently: don't requeue and wait for user to change spec.
		log.Error(err, "Error while attempting cluster HA change")
		return ctrl.Result{}, nil
	}
	managedpostgrescluster.Status.ClusterUpgrade = clusterUpgrade
	return ctrl.Result{RequeueAfter: 3 * time.Minute}, nil
}

// GetSecretKeys gets the secret and returns the expected API key and team id
// or an error if either of those fields or the Secret are missing
func (r *ManagedPostgresClusterReconciler) GetSecretKeys(
	ctx context.Context, managedPostgresCluster *v1beta1.ManagedPostgresCluster,
) (string, string, error) {

	existing := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
		Namespace: managedPostgresCluster.GetNamespace(),
		Name:      managedPostgresCluster.Spec.Secret,
	}}

	err := errors.WithStack(
		r.Client.Get(ctx, client.ObjectKeyFromObject(existing), existing))

	if err == nil {
		if existing.Data["key"] != nil && existing.Data["team"] != nil {
			return string(existing.Data["key"]), string(existing.Data["team"]), nil
		}
		err = fmt.Errorf("error handling secret: found key %t, found team %t",
			existing.Data["key"] == nil,
			existing.Data["team"] == nil)
	}

	return "", "", err
}
