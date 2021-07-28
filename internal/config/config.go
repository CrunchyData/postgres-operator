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

// defaultFromEnv reads the environment variable key when value is empty.
func defaultFromEnv(value, key string) string {
	if value == "" {
		return os.Getenv(key)
	}
	return value
}

// PGBackRestContainerImage returns the container image to use for pgBackRest.
func PGBackRestContainerImage(cluster *v1beta1.PostgresCluster) string {
	image := cluster.Spec.Backups.PGBackRest.Image

	return defaultFromEnv(image, "RELATED_IMAGE_PGBACKREST")
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
