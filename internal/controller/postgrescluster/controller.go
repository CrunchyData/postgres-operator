package postgrescluster

/*
Copyright 2021 Crunchy Data Solutions, Inc.
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
	"io"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
)

const (
	// ControllerName is the name of the PostgresCluster controller
	ControllerName = "postgrescluster-controller"

	// LabelPostgresCluster is used to indicate the name of the PostgresCluster a specific resource
	// is associated with
	LabelPostgresCluster = LabelPrefix + "postgrescluster"

	// LabelPrefix the prefix that should be appended to any labels created by the PostgresCluster
	// controller
	LabelPrefix = "crunchydata.com/"

	// workerCount defines the number of worker queues for the PostgresCluster controller
	workerCount = 2
)

// Reconciler holds resources for the PostgresCluster reconciler
type Reconciler struct {
	Client   client.Client
	Owner    client.FieldOwner
	Recorder record.EventRecorder
	Tracer   trace.Tracer

	PodExec func(
		namespace, pod, container string,
		stdin io.Reader, stdout, stderr io.Writer, command ...string,
	) error
}

// +kubebuilder:rbac:groups=postgres-operator.crunchydata.com,resources=postgresclusters,verbs=get;list;watch
// +kubebuilder:rbac:groups=postgres-operator.crunchydata.com,resources=postgresclusters/status,verbs=get;patch

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
	cluster := &v1alpha1.PostgresCluster{}
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

	// Keep a copy of cluster prior to any manipulations.
	before := cluster.DeepCopy()

	// Check for and handle deletion of cluster. Return early if it is being
	// deleted or there was an error.
	if result, err := r.handleDelete(ctx, cluster); err != nil {
		span.RecordError(err)
		return reconcile.Result{}, err
	} else if result != nil {
		return *result, nil
	}

	var (
		clusterConfigMap     *v1.ConfigMap
		clusterPodService    *v1.Service
		patroniLeaderService *v1.Service
		err                  error
	)

	// TODO(cbandy): Accumulate postgres settings.

	pgHBAs := postgres.HBAs{}
	pgHBAs.Mandatory = append(pgHBAs.Mandatory, *postgres.NewHBA().Local().User("postgres").Method("peer"))
	pgHBAs.Mandatory = append(pgHBAs.Mandatory, *postgres.NewHBA().TCP().Replication().Method("trust"))
	pgHBAs.Default = append(pgHBAs.Default, *postgres.NewHBA().TCP().Method("scram-sha-256"))

	pgParameters := postgres.Parameters{}
	pgParameters.Mandatory = postgres.NewParameterSet()
	pgParameters.Mandatory.Add("wal_level", "logical")
	pgParameters.Default = postgres.NewParameterSet()
	pgParameters.Default.Add("jit", "off")

	if err == nil {
		err = r.reconcilePatroniStatus(ctx, cluster)
	}
	if err == nil {
		clusterConfigMap, err = r.reconcileClusterConfigMap(ctx, cluster, pgHBAs, pgParameters)
	}
	if err == nil {
		clusterPodService, err = r.reconcileClusterPodService(ctx, cluster)
	}
	if err == nil {
		patroniLeaderService, err = r.reconcilePatroniLeaderLease(ctx, cluster)
	}
	if err == nil {
		err = r.reconcileClusterPrimaryService(ctx, cluster, patroniLeaderService)
	}
	if err == nil {
		// TODO(tjmoore4): will need to do something with the returned secret
		_, err = r.reconcilePGUserSecret(ctx, cluster)
	}
	if err == nil {
		err = r.reconcilePatroniDistributedConfiguration(ctx, cluster)
	}
	if err == nil {
		err = r.reconcilePatroniDynamicConfiguration(ctx, cluster, pgHBAs, pgParameters)
	}

	for i := range cluster.Spec.InstanceSets {
		if err == nil {
			_, err = r.reconcileInstanceSet(
				ctx, cluster, &cluster.Spec.InstanceSets[i],
				clusterConfigMap, clusterPodService, patroniLeaderService)
		}
	}

	if err == nil {
		err = updateResult(r.reconcilePGBackRest(ctx, cluster))
	}

	// TODO reconcile pgBouncer

	// TODO reconcile pgadmin4

	// at this point everything reconciled successfully, and we can update the
	// observedGeneration
	cluster.Status.ObservedGeneration = cluster.GetGeneration()

	if err == nil && !equality.Semantic.DeepEqual(before.Status, cluster.Status) {
		// NOTE(cbandy): Kubernetes prior to v1.16.10 and v1.17.6 does not track
		// managed fields on the status subresource: https://issue.k8s.io/88901
		err = errors.WithStack(
			r.Client.Status().Patch(ctx, cluster, client.MergeFrom(before), r.Owner))
	}

	log.V(1).Info("reconciled cluster")

	return result, err
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

// setControllerReference sets owner as a Controller OwnerReference on controlled.
// Only one OwnerReference can be a controller, so it returns an error if another
// is already set.
func (r *Reconciler) setControllerReference(
	owner *v1alpha1.PostgresCluster, controlled client.Object,
) error {
	return controllerutil.SetControllerReference(owner, controlled, r.Client.Scheme())
}

// SetupWithManager adds the PostgresCluster controller to the provided runtime manager
func (r *Reconciler) SetupWithManager(mgr manager.Manager) error {
	if r.PodExec == nil {
		var err error
		r.PodExec, err = newPodExecutor(mgr.GetConfig())
		if err != nil {
			return err
		}
	}

	return builder.ControllerManagedBy(mgr).
		For(&v1alpha1.PostgresCluster{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: workerCount,
		}).
		Owns(&v1.ConfigMap{}).
		Owns(&v1.Endpoints{}).
		Owns(&v1.Secret{}).
		Owns(&v1.Service{}).
		Owns(&appsv1.StatefulSet{}).
		Watches(&source.Kind{Type: &appsv1.StatefulSet{}},
			r.statefulSetControllerRefHandlerFuncs()). // watch all StatefulSets
		Complete(r)
}
