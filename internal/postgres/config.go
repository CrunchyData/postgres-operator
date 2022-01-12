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

package postgres

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/internal/naming"
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

	// downwardAPIPath is where to mount the downwardAPI volume.
	downwardAPIPath = "/etc/database-containerinfo"

	// SocketDirectory is where to bind and connect to UNIX sockets.
	SocketDirectory = "/tmp/postgres"

	// ReplicationUser is the PostgreSQL role that will be created by Patroni
	// for streaming replication and for `pg_rewind`.
	ReplicationUser = "_crunchyrepl"

	// configMountPath is where to mount additional config files
	configMountPath = "/etc/postgres"
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
		// Setting the KRB5_CONFIG for kerberos
		// - https://web.mit.edu/kerberos/krb5-current/doc/admin/conf_files/krb5_conf.html
		{
			Name:  "KRB5_CONFIG",
			Value: configMountPath + "/krb5.conf",
		},
		// In testing it was determined that we need to set this env var for the replay cache
		// otherwise it defaults to the read-only location `/var/tmp/`
		// - https://web.mit.edu/kerberos/krb5-current/doc/basic/rcache_def.html#replay-cache-types
		{
			Name:  "KRB5RCACHEDIR",
			Value: "/tmp",
		},
	}
}

// reloadCommand returns an entrypoint that convinces PostgreSQL to reload
// certificate files when they change. The process will appear as name in `ps`
// and `top`.
func reloadCommand(name string) []string {
	// Use a Bash loop to periodically check the mtime of the mounted
	// certificate volume. When it changes, copy the replication certificate,
	// signal PostgreSQL, and print the observed timestamp.
	//
	// PostgreSQL v10 reads its server certificate files during reload (SIGHUP).
	// - https://www.postgresql.org/docs/current/ssl-tcp.html#SSL-SERVER-FILES
	// - https://www.postgresql.org/docs/current/app-postgres.html
	//
	// PostgreSQL reads its replication credentials every time it opens a
	// replication connection. It does not need to be signaled when the
	// certificate contents change.
	//
	// The copy is necessary because Kubernetes sets g+r when fsGroup is enabled,
	// but PostgreSQL requires client keys to not be readable by other users.
	// - https://www.postgresql.org/docs/current/libpq-ssl.html
	// - https://issue.k8s.io/57923
	//
	// Coreutils `sleep` uses a lot of memory, so the following opens a file
	// descriptor and uses the timeout of the builtin `read` to wait. That same
	// descriptor gets closed and reopened to use the builtin `[ -nt` to check
	// mtimes.
	// - https://unix.stackexchange.com/a/407383
	script := fmt.Sprintf(`
declare -r directory=%q
exec {fd}<> <(:)
while read -r -t 5 -u "${fd}" || true; do
  if [ "${directory}" -nt "/proc/self/fd/${fd}" ] &&
    install -D --mode=0600 -t %q "${directory}"/{%s,%s,%s} &&
    pkill -HUP --exact --parent=1 postgres
  then
    exec {fd}>&- && exec {fd}<> <(:)
    stat --format='Loaded certificates dated %%y' "${directory}"
  fi
done
`,
		naming.CertMountPath,
		naming.ReplicationTmp,
		naming.ReplicationCertPath,
		naming.ReplicationPrivateKeyPath,
		naming.ReplicationCACertPath,
	)

	// Elide the above script from `ps` and `top` by wrapping it in a function
	// and calling that.
	wrapper := `monitor() {` + script + `}; export -f monitor; exec -a "$0" bash -ceu monitor`

	return []string{"bash", "-ceu", "--", wrapper, name}
}

// startupCommand returns an entrypoint that prepares the filesystem for
// PostgreSQL.
func startupCommand(
	cluster *v1beta1.PostgresCluster, instance *v1beta1.PostgresInstanceSetSpec,
) []string {
	version := fmt.Sprint(cluster.Spec.PostgresVersion)
	walDir := WALDirectory(cluster, instance)

	args := []string{version, walDir, naming.PGBackRestPGDataLogPath}
	script := strings.Join([]string{
		`declare -r expected_major_version="$1" pgwal_directory="$2" pgbrLog_directory="$3"`,

		// Function to log values in a basic structured format.
		`results() { printf '::postgres-operator: %s::%s\n' "$@"; }`,

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

		// Determine if the data directory has been prepared for bootstrapping the cluster
		`bootstrap_dir="${postgres_data_directory}_bootstrap"`,
		`[ -d "${bootstrap_dir}" ] && results 'bootstrap directory' "${bootstrap_dir}"`,
		`[ -d "${bootstrap_dir}" ] && postgres_data_directory="${bootstrap_dir}"`,

		// PostgreSQL requires its directory to be writable by only itself.
		// Pod "securityContext.fsGroup" sets g+w on directories for *some*
		// storage providers.
		// - https://www.postgresql.org/docs/current/creating-cluster.html
		// - https://git.postgresql.org/gitweb/?p=postgresql.git;f=src/backend/utils/init/miscinit.c;hb=REL_13_0#l319
		// - https://issue.k8s.io/93802#issuecomment-717646167
		`install --directory --mode=0700 "${postgres_data_directory}"`,

		// Create the pgBackRest log directory.
		`results 'pgBackRest log directory' "${pgbrLog_directory}"`,
		`install --directory --mode=0775 "${pgbrLog_directory}"`,

		// Copy replication client certificate files
		// from the /pgconf/tls/replication directory to the /tmp/replication directory in order
		// to set proper file permissions. This is required because the group permission settings
		// applied via the defaultMode option are not honored as expected, resulting in incorrect
		// group read permissions.
		// See https://github.com/kubernetes/kubernetes/issues/57923
		// TODO(tjmoore4): remove this implementation when/if defaultMode permissions are set as
		// expected for the mounted volume.
		fmt.Sprintf(`install -D --mode=0600 -t %q %q/{%s,%s,%s}`,
			naming.ReplicationTmp, naming.CertMountPath+naming.ReplicationDirectory,
			naming.ReplicationCert, naming.ReplicationPrivateKey,
			naming.ReplicationCACert),

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
