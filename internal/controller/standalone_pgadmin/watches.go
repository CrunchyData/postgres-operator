// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package standalone_pgadmin

import (
	"context"

	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// watchPostgresClusters returns a [handler.EventHandler] for PostgresClusters.
func (r *PGAdminReconciler) watchPostgresClusters() handler.Funcs {
	handle := func(ctx context.Context, cluster client.Object, q workqueue.RateLimitingInterface) {
		for _, pgadmin := range r.findPGAdminsForPostgresCluster(ctx, cluster) {

			q.Add(ctrl.Request{
				NamespacedName: client.ObjectKeyFromObject(pgadmin),
			})
		}
	}

	return handler.Funcs{
		CreateFunc: func(ctx context.Context, e event.CreateEvent, q workqueue.RateLimitingInterface) {
			handle(ctx, e.Object, q)
		},
		UpdateFunc: func(ctx context.Context, e event.UpdateEvent, q workqueue.RateLimitingInterface) {
			handle(ctx, e.ObjectNew, q)
		},
		DeleteFunc: func(ctx context.Context, e event.DeleteEvent, q workqueue.RateLimitingInterface) {
			handle(ctx, e.Object, q)
		},
	}
}

// watchForRelatedSecret handles create/update/delete events for secrets,
// passing the Secret ObjectKey to findPGAdminsForSecret
func (r *PGAdminReconciler) watchForRelatedSecret() handler.EventHandler {
	handle := func(ctx context.Context, secret client.Object, q workqueue.RateLimitingInterface) {
		key := client.ObjectKeyFromObject(secret)

		for _, pgadmin := range r.findPGAdminsForSecret(ctx, key) {
			q.Add(ctrl.Request{
				NamespacedName: client.ObjectKeyFromObject(pgadmin),
			})
		}
	}

	return handler.Funcs{
		CreateFunc: func(ctx context.Context, e event.CreateEvent, q workqueue.RateLimitingInterface) {
			handle(ctx, e.Object, q)
		},
		UpdateFunc: func(ctx context.Context, e event.UpdateEvent, q workqueue.RateLimitingInterface) {
			handle(ctx, e.ObjectNew, q)
		},
		// If the secret is deleted, we want to reconcile
		// in order to emit an event/status about this problem.
		// We will also emit a matching event/status about this problem
		// when we reconcile the cluster and can't find the secret.
		// That way, users will get two alerts: one when the secret is deleted
		// and another when the cluster is being reconciled.
		DeleteFunc: func(ctx context.Context, e event.DeleteEvent, q workqueue.RateLimitingInterface) {
			handle(ctx, e.Object, q)
		},
	}
}

//+kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="pgadmins",verbs={list}

// findPGAdminsForSecret returns PGAdmins that have a user or users that have their password
// stored in the Secret
func (r *PGAdminReconciler) findPGAdminsForSecret(
	ctx context.Context, secret client.ObjectKey,
) []*v1beta1.PGAdmin {
	var matching []*v1beta1.PGAdmin
	var pgadmins v1beta1.PGAdminList

	// NOTE: If this becomes slow due to a large number of PGAdmins in a single
	// namespace, we can configure the [ctrl.Manager] field indexer and pass a
	// [fields.Selector] here.
	// - https://book.kubebuilder.io/reference/watching-resources/externally-managed.html
	if err := r.List(ctx, &pgadmins, &client.ListOptions{
		Namespace: secret.Namespace,
	}); err == nil {
		for i := range pgadmins.Items {
			for j := range pgadmins.Items[i].Spec.Users {
				if pgadmins.Items[i].Spec.Users[j].PasswordRef.LocalObjectReference.Name == secret.Name {
					matching = append(matching, &pgadmins.Items[i])
					break
				}
			}
		}
	}
	return matching
}
