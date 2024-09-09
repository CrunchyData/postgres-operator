// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

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
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/crunchydata/postgres-operator/internal/bridge"
	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	pgoRuntime "github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// CrunchyBridgeClusterReconciler reconciles a CrunchyBridgeCluster object
type CrunchyBridgeClusterReconciler struct {
	client.Client

	Owner client.FieldOwner

	// For this iteration, we will only be setting conditions rather than
	// setting conditions and emitting events. That may change in the future,
	// so we're leaving this EventRecorder here for now.
	// record.EventRecorder

	// NewClient is called each time a new Client is needed.
	NewClient func() bridge.ClientInterface
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
		WatchesRawSource(
			pgoRuntime.NewTickerImmediate(5*time.Minute, event.GenericEvent{}, r.Watch()),
		).
		// Watch secrets and filter for secrets mentioned by CrunchyBridgeClusters
		Watches(
			&corev1.Secret{},
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
			log.Info("deleting", "result", fmt.Sprintf("%+v", *result))
		}
		return *result, err
	}

	// Wonder if there's a better way to handle adding/checking/removing statuses
	// We did something in the upgrade controller
	// Exit early if we can't create from this K8s object
	// unless this K8s object has been changed (compare ObservedGeneration)
	invalid := meta.FindStatusCondition(crunchybridgecluster.Status.Conditions,
		v1beta1.ConditionReady)
	if invalid != nil &&
		invalid.Status == metav1.ConditionFalse &&
		invalid.Reason == "ClusterInvalid" &&
		invalid.ObservedGeneration == crunchybridgecluster.GetGeneration() {
		return ctrl.Result{}, nil
	}

	// check for an upgrade error and return until observedGeneration has
	// been incremented.
	invalidUpgrade := meta.FindStatusCondition(crunchybridgecluster.Status.Conditions,
		v1beta1.ConditionUpgrading)
	if invalidUpgrade != nil &&
		invalidUpgrade.Status == metav1.ConditionFalse &&
		invalidUpgrade.Reason == "UpgradeError" &&
		invalidUpgrade.ObservedGeneration == crunchybridgecluster.GetGeneration() {
		return ctrl.Result{}, nil
	}

	// We should only be missing the ID if no create has been issued
	// or the create was interrupted and we haven't received the ID.
	if crunchybridgecluster.Status.ID == "" {
		// Check if a cluster with the same name already exists
		controllerResult, err := r.handleDuplicateClusterName(ctx, key, team, crunchybridgecluster)
		if err != nil || controllerResult != nil {
			return *controllerResult, err
		}

		// if we've gotten here then no cluster exists with that name and we're missing the ID, ergo, create cluster
		return r.handleCreateCluster(ctx, key, team, crunchybridgecluster), nil
	}

	// If we reach this point, our CrunchyBridgeCluster object has an ID, so we want
	// to fill in the details for the cluster, cluster status, and cluster upgrades
	// from the Bridge API.

	// Get Cluster
	err = r.handleGetCluster(ctx, key, crunchybridgecluster)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Get Cluster Status
	err = r.handleGetClusterStatus(ctx, key, crunchybridgecluster)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Get Cluster Upgrade
	err = r.handleGetClusterUpgrade(ctx, key, crunchybridgecluster)
	if err != nil {
		return ctrl.Result{}, err
	}

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
		return runtime.RequeueWithoutBackoff(3 * time.Minute), nil
	}

	// Check if there's an upgrade difference for the three upgradeable fields that hit the upgrade endpoint
	// Why PostgresVersion and MajorVersion? Because MajorVersion in the Status is sure to be
	// an int of the major version, whereas Status.Responses.Cluster.PostgresVersion might be the ID
	if (crunchybridgecluster.Spec.Storage != *crunchybridgecluster.Status.Storage) ||
		crunchybridgecluster.Spec.Plan != crunchybridgecluster.Status.Plan ||
		crunchybridgecluster.Spec.PostgresVersion != crunchybridgecluster.Status.MajorVersion {
		return r.handleUpgrade(ctx, key, crunchybridgecluster), nil
	}

	// Are there diffs between the cluster response from the Bridge API and the spec?
	// HA diffs are sent to /clusters/{cluster_id}/actions/[enable|disable]-ha
	// so have to know (a) to send and (b) which to send to
	if crunchybridgecluster.Spec.IsHA != *crunchybridgecluster.Status.IsHA {
		return r.handleUpgradeHA(ctx, key, crunchybridgecluster), nil
	}

	// Check if there's a difference in is_protected, name, maintenance_window_start, etc.
	// see https://docs.crunchybridge.com/api/cluster#update-cluster
	// updates to these fields that hit the PATCH `clusters/<id>` endpoint
	if crunchybridgecluster.Spec.IsProtected != *crunchybridgecluster.Status.IsProtected ||
		crunchybridgecluster.Spec.ClusterName != crunchybridgecluster.Status.ClusterName {
		return r.handleUpdate(ctx, key, crunchybridgecluster), nil
	}

	log.Info("Reconciled")
	// TODO(crunchybridgecluster): do we always want to requeue? Does the Watch mean we
	// don't need this, or do we want both?
	return runtime.RequeueWithoutBackoff(3 * time.Minute), nil
}

