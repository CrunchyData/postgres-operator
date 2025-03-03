// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestFetchKeyCommand(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
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
	})

	t.Run("blank", func(t *testing.T) {
		var spec1 v1beta1.PostgresClusterSpec
		require.UnmarshalInto(t, &spec1, `{
			patroni: {
				dynamicConfiguration: {
					postgresql: {
						parameters: {
							encryption_key_command: "",
						},
					},
				},
			},
		}`)
		assert.Equal(t, "", FetchKeyCommand(&spec1))

		var spec2 v1beta1.PostgresClusterSpec
		require.UnmarshalInto(t, &spec2, `{
			config: {
				parameters: {
					encryption_key_command: "",
				},
			},
		}`)
		assert.Equal(t, "", FetchKeyCommand(&spec2))
	})

	t.Run("exists", func(t *testing.T) {
		var spec1 v1beta1.PostgresClusterSpec
		require.UnmarshalInto(t, &spec1, `{
			patroni: {
				dynamicConfiguration: {
					postgresql: {
						parameters: {
							encryption_key_command: "echo mykey",
						},
					},
				},
			},
		}`)
		assert.Equal(t, "echo mykey", FetchKeyCommand(&spec1))

		var spec2 v1beta1.PostgresClusterSpec
		require.UnmarshalInto(t, &spec2, `{
			config: {
				parameters: {
					encryption_key_command: "cat somefile",
				},
			},
		}`)
		assert.Equal(t, "cat somefile", FetchKeyCommand(&spec2))
	})

	t.Run("config.parameters takes precedence", func(t *testing.T) {
		var spec v1beta1.PostgresClusterSpec
		require.UnmarshalInto(t, &spec, `{
			config: {
				parameters: {
					encryption_key_command: "cat somefile",
				},
			},
			patroni: {
				dynamicConfiguration: {
					postgresql: {
						parameters: {
							encryption_key_command: "echo mykey",
						},
					},
				},
			},
		}`)
		assert.Equal(t, "cat somefile", FetchKeyCommand(&spec))
	})
}

func TestPGAdminContainerImage(t *testing.T) {
	cluster := &v1beta1.PostgresCluster{}

	t.Setenv("RELATED_IMAGE_PGADMIN", "")
	os.Unsetenv("RELATED_IMAGE_PGADMIN")
	assert.Equal(t, PGAdminContainerImage(cluster), "")

	t.Setenv("RELATED_IMAGE_PGADMIN", "")
	assert.Equal(t, PGAdminContainerImage(cluster), "")

	t.Setenv("RELATED_IMAGE_PGADMIN", "env-var-pgadmin")
	assert.Equal(t, PGAdminContainerImage(cluster), "env-var-pgadmin")

	require.UnmarshalInto(t, &cluster.Spec, `{
		userInterface: { pgAdmin: { image: spec-image } },
	}`)
	assert.Equal(t, PGAdminContainerImage(cluster), "spec-image")
}

func TestPGBackRestContainerImage(t *testing.T) {
	cluster := &v1beta1.PostgresCluster{}

	t.Setenv("RELATED_IMAGE_PGBACKREST", "")
	os.Unsetenv("RELATED_IMAGE_PGBACKREST")
	assert.Equal(t, PGBackRestContainerImage(cluster), "")

	t.Setenv("RELATED_IMAGE_PGBACKREST", "")
	assert.Equal(t, PGBackRestContainerImage(cluster), "")

	t.Setenv("RELATED_IMAGE_PGBACKREST", "env-var-pgbackrest")
	assert.Equal(t, PGBackRestContainerImage(cluster), "env-var-pgbackrest")

	require.UnmarshalInto(t, &cluster.Spec, `{
		backups: { pgbackrest: { image: spec-image } },
	}`)
	assert.Equal(t, PGBackRestContainerImage(cluster), "spec-image")
}

