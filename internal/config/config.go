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

package config

import (
	"fmt"
	"os"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// a list of container images that are available
// The Red Hat Marketplace requires environment variables to be used for any
// image except the Operator's own image (see
//  https://redhat-connect.gitbook.io/certified-operator-guide/troubleshooting-and-resources/offline-enabled-operators#golang-operators)
// Any container images that the Operator might require to perform its functions
// must be made available. Because a user could potentially use any of the images
// below, each is defined separately. (See
//	https://docs.openshift.com/container-platform/4.7/operators/operator_sdk/osdk-generating-csvs.html#olm-enabling-operator-for-restricted-network_osdk-generating-csvs)
// This approach works with the concept of image streams and can allow for automatic
// updates when the container image is changed (See
//  https://docs.openshift.com/container-platform/4.7/openshift_images/images-understand.html#images-imagestream-use_images-understand)

const (
	// This is the base string for each version of Postgres.
	// For standard PostgreSQL images, the format is
	// RELATED_IMAGE_POSTGRES_<Version>
	// where Version is the PostgreSQL version set in Spec.PostgresVersion.
	// Example: RELATED_IMAGE_POSTGRES_13
	// For PostGIS enabled PostgreSQL images, the format is
	// RELATED_IMAGE_POSTGRES_<Version>_GIS_<GIS Version> where "Version" is the
	// PostgreSQL version set in Spec.PostgresVersion and "GIS Version"
	// is the PostGIS version set in Spec.PostGISVersion.
	// Example: RELATED_IMAGE_POSTGRES_13_GIS_3.1
	CrunchyPostgres = "RELATED_IMAGE_POSTGRES_"

	// Remaining images
	CrunchyPGBouncer  = "RELATED_IMAGE_PGBOUNCER"
	CrunchyPGBackRest = "RELATED_IMAGE_PGBACKREST"
	CrunchyPGExporter = "RELATED_IMAGE_PGEXPORTER"
)

// defaultFromEnv returns the value of the given environment
// variable if the spec value given is empty
func defaultFromEnv(value, key string) string {
	if value == "" {
		return os.Getenv(key)
	}
	return value
}

// PGBackRestContainerImage returns the container image to use for pgBackRest.
// It takes the current image defined in the postgrescluster spec and the
// associated related image override environment variable. If the image
// defined on the spec is empty, the value stored in the environment variable
// is returned.
func PGBackRestContainerImage(cluster *v1beta1.PostgresCluster) string {
	return defaultFromEnv(cluster.Spec.Backups.PGBackRest.Image,
		CrunchyPGBackRest)
}

// PGBouncerContainerImage returns the container image to use for pgBouncer.
// It takes the current image defined in the postgrescluster spec and the
// associated related image override environment variable. If the image
// defined on the spec is empty, the value stored in the environment variable
// is returned.
func PGBouncerContainerImage(cluster *v1beta1.PostgresCluster) string {
	return defaultFromEnv(cluster.Spec.Proxy.PGBouncer.Image,
		CrunchyPGBouncer)
}

// PGExporterContainerImage returns the container image to use for the
// PostgreSQL Exporter. It takes the current image defined in the
// postgrescluster spec and the associated related image override environment
// variable. If the image defined on the spec is empty, the value stored in the
// environment variable is returned.
func PGExporterContainerImage(cluster *v1beta1.PostgresCluster) string {
	return defaultFromEnv(cluster.Spec.Monitoring.PGMonitor.Exporter.Image,
		CrunchyPGExporter)
}

// PostgresContainerImage returns the container image to use for PostgreSQL
// based images. It takes in the postgrescluster CRD and first checks for the
// image value on the spec. If empty, it attempts to retrieve the relevant
// environment variable value for the required image. This is determined by
// gathering the defined Postgres version and, if it exists, PostGIS version
// from the spec. Depending on these configured items, the relevant value is
// pulled from the environment variable.
func PostgresContainerImage(cluster *v1beta1.PostgresCluster) string {

	if cluster.Spec.Image != "" {
		return cluster.Spec.Image
	}

	key := fmt.Sprintf("%s%v", CrunchyPostgres, cluster.Spec.PostgresVersion)

	if version := cluster.Spec.PostGISVersion; version != "" {
		key += "_GIS_" + version
	}

	return os.Getenv(key)
}
