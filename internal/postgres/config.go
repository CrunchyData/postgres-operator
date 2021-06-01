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
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

const (
	// bashSafeLink is a Bash function that moves an existing file or directory
	// and replaces it with a symbolic link.
	bashSafeLink = `
safelink() (
  local desired="$1" name="$2" current
  current=$(realpath "${name}")
  if [ "${current}" = "${desired}" ]; then return; fi
  set -x; mv --no-target-directory "${current}" "${desired}"
  ln --no-dereference --force --symbolic "${desired}" "${name}"
)
`

	// dataMountPath is where to mount the main data volume.
	dataMountPath = "/pgdata"

	// walMountPath is where to mount the optional WAL volume.
	walMountPath = "/pgwal"

	// SocketDirectory is where to bind and connect to UNIX sockets.
	SocketDirectory = "/tmp/postgres"

	// ReplicationUser is the PostgreSQL role that will be created by Patroni
	// for streaming replication and for `pg_rewind`.
	ReplicationUser = "_crunchyrepl"
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

// WALDirectory returns the absolute path to the directory where an instance
// stores its WAL files.
// - https://www.postgresql.org/docs/current/wal.html
func WALDirectory(
	cluster *v1beta1.PostgresCluster, instance *v1beta1.PostgresInstanceSetSpec,
) string {
	// When no WAL volume is specified, store WAL files on the main data volume.
	walStorage := dataMountPath
	if instance.WALVolumeClaimSpec != nil {
		walStorage = walMountPath
	}
	return fmt.Sprintf("%s/pg%d_wal", walStorage, cluster.Spec.PostgresVersion)
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
			Name:  "PGHOST",
			Value: SocketDirectory,
		},
		{
			Name:  "PGPORT",
			Value: fmt.Sprint(*cluster.Spec.Port),
		},
	}
}

// startupCommand returns an entrypoint that prepares the filesystem for
// PostgreSQL.
func startupCommand(
	cluster *v1beta1.PostgresCluster, instance *v1beta1.PostgresInstanceSetSpec,
) []string {
	version := fmt.Sprint(cluster.Spec.PostgresVersion)
	walDir := WALDirectory(cluster, instance)

	args := []string{version, walDir}
	script := strings.Join([]string{
		`declare -r expected_major_version="$1" pgwal_directory="$2"`,

		// Function to log values in a basic structured format.
		`results() { printf '::postgres-operator: %s::%s\n' "$@"; }`,

		// PostgreSQL prefers its files to be writable by only itself.
		// - https://www.postgresql.org/docs/current/creating-cluster.html
		`directory() { [ -d "$1" ] || install --directory --mode=0700 "$1"; }`,

		// Function to change a directory symlink while keeping the directory content.
		strings.TrimSpace(bashSafeLink),

		// Log the effective user ID and all the group IDs.
		`echo Initializing ...`,
		`results 'uid' "$(id -u)" 'gid' "$(id -G)"`,

		// Abort when the PostgreSQL version installed in the image does not
		// match the cluster spec.
		`results 'postgres path' "$(command -v postgres)"`,
		`results 'postgres version' "${postgres_version:=$(postgres --version)}"`,
		`[[ "${postgres_version}" == *") ${expected_major_version}."* ]]`,

		// Abort when the configured data directory is not $PGDATA.
		// - https://www.postgresql.org/docs/current/runtime-config-file-locations.html
		`results 'config directory' "${PGDATA:?}"`,
		`postgres_data_directory=$([ -d "${PGDATA}" ] && postgres -C data_directory || echo "${PGDATA}")`,
		`results 'data directory' "${postgres_data_directory}"`,
		`[ "${postgres_data_directory}" = "${PGDATA}" ]`,

		// pgBackRest expects the data directory to exist before doing a restore.
		`directory "${postgres_data_directory}"`,

		// When the data directory is empty, there's nothing more to do.
		`[ -f "${postgres_data_directory}/PG_VERSION" ] || exit 0`,

		// Abort when the data directory is not empty and its version does not
		// match the cluster spec.
		`results 'data version' "${postgres_data_version:=$(< "${postgres_data_directory}/PG_VERSION")}"`,
		`[ "${postgres_data_version}" = "${expected_major_version}" ]`,

		// Safely move the WAL directory onto the intended volume. PostgreSQL
		// always writes WAL files in the "pg_wal" directory inside the data
		// directory. The recommended way to relocate it is with a symbolic
		// link. `initdb` and `pg_basebackup` have a `--waldir` flag that does
		// the same.
		// - https://www.postgresql.org/docs/current/wal-internals.html
		// - https://git.postgresql.org/gitweb/?p=postgresql.git;f=src/bin/initdb/initdb.c;hb=REL_13_0#l2718
		// - https://git.postgresql.org/gitweb/?p=postgresql.git;f=src/bin/pg_basebackup/pg_basebackup.c;hb=REL_13_0#l2621
		`safelink "${pgwal_directory}" "${postgres_data_directory}/pg_wal"`,
		`results 'wal directory' "$(realpath "${postgres_data_directory}/pg_wal")"`,
	}, "\n")

	return append([]string{"bash", "-ceu", "--", script, "startup"}, args...)
}
