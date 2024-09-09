// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

const (
	// bashHalt is a Bash function that prints its arguments to stderr then
	// exits with a non-zero status. It uses the exit status of the prior
	// command if that was not zero.
	bashHalt = `halt() { local rc=$?; >&2 echo "$@"; exit "${rc/#0/1}"; }`

	// bashPermissions is a Bash function that prints the permissions of a file
	// or directory and all its parent directories, except the root directory.
	bashPermissions = `permissions() {` +
		` while [[ -n "$1" ]]; do set "${1%/*}" "$@"; done; shift;` +
		` stat -Lc '%A %4u %4g %n' "$@";` +
		` }`

	// bashRecreateDirectory is a Bash function that moves the contents of an
	// existing directory into a newly created directory of the same name.
	bashRecreateDirectory = `
recreate() (
  local tmp; tmp=$(mktemp -d -p "${1%/*}"); GLOBIGNORE='.:..'; set -x
  chmod "$2" "${tmp}"; mv "$1"/* "${tmp}"; rmdir "$1"; mv "${tmp}" "$1"
)
`

	// bashSafeLink is a Bash function that moves an existing file or directory
	// and replaces it with a symbolic link.
	bashSafeLink = `
safelink() (
  local desired="$1" name="$2" current
  current=$(realpath "${name}")
  if [[ "${current}" == "${desired}" ]]; then return; fi
  set -x; mv --no-target-directory "${current}" "${desired}"
  ln --no-dereference --force --symbolic "${desired}" "${name}"
)
`

	// dataMountPath is where to mount the main data volume.
	dataMountPath = "/pgdata"

	// dataMountPath is where to mount the main data volume.
	tablespaceMountPath = "/tablespaces"

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
	return fmt.Sprintf("%s/pg%d_wal", WALStorage(instance), cluster.Spec.PostgresVersion)
}

