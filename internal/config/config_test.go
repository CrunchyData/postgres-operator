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
