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
	"fmt"
	"os"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// defaultFromEnv reads the environment variable key when value is empty.
func defaultFromEnv(value, key string) string {
	if value == "" {
		return os.Getenv(key)
	}
	return value
}

// Red Hat Marketplace requires operators to use environment variables be used
// for any image other than the operator itself. Those variables must start with
// "RELATED_IMAGE_" so that OSBS can transform their tag values into digests
// for a "disconncted" OLM CSV.

// - https://redhat-connect.gitbook.io/certified-operator-guide/troubleshooting-and-resources/offline-enabled-operators
// - https://osbs.readthedocs.io/en/latest/users.html#pullspec-locations

// PGBackRestContainerImage returns the container image to use for pgBackRest.
func PGBackRestContainerImage(cluster *v1beta1.PostgresCluster) string {
	image := cluster.Spec.Backups.PGBackRest.Image

	return defaultFromEnv(image, "RELATED_IMAGE_PGBACKREST")
}

// PGAdminContainerImage returns the container image to use for pgAdmin.
func PGAdminContainerImage(cluster *v1beta1.PostgresCluster) string {
	var image string
	if cluster.Spec.UserInterface != nil &&
		cluster.Spec.UserInterface.PGAdmin != nil {
		image = cluster.Spec.UserInterface.PGAdmin.Image
	}

	return defaultFromEnv(image, "RELATED_IMAGE_PGADMIN")
}

// PGBouncerContainerImage returns the container image to use for pgBouncer.
func PGBouncerContainerImage(cluster *v1beta1.PostgresCluster) string {
	var image string
	if cluster.Spec.Proxy != nil &&
		cluster.Spec.Proxy.PGBouncer != nil {
		image = cluster.Spec.Proxy.PGBouncer.Image
	}

	return defaultFromEnv(image, "RELATED_IMAGE_PGBOUNCER")
}

// PGExporterContainerImage returns the container image to use for the
// PostgreSQL Exporter.
func PGExporterContainerImage(cluster *v1beta1.PostgresCluster) string {
	var image string
	if cluster.Spec.Monitoring != nil &&
		cluster.Spec.Monitoring.PGMonitor != nil &&
		cluster.Spec.Monitoring.PGMonitor.Exporter != nil {
		image = cluster.Spec.Monitoring.PGMonitor.Exporter.Image
	}

	return defaultFromEnv(image, "RELATED_IMAGE_PGEXPORTER")
}

// PostgresContainerImage returns the container image to use for PostgreSQL.
func PostgresContainerImage(cluster *v1beta1.PostgresCluster) string {
	image := cluster.Spec.Image
	key := "RELATED_IMAGE_POSTGRES_" + fmt.Sprint(cluster.Spec.PostgresVersion)

	if version := cluster.Spec.PostGISVersion; version != "" {
		key += "_GIS_" + version
	}

	return defaultFromEnv(image, key)
}

// PGONamespace returns the namespace where the PGO is running,
// based on the env var from the DownwardAPI
// If no env var is found, returns ""
func PGONamespace() string {
	return os.Getenv("PGO_NAMESPACE")
}
