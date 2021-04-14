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
	"reflect"
	"strconv"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// TestPGBackRestConfiguration goes through the various steps of the current
// pgBackRest configuration setup and verifies the expected values are set in
// the expected configmap and volumes
func TestPGBackRestConfiguration(t *testing.T) {

	// set cluster name and namespace values in postgrescluster spec
	postgresCluster := &v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testclustername,
			Namespace: "postgres-operator-test-" + rand.String(6),
		},
		Spec: v1beta1.PostgresClusterSpec{
			PostgresVersion: 12,
			Archive: v1beta1.Archive{
				PGBackRest: v1beta1.PGBackRestArchive{
					Repos: []v1beta1.RepoVolume{{
						Name: "repo1",
					}},
					RepoHost: &v1beta1.RepoHost{
						Dedicated: &v1beta1.DedicatedRepo{},
					},
				},
			},
		},
	}

	// the initially created configmap
	var cmInitial *v1.ConfigMap
	// the returned configmap
	var cmReturned v1.ConfigMap
	// pod spec for testing projected volumes and volume mounts
	pod := &v1.PodSpec{}

	testInstanceName := "test-instance-abc"
	testRepoName := "repo-host"

	t.Run("pgbackrest configmap checks", func(t *testing.T) {

		// setup the test environment and ensure a clean teardown
		testEnv, testClient := setupTestEnv(t)

		// define the cleanup steps to run once the tests complete
		t.Cleanup(func() {
			teardownTestEnv(t, testEnv)
		})

		t.Run("create pgbackrest configmap struct", func(t *testing.T) {
			// create an array of one host string vlaue
			pghosts := []string{testInstanceName}
			// create the configmap struct
			cmInitial = CreatePGBackRestConfigMapIntent(postgresCluster, testRepoName, pghosts)

			// check that there is configmap data
			assert.Assert(t, cmInitial.Data != nil)
		})

		t.Run("create pgbackrest configmap", func(t *testing.T) {

			ns := &v1.Namespace{}
			ns.Name = naming.PGBackRestConfig(postgresCluster).Namespace
			assert.NilError(t, testClient.Create(context.Background(), ns))
			t.Cleanup(func() { assert.Check(t, testClient.Delete(context.Background(), ns)) })

			// create the configmap
			err := testClient.Patch(context.Background(), cmInitial, client.Apply, client.ForceOwnership, client.FieldOwner(testFieldOwner))

			assert.NilError(t, err)
		})

		t.Run("get pgbackrest configmap", func(t *testing.T) {

			objectKey := client.ObjectKey{
				Namespace: naming.PGBackRestConfig(postgresCluster).Namespace,
				Name:      naming.PGBackRestConfig(postgresCluster).Name,
			}

			err := testClient.Get(context.Background(), objectKey, &cmReturned)

			assert.NilError(t, err)
		})

		// finally, verify initial and returned match
		assert.Assert(t, reflect.DeepEqual(cmInitial.Data, cmReturned.Data))

	})

	t.Run("check pgbackrest configmap repo configuration", func(t *testing.T) {

		assert.Equal(t, getCMData(cmReturned, CMRepoKey),
			`[global]
log-path=/tmp
repo1-path=/pgbackrest/repo1

[db]
pg1-host=`+testInstanceName+`-0.testcluster-pods
pg1-path=/pgdata/pg`+strconv.Itoa(postgresCluster.Spec.PostgresVersion)+`
pg1-port=5432
pg1-socket-path=/tmp
`)
	})

	t.Run("check pgbackrest configmap primary configuration", func(t *testing.T) {

		assert.Equal(t, getCMData(cmReturned, testInstanceName+".conf"),
			`[global]
log-path=/tmp
repo1-host=`+testRepoName+`-0.testcluster-pods
repo1-host-user=postgres
repo1-path=/pgbackrest/repo1

[db]
pg1-path=/pgdata/pg`+strconv.Itoa(postgresCluster.Spec.PostgresVersion)+`
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
		- mountPath: /etc/pgbackrest/conf.d
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
		- mountPath: /etc/pgbackrest/conf.d
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
		- mountPath: /etc/pgbackrest/conf.d
  name: pgbackrest-config
  readOnly: true
		`)+"\n"))
	})
}
