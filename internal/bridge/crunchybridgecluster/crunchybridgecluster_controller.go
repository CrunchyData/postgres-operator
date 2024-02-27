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

package crunchybridgecluster

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
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
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// CrunchyBridgeClusterReconciler reconciles a CrunchyBridgeCluster object
type CrunchyBridgeClusterReconciler struct {
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

//+kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="crunchybridgeclusters",verbs={list,watch}
//+kubebuilder:rbac:groups="",resources="secrets",verbs={list,watch}

// SetupWithManager sets up the controller with the Manager.
func (r *CrunchyBridgeClusterReconciler) SetupWithManager(
	mgr ctrl.Manager,
) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.CrunchyBridgeCluster{}).
		Owns(&corev1.Secret{}).
		// Wake periodically to check Bridge API for all CrunchyBridgeClusters.
		// Potentially replace with different requeue times, remove the Watch function
		// Smarter: retry after a certain time for each cluster: https://gist.github.com/cbandy/a5a604e3026630c5b08cfbcdfffd2a13
		Watches(
			pgoRuntime.NewTickerImmediate(5*time.Minute, event.GenericEvent{}),
			r.Watch(),
		).
		// Watch secrets and filter for secrets mentioned by CrunchyBridgeClusters
		Watches(
			&source.Kind{Type: &corev1.Secret{}},
			r.watchForRelatedSecret(),
		).
		Complete(r)
}

// The owner reference created by controllerutil.SetControllerReference blocks
// deletion. The OwnerReferencesPermissionEnforcement plugin requires that the
// creator of such a reference have either "delete" permission on the owner or
// "update" permission on the owner's "finalizers" subresource.
// - https://docs.k8s.io/reference/access-authn-authz/admission-controllers/
// +kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="crunchybridgeclusters/finalizers",verbs={update}

// setControllerReference sets owner as a Controller OwnerReference on controlled.
// Only one OwnerReference can be a controller, so it returns an error if another
// is already set.
func (r *CrunchyBridgeClusterReconciler) setControllerReference(
	owner *v1beta1.CrunchyBridgeCluster, controlled client.Object,
) error {
	return controllerutil.SetControllerReference(owner, controlled, r.Client.Scheme())
}

