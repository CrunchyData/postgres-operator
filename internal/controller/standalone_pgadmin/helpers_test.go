// Copyright 2023 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package standalone_pgadmin

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/internal/testing/require"
)

// Scale extends d according to PGO_TEST_TIMEOUT_SCALE.
//
// TODO(tjmoore4): This function is duplicated from a version that takes a PostgresCluster object.
var Scale = func(d time.Duration) time.Duration { return d }

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