// reconcileBridgeConnectionSecret looks for the Bridge connection secret specified by the cluster,
// and returns the API key and Team ID found in the secret, or sets conditions and returns an error
// if the secret is invalid.
func (r *CrunchyBridgeClusterReconciler) reconcileBridgeConnectionSecret(
	ctx context.Context, crunchybridgecluster *v1beta1.CrunchyBridgeCluster,
) (string, string, error) {
	key, team, err := r.GetSecretKeys(ctx, crunchybridgecluster)
	if err != nil {
		meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
			ObservedGeneration: crunchybridgecluster.GetGeneration(),
			Type:               v1beta1.ConditionReady,
			Status:             metav1.ConditionUnknown,
			Reason:             "SecretInvalid",
			Message: fmt.Sprintf(
				"The condition of the cluster is unknown because the secret is invalid: %v", err),
		})
		meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
			Type:               v1beta1.ConditionUpgrading,
			Status:             metav1.ConditionUnknown,
			ObservedGeneration: crunchybridgecluster.GetGeneration(),
			LastTransitionTime: metav1.Time{},
			Reason:             "SecretInvalid",
			Message: fmt.Sprintf(
				"The condition of the upgrade(s) is unknown because the secret is invalid: %v", err),
		})

		return "", "", err
	}

	return key, team, err
}

// handleDuplicateClusterName checks Bridge for any already existing clusters that
// have the same name. It returns (nil, nil) when no cluster is found with the same
// name. It returns a controller result, indicating we should exit the reconcile loop,
// if a cluster with a duplicate name is found. The caller is responsible for
// returning controller result objects and errors to controller-runtime.
func (r *CrunchyBridgeClusterReconciler) handleDuplicateClusterName(ctx context.Context,
	apiKey, teamId string, crunchybridgecluster *v1beta1.CrunchyBridgeCluster,
) (*ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	clusters, err := r.NewClient().ListClusters(ctx, apiKey, teamId)
	if err != nil {
		meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
			ObservedGeneration: crunchybridgecluster.GetGeneration(),
			Type:               v1beta1.ConditionReady,
			Status:             metav1.ConditionUnknown,
			Reason:             "UnknownClusterState",
			Message:            fmt.Sprintf("Issue listing existing clusters in Bridge: %v", err),
		})
		log.Error(err, "issue listing existing clusters in Bridge")
		return &ctrl.Result{}, err
	}

	for _, cluster := range clusters {
		if crunchybridgecluster.Spec.ClusterName == cluster.ClusterName {
			// Cluster with the same name exists so check for adoption annotation
			adoptionID, annotationExists := crunchybridgecluster.Annotations[naming.CrunchyBridgeClusterAdoptionAnnotation]
			if annotationExists && strings.EqualFold(adoptionID, cluster.ID) {
				// Annotation is present with correct ID value; adopt cluster by assigning ID to status.
				crunchybridgecluster.Status.ID = cluster.ID
				// Requeue now that we have a cluster ID assigned
				return &ctrl.Result{Requeue: true}, nil
			}

			// If we made it here, the adoption annotation either doesn't exist or its value is incorrect.
			// The user must either add it or change the name on the CR.

			// Set invalid status condition and create log message.
			meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
				ObservedGeneration: crunchybridgecluster.GetGeneration(),
				Type:               v1beta1.ConditionReady,
				Status:             metav1.ConditionFalse,
				Reason:             "DuplicateClusterName",
				Message: fmt.Sprintf("A cluster with the same name already exists for this team (Team ID: %v). "+
					"Give the CrunchyBridgeCluster CR a unique name, or if you would like to take control of the "+
					"existing cluster, add the 'postgres-operator.crunchydata.com/adopt-bridge-cluster' "+
					"annotation and set its value to the existing cluster's ID (Cluster ID: %v).", teamId, cluster.ID),
			})

			log.Info(fmt.Sprintf("A cluster with the same name already exists for this team (Team ID: %v). "+
				"Give the CrunchyBridgeCluster CR a unique name, or if you would like to take control "+
				"of the existing cluster, add the 'postgres-operator.crunchydata.com/adopt-bridge-cluster' "+
				"annotation and set its value to the existing cluster's ID (Cluster ID: %v).", teamId, cluster.ID))

			// We have an invalid cluster spec so we don't want to requeue
			return &ctrl.Result{}, nil
		}
	}

	return nil, nil
}

