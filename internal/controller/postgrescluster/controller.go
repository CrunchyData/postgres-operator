// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"go.opentelemetry.io/otel/trace"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/pgaudit"
	"github.com/crunchydata/postgres-operator/internal/pgbackrest"
	"github.com/crunchydata/postgres-operator/internal/pgbouncer"
	"github.com/crunchydata/postgres-operator/internal/pgmonitor"
	"github.com/crunchydata/postgres-operator/internal/pki"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/internal/registration"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

const (
	// ControllerName is the name of the PostgresCluster controller
	ControllerName = "postgrescluster-controller"
)

// Reconciler holds resources for the PostgresCluster reconciler
type Reconciler struct {
	Client          client.Client
	DiscoveryClient *discovery.DiscoveryClient
	IsOpenShift     bool
	Owner           client.FieldOwner
	PodExec         func(
		ctx context.Context, namespace, pod, container string,
		stdin io.Reader, stdout, stderr io.Writer, command ...string,
	) error
	Recorder     record.EventRecorder
	Registration registration.Registration
	Tracer       trace.Tracer
}

// +kubebuilder:rbac:groups="",resources="events",verbs={create,patch}
// +kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="postgresclusters",verbs={get,list,watch}
// +kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="postgresclusters/status",verbs={patch}

// Reconcile reconciles a ConfigMap in a namespace managed by the PostgreSQL Operator
func (r *Reconciler) Reconcile(
	ctx context.Context, request reconcile.Request) (reconcile.Result, error,
) {
	ctx, span := r.Tracer.Start(ctx, "Reconcile")
	log := logging.FromContext(ctx)
	defer span.End()

	// get the postgrescluster from the cache
	cluster := &v1beta1.PostgresCluster{}
	if err := r.Client.Get(ctx, request.NamespacedName, cluster); err != nil {
		// NotFound cannot be fixed by requeuing so ignore it. During background
		// deletion, we receive delete events from cluster's dependents after
		// cluster is deleted.
		if err = client.IgnoreNotFound(err); err != nil {
			log.Error(err, "unable to fetch PostgresCluster")
			span.RecordError(err)
		}
		return runtime.ErrorWithBackoff(err)
	}

	// Set any defaults that may not have been stored in the API. No DeepCopy
	// is necessary because controller-runtime makes a copy before returning
	// from its cache.
	cluster.Default()

	if cluster.Spec.OpenShift == nil {
		cluster.Spec.OpenShift = &r.IsOpenShift
	}

	// Keep a copy of cluster prior to any manipulations.
	before := cluster.DeepCopy()

	// NOTE(cbandy): When a namespace is deleted, objects owned by a
	// PostgresCluster may be deleted before the PostgresCluster is deleted.
	// When this happens, any attempt to reconcile those objects is rejected
	// as Forbidden: "unable to create new content in namespace … because it is
	// being terminated".

	// Check for and handle deletion of cluster. Return early if it is being
	// deleted or there was an error.
	if result, err := r.handleDelete(ctx, cluster); err != nil {
		span.RecordError(err)
		log.Error(err, "deleting")
		return runtime.ErrorWithBackoff(err)

	} else if result != nil {
		if log := log.V(1); log.Enabled() {
			log.Info("deleting", "result", fmt.Sprintf("%+v", *result))
		}
		return *result, nil
	}

	// Perform initial validation on a cluster
	// TODO: Move this to a defaulting (mutating admission) webhook
	// to leverage regular validation.

	// verify all needed image values are defined
	if err := config.VerifyImageValues(cluster); err != nil {
		// warning event with missing image information
		r.Recorder.Event(cluster, corev1.EventTypeWarning, "MissingRequiredImage",
			err.Error())
		// specifically allow reconciliation if the cluster is shutdown to
		// facilitate upgrades, otherwise return
		if !initialize.FromPointer(cluster.Spec.Shutdown) {
			return runtime.ErrorWithBackoff(err)
		}
	}

	if cluster.Spec.Standby != nil &&
		cluster.Spec.Standby.Enabled &&
		cluster.Spec.Standby.Host == "" &&
		cluster.Spec.Standby.RepoName == "" {
		// When a standby cluster is requested but a repoName or host is not provided
		// the cluster will be created as a non-standby. Reject any clusters with
		// this configuration and provide an event
		path := field.NewPath("spec", "standby")
		err := field.Invalid(path, cluster.Name, "Standby requires a host or repoName to be enabled")
		r.Recorder.Event(cluster, corev1.EventTypeWarning, "InvalidStandbyConfiguration", err.Error())
		return runtime.ErrorWithBackoff(err)
	}

	var (
		clusterConfigMap             *corev1.ConfigMap
		clusterReplicationSecret     *corev1.Secret
		clusterPodService            *corev1.Service
		clusterVolumes               []corev1.PersistentVolumeClaim
		instanceServiceAccount       *corev1.ServiceAccount
		instances                    *observedInstances
		patroniLeaderService         *corev1.Service
		primaryCertificate           *corev1.SecretProjection
		primaryService               *corev1.Service
		replicaService               *corev1.Service
		rootCA                       *pki.RootCertificateAuthority
		monitoringSecret             *corev1.Secret
		exporterQueriesConfig        *corev1.ConfigMap
		exporterWebConfig            *corev1.ConfigMap
		err                          error
		backupsSpecFound             bool
		backupsReconciliationAllowed bool
	)

	patchClusterStatus := func() error {
		if !equality.Semantic.DeepEqual(before.Status, cluster.Status) {
			// NOTE(cbandy): Kubernetes prior to v1.16.10 and v1.17.6 does not track
			// managed fields on the status subresource: https://issue.k8s.io/88901
			if err := r.Client.Status().Patch(
				ctx, cluster, client.MergeFrom(before), r.Owner); err != nil {
				log.Error(err, "patching cluster status")
				return err
			}
			log.V(1).Info("patched cluster status")
		}
		return nil
	}

	if r.Registration != nil && r.Registration.Required(r.Recorder, cluster, &cluster.Status.Conditions) {
		registration.SetAdvanceWarning(r.Recorder, cluster, &cluster.Status.Conditions)
	}
	cluster.Status.RegistrationRequired = nil
	cluster.Status.TokenRequired = ""

	// if the cluster is paused, set a condition and return
	if cluster.Spec.Paused != nil && *cluster.Spec.Paused {
		meta.SetStatusCondition(&cluster.Status.Conditions, metav1.Condition{
			Type:    v1beta1.PostgresClusterProgressing,
			Status:  metav1.ConditionFalse,
			Reason:  "Paused",
			Message: "No spec changes will be applied and no other statuses will be updated.",

			ObservedGeneration: cluster.GetGeneration(),
		})
		return runtime.ErrorWithBackoff(patchClusterStatus())
	} else {
		meta.RemoveStatusCondition(&cluster.Status.Conditions, v1beta1.PostgresClusterProgressing)
	}

	if err == nil {
		backupsSpecFound, backupsReconciliationAllowed, err = r.BackupsEnabled(ctx, cluster)

		// If we cannot reconcile because the backup reconciliation is paused, set a condition and exit
		if !backupsReconciliationAllowed {
			meta.SetStatusCondition(&cluster.Status.Conditions, metav1.Condition{
				Type:   v1beta1.PostgresClusterProgressing,
				Status: metav1.ConditionFalse,
				Reason: "Paused",
				Message: "Reconciliation is paused: please fill in spec.backups " +
					"or add the postgres-operator.crunchydata.com/authorizeBackupRemoval " +
					"annotation to authorize backup removal.",

				ObservedGeneration: cluster.GetGeneration(),
			})
			return runtime.ErrorWithBackoff(patchClusterStatus())
		} else {
			meta.RemoveStatusCondition(&cluster.Status.Conditions, v1beta1.PostgresClusterProgressing)
		}
	}

	pgHBAs := postgres.NewHBAs()
	pgmonitor.PostgreSQLHBAs(cluster, &pgHBAs)
	pgbouncer.PostgreSQL(cluster, &pgHBAs)

	pgParameters := postgres.NewParameters()
	pgaudit.PostgreSQLParameters(&pgParameters)
	pgbackrest.PostgreSQL(cluster, &pgParameters, backupsSpecFound)
	pgmonitor.PostgreSQLParameters(cluster, &pgParameters)

	// Set huge_pages = try if a hugepages resource limit > 0, otherwise set "off"
	postgres.SetHugePages(cluster, &pgParameters)

	if err == nil {
		rootCA, err = r.reconcileRootCertificate(ctx, cluster)
	}

	if err == nil {
		// Since any existing data directories must be moved prior to bootstrapping the
		// cluster, further reconciliation will not occur until the directory move Jobs
		// (if configured) have completed. Func reconcileDirMoveJobs() will therefore
		// return a bool indicating that the controller should return early while any
		// required Jobs are running, after which it will indicate that an early
		// return is no longer needed, and reconciliation can proceed normally.
		returnEarly, err := r.reconcileDirMoveJobs(ctx, cluster)
		if err != nil || returnEarly {
			return runtime.ErrorWithBackoff(errors.Join(err, patchClusterStatus()))
		}
	}
	if err == nil {
		clusterVolumes, err = r.observePersistentVolumeClaims(ctx, cluster)
	}
	if err == nil {
		clusterVolumes, err = r.configureExistingPVCs(ctx, cluster, clusterVolumes)
	}
	if err == nil {
		instances, err = r.observeInstances(ctx, cluster)
	}

	result := reconcile.Result{}

	if err == nil {
		var requeue time.Duration
		if requeue, err = r.reconcilePatroniStatus(ctx, cluster, instances); err == nil && requeue > 0 {
			result.RequeueAfter = requeue
		}
	}
	if err == nil {
		err = r.reconcilePatroniSwitchover(ctx, cluster, instances)
	}
	// reconcile the Pod service before reconciling any data source in case it is necessary
	// to start Pods during data source reconciliation that require network connections (e.g.
	// if it is necessary to start a dedicated repo host to bootstrap a new cluster using its
	// own existing backups).
	if err == nil {
		clusterPodService, err = r.reconcileClusterPodService(ctx, cluster)
	}
	// reconcile the RBAC resources before reconciling any data source in case
	// restore/move Job pods require the ServiceAccount to access any data source.
	// e.g., we are restoring from an S3 source using an IAM for access
	// - https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts-technical-overview.html
	if err == nil {
		instanceServiceAccount, err = r.reconcileRBACResources(ctx, cluster)
	}
	// First handle reconciling any data source configured for the PostgresCluster.  This includes
	// reconciling the data source defined to bootstrap a new cluster, as well as a reconciling
	// a data source to perform restore in-place and re-bootstrap the cluster.
	if err == nil {
		// Since the PostgreSQL data source needs to be populated prior to bootstrapping the
		// cluster, further reconciliation will not occur until the data source (if configured) is
		// initialized.  Func reconcileDataSource() will therefore return a bool indicating that
		// the controller should return early while data initialization is in progress, after
		// which it will indicate that an early return is no longer needed, and reconciliation
		// can proceed normally.
		returnEarly, err := r.reconcileDataSource(ctx, cluster, instances, clusterVolumes, rootCA, backupsSpecFound)
		if err != nil || returnEarly {
			return runtime.ErrorWithBackoff(errors.Join(err, patchClusterStatus()))
		}
	}
	if err == nil {
		clusterConfigMap, err = r.reconcileClusterConfigMap(ctx, cluster, pgHBAs, pgParameters)
	}
	if err == nil {
		clusterReplicationSecret, err = r.reconcileReplicationSecret(ctx, cluster, rootCA)
	}
	if err == nil {
		patroniLeaderService, err = r.reconcilePatroniLeaderLease(ctx, cluster)
	}
	if err == nil {
		primaryService, err = r.reconcileClusterPrimaryService(ctx, cluster, patroniLeaderService)
	}
	if err == nil {
		replicaService, err = r.reconcileClusterReplicaService(ctx, cluster)
	}
	if err == nil {
		primaryCertificate, err = r.reconcileClusterCertificate(ctx, rootCA, cluster, primaryService, replicaService)
	}
	if err == nil {
		err = r.reconcilePatroniDistributedConfiguration(ctx, cluster)
	}
	if err == nil {
		err = r.reconcilePatroniDynamicConfiguration(ctx, cluster, instances, pgHBAs, pgParameters)
	}
	if err == nil {
		monitoringSecret, err = r.reconcileMonitoringSecret(ctx, cluster)
	}
	if err == nil {
		exporterQueriesConfig, err = r.reconcileExporterQueriesConfig(ctx, cluster)
	}
	if err == nil {
		exporterWebConfig, err = r.reconcileExporterWebConfig(ctx, cluster)
	}
	if err == nil {
		err = r.reconcileInstanceSets(
			ctx, cluster, clusterConfigMap, clusterReplicationSecret, rootCA,
			clusterPodService, instanceServiceAccount, instances, patroniLeaderService,
			primaryCertificate, clusterVolumes, exporterQueriesConfig, exporterWebConfig,
			backupsSpecFound,
		)
	}

	if err == nil {
		err = r.reconcilePostgresDatabases(ctx, cluster, instances)
	}
	if err == nil {
		err = r.reconcilePostgresUsers(ctx, cluster, instances)
	}

	if err == nil {
		var next reconcile.Result
		if next, err = r.reconcilePGBackRest(ctx, cluster,
			instances, rootCA, backupsSpecFound); err == nil && !next.IsZero() {
			result.Requeue = result.Requeue || next.Requeue
			if next.RequeueAfter > 0 {
				result.RequeueAfter = next.RequeueAfter
			}
		}
	}
	if err == nil {
		err = r.reconcileVolumeSnapshots(ctx, cluster, instances, clusterVolumes)
	}
	if err == nil {
		err = r.reconcilePGBouncer(ctx, cluster, instances, primaryCertificate, rootCA)
	}
	if err == nil {
		err = r.reconcilePGMonitor(ctx, cluster, instances, monitoringSecret)
	}
	if err == nil {
		err = r.reconcileDatabaseInitSQL(ctx, cluster, instances)
	}
	if err == nil {
		err = r.reconcilePGAdmin(ctx, cluster)
	}
	if err == nil {
		// This is after [Reconciler.rolloutInstances] to ensure that recreating
		// Pods takes precedence.
		err = r.handlePatroniRestarts(ctx, cluster, instances)
	}

	// at this point everything reconciled successfully, and we can update the
	// observedGeneration
	cluster.Status.ObservedGeneration = cluster.GetGeneration()

	log.V(1).Info("reconciled cluster")

	return result, errors.Join(err, patchClusterStatus())
}

