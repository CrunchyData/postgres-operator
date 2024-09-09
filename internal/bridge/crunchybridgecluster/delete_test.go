// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package crunchybridgecluster

import (
	"context"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/crunchydata/postgres-operator/internal/bridge"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
)

func TestHandleDeleteCluster(t *testing.T) {
	ctx := context.Background()
	tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	ns := setupNamespace(t, tClient).Name

	firstClusterInBridge := testClusterApiResource()
	firstClusterInBridge.ClusterName = "bridge-cluster-1"
	secondClusterInBridge := testClusterApiResource()
	secondClusterInBridge.ClusterName = "bridge-cluster-2"
	secondClusterInBridge.ID = "2345"

	reconciler := &CrunchyBridgeClusterReconciler{
		Client: tClient,
		Owner:  "crunchybridgecluster-controller",
	}
	testBridgeClient := &TestBridgeClient{
		ApiKey:   "9012",
		TeamId:   "5678",
		Clusters: []*bridge.ClusterApiResource{firstClusterInBridge, secondClusterInBridge},
	}
	reconciler.NewClient = func() bridge.ClientInterface {
		return testBridgeClient
	}

	t.Run("SuccessfulDeletion", func(t *testing.T) {
		// Create test cluster in kubernetes
		cluster := testCluster()
		cluster.Namespace = ns
		cluster.Status.ID = "1234"
		cluster.Spec.ClusterName = "bridge-cluster-1"
		assert.NilError(t, tClient.Create(ctx, cluster))

		// Run handleDelete
		controllerResult, err := reconciler.handleDelete(ctx, cluster, "9012")
		assert.NilError(t, err)
		assert.Check(t, controllerResult == nil)

		// Make sure that finalizer was added
		assert.Check(t, controllerutil.ContainsFinalizer(cluster, finalizer))

		// Send delete request to kubernetes
		assert.NilError(t, tClient.Delete(ctx, cluster))

		// Get cluster from kubernetes and assert that the deletion timestamp was added
		assert.NilError(t, tClient.Get(ctx, client.ObjectKeyFromObject(cluster), cluster))
		assert.Check(t, !cluster.ObjectMeta.DeletionTimestamp.IsZero())

		// Note: We must run handleDelete multiple times because we don't want to remove the
		// finalizer until we're sure that the cluster has been deleted from Bridge, so we
		// have to do multiple calls/reconcile loops.
		// Run handleDelete again to delete from Bridge
		cluster.Status.ID = "1234"
		controllerResult, err = reconciler.handleDelete(ctx, cluster, "9012")
		assert.NilError(t, err)
		assert.Equal(t, controllerResult.RequeueAfter, 1*time.Second)
		assert.Equal(t, len(testBridgeClient.Clusters), 1)
		assert.Equal(t, testBridgeClient.Clusters[0].ClusterName, "bridge-cluster-2")

		// Run handleDelete one last time to remove finalizer
		controllerResult, err = reconciler.handleDelete(ctx, cluster, "9012")
		assert.NilError(t, err)
		assert.Equal(t, *controllerResult, ctrl.Result{})

		// Make sure that finalizer was removed
		assert.Check(t, !controllerutil.ContainsFinalizer(cluster, finalizer))
	})

	t.Run("UnsuccessfulDeletion", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns
		cluster.Status.ID = "2345"
		cluster.Spec.ClusterName = "bridge-cluster-2"
		assert.NilError(t, tClient.Create(ctx, cluster))

		// Run handleDelete
		controllerResult, err := reconciler.handleDelete(ctx, cluster, "9012")
		assert.NilError(t, err)
		assert.Check(t, controllerResult == nil)

		// Make sure that finalizer was added
		assert.Check(t, controllerutil.ContainsFinalizer(cluster, finalizer))

		// Send delete request to kubernetes
		assert.NilError(t, tClient.Delete(ctx, cluster))

		// Get cluster from kubernetes and assert that the deletion timestamp was added
		assert.NilError(t, tClient.Get(ctx, client.ObjectKeyFromObject(cluster), cluster))
		assert.Check(t, !cluster.ObjectMeta.DeletionTimestamp.IsZero())

		// Run handleDelete again to attempt to delete from Bridge, but provide bad api key
		cluster.Status.ID = "2345"
		controllerResult, err = reconciler.handleDelete(ctx, cluster, "bad_api_key")
		assert.ErrorContains(t, err, "boom")
		assert.Equal(t, *controllerResult, ctrl.Result{})

		// Run handleDelete a couple times with good api key so test can cleanup properly.
		// Note: We must run handleDelete multiple times because we don't want to remove the
		// finalizer until we're sure that the cluster has been deleted from Bridge, so we
		// have to do multiple calls/reconcile loops.
		// delete from bridge
		_, err = reconciler.handleDelete(ctx, cluster, "9012")
		assert.NilError(t, err)

		// remove finalizer
		_, err = reconciler.handleDelete(ctx, cluster, "9012")
		assert.NilError(t, err)

		// Make sure that finalizer was removed
		assert.Check(t, !controllerutil.ContainsFinalizer(cluster, finalizer))
	})
}
