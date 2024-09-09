// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"testing"

	"gotest.tools/v3/assert"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func saveEnv(t testing.TB, key string) {
	t.Helper()
	previous, ok := os.LookupEnv(key)
	t.Cleanup(func() {
		if ok {
			os.Setenv(key, previous)
		} else {
			os.Unsetenv(key)
		}
	})
}

func setEnv(t testing.TB, key, value string) {
	t.Helper()
	saveEnv(t, key)
	assert.NilError(t, os.Setenv(key, value))
}

func unsetEnv(t testing.TB, key string) {
	t.Helper()
	saveEnv(t, key)
	assert.NilError(t, os.Unsetenv(key))
}

func TestFetchKeyCommand(t *testing.T) {

	spec1 := v1beta1.PostgresClusterSpec{}
	assert.Assert(t, FetchKeyCommand(&spec1) == "")

	spec2 := v1beta1.PostgresClusterSpec{
		Patroni: &v1beta1.PatroniSpec{},
	}
	assert.Assert(t, FetchKeyCommand(&spec2) == "")

	spec3 := v1beta1.PostgresClusterSpec{
		Patroni: &v1beta1.PatroniSpec{
			DynamicConfiguration: map[string]any{},
		},
	}
	assert.Assert(t, FetchKeyCommand(&spec3) == "")

	spec4 := v1beta1.PostgresClusterSpec{
		Patroni: &v1beta1.PatroniSpec{
			DynamicConfiguration: map[string]any{
				"postgresql": map[string]any{},
			},
		},
	}
	assert.Assert(t, FetchKeyCommand(&spec4) == "")

	spec5 := v1beta1.PostgresClusterSpec{
		Patroni: &v1beta1.PatroniSpec{
			DynamicConfiguration: map[string]any{
				"postgresql": map[string]any{
					"parameters": map[string]any{},
				},
			},
		},
	}
	assert.Assert(t, FetchKeyCommand(&spec5) == "")

	spec6 := v1beta1.PostgresClusterSpec{
		Patroni: &v1beta1.PatroniSpec{
			DynamicConfiguration: map[string]any{
				"postgresql": map[string]any{
					"parameters": map[string]any{
						"encryption_key_command": "",
					},
				},
			},
		},
	}
	assert.Assert(t, FetchKeyCommand(&spec6) == "")

	spec7 := v1beta1.PostgresClusterSpec{
		Patroni: &v1beta1.PatroniSpec{
			DynamicConfiguration: map[string]any{
				"postgresql": map[string]any{
					"parameters": map[string]any{
						"encryption_key_command": "echo mykey",
					},
				},
			},
		},
	}
	assert.Assert(t, FetchKeyCommand(&spec7) == "echo mykey")

}

func TestPGAdminContainerImage(t *testing.T) {
	cluster := &v1beta1.PostgresCluster{}

	unsetEnv(t, "RELATED_IMAGE_PGADMIN")
	assert.Equal(t, PGAdminContainerImage(cluster), "")

	setEnv(t, "RELATED_IMAGE_PGADMIN", "")
	assert.Equal(t, PGAdminContainerImage(cluster), "")

	setEnv(t, "RELATED_IMAGE_PGADMIN", "env-var-pgadmin")
	assert.Equal(t, PGAdminContainerImage(cluster), "env-var-pgadmin")

	assert.NilError(t, yaml.Unmarshal([]byte(`{
		userInterface: { pgAdmin: { image: spec-image } },
	}`), &cluster.Spec))
	assert.Equal(t, PGAdminContainerImage(cluster), "spec-image")
}

func TestPGBackRestContainerImage(t *testing.T) {
	cluster := &v1beta1.PostgresCluster{}

	unsetEnv(t, "RELATED_IMAGE_PGBACKREST")
	assert.Equal(t, PGBackRestContainerImage(cluster), "")

	setEnv(t, "RELATED_IMAGE_PGBACKREST", "")
	assert.Equal(t, PGBackRestContainerImage(cluster), "")

	setEnv(t, "RELATED_IMAGE_PGBACKREST", "env-var-pgbackrest")
	assert.Equal(t, PGBackRestContainerImage(cluster), "env-var-pgbackrest")

	assert.NilError(t, yaml.Unmarshal([]byte(`{
		backups: { pgBackRest: { image: spec-image } },
	}`), &cluster.Spec))
	assert.Equal(t, PGBackRestContainerImage(cluster), "spec-image")
}

func TestPGBouncerContainerImage(t *testing.T) {
	cluster := &v1beta1.PostgresCluster{}

	unsetEnv(t, "RELATED_IMAGE_PGBOUNCER")
	assert.Equal(t, PGBouncerContainerImage(cluster), "")

	setEnv(t, "RELATED_IMAGE_PGBOUNCER", "")
	assert.Equal(t, PGBouncerContainerImage(cluster), "")

	setEnv(t, "RELATED_IMAGE_PGBOUNCER", "env-var-pgbouncer")
	assert.Equal(t, PGBouncerContainerImage(cluster), "env-var-pgbouncer")

	assert.NilError(t, yaml.Unmarshal([]byte(`{
		proxy: { pgBouncer: { image: spec-image } },
	}`), &cluster.Spec))
	assert.Equal(t, PGBouncerContainerImage(cluster), "spec-image")
}