// deleteControlled safely deletes object when it is controlled by cluster.
func (r *Reconciler) deleteControlled(
	ctx context.Context, cluster *v1beta1.PostgresCluster, object client.Object,
) error {
	if metav1.IsControlledBy(object, cluster) {
		uid := object.GetUID()
		version := object.GetResourceVersion()
		exactly := client.Preconditions{UID: &uid, ResourceVersion: &version}

		return r.Client.Delete(ctx, object, exactly)
	}

	return nil
}

// patch sends patch to object's endpoint in the Kubernetes API and updates
// object with any returned content. The fieldManager is set to r.Owner, but
// can be overridden in options.
// - https://docs.k8s.io/reference/using-api/server-side-apply/#managers
func (r *Reconciler) patch(
	ctx context.Context, object client.Object,
	patch client.Patch, options ...client.PatchOption,
) error {
	options = append([]client.PatchOption{r.Owner}, options...)
	return r.Client.Patch(ctx, object, patch, options...)
}

// The owner reference created by controllerutil.SetControllerReference blocks
// deletion. The OwnerReferencesPermissionEnforcement plugin requires that the
// creator of such a reference have either "delete" permission on the owner or
// "update" permission on the owner's "finalizers" subresource.
// - https://docs.k8s.io/reference/access-authn-authz/admission-controllers/
// +kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="postgresclusters/finalizers",verbs={update}