// watchForRelatedSecret handles create/update/delete events for secrets,
// passing the Secret ObjectKey to findCrunchyBridgeClustersForSecret
func (r *CrunchyBridgeClusterReconciler) watchForRelatedSecret() handler.EventHandler {
	handle := func(secret client.Object, q workqueue.RateLimitingInterface) {
		ctx := context.Background()
		key := client.ObjectKeyFromObject(secret)

		for _, cluster := range r.findCrunchyBridgeClustersForSecret(ctx, key) {
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

//+kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="crunchybridgeclusters",verbs={list}

// findCrunchyBridgeClustersForSecret returns CrunchyBridgeClusters
// that are connected to the Secret
func (r *CrunchyBridgeClusterReconciler) findCrunchyBridgeClustersForSecret(
	ctx context.Context, secret client.ObjectKey,
) []*v1beta1.CrunchyBridgeCluster {
	var matching []*v1beta1.CrunchyBridgeCluster
	var clusters v1beta1.CrunchyBridgeClusterList

	// NOTE: If this becomes slow due to a large number of CrunchyBridgeClusters in a single
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

//+kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="crunchybridgeclusters",verbs={list}

// Watch enqueues all existing CrunchyBridgeClusters for reconciles.
func (r *CrunchyBridgeClusterReconciler) Watch() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(client.Object) []reconcile.Request {
		ctx := context.Background()

		crunchyBridgeClusterList := &v1beta1.CrunchyBridgeClusterList{}
		_ = r.List(ctx, crunchyBridgeClusterList)

		reconcileRequests := []reconcile.Request{}
		for index := range crunchyBridgeClusterList.Items {
			reconcileRequests = append(reconcileRequests,
				reconcile.Request{
					NamespacedName: client.ObjectKeyFromObject(
						&crunchyBridgeClusterList.Items[index],
					),
				},
			)
		}

		return reconcileRequests
	})
}

//+kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="crunchybridgeclusters",verbs={get,patch,update}
//+kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="crunchybridgeclusters/status",verbs={patch,update}
//+kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="crunchybridgeclusters/finalizers",verbs={patch,update}
//+kubebuilder:rbac:groups="",resources="secrets",verbs={get}

// Reconcile does the work to move the current state of the world toward the
// desired state described in a [v1beta1.CrunchyBridgeCluster] identified by req.
func (r *CrunchyBridgeClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Retrieve the crunchybridgecluster from the client cache, if it exists. A deferred
	// function below will send any changes to its Status field.
	//
	// NOTE: No DeepCopy is necessary here because controller-runtime makes a
	// copy before returning from its cache.
	// - https://github.com/kubernetes-sigs/controller-runtime/issues/1235
	crunchybridgecluster := &v1beta1.CrunchyBridgeCluster{}
	err := r.Get(ctx, req.NamespacedName, crunchybridgecluster)

	if err == nil {
		// Write any changes to the crunchybridgecluster status on the way out.
		before := crunchybridgecluster.DeepCopy()
		defer func() {
			if !equality.Semantic.DeepEqual(before.Status, crunchybridgecluster.Status) {
				status := r.Status().Patch(ctx, crunchybridgecluster, client.MergeFrom(before), r.Owner)

				if err == nil && status != nil {
					err = status
				} else if status != nil {
					log.Error(status, "Patching CrunchyBridgeCluster status")
				}
			}
		}()
	} else {
		// NotFound cannot be fixed by requeuing so ignore it. During background
		// deletion, we receive delete events from crunchybridgecluster's dependents after
		// crunchybridgecluster is deleted.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Get and validate connection secret for requests
	key, team, err := r.reconcileBridgeConnectionSecret(ctx, crunchybridgecluster)
	if err != nil {
		log.Error(err, "issue reconciling bridge connection secret")

		// Don't automatically requeue Secret issues. We are watching for
		// related secrets, so will requeue when a related secret is touched.
		// lint:ignore nilerr Return err as status, no requeue needed
		return ctrl.Result{}, nil
	}

	// Check for and handle deletion of cluster. Return early if it is being
	// deleted or there was an error. Make sure finalizer is added if cluster
	// is not being deleted.
	if result, err := r.handleDelete(ctx, crunchybridgecluster, key); err != nil {
		log.Error(err, "deleting")
		return ctrl.Result{}, err
	} else if result != nil {
		if log := log.V(1); log.Enabled() {
			if result.RequeueAfter > 0 {
				// RequeueAfter implies Requeue, but set both to make the next
				// log message more clear.
				result.Requeue = true
			}
			log.Info("deleting", "result", fmt.Sprintf("%+v", *result))
		}
		return *result, err
	}

	// Wonder if there's a better way to handle adding/checking/removing statuses
	// We did something in the upgrade controller
	// Exit early if we can't create from this K8s object
	// unless this K8s object has been changed (compare ObservedGeneration)
	invalid := meta.FindStatusCondition(crunchybridgecluster.Status.Conditions,
		v1beta1.ConditionCreating)
	if invalid != nil &&
		invalid.Status == metav1.ConditionFalse &&
		invalid.Message == "ClusterInvalid" &&
		invalid.ObservedGeneration == crunchybridgecluster.GetGeneration() {
		return ctrl.Result{}, nil
	}

	// Remove cluster invalid status if found
	if invalid != nil &&
		invalid.Status == metav1.ConditionFalse &&
		invalid.Reason == "ClusterInvalid" {
		meta.RemoveStatusCondition(&crunchybridgecluster.Status.Conditions,
			v1beta1.ConditionCreating)
	}

	// We should only be missing the ID if no create has been issued
	// or the create was interrupted and we haven't received the ID.
	if crunchybridgecluster.Status.ID == "" {
		// Check if the cluster exists
		clusters, err := r.NewClient().ListClusters(ctx, key, team)
		if err != nil {
			log.Error(err, "issue listing existing clusters in Bridge")
			return ctrl.Result{}, err
		}

		for _, cluster := range clusters {
			if crunchybridgecluster.Spec.ClusterName == cluster.ClusterName {
				// Cluster with the same name exists so check for adoption annotation
				adoptionID, annotationExists := crunchybridgecluster.Annotations[naming.CrunchyBridgeClusterAdoptionAnnotation]
				if annotationExists && strings.EqualFold(adoptionID, cluster.ID) {
					// Annotation is present with correct ID value; adopt cluster by assigning ID to status.
					crunchybridgecluster.Status.ID = cluster.ID
					// Requeue now that we have a cluster ID assigned
					return ctrl.Result{Requeue: true}, nil
				}

				// If we made it here, the adoption annotation either doesn't exist or its value is incorrect.
				// The user must either add it or change the name on the CR.

				// Set invalid status condition and create log message.
				meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
					ObservedGeneration: crunchybridgecluster.GetGeneration(),
					Type:               v1beta1.ConditionCreating,
					Status:             metav1.ConditionFalse,
					Reason:             "ClusterInvalid",
					Message: fmt.Sprintf("A cluster with the same name already exists for this team (Team ID: %v). "+
						"Give the CrunchyBridgeCluster CR a unique name, or if you would like to take control of the "+
						"existing cluster, add the 'postgres-operator.crunchydata.com/adopt-bridge-cluster' "+
						"annotation and set its value to the existing cluster's ID (Cluster ID: %v).", team, cluster.ID),
				})

				log.Info(fmt.Sprintf("A cluster with the same name already exists for this team (Team ID: %v). "+
					"Give the CrunchyBridgeCluster CR a unique name, or if you would like to take control "+
					"of the existing cluster, add the 'postgres-operator.crunchydata.com/adopt-bridge-cluster' "+
					"annotation and set its value to the existing cluster's ID (Cluster ID: %v).", team, cluster.ID))

				// We have an invalid cluster spec so we don't want to requeue
				return ctrl.Result{}, nil
			}
		}

		// if we've gotten here then no cluster exists with that name and we're missing the ID, ergo, create cluster
		createClusterRequestPayload := &bridge.PostClustersRequestPayload{
			IsHA:            crunchybridgecluster.Spec.IsHA,
			Name:            crunchybridgecluster.Spec.ClusterName,
			Plan:            crunchybridgecluster.Spec.Plan,
			PostgresVersion: intstr.FromInt(crunchybridgecluster.Spec.PostgresVersion),
			Provider:        crunchybridgecluster.Spec.Provider,
			Region:          crunchybridgecluster.Spec.Region,
			Storage:         bridge.ToGibibytes(crunchybridgecluster.Spec.Storage),
			Team:            team,
		}
		cluster, err := r.NewClient().CreateCluster(ctx, key, createClusterRequestPayload)
		if err != nil {
			log.Error(err, "issue creating cluster in Bridge")
			// TODO(crunchybridgecluster): probably shouldn't set this condition unless response from Bridge
			// indicates the payload is wrong
			// Otherwise want a different condition
			meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
				ObservedGeneration: crunchybridgecluster.GetGeneration(),
				Type:               v1beta1.ConditionCreating,
				Status:             metav1.ConditionFalse,
				Reason:             "ClusterInvalid",
				Message: fmt.Sprintf(
					"Cannot create from spec for some reason: %v", "TODO(crunchybridgecluster)"),
			})

			// TODO(crunchybridgecluster): If the payload is wrong, we don't want to requeue, so pass nil error
			// If the transmission hit a transient problem, we do want to requeue
			return ctrl.Result{}, nil
		}
		crunchybridgecluster.Status.ID = cluster.ID

		meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
			ObservedGeneration: crunchybridgecluster.GetGeneration(),
			Type:               v1beta1.ConditionUpdating,
			Status:             metav1.ConditionUnknown,
			Reason:             "NoUpgradesInProgress",
			Message: fmt.Sprintf(
				"No upgrades in Progress for Crunchy Bridge Cluster %v", crunchybridgecluster.Name),
		})
		return ctrl.Result{RequeueAfter: 3 * time.Minute}, nil
	}

	// If we reach this point, our CrunchyBridgeCluster object has an ID, so we want
	// to fill in the details for the cluster, cluster status, and cluster upgrades
	// from the Bridge API.

	// Get Cluster
	clusterDetails, err := r.NewClient().GetCluster(ctx, key, crunchybridgecluster.Status.ID)
	if err != nil {
		log.Error(err, "issue getting cluster information from Bridge")
		return ctrl.Result{}, err
	}
	clusterDetails.AddDataToClusterStatus(crunchybridgecluster)

	// Get Cluster Status
	clusterStatus, err := r.NewClient().GetClusterStatus(ctx, key, crunchybridgecluster.Status.ID)
	if err != nil {
		log.Error(err, "issue getting cluster status from Bridge")
		return ctrl.Result{}, err
	}
	clusterStatus.AddDataToClusterStatus(crunchybridgecluster)
	// TODO: Update the ConditionReady status here

	// Get Cluster Upgrade
	clusterUpgradeDetails, err := r.NewClient().GetClusterUpgrade(ctx, key, crunchybridgecluster.Status.ID)
	if err != nil {
		log.Error(err, "issue getting cluster upgrade from Bridge")
		return ctrl.Result{}, err
	}
	clusterUpgradeDetails.AddDataToClusterStatus(crunchybridgecluster)
	// TODO: Update the ConditionUpdating status here

	// Reconcile roles and their secrets
	err = r.reconcilePostgresRoles(ctx, key, crunchybridgecluster)
	if err != nil {
		log.Error(err, "issue reconciling postgres user roles/secrets")
		return ctrl.Result{}, err
	}

	// For now, we skip updating until the upgrade status is cleared.
	// For the future, we may want to update in-progress upgrades,
	// and for that we will need a way tell that an upgrade in progress
	// is the one we want to update.
	// Consider: Perhaps add `generation` field to upgrade status?
	// Checking this here also means that if an upgrade is requested through the GUI/API
	// then we will requeue and wait for it to be done.
	// TODO(crunchybridgecluster): Do we want the operator to interrupt
	// upgrades created through the GUI/API?
	if len(crunchybridgecluster.Status.OngoingUpgrade) != 0 {
		return ctrl.Result{RequeueAfter: 3 * time.Minute}, nil
	}

	// Check if there's an upgrade difference for the three upgradeable fields that hit the upgrade endpoint
	// Why PostgresVersion and MajorVersion? Because MajorVersion in the Status is sure to be
	// an int of the major version, whereas Status.Responses.Cluster.PostgresVersion might be the ID
	if (crunchybridgecluster.Spec.Storage != *crunchybridgecluster.Status.Storage) ||
		crunchybridgecluster.Spec.Plan != crunchybridgecluster.Status.Plan ||
		crunchybridgecluster.Spec.PostgresVersion != crunchybridgecluster.Status.MajorVersion {
		return r.handleUpgrade(ctx, key, crunchybridgecluster)
	}

	// Are there diffs between the cluster response from the Bridge API and the spec?
	// HA diffs are sent to /clusters/{cluster_id}/actions/[enable|disable]-ha
	// so have to know (a) to send and (b) which to send to
	if crunchybridgecluster.Spec.IsHA != *crunchybridgecluster.Status.IsHA {
		return r.handleUpgradeHA(ctx, key, crunchybridgecluster)
	}
	meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
		ObservedGeneration: crunchybridgecluster.GetGeneration(),
		Type:               v1beta1.ConditionUpdating,
		Status:             metav1.ConditionUnknown,
		Reason:             "NoUpgradesInProgress",
		Message: fmt.Sprintf(
			"No upgrades in Progress for Crunchy Bridge Cluster %v", crunchybridgecluster.Name),
	})

	// Check if there's a difference in is_protected, name, maintenance_window_start, etc.
	// see https://docs.crunchybridge.com/api/cluster#update-cluster
	// updates to these fields that hit the PATCH `clusters/<id>` endpoint
	// TODO(crunchybridgecluster)

	log.Info("Reconciled")
	// TODO(crunchybridgecluster): do we always want to requeue? Does the Watch mean we
	// don't need this, or do we want both?
	return ctrl.Result{RequeueAfter: 3 * time.Minute}, nil
}

