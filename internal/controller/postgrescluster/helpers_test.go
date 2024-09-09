// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

var (
	//TODO(tjmoore4): With the new RELATED_IMAGES defaulting behavior, tests could be refactored
	// to reference those environment variables instead of hard coded image values
	CrunchyPostgresHAImage = "registry.developers.crunchydata.com/crunchydata/crunchy-postgres:ubi8-13.6-1"
	CrunchyPGBackRestImage = "registry.developers.crunchydata.com/crunchydata/crunchy-pgbackrest:ubi8-2.38-0"
	CrunchyPGBouncerImage  = "registry.developers.crunchydata.com/crunchydata/crunchy-pgbouncer:ubi8-1.16-2"
)

// Scale extends d according to PGO_TEST_TIMEOUT_SCALE.
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
func setupKubernetes(t testing.TB) (*rest.Config, client.Client) {
	t.Helper()

	// Start and/or connect to a Kubernetes API, or Skip when that's not configured.
	cfg, cc := require.Kubernetes2(t)

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

	return cfg, cc
}

// setupNamespace creates a random namespace that will be deleted by t.Cleanup.
//
// Deprecated: Use [require.Namespace] instead.
func setupNamespace(t testing.TB, cc client.Client) *corev1.Namespace {
	t.Helper()
	return require.Namespace(t, cc)
}

func testVolumeClaimSpec() corev1.PersistentVolumeClaimSpec {
	// Defines a volume claim spec that can be used to create instances
	return corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		Resources: corev1.VolumeResourceRequirements{
			Requests: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceStorage: resource.MustParse("1Gi"),
			},
		},
	}
}
func testCluster() *v1beta1.PostgresCluster {
	// Defines a base cluster spec that can be used by tests to generate a
	// cluster with an expected number of instances
	cluster := v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hippo",
		},
		Spec: v1beta1.PostgresClusterSpec{
			PostgresVersion: 13,
			Image:           CrunchyPostgresHAImage,
			ImagePullSecrets: []corev1.LocalObjectReference{{
				Name: "myImagePullSecret"},
			},
			InstanceSets: []v1beta1.PostgresInstanceSetSpec{{
				Name:                "instance1",
				Replicas:            initialize.Int32(1),
				DataVolumeClaimSpec: testVolumeClaimSpec(),
			}},
			Backups: v1beta1.Backups{
				PGBackRest: v1beta1.PGBackRestArchive{
					Image: CrunchyPGBackRestImage,
					Repos: []v1beta1.PGBackRestRepo{{
						Name: "repo1",
						Volume: &v1beta1.RepoPVC{
							VolumeClaimSpec: testVolumeClaimSpec(),
						},
					}},
				},
			},
			Proxy: &v1beta1.PostgresProxySpec{
				PGBouncer: &v1beta1.PGBouncerPodSpec{
					Image: CrunchyPGBouncerImage,
				},
			},
		},
	}
	return cluster.DeepCopy()
}

// setupManager creates the runtime manager used during controller testing
func setupManager(t *testing.T, cfg *rest.Config,
	controllerSetup func(mgr manager.Manager)) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	// Disable health endpoints
	options := runtime.Options{}
	options.HealthProbeBindAddress = "0"
	options.Metrics.BindAddress = "0"

	mgr, err := runtime.NewManager(cfg, options)
	if err != nil {
		t.Fatal(err)
	}

	controllerSetup(mgr)

	go func() {
		if err := mgr.Start(ctx); err != nil {
			t.Error(err)
		}
	}()
	t.Log("Manager started")

	return ctx, cancel
}

// teardownManager stops the runtime manager when the tests
// have completed
func teardownManager(cancel context.CancelFunc, t *testing.T) {
	cancel()
	t.Log("Manager stopped")
}
