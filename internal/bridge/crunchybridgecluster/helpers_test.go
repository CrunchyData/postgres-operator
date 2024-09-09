// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package crunchybridgecluster

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/internal/bridge"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// Scale extends d according to PGO_TEST_TIMEOUT_SCALE.
var Scale = func(d time.Duration) time.Duration { return d }

// This function was duplicated from the postgrescluster package.
// TODO: Pull these duplicated functions out into a separate, shared package.
func init() {
	setting := os.Getenv("PGO_TEST_TIMEOUT_SCALE")
	factor, _ := strconv.ParseFloat(setting, 64)

	if setting != "" {
		if factor <= 0 {
			panic("PGO_TEST_TIMEOUT_SCALE must be a fractional number greater than zero")
		}

		Scale = func(d time.Duration) time.Duration {
			return time.Duration(factor * float64(d))
		}
	}
}

// setupKubernetes starts or connects to a Kubernetes API and returns a client
// that uses it. See [require.Kubernetes] for more details.
func setupKubernetes(t testing.TB) client.Client {
	t.Helper()

	// Start and/or connect to a Kubernetes API, or Skip when that's not configured.
	cc := require.Kubernetes(t)

	// Log the status of any test namespaces after this test fails.
	t.Cleanup(func() {
		if t.Failed() {
			var namespaces corev1.NamespaceList
			_ = cc.List(context.Background(), &namespaces, client.HasLabels{"postgres-operator-test"})

			type shaped map[string]corev1.NamespaceStatus
			result := make([]shaped, len(namespaces.Items))

			for i, ns := range namespaces.Items {
				result[i] = shaped{ns.Labels["postgres-operator-test"]: ns.Status}
			}

			formatted, _ := yaml.Marshal(result)
			t.Logf("Test Namespaces:\n%s", formatted)
		}
	})

	return cc
}

// setupNamespace creates a random namespace that will be deleted by t.Cleanup.
//
// Deprecated: Use [require.Namespace] instead.
func setupNamespace(t testing.TB, cc client.Client) *corev1.Namespace {
	t.Helper()
	return require.Namespace(t, cc)
}

// testCluster defines a base cluster spec that can be used by tests to
// generate a CrunchyBridgeCluster CR
func testCluster() *v1beta1.CrunchyBridgeCluster {
	cluster := v1beta1.CrunchyBridgeCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hippo-cr",
		},
		Spec: v1beta1.CrunchyBridgeClusterSpec{
			ClusterName:     "hippo-cluster",
			IsHA:            false,
			PostgresVersion: 15,
			Plan:            "standard-8",
			Provider:        "aws",
			Region:          "us-east-2",
			Secret:          "crunchy-bridge-api-key",
			Storage:         resource.MustParse("10Gi"),
		},
	}
	return cluster.DeepCopy()
}

func testClusterApiResource() *bridge.ClusterApiResource {
	cluster := bridge.ClusterApiResource{
		ID:           "1234",
		Host:         "example.com",
		IsHA:         initialize.Bool(false),
		IsProtected:  initialize.Bool(false),
		MajorVersion: 15,
		ClusterName:  "hippo-cluster",
		Plan:         "standard-8",
		Provider:     "aws",
		Region:       "us-east-2",
		Storage:      10,
		Team:         "5678",
	}
	return &cluster
}

func testClusterStatusApiResource(clusterId string) *bridge.ClusterStatusApiResource {
	teamId := "5678"
	state := "ready"

	clusterStatus := bridge.ClusterStatusApiResource{
		DiskUsage: &bridge.ClusterDiskUsageApiResource{
			DiskAvailableMB: 16,
			DiskTotalSizeMB: 16,
			DiskUsedMB:      0,
		},
		OldestBackup: "oldbackup",
		OngoingUpgrade: &bridge.ClusterUpgradeApiResource{
			ClusterID:  clusterId,
			Operations: []*v1beta1.UpgradeOperation{},
			Team:       teamId,
		},
		State: state,
	}

	return &clusterStatus
}

func testClusterUpgradeApiResource(clusterId string) *bridge.ClusterUpgradeApiResource {
	teamId := "5678"

	clusterUpgrade := bridge.ClusterUpgradeApiResource{
		ClusterID: clusterId,
		Operations: []*v1beta1.UpgradeOperation{
			{
				Flavor:       "resize",
				StartingFrom: "",
				State:        "in_progress",
			},
		},
		Team: teamId,
	}

	return &clusterUpgrade
}

func testClusterRoleApiResource() *bridge.ClusterRoleApiResource {
	clusterId := "1234"
	teamId := "5678"
	roleName := "application"

	clusterRole := bridge.ClusterRoleApiResource{
		AccountEmail: "test@email.com",
		AccountId:    "12345678",
		ClusterId:    clusterId,
		Flavor:       "chocolate",
		Name:         roleName,
		Password:     "application-password",
		Team:         teamId,
		URI:          "connection-string",
	}

	return &clusterRole
}
