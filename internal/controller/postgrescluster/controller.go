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
	"reflect"

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

	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
)

const workerCount = 2

// Reconciler holds resources for the PostgresCluster reconciler
type Reconciler struct {
	Client   client.Client
	Owner    client.FieldOwner
	Recorder record.EventRecorder
	Tracer   trace.Tracer
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

	// get the postgrescluster from the cache
	postgresCluster := &v1alpha1.PostgresCluster{}
	if err := r.Client.Get(ctx, request.NamespacedName, postgresCluster); err != nil {
		log.Error(err, "cannot retrieve postgrescluster")
		span.RecordError(err)

		// returning an error will cause the work to be requeued
		return reconcile.Result{}, err
	}

	// an example of creating an event
	r.Recorder.Eventf(postgresCluster, v1.EventTypeNormal, "Initializing",
		"Initializing postgrescluster %s", request.NamespacedName)

	// call business logic to reconcile the postgrescluster
	cluster := postgresCluster.DeepCopy()
	cluster.Default()

	// Keep a copy of cluster prior to any manipulations.
	before := cluster.DeepCopy()

	var (
		clusterPodService *v1.Service
		clusterService    *v1.Service
		clusterConfigMap  *v1.ConfigMap
		err               error
	)

	if err == nil {
		clusterService, err = r.reconcileClusterService(ctx, cluster)
	}
	if err == nil {
		clusterConfigMap, err = r.reconcileClusterConfigMap(ctx, cluster)
	}
	if err == nil {
		clusterPodService, err = r.reconcileClusterPodService(ctx, cluster)
	}
	if err == nil {
		err = r.reconcilePatroniDistributedConfiguration(ctx, cluster)
	}

	for i := range cluster.Spec.InstanceSets {
		if err == nil {
			_, err = r.reconcileInstanceSet(
				ctx, cluster, &cluster.Spec.InstanceSets[i],
				clusterConfigMap, clusterPodService, clusterService)
		}
	}

	if err == nil {
		cluster.Status.ObservedGeneration = cluster.Generation

		// NOTE(cbandy): Kubernetes prior to v1.16.10 and v1.17.6 does not track
		// managed fields on the status subresource: https://issue.k8s.io/88901
		if !equality.Semantic.DeepEqual(before.Status, cluster.Status) {
			err = errors.WithStack(
				r.Client.Status().Patch(
					ctx, cluster, client.MergeFrom(before), r.Owner))
		}
	}

	if err == nil {
		log.V(1).Info("reconciled cluster")
	} else {
		log.Error(err, "reconciling cluster")
		span.RecordError(err)
	}

	return reconcile.Result{}, err
}

// apply sends an apply patch to object's endpoint in the Kubernetes API and
// updates object with any returned content. The fieldManager is set to
// r.Owner, but can be overridden in options.
// - https://docs.k8s.io/reference/using-api/server-side-apply/#managers
func (r *Reconciler) apply(
	ctx context.Context, object client.Object, options ...client.PatchOption,
) error {
	zero := reflect.New(reflect.TypeOf(object).Elem()).Interface()
	data, err := client.MergeFrom(zero.(client.Object)).Data(object)
	patch := client.RawPatch(client.Apply.Type(), data)

	if err == nil {
		err = r.patch(ctx, object, patch, options...)
	}
	return err
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
	return builder.ControllerManagedBy(mgr).
		For(&v1alpha1.PostgresCluster{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: workerCount,
		}).
		Owns(&v1.ConfigMap{}).
		Owns(&v1.Service{}).
		Owns(&appsv1.StatefulSet{}).
		Complete(r)
}