func TestPGBouncerContainerImage(t *testing.T) {
	cluster := &v1beta1.PostgresCluster{}

	t.Setenv("RELATED_IMAGE_PGBOUNCER", "")
	os.Unsetenv("RELATED_IMAGE_PGBOUNCER")
	assert.Equal(t, PGBouncerContainerImage(cluster), "")

	t.Setenv("RELATED_IMAGE_PGBOUNCER", "")
	assert.Equal(t, PGBouncerContainerImage(cluster), "")

	t.Setenv("RELATED_IMAGE_PGBOUNCER", "env-var-pgbouncer")
	assert.Equal(t, PGBouncerContainerImage(cluster), "env-var-pgbouncer")

	require.UnmarshalInto(t, &cluster.Spec, `{
		proxy: { pgBouncer: { image: spec-image } },
	}`)
	assert.Equal(t, PGBouncerContainerImage(cluster), "spec-image")
}

func TestPGExporterContainerImage(t *testing.T) {
	cluster := &v1beta1.PostgresCluster{}

	t.Setenv("RELATED_IMAGE_PGEXPORTER", "")
	os.Unsetenv("RELATED_IMAGE_PGEXPORTER")
	assert.Equal(t, PGExporterContainerImage(cluster), "")

	t.Setenv("RELATED_IMAGE_PGEXPORTER", "")
	assert.Equal(t, PGExporterContainerImage(cluster), "")

	t.Setenv("RELATED_IMAGE_PGEXPORTER", "env-var-pgexporter")
	assert.Equal(t, PGExporterContainerImage(cluster), "env-var-pgexporter")

	require.UnmarshalInto(t, &cluster.Spec, `{
		monitoring: { pgmonitor: { exporter: { image: spec-image } } },
	}`)
	assert.Equal(t, PGExporterContainerImage(cluster), "spec-image")
}

func TestStandalonePGAdminContainerImage(t *testing.T) {
	pgadmin := &v1beta1.PGAdmin{}

	t.Setenv("RELATED_IMAGE_STANDALONE_PGADMIN", "")
	os.Unsetenv("RELATED_IMAGE_STANDALONE_PGADMIN")
	assert.Equal(t, StandalonePGAdminContainerImage(pgadmin), "")

	t.Setenv("RELATED_IMAGE_STANDALONE_PGADMIN", "")
	assert.Equal(t, StandalonePGAdminContainerImage(pgadmin), "")

	t.Setenv("RELATED_IMAGE_STANDALONE_PGADMIN", "env-var-pgadmin")
	assert.Equal(t, StandalonePGAdminContainerImage(pgadmin), "env-var-pgadmin")

	require.UnmarshalInto(t, &pgadmin.Spec, `{
		image: spec-image
	}`)
	assert.Equal(t, StandalonePGAdminContainerImage(pgadmin), "spec-image")
}

func TestPostgresContainerImage(t *testing.T) {
	cluster := &v1beta1.PostgresCluster{}
	cluster.Spec.PostgresVersion = 12

	t.Setenv("RELATED_IMAGE_POSTGRES_12", "")
	os.Unsetenv("RELATED_IMAGE_POSTGRES_12")
	assert.Equal(t, PostgresContainerImage(cluster), "")

	t.Setenv("RELATED_IMAGE_POSTGRES_12", "")
	assert.Equal(t, PostgresContainerImage(cluster), "")

	t.Setenv("RELATED_IMAGE_POSTGRES_12", "env-var-postgres")
	assert.Equal(t, PostgresContainerImage(cluster), "env-var-postgres")

	cluster.Spec.Image = "spec-image"
	assert.Equal(t, PostgresContainerImage(cluster), "spec-image")

	cluster.Spec.Image = ""
	cluster.Spec.PostGISVersion = "3.0"
	t.Setenv("RELATED_IMAGE_POSTGRES_12_GIS_3.0", "env-var-postgis")
	assert.Equal(t, PostgresContainerImage(cluster), "env-var-postgis")

	cluster.Spec.Image = "spec-image"
	assert.Equal(t, PostgresContainerImage(cluster), "spec-image")
}

func TestVerifyImageValues(t *testing.T) {
	cluster := &v1beta1.PostgresCluster{}

	verifyImageCheck := func(t *testing.T, envVar, errString string, cluster *v1beta1.PostgresCluster) {

		t.Setenv(envVar, "")
		os.Unsetenv(envVar)
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