func (r *CrunchyBridgeClusterReconciler) reconcileBridgeConnectionSecret(
	ctx context.Context, crunchybridgecluster *v1beta1.CrunchyBridgeCluster,
) (string, string, error) {
	key, team, err := r.GetSecretKeys(ctx, crunchybridgecluster)
	if err != nil {
		meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
			ObservedGeneration: crunchybridgecluster.GetGeneration(),
			Type:               v1beta1.ConditionCreating,
			Status:             metav1.ConditionFalse,
			Reason:             "SecretInvalid",
			Message: fmt.Sprintf(
				"Cannot create with bad secret: %v", "TODO(crunchybridgecluster)"),
		})

		return "", "", err
	}

	// Remove SecretInvalid condition if found
	invalid := meta.FindStatusCondition(crunchybridgecluster.Status.Conditions,
		v1beta1.ConditionCreating)
	if invalid != nil && invalid.Status == metav1.ConditionFalse && invalid.Reason == "SecretInvalid" {
		meta.RemoveStatusCondition(&crunchybridgecluster.Status.Conditions,
			v1beta1.ConditionCreating)
	}

	return key, team, err
}

// handleUpgrade handles upgrades that hit the "POST /clusters/<id>/upgrade" endpoint
func (r *CrunchyBridgeClusterReconciler) handleUpgrade(ctx context.Context,
	apiKey string,
	crunchybridgecluster *v1beta1.CrunchyBridgeCluster,
) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Handling upgrade request")

	upgradeRequest := &bridge.PostClustersUpgradeRequestPayload{
		Plan:            crunchybridgecluster.Spec.Plan,
		PostgresVersion: intstr.FromInt(crunchybridgecluster.Spec.PostgresVersion),
		Storage:         bridge.ToGibibytes(crunchybridgecluster.Spec.Storage),
	}

	clusterUpgrade, err := r.NewClient().UpgradeCluster(ctx, apiKey,
		crunchybridgecluster.Status.ID, upgradeRequest)
	if err != nil {
		// TODO(crunchybridgecluster): consider what errors we might get
		// and what different results/requeue times we want to return.
		// Currently: don't requeue and wait for user to change spec.
		log.Error(err, "Error while attempting cluster upgrade")
		return ctrl.Result{}, nil
	}
	clusterUpgrade.AddDataToClusterStatus(crunchybridgecluster)

	for _, operation := range clusterUpgrade.Operations {
		meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
			ObservedGeneration: crunchybridgecluster.GetGeneration(),
			Type:               v1beta1.ConditionUpdating,
			Status:             metav1.ConditionTrue,
			Reason:             operation.Flavor,
			Message: fmt.Sprintf(
				"Performing an upgrade of type %v with a state of %v on Crunchy Bridge Cluster %v",
				operation.Flavor, operation.State, crunchybridgecluster.Name),
		})
	}
	return ctrl.Result{RequeueAfter: 3 * time.Minute}, nil
}

