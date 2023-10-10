// Copyright 2023 Crunchy Data Solutions, Inc.
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
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestGenerateConfig(t *testing.T) {
	require.ParallelCapacity(t, 0)

	expectedString := `{
  "ALLOWED_HOSTS": [
    "225.0.0.0/8",
    "226.0.0.0/7",
    "228.0.0.0/6"
  ],
  "SERVER_MODE": true,
  "UPGRADE_CHECK_ENABLED": false,
  "UPGRADE_CHECK_KEY": "",
  "UPGRADE_CHECK_URL": ""
}
`
	pgadmin := new(v1beta1.PGAdmin)
	pgadmin.Spec.Config.Settings = map[string]interface{}{
		"ALLOWED_HOSTS": []interface{}{"225.0.0.0/8", "226.0.0.0/7", "228.0.0.0/6"},
	}
	actualString, err := generateConfig(pgadmin)
	assert.NilError(t, err)
	assert.Equal(t, actualString, expectedString)
}

func TestGenerateClusterConfig(t *testing.T) {
	require.ParallelCapacity(t, 0)

	cluster := testCluster()
	cluster.Namespace = "postgres-operator"
	clusterList := &v1beta1.PostgresClusterList{
		Items: []v1beta1.PostgresCluster{*cluster, *cluster},
	}
	clusters := map[string]*v1beta1.PostgresClusterList{
		"shared": clusterList,
		"test":   clusterList,
		"hello":  clusterList,
	}

	expectedString := `{
  "Servers": {
    "1": {
      "Group": "hello",
      "Host": "hippo-primary.postgres-operator.svc",
      "MaintenanceDB": "postgres",
      "Name": "hippo",
      "Port": 5432,
      "SSLMode": "prefer",
      "Shared": true,
      "Username": "hippo"
    },
    "2": {
      "Group": "hello",
      "Host": "hippo-primary.postgres-operator.svc",
      "MaintenanceDB": "postgres",
      "Name": "hippo",
      "Port": 5432,
      "SSLMode": "prefer",
      "Shared": true,
      "Username": "hippo"
    },
    "3": {
      "Group": "shared",
      "Host": "hippo-primary.postgres-operator.svc",
      "MaintenanceDB": "postgres",
      "Name": "hippo",
      "Port": 5432,
      "SSLMode": "prefer",
      "Shared": true,
      "Username": "hippo"
    },
    "4": {
      "Group": "shared",
      "Host": "hippo-primary.postgres-operator.svc",
      "MaintenanceDB": "postgres",
      "Name": "hippo",
      "Port": 5432,
      "SSLMode": "prefer",
      "Shared": true,
      "Username": "hippo"
    },
    "5": {
      "Group": "test",
      "Host": "hippo-primary.postgres-operator.svc",
      "MaintenanceDB": "postgres",
      "Name": "hippo",
      "Port": 5432,
      "SSLMode": "prefer",
      "Shared": true,
      "Username": "hippo"
    },
    "6": {
      "Group": "test",
      "Host": "hippo-primary.postgres-operator.svc",
      "MaintenanceDB": "postgres",
      "Name": "hippo",
      "Port": 5432,
      "SSLMode": "prefer",
      "Shared": true,
      "Username": "hippo"
    }
  }
}
`
	actualString, err := generateClusterConfig(clusters)
	assert.NilError(t, err)
	assert.Equal(t, actualString, expectedString)
}

func TestGeneratePGAdminConfigMap(t *testing.T) {
	require.ParallelCapacity(t, 0)

	pgadmin := new(v1beta1.PGAdmin)
	pgadmin.Namespace = "some-ns"
	pgadmin.Name = "pg1"
	clusters := map[string]*v1beta1.PostgresClusterList{}
	t.Run("Data,ObjectMeta,TypeMeta", func(t *testing.T) {
		pgadmin := pgadmin.DeepCopy()

		configmap, err := configmap(pgadmin, clusters)

		assert.NilError(t, err)
		assert.Assert(t, cmp.MarshalMatches(configmap.TypeMeta, `
apiVersion: v1
kind: ConfigMap
		`))
		assert.Assert(t, cmp.MarshalMatches(configmap.ObjectMeta, `
creationTimestamp: null
labels:
  postgres-operator.crunchydata.com/role: pgadmin
  postgres-operator.crunchydata.com/standalone-pgadmin: pg1
name: pg1-standalone-pgadmin
namespace: some-ns
		`))

		assert.Assert(t, len(configmap.Data) > 0, "expected some configuration")
	})

	t.Run("Annotations,Labels", func(t *testing.T) {
		pgadmin := pgadmin.DeepCopy()
		pgadmin.Spec.Metadata = &v1beta1.Metadata{
			Annotations: map[string]string{"a": "v1", "b": "v2"},
			Labels:      map[string]string{"c": "v3", "d": "v4"},
		}

		configmap, err := configmap(pgadmin, clusters)

		assert.NilError(t, err)
		// Annotations present in the metadata.
		assert.DeepEqual(t, configmap.ObjectMeta.Annotations, map[string]string{
			"a": "v1", "b": "v2",
		})

		// Labels present in the metadata.
		assert.DeepEqual(t, configmap.ObjectMeta.Labels, map[string]string{
			"c": "v3", "d": "v4",
			"postgres-operator.crunchydata.com/standalone-pgadmin": "pg1",
			"postgres-operator.crunchydata.com/role":               "pgadmin",
		})
	})
}
