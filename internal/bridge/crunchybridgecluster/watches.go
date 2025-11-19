// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package crunchybridgecluster

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

//+kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="crunchybridgeclusters",verbs={list}

// findCrunchyBridgeClustersForSecret returns CrunchyBridgeClusters
// that are connected to the Secret
func (r *CrunchyBridgeClusterReconciler) findCrunchyBridgeClustersForSecret(
	ctx context.Context, secret client.ObjectKey,
) []*v1beta1.CrunchyBridgeCluster {
	var matching []*v1beta1.CrunchyBridgeCluster
	var clusters v1beta1.CrunchyBridgeClusterList

	// NOTE: If this becomes slow due to a large number of CrunchyBridgeClusters in a single
	// namespace, we can configure the [manager.Manager] field indexer and pass a
	// [fields.Selector] here.
	// - https://book.kubebuilder.io/reference/watching-resources/externally-managed.html
	if err := r.Reader.List(ctx, &clusters, &client.ListOptions{
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