// handleUpgradeHA handles upgrades that hit the
// "PUT /clusters/<id>/actions/[enable|disable]-ha" endpoint
func (r *CrunchyBridgeClusterReconciler) handleUpgradeHA(ctx context.Context,
	apiKey string,
	crunchybridgecluster *v1beta1.CrunchyBridgeCluster,
) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Handling HA change request")

	action := "enable-ha"
	if !crunchybridgecluster.Spec.IsHA {
		action = "disable-ha"
	}

	clusterUpgrade, err := r.NewClient().UpgradeClusterHA(ctx, apiKey, crunchybridgecluster.Status.ID, action)
	if err != nil {
		// TODO(crunchybridgecluster): consider what errors we might get
		// and what different results/requeue times we want to return.
		// Currently: don't requeue and wait for user to change spec.
		log.Error(err, "Error while attempting cluster HA change")
		return ctrl.Result{}, nil
	}
	clusterUpgrade.AddDataToClusterStatus(crunchybridgecluster)

	meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
		ObservedGeneration: crunchybridgecluster.GetGeneration(),
		Type:               v1beta1.ConditionUpdating,
		Status:             metav1.ConditionTrue,
		Reason:             "UpgradeInProgress",
		Message: fmt.Sprintf(
			"HA upgrade in progress to %v on the Crunchy Bridge Cluster %v", action, crunchybridgecluster.Name),
	})
	return ctrl.Result{RequeueAfter: 3 * time.Minute}, nil
}

// GetSecretKeys gets the secret and returns the expected API key and team id
// or an error if either of those fields or the Secret are missing
func (r *CrunchyBridgeClusterReconciler) GetSecretKeys(
	ctx context.Context, crunchyBridgeCluster *v1beta1.CrunchyBridgeCluster,
) (string, string, error) {

	existing := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
		Namespace: crunchyBridgeCluster.GetNamespace(),
		Name:      crunchyBridgeCluster.Spec.Secret,
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

// deleteControlled safely deletes object when it is controlled by cluster.
func (r *CrunchyBridgeClusterReconciler) deleteControlled(
	ctx context.Context, crunchyBridgeCluster *v1beta1.CrunchyBridgeCluster, object client.Object,
) error {
	if metav1.IsControlledBy(object, crunchyBridgeCluster) {
		uid := object.GetUID()
		version := object.GetResourceVersion()
		exactly := client.Preconditions{UID: &uid, ResourceVersion: &version}

		return r.Client.Delete(ctx, object, exactly)
	}

	return nil
}
