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

	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

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
	if err := r.List(ctx, &clusters, &client.ListOptions{
		Namespace: secret.Namespace,
	}); err == nil {
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
		log := ctrl.LoggerFrom(ctx)

		crunchyBridgeClusterList := &v1beta1.CrunchyBridgeClusterList{}
		if err := r.List(ctx, crunchyBridgeClusterList); err != nil {
			log.Error(err, "Error listing CrunchyBridgeClusters.")
		}

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