// WALStorage returns the absolute path to the disk where an instance stores its
// WAL files. Use [WALDirectory] for the exact directory that Postgres uses.
func WALStorage(instance *v1beta1.PostgresInstanceSetSpec) string {
	if instance.WALVolumeClaimSpec != nil {
		return walMountPath
	}
	// When no WAL volume is specified, store WAL files on the main data volume.
	return dataMountPath
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
		// This allows a custom CA certificate to be mounted for Postgres LDAP
		// authentication via spec.config.files.
		// - https://wiki.postgresql.org/wiki/LDAP_Authentication_against_AD
		//
		// When setting the TLS_CACERT for LDAP as an environment variable, 'LDAP'
		// must be appended as a prefix.
		// - https://www.openldap.org/software/man.cgi?query=ldap.conf
		//
		// Testing with LDAPTLS_CACERTDIR did not work as expected during testing.
		{
			Name:  "LDAPTLS_CACERT",
			Value: configMountPath + "/ldap/ca.crt",
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
# Parameters for curl when managing autogrow annotation.
APISERVER="https://kubernetes.default.svc"
SERVICEACCOUNT="/var/run/secrets/kubernetes.io/serviceaccount"
NAMESPACE=$(cat ${SERVICEACCOUNT}/namespace)
TOKEN=$(cat ${SERVICEACCOUNT}/token)
CACERT=${SERVICEACCOUNT}/ca.crt

declare -r directory=%q
exec {fd}<> <(:||:)
while read -r -t 5 -u "${fd}" ||:; do
  # Manage replication certificate.
  if [[ "${directory}" -nt "/proc/self/fd/${fd}" ]] &&
    install -D --mode=0600 -t %q "${directory}"/{%s,%s,%s} &&
    pkill -HUP --exact --parent=1 postgres
  then
    exec {fd}>&- && exec {fd}<> <(:||:)
    stat --format='Loaded certificates dated %%y' "${directory}"
  fi

  # Manage autogrow annotation.
  # Return size in Mebibytes.
  size=$(df --human-readable --block-size=M /pgdata | awk 'FNR == 2 {print $2}')
  use=$(df --human-readable /pgdata | awk 'FNR == 2 {print $5}')
  sizeInt="${size//M/}"
  # Use the sed punctuation class, because the shell will not accept the percent sign in an expansion.
  useInt=$(echo $use | sed 's/[[:punct:]]//g')
  triggerExpansion="$((useInt > 75))"
  if [ $triggerExpansion -eq 1 ]; then
    newSize="$(((sizeInt / 2)+sizeInt))"
    newSizeMi="${newSize}Mi"
    d='[{"op": "add", "path": "/metadata/annotations/suggested-pgdata-pvc-size", "value": "'"$newSizeMi"'"}]'
    curl --cacert ${CACERT} --header "Authorization: Bearer ${TOKEN}" -XPATCH "${APISERVER}/api/v1/namespaces/${NAMESPACE}/pods/${HOSTNAME}?fieldManager=kubectl-annotate" -H "Content-Type: application/json-patch+json" --data "$d"
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
	ctx context.Context,
	cluster *v1beta1.PostgresCluster, instance *v1beta1.PostgresInstanceSetSpec,
) []string {
	version := fmt.Sprint(cluster.Spec.PostgresVersion)
	walDir := WALDirectory(cluster, instance)

	// If the user requests tablespaces, we want to make sure the directories exist with the
	// correct owner and permissions.
	tablespaceCmd := ""
	if feature.Enabled(ctx, feature.TablespaceVolumes) {
		// This command checks if a dir exists and if not, creates it;
		// if the dir does exist, then we `recreate` it to make sure the owner is correct;
		// if the dir exists with the wrong owner and is not writeable, we error.
		// This is the same behavior we use for the main PGDATA directory.
		// Note: Postgres requires the tablespace directory to be "an existing, empty directory
		// that is owned by the PostgreSQL operating system user."
		// - https://www.postgresql.org/docs/current/manage-ag-tablespaces.html
		// However, unlike the PGDATA directory, Postgres will _not_ error out
		// if the permissions are wrong on the tablespace directory.
		// Instead, when a tablespace is created in Postgres, Postgres will `chmod` the
		// tablespace directory to match permissions on the PGDATA directory (either 700 or 750).
		// Postgres setting the tablespace directory permissions:
		// - https://git.postgresql.org/gitweb/?p=postgresql.git;f=src/backend/commands/tablespace.c;hb=REL_14_0#l600
		// Postgres choosing directory permissions:
		// - https://git.postgresql.org/gitweb/?p=postgresql.git;f=src/common/file_perm.c;hb=REL_14_0#l27
		// Note: This permission change seems to happen only when the tablespace is created in Postgres.
		// If the user manually `chmod`'ed the directory after the creation of the tablespace, Postgres
		// would not attempt to change the directory permissions.
		// Note: as noted below, we mount the tablespace directory to the mountpoint `/tablespaces/NAME`,
		// and so we add the subdirectory `data` in order to set the permissions.
		checkInstallRecreateCmd := strings.Join([]string{
			`if [[ ! -e "${tablespace_dir}" || -O "${tablespace_dir}" ]]; then`,
			`install --directory --mode=0700 "${tablespace_dir}"`,
			`elif [[ -w "${tablespace_dir}" && -g "${tablespace_dir}" ]]; then`,
			`recreate "${tablespace_dir}" '0700'`,
			`else (halt Permissions!); fi ||`,
			`halt "$(permissions "${tablespace_dir}" ||:)"`,
		}, "\n")

		for _, tablespace := range instance.TablespaceVolumes {
			// The path for tablespaces volumes is /tablespaces/NAME/data
			// -- the `data` path is added so that we can arrange the permissions.
			tablespaceCmd = tablespaceCmd + "\ntablespace_dir=/tablespaces/" + tablespace.Name + "/data" + "\n" +
				checkInstallRecreateCmd
		}
	}

	pg_rewind_override := ""
	if config.FetchKeyCommand(&cluster.Spec) != "" {
		// Quoting "EOF" disables parameter substitution during write.
		// - https://tldp.org/LDP/abs/html/here-docs.html#EX71C
		pg_rewind_override = `cat << "EOF" > /tmp/pg_rewind_tde.sh
#!/bin/sh
pg_rewind -K "$(postgres -C encryption_key_command)" "$@"
EOF
chmod +x /tmp/pg_rewind_tde.sh
`
	}

	args := []string{version, walDir, naming.PGBackRestPGDataLogPath}
	script := strings.Join([]string{
		`declare -r expected_major_version="$1" pgwal_directory="$2" pgbrLog_directory="$3"`,

		// Function to print the permissions of a file or directory and its parents.
		bashPermissions,

		// Function to print a message to stderr then exit non-zero.
		bashHalt,

		// Function to log values in a basic structured format.
		`results() { printf '::postgres-operator: %s::%s\n' "$@"; }`,

		// Function to change the owner of an existing directory.
		strings.TrimSpace(bashRecreateDirectory),

		// Function to change a directory symlink while keeping the directory contents.
		strings.TrimSpace(bashSafeLink),

		// Log the effective user ID and all the group IDs.
		`echo Initializing ...`,
		`results 'uid' "$(id -u ||:)" 'gid' "$(id -G ||:)"`,

		// The pgbackrest spool path should be co-located with wal. If a wal volume exists, symlink the spool-path to it.
		`if [[ "${pgwal_directory}" == *"pgwal/"* ]] && [[ ! -d "/pgwal/pgbackrest-spool" ]];then rm -rf "/pgdata/pgbackrest-spool" && mkdir -p "/pgwal/pgbackrest-spool" && ln --force --symbolic "/pgwal/pgbackrest-spool" "/pgdata/pgbackrest-spool";fi`,
		// When a pgwal volume is removed, the symlink will be broken; force pgbackrest to recreate spool-path.
		`if [[ ! -e "/pgdata/pgbackrest-spool" ]];then rm -rf /pgdata/pgbackrest-spool;fi`,

		// Abort when the PostgreSQL version installed in the image does not
		// match the cluster spec.
		`results 'postgres path' "$(command -v postgres ||:)"`,
		`results 'postgres version' "${postgres_version:=$(postgres --version ||:)}"`,
		`[[ "${postgres_version}" =~ ") ${expected_major_version}"($|[^0-9]) ]] ||`,
		`halt Expected PostgreSQL version "${expected_major_version}"`,

		// Abort when the configured data directory is not $PGDATA.
		// - https://www.postgresql.org/docs/current/runtime-config-file-locations.html
		`results 'config directory' "${PGDATA:?}"`,
		`postgres_data_directory=$([[ -d "${PGDATA}" ]] && postgres -C data_directory || echo "${PGDATA}")`,
		`results 'data directory' "${postgres_data_directory}"`,
		`[[ "${postgres_data_directory}" == "${PGDATA}" ]] ||`,
		`halt Expected matching config and data directories`,

		// Determine if the data directory has been prepared for bootstrapping the cluster
		`bootstrap_dir="${postgres_data_directory}_bootstrap"`,
		`[[ -d "${bootstrap_dir}" ]] && results 'bootstrap directory' "${bootstrap_dir}"`,
		`[[ -d "${bootstrap_dir}" ]] && postgres_data_directory="${bootstrap_dir}"`,

		// PostgreSQL requires its directory to be writable by only itself.
		// Pod "securityContext.fsGroup" sets g+w on directories for *some*
		// storage providers. Ensure the current user owns the directory, and
		// remove group permissions.
		// - https://www.postgresql.org/docs/current/creating-cluster.html
		// - https://git.postgresql.org/gitweb/?p=postgresql.git;f=src/backend/postmaster/postmaster.c;hb=REL_10_0#l1522
		// - https://git.postgresql.org/gitweb/?p=postgresql.git;f=src/backend/utils/init/miscinit.c;hb=REL_14_0#l349
		// - https://issue.k8s.io/93802#issuecomment-717646167
		//
		// When the directory does not exist, create it with the correct permissions.
		// When the directory has the correct owner, set the correct permissions.
		`if [[ ! -e "${postgres_data_directory}" || -O "${postgres_data_directory}" ]]; then`,
		`install --directory --mode=0700 "${postgres_data_directory}"`,
		//
		// The directory exists but its owner is wrong. When it is writable,
		// the set-group-ID bit indicates that "fsGroup" probably ran on its
		// contents making them safe to use. In this case, we can make a new
		// directory (owned by this user) and refill it.
		`elif [[ -w "${postgres_data_directory}" && -g "${postgres_data_directory}" ]]; then`,
		`recreate "${postgres_data_directory}" '0700'`,
		//
		// The directory exists, its owner is wrong, and it is not writable.
		`else (halt Permissions!); fi ||`,
		`halt "$(permissions "${postgres_data_directory}" ||:)"`,

		// Create the pgBackRest log directory.
		`results 'pgBackRest log directory' "${pgbrLog_directory}"`,
		`install --directory --mode=0775 "${pgbrLog_directory}" ||`,
		`halt "$(permissions "${pgbrLog_directory}" ||:)"`,

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

		// Add the pg_rewind wrapper script, if TDE is enabled.
		pg_rewind_override,

		tablespaceCmd,
		// When the data directory is empty, there's nothing more to do.
		`[[ -f "${postgres_data_directory}/PG_VERSION" ]] || exit 0`,

		// Abort when the data directory is not empty and its version does not
		// match the cluster spec.
		`results 'data version' "${postgres_data_version:=$(< "${postgres_data_directory}/PG_VERSION")}"`,
		`[[ "${postgres_data_version}" == "${expected_major_version}" ]] ||`,
		`halt Expected PostgreSQL data version "${expected_major_version}"`,

		// For a restore from datasource:
		// Patroni will complain if there's no `postgresql.conf` file
		// and PGDATA may be missing that file if this is a restored database
		// where the conf file was kept elsewhere.
		`[[ ! -f "${postgres_data_directory}/postgresql.conf" ]] &&`,
		`touch "${postgres_data_directory}/postgresql.conf"`,

		// Safely move the WAL directory onto the intended volume. PostgreSQL
		// always writes WAL files in the "pg_wal" directory inside the data
		// directory. The recommended way to relocate it is with a symbolic
		// link. `initdb` and `pg_basebackup` have a `--waldir` flag that does
		// the same.
		// - https://www.postgresql.org/docs/current/wal-internals.html
		// - https://git.postgresql.org/gitweb/?p=postgresql.git;f=src/bin/initdb/initdb.c;hb=REL_13_0#l2718
		// - https://git.postgresql.org/gitweb/?p=postgresql.git;f=src/bin/pg_basebackup/pg_basebackup.c;hb=REL_13_0#l2621
		`safelink "${pgwal_directory}" "${postgres_data_directory}/pg_wal"`,
		`results 'wal directory' "$(realpath "${postgres_data_directory}/pg_wal" ||:)"`,

		// Early versions of PGO create replicas with a recovery signal file.
		// Patroni also creates a standby signal file before starting Postgres,
		// causing Postgres to remove only one, the standby. Remove the extra
		// signal file now, if it exists, and let Patroni manage the standby
		// signal file instead.
		// - https://git.postgresql.org/gitweb/?p=postgresql.git;f=src/backend/access/transam/xlog.c;hb=REL_12_0#l5318
		// TODO(cbandy): Remove this after 5.0 is EOL.
		`rm -f "${postgres_data_directory}/recovery.signal"`,
	}, "\n")

	return append([]string{"bash", "-ceu", "--", script, "startup"}, args...)
}
