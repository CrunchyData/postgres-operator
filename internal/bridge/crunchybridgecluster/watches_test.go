// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package crunchybridgecluster

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/testing/require"
)

func TestFindCrunchyBridgeClustersForSecret(t *testing.T) {
	ctx := context.Background()
	tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	ns := setupNamespace(t, tClient)
	reconciler := &CrunchyBridgeClusterReconciler{Client: tClient}

	secret := &corev1.Secret{}
	secret.Namespace = ns.Name
	secret.Name = "crunchy-bridge-api-key"

	assert.NilError(t, tClient.Create(ctx, secret))
	secretObjectKey := client.ObjectKeyFromObject(secret)

	t.Run("NoClusters", func(t *testing.T) {
		clusters := reconciler.findCrunchyBridgeClustersForSecret(ctx, secretObjectKey)

		assert.Equal(t, len(clusters), 0)
	})

	t.Run("OneCluster", func(t *testing.T) {
		cluster1 := testCluster()
		cluster1.Namespace = ns.Name
		cluster1.Name = "first-cluster"
		assert.NilError(t, tClient.Create(ctx, cluster1))

		clusters := reconciler.findCrunchyBridgeClustersForSecret(ctx, secretObjectKey)

		assert.Equal(t, len(clusters), 1)
		assert.Equal(t, clusters[0].Name, "first-cluster")
	})

	t.Run("TwoClusters", func(t *testing.T) {
		cluster2 := testCluster()
		cluster2.Namespace = ns.Name
		cluster2.Name = "second-cluster"
		assert.NilError(t, tClient.Create(ctx, cluster2))
		clusters := reconciler.findCrunchyBridgeClustersForSecret(ctx, secretObjectKey)

		assert.Equal(t, len(clusters), 2)
		clusterCount := map[string]int{}
		for _, cluster := range clusters {
			clusterCount[cluster.Name] += 1
		}
		assert.Equal(t, clusterCount["first-cluster"], 1)
		assert.Equal(t, clusterCount["second-cluster"], 1)
	})

	t.Run("ClusterWithDifferentSecretNameNotIncluded", func(t *testing.T) {
		cluster3 := testCluster()
		cluster3.Namespace = ns.Name
		cluster3.Name = "third-cluster"
		cluster3.Spec.Secret = "different-secret-name"
		assert.NilError(t, tClient.Create(ctx, cluster3))
		clusters := reconciler.findCrunchyBridgeClustersForSecret(ctx, secretObjectKey)

		assert.Equal(t, len(clusters), 2)
		clusterCount := map[string]int{}
		for _, cluster := range clusters {
			clusterCount[cluster.Name] += 1
		}
		assert.Equal(t, clusterCount["first-cluster"], 1)
		assert.Equal(t, clusterCount["second-cluster"], 1)
		assert.Equal(t, clusterCount["third-cluster"], 0)
	})
}
