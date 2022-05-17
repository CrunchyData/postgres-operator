/*
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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

package postgrescluster

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
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

// marshalMatches converts actual to YAML and compares that to expected.
func marshalMatches(actual interface{}, expected string) cmp.Comparison {
	return cmp.MarshalMatches(actual, expected)
}

var kubernetes struct {
	sync.Mutex

	env   *envtest.Environment
	count int
}

// setupKubernetes starts or connects to a Kubernetes API and returns a client
// that uses it. When starting a local API, the client is a member of the
// "system:masters" group. It also creates any CRDs present in the
// "/config/crd/bases" directory. When any of these fail, it calls t.Fatal.
// It deletes CRDs and stops the local API using t.Cleanup.
func setupKubernetes(t testing.TB) (*envtest.Environment, client.Client) {
	t.Helper()

	kubernetes.Lock()
	defer kubernetes.Unlock()

	if kubernetes.env == nil {
		env := &envtest.Environment{
			CRDDirectoryPaths: []string{
				filepath.Join("..", "..", "..", "config", "crd", "bases"),
			},
		}

		_, err := env.Start()
		assert.NilError(t, err)

		kubernetes.env = env
	}

	kubernetes.count++

	t.Cleanup(func() {
		kubernetes.Lock()
		defer kubernetes.Unlock()

		if t.Failed() {
			if cc, err := client.New(kubernetes.env.Config, client.Options{}); err == nil {
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
		}

		kubernetes.count--

		if kubernetes.count == 0 {
			assert.Check(t, kubernetes.env.Stop())
			kubernetes.env = nil
		}
	})

	scheme, err := runtime.CreatePostgresOperatorScheme()
	assert.NilError(t, err)

	client, err := client.New(kubernetes.env.Config, client.Options{Scheme: scheme})
	assert.NilError(t, err)

	return kubernetes.env, client
}

// setupNamespace creates a random namespace that will be deleted by t.Cleanup.
// When creation fails, it calls t.Fatal. The caller may delete the namespace
// at any time.
func setupNamespace(t testing.TB, cc client.Client) *corev1.Namespace {
	t.Helper()
	ns := &corev1.Namespace{}
	ns.GenerateName = "postgres-operator-test-"
	ns.Labels = map[string]string{"postgres-operator-test": t.Name()}

	ctx := context.Background()
	assert.NilError(t, cc.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, client.IgnoreNotFound(cc.Delete(ctx, ns))) })

	return ns
}

func testVolumeClaimSpec() corev1.PersistentVolumeClaimSpec {
	// Defines a volume claim spec that can be used to create instances
	return corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		Resources: corev1.ResourceRequirements{
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
	contollerSetup func(mgr manager.Manager)) (context.Context, context.CancelFunc) {

	mgr, err := runtime.CreateRuntimeManager("", cfg, true)
	if err != nil {
		t.Fatal(err)
	}

	contollerSetup(mgr)

	ctx, cancel := context.WithCancel(context.Background())
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