func TestPGExporterContainerImage(t *testing.T) {
	cluster := &v1beta1.PostgresCluster{}

	unsetEnv(t, "RELATED_IMAGE_PGEXPORTER")
	assert.Equal(t, PGExporterContainerImage(cluster), "")

	setEnv(t, "RELATED_IMAGE_PGEXPORTER", "")
	assert.Equal(t, PGExporterContainerImage(cluster), "")

	setEnv(t, "RELATED_IMAGE_PGEXPORTER", "env-var-pgexporter")
	assert.Equal(t, PGExporterContainerImage(cluster), "env-var-pgexporter")

	assert.NilError(t, yaml.Unmarshal([]byte(`{
		monitoring: { pgMonitor: { exporter: { image: spec-image } } },
	}`), &cluster.Spec))
	assert.Equal(t, PGExporterContainerImage(cluster), "spec-image")
}

func TestStandalonePGAdminContainerImage(t *testing.T) {
	pgadmin := &v1beta1.PGAdmin{}

	unsetEnv(t, "RELATED_IMAGE_STANDALONE_PGADMIN")
	assert.Equal(t, StandalonePGAdminContainerImage(pgadmin), "")

	setEnv(t, "RELATED_IMAGE_STANDALONE_PGADMIN", "")
	assert.Equal(t, StandalonePGAdminContainerImage(pgadmin), "")

	setEnv(t, "RELATED_IMAGE_STANDALONE_PGADMIN", "env-var-pgadmin")
	assert.Equal(t, StandalonePGAdminContainerImage(pgadmin), "env-var-pgadmin")

	assert.NilError(t, yaml.Unmarshal([]byte(`{
		image: spec-image
	}`), &pgadmin.Spec))
	assert.Equal(t, StandalonePGAdminContainerImage(pgadmin), "spec-image")
}

func TestPostgresContainerImage(t *testing.T) {
	cluster := &v1beta1.PostgresCluster{}
	cluster.Spec.PostgresVersion = 12

	unsetEnv(t, "RELATED_IMAGE_POSTGRES_12")
	assert.Equal(t, PostgresContainerImage(cluster), "")

	setEnv(t, "RELATED_IMAGE_POSTGRES_12", "")
	assert.Equal(t, PostgresContainerImage(cluster), "")

	setEnv(t, "RELATED_IMAGE_POSTGRES_12", "env-var-postgres")
	assert.Equal(t, PostgresContainerImage(cluster), "env-var-postgres")

	cluster.Spec.Image = "spec-image"
	assert.Equal(t, PostgresContainerImage(cluster), "spec-image")

	cluster.Spec.Image = ""
	cluster.Spec.PostGISVersion = "3.0"
	setEnv(t, "RELATED_IMAGE_POSTGRES_12_GIS_3.0", "env-var-postgis")
	assert.Equal(t, PostgresContainerImage(cluster), "env-var-postgis")

	cluster.Spec.Image = "spec-image"
	assert.Equal(t, PostgresContainerImage(cluster), "spec-image")
}

func TestVerifyImageValues(t *testing.T) {
	cluster := &v1beta1.PostgresCluster{}

	verifyImageCheck := func(t *testing.T, envVar, errString string, cluster *v1beta1.PostgresCluster) {
		unsetEnv(t, envVar)
		err := VerifyImageValues(cluster)
		assert.ErrorContains(t, err, errString)
	}

	t.Run("crunchy-postgres", func(t *testing.T) {
		cluster.Spec.PostgresVersion = 14
		verifyImageCheck(t, "RELATED_IMAGE_POSTGRES_14", "crunchy-postgres", cluster)
	})

	t.Run("crunchy-postgres-gis", func(t *testing.T) {
		cluster.Spec.PostGISVersion = "3.3"
		verifyImageCheck(t, "RELATED_IMAGE_POSTGRES_14_GIS_3.3", "crunchy-postgres-gis", cluster)
	})

	t.Run("crunchy-pgbackrest", func(t *testing.T) {
		verifyImageCheck(t, "RELATED_IMAGE_PGBACKREST", "crunchy-pgbackrest", cluster)
	})

	t.Run("crunchy-pgbouncer", func(t *testing.T) {
		cluster.Spec.Proxy = new(v1beta1.PostgresProxySpec)
		cluster.Spec.Proxy.PGBouncer = new(v1beta1.PGBouncerPodSpec)
		verifyImageCheck(t, "RELATED_IMAGE_PGBOUNCER", "crunchy-pgbouncer", cluster)
	})

	t.Run("crunchy-pgadmin4", func(t *testing.T) {
		cluster.Spec.UserInterface = new(v1beta1.UserInterfaceSpec)
		cluster.Spec.UserInterface.PGAdmin = new(v1beta1.PGAdminPodSpec)
		verifyImageCheck(t, "RELATED_IMAGE_PGADMIN", "crunchy-pgadmin4", cluster)
	})

	t.Run("crunchy-postgres-exporter", func(t *testing.T) {
		cluster.Spec.Monitoring = new(v1beta1.MonitoringSpec)
		cluster.Spec.Monitoring.PGMonitor = new(v1beta1.PGMonitorSpec)
		cluster.Spec.Monitoring.PGMonitor.Exporter = new(v1beta1.ExporterSpec)
		verifyImageCheck(t, "RELATED_IMAGE_PGEXPORTER", "crunchy-postgres-exporter", cluster)
	})

	t.Run("multiple images", func(t *testing.T) {
		err := VerifyImageValues(cluster)
		assert.ErrorContains(t, err, "crunchy-postgres-gis")
		assert.ErrorContains(t, err, "crunchy-pgbackrest")
		assert.ErrorContains(t, err, "crunchy-pgbouncer")
		assert.ErrorContains(t, err, "crunchy-pgadmin4")
		assert.ErrorContains(t, err, "crunchy-postgres-exporter")
	})

}