// setControllerReference sets owner as a Controller OwnerReference on controlled.
// Only one OwnerReference can be a controller, so it returns an error if another
// is already set.
func (r *Reconciler) setControllerReference(
	owner *v1beta1.PostgresCluster, controlled client.Object,
) error {
	return controllerutil.SetControllerReference(owner, controlled, r.Client.Scheme())
}

// setOwnerReference sets an OwnerReference on the object without setting the
// owner as a controller. This allows for multiple OwnerReferences on an object.
func (r *Reconciler) setOwnerReference(
	owner *v1beta1.PostgresCluster, controlled client.Object,
) error {
	return controllerutil.SetOwnerReference(owner, controlled, r.Client.Scheme())
}

// +kubebuilder:rbac:groups="",resources="configmaps",verbs={get,list,watch}
// +kubebuilder:rbac:groups="",resources="endpoints",verbs={get,list,watch}
// +kubebuilder:rbac:groups="",resources="persistentvolumeclaims",verbs={get,list,watch}
// +kubebuilder:rbac:groups="",resources="secrets",verbs={get,list,watch}
// +kubebuilder:rbac:groups="",resources="services",verbs={get,list,watch}
// +kubebuilder:rbac:groups="",resources="serviceaccounts",verbs={get,list,watch}
// +kubebuilder:rbac:groups="apps",resources="deployments",verbs={get,list,watch}
// +kubebuilder:rbac:groups="apps",resources="statefulsets",verbs={get,list,watch}
// +kubebuilder:rbac:groups="batch",resources="jobs",verbs={get,list,watch}
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources="roles",verbs={get,list,watch}
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources="rolebindings",verbs={get,list,watch}
// +kubebuilder:rbac:groups="batch",resources="cronjobs",verbs={get,list,watch}
// +kubebuilder:rbac:groups="policy",resources="poddisruptionbudgets",verbs={get,list,watch}

