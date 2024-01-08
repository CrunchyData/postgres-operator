//go:build envtest
// +build envtest

// Copyright 2023 - 2024 Crunchy Data Solutions, Inc.
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

package standalone_pgadmin

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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
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
//
// TODO(tjmoore4): This function is duplicated from a version that takes a PostgresCluster object.
func setupKubernetes(t testing.TB) client.Client {
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

	return client
}

// setupNamespace creates a random namespace that will be deleted by t.Cleanup.
// When creation fails, it calls t.Fatal. The caller may delete the namespace
// at any time.
//
// TODO(tjmoore4): This function is duplicated from a version that takes a PostgresCluster object.
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
