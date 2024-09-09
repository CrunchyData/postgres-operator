// Copyright 2023 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

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

	t.Run("Default", func(t *testing.T) {
		pgadmin := new(v1beta1.PGAdmin)
		result, err := generateConfig(pgadmin)

		assert.NilError(t, err)
		assert.Equal(t, result, `{
  "DEFAULT_SERVER": "0.0.0.0",
  "SERVER_MODE": true,
  "UPGRADE_CHECK_ENABLED": false,
  "UPGRADE_CHECK_KEY": "",
  "UPGRADE_CHECK_URL": ""
}`+"\n")
	})

	t.Run("Mandatory", func(t *testing.T) {
		pgadmin := new(v1beta1.PGAdmin)
		pgadmin.Spec.Config.Settings = map[string]any{
			"SERVER_MODE":           false,
			"UPGRADE_CHECK_ENABLED": true,
		}
		result, err := generateConfig(pgadmin)

		assert.NilError(t, err)
		assert.Equal(t, result, `{
  "DEFAULT_SERVER": "0.0.0.0",
  "SERVER_MODE": true,
  "UPGRADE_CHECK_ENABLED": false,
  "UPGRADE_CHECK_KEY": "",
  "UPGRADE_CHECK_URL": ""
}`+"\n")
	})

	t.Run("Specified", func(t *testing.T) {
		pgadmin := new(v1beta1.PGAdmin)
		pgadmin.Spec.Config.Settings = map[string]any{
			"ALLOWED_HOSTS":  []any{"225.0.0.0/8", "226.0.0.0/7", "228.0.0.0/6"},
			"DEFAULT_SERVER": "::",
		}
		result, err := generateConfig(pgadmin)

		assert.NilError(t, err)
		assert.Equal(t, result, `{
  "ALLOWED_HOSTS": [
    "225.0.0.0/8",
    "226.0.0.0/7",
    "228.0.0.0/6"
  ],
  "DEFAULT_SERVER": "::",
  "SERVER_MODE": true,
  "UPGRADE_CHECK_ENABLED": false,
  "UPGRADE_CHECK_KEY": "",
  "UPGRADE_CHECK_URL": ""
}`+"\n")
	})
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
  postgres-operator.crunchydata.com/pgadmin: pg1
  postgres-operator.crunchydata.com/role: pgadmin
name: pgadmin-
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
			"postgres-operator.crunchydata.com/pgadmin": "pg1",
			"postgres-operator.crunchydata.com/role":    "pgadmin",
		})
	})
}

func TestGenerateGunicornConfig(t *testing.T) {
	require.ParallelCapacity(t, 0)

	t.Run("Default", func(t *testing.T) {
		pgAdmin := &v1beta1.PGAdmin{}
		pgAdmin.Name = "test"
		pgAdmin.Namespace = "postgres-operator"

		expectedString := `{
  "bind": "0.0.0.0:5050",
  "threads": 25,
  "workers": 1
}
`
		actualString, err := generateGunicornConfig(pgAdmin)
		assert.NilError(t, err)
		assert.Equal(t, actualString, expectedString)
	})

	t.Run("Add Settings", func(t *testing.T) {
		pgAdmin := &v1beta1.PGAdmin{}
		pgAdmin.Name = "test"
		pgAdmin.Namespace = "postgres-operator"
		pgAdmin.Spec.Config.Gunicorn = map[string]any{
			"keyfile":  "/path/to/keyfile",
			"certfile": "/path/to/certfile",
		}

		expectedString := `{
  "bind": "0.0.0.0:5050",
  "certfile": "/path/to/certfile",
  "keyfile": "/path/to/keyfile",
  "threads": 25,
  "workers": 1
}
`
		actualString, err := generateGunicornConfig(pgAdmin)
		assert.NilError(t, err)
		assert.Equal(t, actualString, expectedString)
	})

	t.Run("Update Defaults", func(t *testing.T) {
		pgAdmin := &v1beta1.PGAdmin{}
		pgAdmin.Name = "test"
		pgAdmin.Namespace = "postgres-operator"
		pgAdmin.Spec.Config.Gunicorn = map[string]any{
			"bind":    "127.0.0.1:5051",
			"threads": 30,
		}

		expectedString := `{
  "bind": "127.0.0.1:5051",
  "threads": 30,
  "workers": 1
}
`
		actualString, err := generateGunicornConfig(pgAdmin)
		assert.NilError(t, err)
		assert.Equal(t, actualString, expectedString)
	})

	t.Run("Update Mandatory", func(t *testing.T) {
		pgAdmin := &v1beta1.PGAdmin{}
		pgAdmin.Name = "test"
		pgAdmin.Namespace = "postgres-operator"
		pgAdmin.Spec.Config.Gunicorn = map[string]any{
			"workers": "100",
		}

		expectedString := `{
  "bind": "0.0.0.0:5050",
  "threads": 25,
  "workers": 1
}
`
		actualString, err := generateGunicornConfig(pgAdmin)
		assert.NilError(t, err)
		assert.Equal(t, actualString, expectedString)
	})

}