// SetupWithManager adds the PostgresCluster controller to the provided runtime manager
func (r *Reconciler) SetupWithManager(mgr manager.Manager) error {
	if r.PodExec == nil {
		var err error
		r.PodExec, err = runtime.NewPodExecutor(mgr.GetConfig())
		if err != nil {
			return err
		}
	}

	if r.DiscoveryClient == nil {
		var err error
		r.DiscoveryClient, err = discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
		if err != nil {
			return err
		}
	}

	return builder.ControllerManagedBy(mgr).
		For(&v1beta1.PostgresCluster{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Endpoints{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&batchv1.Job{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Owns(&batchv1.CronJob{}).
		Owns(&policyv1.PodDisruptionBudget{}).
		Watches(&corev1.Pod{}, r.watchPods()).
		Watches(&appsv1.StatefulSet{},
			r.controllerRefHandlerFuncs()). // watch all StatefulSets
		Complete(r)
}

// GroupVersionKindExists checks to see whether a given Kind for a given
// GroupVersion exists in the Kubernetes API Server.
func (r *Reconciler) GroupVersionKindExists(groupVersion, kind string) (*bool, error) {
	if r.DiscoveryClient == nil {
		return initialize.Bool(false), nil
	}

	resourceList, err := r.DiscoveryClient.ServerResourcesForGroupVersion(groupVersion)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return initialize.Bool(false), nil
		}

		return nil, err
	}

	for _, resource := range resourceList.APIResources {
		if resource.Kind == kind {
			return initialize.Bool(true), nil
		}
	}

	return initialize.Bool(false), nil
}