// handleCreateCluster handles creating new Crunchy Bridge Clusters
func (r *CrunchyBridgeClusterReconciler) handleCreateCluster(ctx context.Context,
	apiKey, teamId string, crunchybridgecluster *v1beta1.CrunchyBridgeCluster,
) ctrl.Result {
	log := ctrl.LoggerFrom(ctx)

	createClusterRequestPayload := &bridge.PostClustersRequestPayload{
		IsHA:            crunchybridgecluster.Spec.IsHA,
		Name:            crunchybridgecluster.Spec.ClusterName,
		Plan:            crunchybridgecluster.Spec.Plan,
		PostgresVersion: intstr.FromInt(crunchybridgecluster.Spec.PostgresVersion),
		Provider:        crunchybridgecluster.Spec.Provider,
		Region:          crunchybridgecluster.Spec.Region,
		Storage:         bridge.ToGibibytes(crunchybridgecluster.Spec.Storage),
		Team:            teamId,
	}
	cluster, err := r.NewClient().CreateCluster(ctx, apiKey, createClusterRequestPayload)
	if err != nil {
		log.Error(err, "issue creating cluster in Bridge")
		// TODO(crunchybridgecluster): probably shouldn't set this condition unless response from Bridge
		// indicates the payload is wrong
		// Otherwise want a different condition
		meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
			ObservedGeneration: crunchybridgecluster.GetGeneration(),
			Type:               v1beta1.ConditionReady,
			Status:             metav1.ConditionFalse,
			Reason:             "ClusterInvalid",
			Message: fmt.Sprintf(
				"Cannot create from spec: %v", err),
		})

		// TODO(crunchybridgecluster): If the payload is wrong, we don't want to requeue, so pass nil error
		// If the transmission hit a transient problem, we do want to requeue
		return ctrl.Result{}
	}
	crunchybridgecluster.Status.ID = cluster.ID

	meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
		ObservedGeneration: crunchybridgecluster.GetGeneration(),
		Type:               v1beta1.ConditionReady,
		Status:             metav1.ConditionUnknown,
		Reason:             "UnknownClusterState",
		Message:            "The condition of the cluster is unknown.",
	})

	meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
		ObservedGeneration: crunchybridgecluster.GetGeneration(),
		Type:               v1beta1.ConditionUpgrading,
		Status:             metav1.ConditionUnknown,
		Reason:             "UnknownUpgradeState",
		Message:            "The condition of the upgrade(s) is unknown.",
	})

	return runtime.RequeueWithoutBackoff(3 * time.Minute)
}

// handleGetCluster handles getting the cluster details from Bridge and
// updating the cluster CR's Status accordingly
func (r *CrunchyBridgeClusterReconciler) handleGetCluster(ctx context.Context,
	apiKey string, crunchybridgecluster *v1beta1.CrunchyBridgeCluster,
) error {
	log := ctrl.LoggerFrom(ctx)

	clusterDetails, err := r.NewClient().GetCluster(ctx, apiKey, crunchybridgecluster.Status.ID)
	if err != nil {
		meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
			ObservedGeneration: crunchybridgecluster.GetGeneration(),
			Type:               v1beta1.ConditionReady,
			Status:             metav1.ConditionUnknown,
			Reason:             "UnknownClusterState",
			Message:            fmt.Sprintf("Issue getting cluster information from Bridge: %v", err),
		})
		log.Error(err, "issue getting cluster information from Bridge")
		return err
	}
	clusterDetails.AddDataToClusterStatus(crunchybridgecluster)

	return nil
}

