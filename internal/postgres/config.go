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

package postgres

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

const (
	// dataMountPath is where to mount the main data volume.
	dataMountPath = "/pgdata"
)

// ConfigDirectory returns the absolute path to $PGDATA for cluster.
// - https://www.postgresql.org/docs/current/runtime-config-file-locations.html
func ConfigDirectory(cluster *v1beta1.PostgresCluster) string {
	return DataDirectory(cluster)
}

// DataDirectory returns the absolute path to the "data_directory" of cluster.
// - https://www.postgresql.org/docs/current/runtime-config-file-locations.html
func DataDirectory(cluster *v1beta1.PostgresCluster) string {
	return fmt.Sprintf("%s/pg%d", dataMountPath, cluster.Spec.PostgresVersion)
}

// Environment returns the environment variables required to invoke PostgreSQL
// utilities.
func Environment(cluster *v1beta1.PostgresCluster) []corev1.EnvVar {
	return []corev1.EnvVar{
		// - https://www.postgresql.org/docs/current/reference-server.html
		{
			Name:  "PGDATA",
			Value: ConfigDirectory(cluster),
		},

		// - https://www.postgresql.org/docs/current/libpq-envars.html
		{
			Name:  "PGPORT",
			Value: fmt.Sprint(*cluster.Spec.Port),
		},
	}
}
