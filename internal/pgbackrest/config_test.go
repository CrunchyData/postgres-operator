// +build envtest

/*
 Copyright 2021 Crunchy Data Solutions, Inc.
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

package pgbackrest

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
)

// Testing namespace and postgrescluster name
const (
	namespace      = "testnamespace"
	clustername    = "testcluster"
	testFieldOwner = "pgbackrestConfigTestFieldOwner"
)

// getCMData returns the 'Data' content from the specifified configmap
func getCMData(cm v1.ConfigMap, key string) string {

	return cm.Data[key]
}

// simpleMarshalContains takes in a YAML object and checks whether
// it includes the expected string
func simpleMarshalContains(actual interface{}, expected string) bool {
	b, err := yaml.Marshal(actual)

	if err != nil {
		return false
	}

	if string(b) == expected {
		return true
	}
	return false
}

// setupTestEnv configures and starts an EnvTest instance of etcd and the Kubernetes API server
// for test usage, as well as creates a new client instance.
func setupTestEnv(t *testing.T) (*envtest.Environment, *rest.Config, client.Client) {

	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
	}
	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Test environment started")

	pgoScheme, err := runtime.CreatePostgresOperatorScheme()
	if err != nil {
		t.Fatal(err)
	}
	client, err := client.New(cfg, client.Options{Scheme: pgoScheme})
	if err != nil {
		t.Fatal(err)
	}

	return testEnv, cfg, client
}

// setupManager creates new controller runtime manager and returns
// the associated context
func setupManager(t *testing.T, cfg *rest.Config,
	contollerSetup func(mgr manager.Manager)) context.Context {

	mgr, err := runtime.CreateRuntimeManager("", cfg)
	if err != nil {
		t.Fatal(err)
	}

	contollerSetup(mgr)

	ctx := signals.SetupSignalHandler()
	go func() {
		if err := mgr.Start(ctx); err != nil {
			t.Error(err)
		}
	}()
	t.Log("Manager started")

	return ctx
}

// teardownTestEnv stops the test environment when the tests
// have completed
func teardownTestEnv(t *testing.T, testEnv *envtest.Environment) {
	if err := testEnv.Stop(); err != nil {
		t.Error(err)
	}
	t.Log("Test environment stopped")
}

// teardownManager stops the test environment's context
// manager when the tests have completed
func teardownManager(ctx context.Context, t *testing.T) {
	ctx.Done()
	t.Log("Manager stopped")
}

// TestPGBackRestConfiguration goes through the various steps of the current
// pgBackRest configuration setup and verifies the expected values are set in
// the expected configmap and volumes
func TestPGBackRestConfiguration(t *testing.T) {

	// set cluster name and namespace values in postgrescluster spec
	postgresCluster := &v1alpha1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clustername,
			Namespace: namespace,
		},
	}

	// the initially created configmap
	var cmInitial v1.ConfigMap
	// the returned configmap
	var cmReturned v1.ConfigMap
	// pod spec for testing projected volumes and volume mounts
	pod := &v1.PodSpec{}

	t.Run("pgbackrest configmap checks", func(t *testing.T) {

		// setup the test environment and ensure a clean teardown
		testEnv, cfg, testClient := setupTestEnv(t)
		ctx := setupManager(t, cfg, func(mgr manager.Manager) {})

		// define the cleanup steps to run once the tests complete
		t.Cleanup(func() {
			teardownManager(ctx, t)
			teardownTestEnv(t, testEnv)
		})

		t.Run("create pgbackrest configmap struct", func(t *testing.T) {
			// create an array of one host string vlaue
			pghosts := []string{clustername}
			// create the configmap struct
			cmInitial = CreatePGBackRestConfigMapStruct(postgresCluster, pghosts)

			// check that there is configmap data
			assert.Assert(t, cmInitial.Data != nil)
		})

		t.Run("create pgbackrest configmap", func(t *testing.T) {

			// create the test namespace
			err := testClient.Create(ctx, &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			})

			assert.NilError(t, err)

			// create the configmap
			err = testClient.Patch(ctx, &cmInitial, client.Apply, client.ForceOwnership, client.FieldOwner(testFieldOwner))

			assert.NilError(t, err)
		})

		t.Run("get pgbackrest configmap", func(t *testing.T) {

			objectKey := client.ObjectKey{
				Namespace: postgresCluster.GetNamespace(),
				Name:      fmt.Sprintf(cmNameSuffix, postgresCluster.GetName()),
			}

			err := testClient.Get(ctx, objectKey, &cmReturned)

			assert.NilError(t, err)
		})

		// finally, verify initial and returned match
		assert.Assert(t, reflect.DeepEqual(cmInitial.Data, cmReturned.Data))

	})

	t.Run("check pgbackrest configmap default configuration", func(t *testing.T) {

		assert.Equal(t, getCMData(cmReturned, cmJobKey),
			`[global]
log-path=/tmp
`)
	})

	t.Run("check pgbackrest configmap repo configuration", func(t *testing.T) {

		assert.Equal(t, getCMData(cmReturned, cmRepoKey),
			`[global]
log-path=/tmp
repo1-path=/backrestrepo/`+postgresCluster.GetName()+`-backrest-shared-repo

[db]
pg1-host=`+postgresCluster.GetName()+`
pg1-path=/pgdata/`+postgresCluster.GetName()+`
pg1-port=5432
pg1-socket-path=/tmp
`)
	})

	t.Run("check pgbackrest configmap primary configuration", func(t *testing.T) {

		assert.Equal(t, getCMData(cmReturned, cmPrimaryKey),
			`[global]
log-path=/tmp
repo1-host=`+postgresCluster.GetName()+`-backrest-shared-repo
repo1-path=/backrestrepo/`+postgresCluster.GetName()+`-backrest-shared-repo

[db]
pg1-path=/pgdata/`+postgresCluster.GetName()+`
pg1-port=5432
pg1-socket-path=/tmp
`)
	})

	t.Run("check primary config volume", func(t *testing.T) {

		PostgreSQLConfigVolumeAndMount(&cmReturned, pod, "database")

		assert.Assert(t, simpleMarshalContains(&pod.Volumes, strings.TrimSpace(`
		- name: pgbackrest-config
  projected:
    sources:
    - configMap:
        items:
        - key: pgbackrest_primary.conf
          path: /etc/pgbackrest/pgbackrest.conf
        name: `+postgresCluster.GetName()+`-pgbackrest-config
		`)+"\n"))
	})

	t.Run("check primary config volume mount", func(t *testing.T) {

		PostgreSQLConfigVolumeAndMount(&cmReturned, pod, "database")

		container := findOrAppendContainer(&pod.Containers, "database")

		assert.Assert(t, simpleMarshalContains(container.VolumeMounts, strings.TrimSpace(`
		- mountPath: /etc/pgbackrest
  name: pgbackrest-config
  readOnly: true
		`)+"\n"))
	})

	t.Run("check default config volume", func(t *testing.T) {

		JobConfigVolumeAndMount(&cmReturned, pod, "pgbackrest")

		assert.Assert(t, simpleMarshalContains(pod.Volumes, strings.TrimSpace(`
		- name: pgbackrest-config
  projected:
    sources:
    - configMap:
        items:
        - key: pgbackrest_job.conf
          path: /etc/pgbackrest/pgbackrest.conf
        name: `+postgresCluster.GetName()+`-pgbackrest-config
		`)+"\n"))
	})

	t.Run("check default config volume mount", func(t *testing.T) {

		JobConfigVolumeAndMount(&cmReturned, pod, "pgbackrest")

		container := findOrAppendContainer(&pod.Containers, "pgbackrest")

		assert.Assert(t, simpleMarshalContains(container.VolumeMounts, strings.TrimSpace(`
		- mountPath: /etc/pgbackrest
  name: pgbackrest-config
  readOnly: true
		`)+"\n"))
	})

	t.Run("check repo config volume", func(t *testing.T) {

		RepositoryConfigVolumeAndMount(&cmReturned, pod, "pgbackrest")

		assert.Assert(t, simpleMarshalContains(&pod.Volumes, strings.TrimSpace(`
		- name: pgbackrest-config
  projected:
    sources:
    - configMap:
        items:
        - key: pgbackrest_repo.conf
          path: /etc/pgbackrest/pgbackrest.conf
        name: `+postgresCluster.GetName()+`-pgbackrest-config
		`)+"\n"))
	})

	t.Run("check repo config volume mount", func(t *testing.T) {

		RepositoryConfigVolumeAndMount(&cmReturned, pod, "pgbackrest")

		container := findOrAppendContainer(&pod.Containers, "pgbackrest")

		assert.Assert(t, simpleMarshalContains(container.VolumeMounts, strings.TrimSpace(`
		- mountPath: /etc/pgbackrest
  name: pgbackrest-config
  readOnly: true
		`)+"\n"))
	})
}