// handleGetClusterStatus handles getting the cluster status from Bridge and
// updating the cluster CR's Status accordingly
func (r *CrunchyBridgeClusterReconciler) handleGetClusterStatus(ctx context.Context,
	apiKey string, crunchybridgecluster *v1beta1.CrunchyBridgeCluster,
) error {
	log := ctrl.LoggerFrom(ctx)

	clusterStatus, err := r.NewClient().GetClusterStatus(ctx, apiKey, crunchybridgecluster.Status.ID)
	if err != nil {
		meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
			ObservedGeneration: crunchybridgecluster.GetGeneration(),
			Type:               v1beta1.ConditionReady,
			Status:             metav1.ConditionUnknown,
			Reason:             "UnknownClusterState",
			Message:            fmt.Sprintf("Issue getting cluster status from Bridge: %v", err),
		})
		crunchybridgecluster.Status.State = "unknown"
		log.Error(err, "issue getting cluster status from Bridge")
		return err
	}
	clusterStatus.AddDataToClusterStatus(crunchybridgecluster)

	if clusterStatus.State == "ready" {
		meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
			ObservedGeneration: crunchybridgecluster.GetGeneration(),
			Type:               v1beta1.ConditionReady,
			Status:             metav1.ConditionTrue,
			Reason:             clusterStatus.State,
			Message:            fmt.Sprintf("Bridge cluster state is %v.", clusterStatus.State),
		})
	} else {
		meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
			ObservedGeneration: crunchybridgecluster.GetGeneration(),
			Type:               v1beta1.ConditionReady,
			Status:             metav1.ConditionFalse,
			Reason:             clusterStatus.State,
			Message:            fmt.Sprintf("Bridge cluster state is %v.", clusterStatus.State),
		})
	}

	return nil
}

// handleGetClusterUpgrade handles getting the ongoing upgrade operations from Bridge and
// updating the cluster CR's Status accordingly
func (r *CrunchyBridgeClusterReconciler) handleGetClusterUpgrade(ctx context.Context,
	apiKey string,
	crunchybridgecluster *v1beta1.CrunchyBridgeCluster,
) error {
	log := ctrl.LoggerFrom(ctx)

	clusterUpgradeDetails, err := r.NewClient().GetClusterUpgrade(ctx, apiKey, crunchybridgecluster.Status.ID)
	if err != nil {
		meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
			ObservedGeneration: crunchybridgecluster.GetGeneration(),
			Type:               v1beta1.ConditionUpgrading,
			Status:             metav1.ConditionUnknown,
			Reason:             "UnknownUpgradeState",
			Message:            fmt.Sprintf("Issue getting cluster upgrade from Bridge: %v", err),
		})
		log.Error(err, "issue getting cluster upgrade from Bridge")
		return err
	}
	clusterUpgradeDetails.AddDataToClusterStatus(crunchybridgecluster)

	if len(clusterUpgradeDetails.Operations) != 0 {
		meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
			ObservedGeneration: crunchybridgecluster.GetGeneration(),
			Type:               v1beta1.ConditionUpgrading,
			Status:             metav1.ConditionTrue,
			Reason:             clusterUpgradeDetails.Operations[0].Flavor,
			Message: fmt.Sprintf(
				"Performing an upgrade of type %v with a state of %v.",
				clusterUpgradeDetails.Operations[0].Flavor, clusterUpgradeDetails.Operations[0].State),
		})
	} else {
		meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
			ObservedGeneration: crunchybridgecluster.GetGeneration(),
			Type:               v1beta1.ConditionUpgrading,
			Status:             metav1.ConditionFalse,
			Reason:             "NoUpgradesInProgress",
			Message:            "No upgrades being performed",
		})
	}

	return nil
}

