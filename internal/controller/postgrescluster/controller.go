package postgrescluster

/*
Copyright 2020 Crunchy Data Solutions, Inc.
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

	"go.opentelemetry.io/otel/trace"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
)

const workerCount = 2

// Reconciler holds resources for the PostgresCluster reconciler
type Reconciler struct {
	Client   client.Client
	Recorder record.EventRecorder
	Tracer   trace.Tracer
}

// Reconcile reconciles a ConfigMap in a namespace managed by the PostgreSQL Operator
func (r *Reconciler) Reconcile(
	ctx context.Context, request reconcile.Request) (reconcile.Result, error,
) {
	ctx, span := r.Tracer.Start(ctx, "Reconcile")
	log := logging.FromContext(ctx).WithValues("postgrescluster", request.NamespacedName)
	defer span.End()

	// get the postgrescluster from the cache
	postgresCluster := &v1alpha1.PostgresCluster{}
	if err := r.Client.Get(ctx, request.NamespacedName, postgresCluster); err != nil {
		log.Error(err, "cannot retrieve postgrescluster")
		span.RecordError(err)

		// returning an error will cause the work to be requeued
		return reconcile.Result{}, err
	}

	log.V(1).Info("reconciling")

	// an example of creating an event
	r.Recorder.Eventf(postgresCluster, v1.EventTypeNormal, "Initializing",
		"Initializing postgrescluster %s", request.NamespacedName)

	// call business logic to reconcile the postgrescluster

	return reconcile.Result{}, nil
}

// SetupWithManager adds the PostgresCluster controller to the provided runtime manager
func (r *Reconciler) SetupWithManager(mgr manager.Manager) error {
	// create a controller for the PostgresCluster custom resource
	return builder.ControllerManagedBy(mgr).
		For(&v1alpha1.PostgresCluster{}).
		WithEventFilter(predicate.ResourceVersionChangedPredicate{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: workerCount,
		}).
		Complete(r)
}
