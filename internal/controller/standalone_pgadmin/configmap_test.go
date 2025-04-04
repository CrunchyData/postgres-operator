// Copyright 2023 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package standalone_pgadmin

import (
	"context"
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
		result, err := generateConfig(pgadmin, false, 0, 0)

		assert.NilError(t, err)
		assert.Equal(t, result, `{
  "DATA_DIR": "/var/lib/pgadmin",
  "DEFAULT_SERVER": "0.0.0.0",
  "LOG_FILE": "/var/lib/pgadmin/logs/pgadmin.log",
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
		result, err := generateConfig(pgadmin, false, 0, 0)

		assert.NilError(t, err)
		assert.Equal(t, result, `{
  "DATA_DIR": "/var/lib/pgadmin",
  "DEFAULT_SERVER": "0.0.0.0",
  "LOG_FILE": "/var/lib/pgadmin/logs/pgadmin.log",
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
		result, err := generateConfig(pgadmin, false, 0, 0)

		assert.NilError(t, err)
		assert.Equal(t, result, `{
  "ALLOWED_HOSTS": [
    "225.0.0.0/8",
    "226.0.0.0/7",
    "228.0.0.0/6"
  ],
  "DATA_DIR": "/var/lib/pgadmin",
  "DEFAULT_SERVER": "::",
  "LOG_FILE": "/var/lib/pgadmin/logs/pgadmin.log",
  "SERVER_MODE": true,
  "UPGRADE_CHECK_ENABLED": false,
  "UPGRADE_CHECK_KEY": "",
  "UPGRADE_CHECK_URL": ""
}`+"\n")
	})

	t.Run("OTel enabled", func(t *testing.T) {
		pgadmin := new(v1beta1.PGAdmin)
		require.UnmarshalInto(t, &pgadmin.Spec, `{
			instrumentation: {
				logs: { retentionPeriod: 5h },
			},
		}`)
		result, err := generateConfig(pgadmin, true, 4, 60)

		assert.NilError(t, err)
		assert.Equal(t, result, `{
  "CONSOLE_LOG_LEVEL": "WARNING",
  "DATA_DIR": "/var/lib/pgadmin",
  "DEFAULT_SERVER": "0.0.0.0",
  "FILE_LOG_FORMAT_JSON": {
    "level": "levelname",
    "message": "message",
    "name": "name",
    "time": "created"
  },
  "FILE_LOG_LEVEL": "INFO",
  "JSON_LOGGER": true,
  "LOG_FILE": "/var/lib/pgadmin/logs/pgadmin.log",
  "LOG_ROTATION_AGE": 60,
  "LOG_ROTATION_MAX_LOG_FILES": 4,
  "SERVER_MODE": true,
  "UPGRADE_CHECK_ENABLED": false,
  "UPGRADE_CHECK_KEY": "",
  "UPGRADE_CHECK_URL": ""
}`+"\n")
	})
}

func TestGenerateClusterConfig(t *testing.T) {
	require.ParallelCapacity(t, 0)

	cluster := v1beta1.NewPostgresCluster()
	cluster.Namespace = "postgres-operator"
	cluster.Name = "hippo"
	clusters := map[string][]*v1beta1.PostgresCluster{
		"shared": {cluster, cluster},
		"test":   {cluster, cluster},
		"hello":  {cluster, cluster},
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
	clusters := map[string][]*v1beta1.PostgresCluster{}
	ctx := context.Background()
	t.Run("Data,ObjectMeta,TypeMeta", func(t *testing.T) {
		pgadmin := pgadmin.DeepCopy()

		configmap, err := configmap(ctx, pgadmin, clusters)

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

		configmap, err := configmap(ctx, pgadmin, clusters)

		assert.NilError(t, err)
		// Annotations present in the metadata.
		assert.DeepEqual(t, configmap.Annotations, map[string]string{
			"a": "v1", "b": "v2",
		})

		// Labels present in the metadata.
		assert.DeepEqual(t, configmap.Labels, map[string]string{
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
  "logconfig_dict": {},
  "threads": 25,
  "workers": 1
}
`
		actualString, err := generateGunicornConfig(pgAdmin, false, 0, "H")
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
  "logconfig_dict": {},
  "threads": 25,
  "workers": 1
}
`
		actualString, err := generateGunicornConfig(pgAdmin, false, 0, "H")
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
  "logconfig_dict": {},
  "threads": 30,
  "workers": 1
}
`
		actualString, err := generateGunicornConfig(pgAdmin, false, 0, "H")
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
  "logconfig_dict": {},
  "threads": 25,
  "workers": 1
}
`
		actualString, err := generateGunicornConfig(pgAdmin, false, 0, "H")
		assert.NilError(t, err)
		assert.Equal(t, actualString, expectedString)
	})

	t.Run("OTel enabled", func(t *testing.T) {
		pgAdmin := &v1beta1.PGAdmin{}
		pgAdmin.Name = "test"
		pgAdmin.Namespace = "postgres-operator"
		require.UnmarshalInto(t, &pgAdmin.Spec, `{
			instrumentation: {
				logs: { retentionPeriod: 5h },
			},
		}`)
		actualString, err := generateGunicornConfig(pgAdmin, true, 4, "H")

		expectedString := `{
  "bind": "0.0.0.0:5050",
  "logconfig_dict": {
    "formatters": {
      "generic": {
        "class": "logging.Formatter",
        "datefmt": "[%Y-%m-%d %H:%M:%S %z]",
        "format": "%(asctime)s [%(process)d] [%(levelname)s] %(message)s"
      },
      "json": {
        "class": "jsonformatter.JsonFormatter",
        "format": {
          "level": "levelname",
          "message": "message",
          "name": "name",
          "time": "created"
        },
        "separators": [
          ",",
          ":"
        ]
      }
    },
    "handlers": {
      "console": {
        "class": "logging.StreamHandler",
        "formatter": "generic",
        "stream": "ext://sys.stdout"
      },
      "file": {
        "backupCount": 4,
        "class": "logging.handlers.TimedRotatingFileHandler",
        "filename": "/var/lib/pgadmin/logs/gunicorn.log",
        "formatter": "json",
        "interval": 1,
        "when": "H"
      }
    },
    "loggers": {
      "gunicorn.access": {
        "handlers": [
          "file"
        ],
        "level": "INFO",
        "propagate": true,
        "qualname": "gunicorn.access"
      },
      "gunicorn.error": {
        "handlers": [
          "file"
        ],
        "level": "INFO",
        "propagate": true,
        "qualname": "gunicorn.error"
      }
    }
  },
  "threads": 25,
  "workers": 1
}
`

		assert.NilError(t, err)
		assert.Equal(t, actualString, expectedString)
	})

}