// handleUpgrade handles upgrades that hit the "POST /clusters/<id>/upgrade" endpoint
func (r *CrunchyBridgeClusterReconciler) handleUpgrade(ctx context.Context,
	apiKey string,
	crunchybridgecluster *v1beta1.CrunchyBridgeCluster,
) ctrl.Result {
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
		meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
			ObservedGeneration: crunchybridgecluster.GetGeneration(),
			Type:               v1beta1.ConditionUpgrading,
			Status:             metav1.ConditionFalse,
			Reason:             "UpgradeError",
			Message: fmt.Sprintf(
				"Error performing an upgrade: %s", err),
		})
		log.Error(err, "Error while attempting cluster upgrade")
		return ctrl.Result{}
	}
	clusterUpgrade.AddDataToClusterStatus(crunchybridgecluster)

	if len(clusterUpgrade.Operations) != 0 {
		meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
			ObservedGeneration: crunchybridgecluster.GetGeneration(),
			Type:               v1beta1.ConditionUpgrading,
			Status:             metav1.ConditionTrue,
			Reason:             clusterUpgrade.Operations[0].Flavor,
			Message: fmt.Sprintf(
				"Performing an upgrade of type %v with a state of %v.",
				clusterUpgrade.Operations[0].Flavor, clusterUpgrade.Operations[0].State),
		})
	}

	return runtime.RequeueWithoutBackoff(3 * time.Minute)
}

// handleUpgradeHA handles upgrades that hit the
// "PUT /clusters/<id>/actions/[enable|disable]-ha" endpoint
func (r *CrunchyBridgeClusterReconciler) handleUpgradeHA(ctx context.Context,
	apiKey string,
	crunchybridgecluster *v1beta1.CrunchyBridgeCluster,
) ctrl.Result {
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
		meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
			ObservedGeneration: crunchybridgecluster.GetGeneration(),
			Type:               v1beta1.ConditionUpgrading,
			Status:             metav1.ConditionFalse,
			Reason:             "UpgradeError",
			Message: fmt.Sprintf(
				"Error performing an HA upgrade: %s", err),
		})
		log.Error(err, "Error while attempting cluster HA change")
		return ctrl.Result{}
	}
	clusterUpgrade.AddDataToClusterStatus(crunchybridgecluster)
	if len(clusterUpgrade.Operations) != 0 {
		meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
			ObservedGeneration: crunchybridgecluster.GetGeneration(),
			Type:               v1beta1.ConditionUpgrading,
			Status:             metav1.ConditionTrue,
			Reason:             clusterUpgrade.Operations[0].Flavor,
			Message: fmt.Sprintf(
				"Performing an upgrade of type %v with a state of %v.",
				clusterUpgrade.Operations[0].Flavor, clusterUpgrade.Operations[0].State),
		})
	}

	return runtime.RequeueWithoutBackoff(3 * time.Minute)
}

// handleUpdate handles upgrades that hit the "PATCH /clusters/<id>" endpoint
func (r *CrunchyBridgeClusterReconciler) handleUpdate(ctx context.Context,
	apiKey string,
	crunchybridgecluster *v1beta1.CrunchyBridgeCluster,
) ctrl.Result {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Handling update request")

	updateRequest := &bridge.PatchClustersRequestPayload{
		IsProtected: &crunchybridgecluster.Spec.IsProtected,
		Name:        crunchybridgecluster.Spec.ClusterName,
	}

	clusterUpdate, err := r.NewClient().UpdateCluster(ctx, apiKey,
		crunchybridgecluster.Status.ID, updateRequest)
	if err != nil {
		// TODO(crunchybridgecluster): consider what errors we might get
		// and what different results/requeue times we want to return.
		// Currently: don't requeue and wait for user to change spec.
		meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
			ObservedGeneration: crunchybridgecluster.GetGeneration(),
			Type:               v1beta1.ConditionUpgrading,
			Status:             metav1.ConditionFalse,
			Reason:             "UpgradeError",
			Message: fmt.Sprintf(
				"Error performing an upgrade: %s", err),
		})
		log.Error(err, "Error while attempting cluster update")
		return ctrl.Result{}
	}
	clusterUpdate.AddDataToClusterStatus(crunchybridgecluster)
	meta.SetStatusCondition(&crunchybridgecluster.Status.Conditions, metav1.Condition{
		ObservedGeneration: crunchybridgecluster.GetGeneration(),
		Type:               v1beta1.ConditionUpgrading,
		Status:             metav1.ConditionTrue,
		Reason:             "ClusterUpgrade",
		Message: fmt.Sprintf(
			"An upgrade is occurring, the clusters name is %v and the cluster is protected is %v.",
			clusterUpdate.ClusterName, *clusterUpdate.IsProtected),
	})

	return runtime.RequeueWithoutBackoff(3 * time.Minute)
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
		err = fmt.Errorf("error handling secret; expected to find a key and a team: found key %t, found team %t",
			existing.Data["key"] != nil,
			existing.Data["team"] != nil)
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
