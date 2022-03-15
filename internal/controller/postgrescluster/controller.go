package postgrescluster

/*
Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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
	"io"
	"os"
	"strconv"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/pgaudit"
	"github.com/crunchydata/postgres-operator/internal/pgbackrest"
	"github.com/crunchydata/postgres-operator/internal/pgbouncer"
	"github.com/crunchydata/postgres-operator/internal/pgmonitor"
	"github.com/crunchydata/postgres-operator/internal/pki"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

const (
	// ControllerName is the name of the PostgresCluster controller
	ControllerName = "postgrescluster-controller"
)

// Reconciler holds resources for the PostgresCluster reconciler
type Reconciler struct {
	Client      client.Client
	Owner       client.FieldOwner
	Recorder    record.EventRecorder
	Tracer      trace.Tracer
	IsOpenShift bool

	PodExec func(
		namespace, pod, container string,
		stdin io.Reader, stdout, stderr io.Writer, command ...string,
	) error
}

// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=postgres-operator.crunchydata.com,resources=postgresclusters,verbs=get;list;watch
// +kubebuilder:rbac:groups=postgres-operator.crunchydata.com,resources=postgresclusters/status,verbs=patch

// Reconcile reconciles a ConfigMap in a namespace managed by the PostgreSQL Operator
func (r *Reconciler) Reconcile(
	ctx context.Context, request reconcile.Request) (reconcile.Result, error,
) {
	ctx, span := r.Tracer.Start(ctx, "Reconcile")
	log := logging.FromContext(ctx)
	defer span.End()

	// create the result that will be updated following a call to each reconciler
	result := reconcile.Result{}
	updateResult := func(next reconcile.Result, err error) error {
		if err == nil {
			result = updateReconcileResult(result, next)
		}
		return err
	}

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
		return result, err
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
		return reconcile.Result{}, err

	} else if result != nil {
		if log := log.V(1); log.Enabled() {
			if result.RequeueAfter > 0 {
				// RequeueAfter implies Requeue, but set both to make the next
				// log message more clear.
				result.Requeue = true
			}
			log.Info("deleting", "result", fmt.Sprintf("%+v", *result))
		}
		return *result, nil
	}

	var (
		clusterConfigMap         *corev1.ConfigMap
		clusterReplicationSecret *corev1.Secret
		clusterPodService        *corev1.Service
		clusterVolumes           []corev1.PersistentVolumeClaim
		instanceServiceAccount   *corev1.ServiceAccount
		instances                *observedInstances
		patroniLeaderService     *corev1.Service
		primaryCertificate       *corev1.SecretProjection
		primaryService           *corev1.Service
		rootCA                   *pki.RootCertificateAuthority
		monitoringSecret         *corev1.Secret
		err                      error
	)

	// Define the function for the updating the PostgresCluster status. Returns any error that
	// occurs while attempting to patch the status, while otherwise simply returning the
	// Result and error variables that are populated while reconciling the PostgresCluster.
	patchClusterStatus := func() (reconcile.Result, error) {
		if !equality.Semantic.DeepEqual(before.Status, cluster.Status) {
			// NOTE(cbandy): Kubernetes prior to v1.16.10 and v1.17.6 does not track
			// managed fields on the status subresource: https://issue.k8s.io/88901
			if err := errors.WithStack(r.Client.Status().Patch(
				ctx, cluster, client.MergeFrom(before), r.Owner)); err != nil {
				log.Error(err, "patching cluster status")
				return result, err
			}
			log.V(1).Info("patched cluster status")
		}
		return result, err
	}

	pgHBAs := postgres.NewHBAs()
	pgmonitor.PostgreSQLHBAs(cluster, &pgHBAs)
	pgbouncer.PostgreSQL(cluster, &pgHBAs)

	pgParameters := postgres.NewParameters()
	pgaudit.PostgreSQLParameters(&pgParameters)
	pgbackrest.PostgreSQL(cluster, &pgParameters)
	pgmonitor.PostgreSQLParameters(cluster, &pgParameters)

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
		var returnEarly bool
		returnEarly, err = r.reconcileDirMoveJobs(ctx, cluster)
		if err != nil || returnEarly {
			return patchClusterStatus()
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
	if err == nil {
		err = updateResult(r.reconcilePatroniStatus(ctx, cluster, instances))
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
		var returnEarly bool
		returnEarly, err = r.reconcileDataSource(ctx, cluster, instances, clusterVolumes, rootCA)
		if err != nil || returnEarly {
			return patchClusterStatus()
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
		err = r.reconcileClusterReplicaService(ctx, cluster)
	}
	if err == nil {
		primaryCertificate, err = r.reconcileClusterCertificate(ctx, rootCA, cluster, primaryService)
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
		err = r.reconcileInstanceSets(
			ctx, cluster, clusterConfigMap, clusterReplicationSecret,
			rootCA, clusterPodService, instanceServiceAccount, instances,
			patroniLeaderService, primaryCertificate, clusterVolumes)
	}

	if err == nil {
		err = r.reconcilePostgresDatabases(ctx, cluster, instances)
	}
	if err == nil {
		err = r.reconcilePostgresUsers(ctx, cluster, instances)
	}

	if err == nil {
		err = updateResult(r.reconcilePGBackRest(ctx, cluster, instances, rootCA))
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

	return patchClusterStatus()
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
// +kubebuilder:rbac:groups=postgres-operator.crunchydata.com,resources=postgresclusters/finalizers,verbs=update

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

// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=endpoints,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch

// SetupWithManager adds the PostgresCluster controller to the provided runtime manager
func (r *Reconciler) SetupWithManager(mgr manager.Manager) error {
	if r.PodExec == nil {
		var err error
		r.PodExec, err = newPodExecutor(mgr.GetConfig())
		if err != nil {
			return err
		}
	}

	var opts controller.Options

	// TODO(cbandy): Move this to main with controller-runtime v0.9+
	// - https://github.com/kubernetes-sigs/controller-runtime/commit/82fc2564cf
	if s := os.Getenv("PGO_WORKERS"); s != "" {
		if i, err := strconv.Atoi(s); err == nil && i > 0 {
			opts.MaxConcurrentReconciles = i
		} else {
			mgr.GetLogger().Error(err, "PGO_WORKERS must be a positive number")
		}
	}
	if opts.MaxConcurrentReconciles == 0 {
		opts.MaxConcurrentReconciles = 2
	}

	return builder.ControllerManagedBy(mgr).
		For(&v1beta1.PostgresCluster{}).
		WithOptions(opts).
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
		Owns(&batchv1beta1.CronJob{}).
		Owns(&policyv1beta1.PodDisruptionBudget{}).
		Watches(&source.Kind{Type: &corev1.Pod{}}, r.watchPods()).
		Watches(&source.Kind{Type: &appsv1.StatefulSet{}},
			r.controllerRefHandlerFuncs()). // watch all StatefulSets
		Complete(r)
}
