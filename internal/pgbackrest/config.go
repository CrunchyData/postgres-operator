// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgbackrest

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/internal/collector"
	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/internal/shell"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

const (
	// defaultRepo1Path stores the default pgBackRest repo path
	defaultRepo1Path = "/pgbackrest/"

	// DefaultStanzaName is the name of the default pgBackRest stanza
	DefaultStanzaName = "db"

	// CMInstanceKey is the name of the pgBackRest configuration file for a PostgreSQL instance
	CMInstanceKey = "pgbackrest_instance.conf"

	// CMRepoKey is the name of the pgBackRest configuration file for a pgBackRest dedicated
	// repository host
	CMRepoKey = "pgbackrest_repo.conf"

	// configDirectory is the pgBackRest configuration directory.
	configDirectory = "/etc/pgbackrest/conf.d"

	// ConfigHashKey is the name of the file storing the pgBackRest config hash
	ConfigHashKey = "config-hash"

	// repoMountPath is where to mount the pgBackRest repo volume.
	repoMountPath = "/pgbackrest"

	serverConfigAbsolutePath   = configDirectory + "/" + serverConfigProjectionPath
	serverConfigProjectionPath = "~postgres-operator_server.conf"

	serverConfigMapKey = "pgbackrest-server.conf"

	// serverMountPath is the directory containing the TLS server certificate
	// and key. This is outside of configDirectory so the hash calculated by
	// backup jobs does not change when the primary changes.
	serverMountPath = "/etc/pgbackrest/server"
)

const (
	iniGeneratedWarning = "" +
		"# Generated by postgres-operator. DO NOT EDIT.\n" +
		"# Your changes will not be saved.\n"
)

// CreatePGBackRestConfigMapIntent creates a configmap struct with pgBackRest pgbackrest.conf settings in the data field.
// The keys within the data field correspond to the use of that configuration.
// pgbackrest_job.conf is used by certain jobs, such as stanza create and backup
// pgbackrest_primary.conf is used by the primary database pod
// pgbackrest_repo.conf is used by the pgBackRest repository pod
func CreatePGBackRestConfigMapIntent(ctx context.Context, postgresCluster *v1beta1.PostgresCluster,
	repoHostName, configHash, serviceName, serviceNamespace string,
	instanceNames []string) (*corev1.ConfigMap, error) {

	var err error

	meta := naming.PGBackRestConfig(postgresCluster)
	meta.Annotations = naming.Merge(
		postgresCluster.Spec.Metadata.GetAnnotationsOrNil(),
		postgresCluster.Spec.Backups.PGBackRest.Metadata.GetAnnotationsOrNil())
	meta.Labels = naming.Merge(
		postgresCluster.Spec.Metadata.GetLabelsOrNil(),
		postgresCluster.Spec.Backups.PGBackRest.Metadata.GetLabelsOrNil(),
		naming.PGBackRestConfigLabels(postgresCluster.GetName()),
	)

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: meta,
	}

	// create an empty map for the config data
	initialize.Map(&cm.Data)

	pgdataDir := postgres.DataDirectory(postgresCluster)
	// Port will always be populated, since the API will set a default of 5432 if not provided
	pgPort := *postgresCluster.Spec.Port
	cm.Data[CMInstanceKey] = iniGeneratedWarning +
		populatePGInstanceConfigurationMap(
			serviceName, serviceNamespace, repoHostName, pgdataDir,
			config.FetchKeyCommand(&postgresCluster.Spec),
			strconv.Itoa(postgresCluster.Spec.PostgresVersion),
			pgPort, postgresCluster.Spec.Backups.PGBackRest.Repos,
			postgresCluster.Spec.Backups.PGBackRest.Global,
		).String()

	// PostgreSQL instances that have not rolled out expect to mount a server
	// config file. Always populate that file so those volumes stay valid and
	// Kubernetes propagates their contents to those pods. The repo host name
	// given below should always be set, but this guards for cases when it might
	// not be.
	cm.Data[serverConfigMapKey] = ""

	if repoHostName != "" {
		cm.Data[serverConfigMapKey] = iniGeneratedWarning +
			serverConfig(postgresCluster).String()

		cm.Data[CMRepoKey] = iniGeneratedWarning +
			populateRepoHostConfigurationMap(
				serviceName, serviceNamespace,
				pgdataDir, config.FetchKeyCommand(&postgresCluster.Spec),
				strconv.Itoa(postgresCluster.Spec.PostgresVersion),
				pgPort, instanceNames,
				postgresCluster.Spec.Backups.PGBackRest.Repos,
				postgresCluster.Spec.Backups.PGBackRest.Global,
			).String()

		if RepoHostVolumeDefined(postgresCluster) &&
			(feature.Enabled(ctx, feature.OpenTelemetryLogs) ||
				feature.Enabled(ctx, feature.OpenTelemetryMetrics)) {
			err = collector.AddToConfigMap(ctx, collector.NewConfigForPgBackrestRepoHostPod(
				ctx,
				postgresCluster.Spec.Instrumentation,
				postgresCluster.Spec.Backups.PGBackRest.Repos,
			), cm)

			// If OTel logging is enabled, add logrotate config for the RepoHost
			if err == nil &&
				postgresCluster.Spec.Instrumentation != nil &&
				feature.Enabled(ctx, feature.OpenTelemetryLogs) {
				var pgBackRestLogPath string
				for _, repo := range postgresCluster.Spec.Backups.PGBackRest.Repos {
					if repo.Volume != nil {
						pgBackRestLogPath = fmt.Sprintf(naming.PGBackRestRepoLogPath, repo.Name)
						break
					}
				}

				collector.AddLogrotateConfigs(ctx, postgresCluster.Spec.Instrumentation, cm, []collector.LogrotateConfig{{
					LogFiles: []string{pgBackRestLogPath + "/*.log"},
				}})
			}
		}
	}

	cm.Data[ConfigHashKey] = configHash

	return cm, err
}

// MakePGBackrestLogDir creates the pgBackRest default log path directory used when a
// dedicated repo host is configured.
func MakePGBackrestLogDir(template *corev1.PodTemplateSpec,
	cluster *v1beta1.PostgresCluster) string {

	var pgBackRestLogPath string
	for _, repo := range cluster.Spec.Backups.PGBackRest.Repos {
		if repo.Volume != nil {
			pgBackRestLogPath = fmt.Sprintf(naming.PGBackRestRepoLogPath, repo.Name)
			break
		}
	}

	container := corev1.Container{
		// TODO(log-rotation): The second argument here should be the path
		// of the volume mount. Find a way to calculate that consistently.
		Command:         []string{"bash", "-c", shell.MakeDirectories(0o775, path.Dir(pgBackRestLogPath), pgBackRestLogPath)},
		Image:           config.PGBackRestContainerImage(cluster),
		ImagePullPolicy: cluster.Spec.ImagePullPolicy,
		Name:            naming.ContainerPGBackRestLogDirInit,
		SecurityContext: initialize.RestrictedSecurityContext(),
	}

	// Set the container resources to the 'pgbackrest' container configuration.
	for i, c := range template.Spec.Containers {
		if c.Name == naming.PGBackRestRepoContainerName {
			container.Resources = template.Spec.Containers[i].Resources
			break
		}
	}
	template.Spec.InitContainers = append(template.Spec.InitContainers, container)

	return pgBackRestLogPath
}

// RestoreCommand returns the command for performing a pgBackRest restore.  In addition to calling
// the pgBackRest restore command with any pgBackRest options provided, the script also does the
// following:
//   - Removes the patroni.dynamic.json file if present.  This ensures the configuration from the
//     cluster being restored from is not utilized when bootstrapping a new cluster, and the
//     configuration for the new cluster is utilized instead.
//   - Starts the database and allows recovery to complete.  A temporary postgresql.conf file
//     with the minimum settings needed to safely start the database is created and utilized.
//   - Renames the data directory as needed to bootstrap the cluster using the restored database.
//     This ensures compatibility with the "existing" bootstrap method that is included in the
//     Patroni config when bootstrapping a cluster using an existing data directory.
func RestoreCommand(pgdata, hugePagesSetting, fetchKeyCommand string, _ []*corev1.PersistentVolumeClaim, args ...string) []string {
	ps := postgres.NewParameterSet()
	ps.Add("data_directory", pgdata)
	ps.Add("huge_pages", hugePagesSetting)

	// Keep history and WAL files until the cluster starts with its normal
	// archiving enabled.
	ps.Add("archive_command", "false -- store WAL files locally for now")
	ps.Add("archive_mode", "on")

	// Enable "hot_standby" so we can connect to Postgres and observe its
	// progress during recovery.
	ps.Add("hot_standby", "on")

	if fetchKeyCommand != "" {
		ps.Add("encryption_key_command", fetchKeyCommand)
	}

	configure := strings.Join([]string{
		// With "hot_standby" on, some parameters cannot be smaller than they were
		// when Postgres was backed up. Configure these to match values reported by
		// "pg_controldata" before starting Postgres. These parameters are also
		// written to WAL files and may change during recovery. When they increase,
		// Postgres exits and we reconfigure it here.
		// - https://www.postgresql.org/docs/current/app-pgcontroldata.html
		`control=$(LC_ALL=C pg_controldata)`,
		`read -r max_conn <<< "${control##*max_connections setting:}"`,
		`read -r max_lock <<< "${control##*max_locks_per_xact setting:}"`,
		`read -r max_ptxn <<< "${control##*max_prepared_xacts setting:}"`,
		`read -r max_work <<< "${control##*max_worker_processes setting:}"`,

		// During recovery, only allow connections over the the domain socket.
		`echo > /tmp/pg_hba.restore.conf 'local all "postgres" peer'`,

		// Combine parameters from Go with those detected in Bash.
		`cat >  /tmp/postgres.restore.conf <<'EOF'`, ps.String(), `EOF`,
		`cat >> /tmp/postgres.restore.conf <<EOF`,
		`hba_file = '/tmp/pg_hba.restore.conf'`,
		`max_connections = '${max_conn}'`,
		`max_locks_per_transaction = '${max_lock}'`,
		`max_prepared_transactions = '${max_ptxn}'`,
		`max_worker_processes = '${max_work}'`,
		`EOF`,

		`version=$(< "${PGDATA}/PG_VERSION")`,

		// PostgreSQL v12 introduced the "max_wal_senders" parameter.
		`if [[ "${version}" -ge 12 ]]; then`,
		`read -r max_wals <<< "${control##*max_wal_senders setting:}"`,
		`echo >> /tmp/postgres.restore.conf "max_wal_senders = '${max_wals}'"`,
		`fi`,

		// TODO(sockets): PostgreSQL v14 is able to connect over abstract sockets in the network namespace.
		`PGHOST=$([[ "${version}" -ge 14 ]] && echo '/tmp' || echo '/tmp')`,
		`echo >> /tmp/postgres.restore.conf "unix_socket_directories = '${PGHOST}'"`,
	}, "\n")

	script := strings.Join([]string{
		`declare -r PGDATA="$1" opts="$2"; export PGDATA PGHOST`,

		// Remove any "postmaster.pid" file leftover from a prior failure.
		`rm -f "${PGDATA}/postmaster.pid"`,

		// Run the restore and print its arguments.
		`bash -xc "pgbackrest restore ${opts}"`,

		// Ignore any Patroni settings present in the backup.
		`rm -f "${PGDATA}/patroni.dynamic.json"`,

		// By default, pg_ctl waits 60 seconds for Postgres to stop or start.
		// We want to be certain when Postgres is running or not, so we use
		// a very large timeout (365 days) to effectively wait forever. With
		// this, the result of "pg_ctl --wait" indicates the state of Postgres.
		// - https://www.postgresql.org/docs/current/app-pg-ctl.html
		fmt.Sprintf(`export PGCTLTIMEOUT=%d`, 365*24*time.Hour/time.Second),

		// Configure and start Postgres until we can see that it has finished
		// replaying WAL.
		//
		// PostgreSQL v13 and earlier exit when they need reconfiguration with
		// "hot_standby" on. This can cause pg_ctl to fail, so we compare the
		// LSN from before and after calling it. If the LSN changed, Postgres
		// ran and was able to replay WAL before exiting. In that case, configure
		// Postgres and start it again to see if it can make more progress.
		//
		// If Postgres exits after pg_ctl succeeds, psql returns nothing which
		// resets the "recovering" variable. Configure Postgres and start it again.
		`until [[ "${recovering=}" == 'f' ]]; do`,
		`  if [[ -z "${recovering}" ]]; then`, configure,
		`    read -r stopped <<< "${control##*recovery ending location:}"`,
		`    pg_ctl start --silent --wait --options='-c config_file=/tmp/postgres.restore.conf' || failed=$?`,
		`    [[ "${started-}" == "${stopped}" && -n "${failed-}" ]] && exit "${failed}"`,
		`    started="${stopped}" && [[ -n "${failed-}" ]] && failed= && continue`,
		`  fi`,
		// Ask Postgres if it is still recovering. PostgreSQL v14 pauses when it
		// needs reconfiguration with "hot_standby" on, and resuming replay causes
		// it to exit like prior versions.
		// - https://www.postgresql.org/docs/current/hot-standby.html
		//
		// NOTE: "pg_wal_replay_resume()" returns void which cannot be compared to
		// null. Instead, cast it to text and compare that for a boolean result.
		`  recovering=$(psql -Atc "SELECT CASE`,
		`    WHEN NOT pg_catalog.pg_is_in_recovery() THEN false`,
		`    WHEN NOT pg_catalog.pg_is_wal_replay_paused() THEN true`,
		`    ELSE pg_catalog.pg_wal_replay_resume()::text = ''`,
		`  END" && sleep 1) ||:`,
		`done`,

		// Replay is done. Stop Postgres gracefully and move the data directory
		// into position for our Patroni bootstrap method.
		`pg_ctl stop --silent --wait`,
		`mv "${PGDATA}" "${PGDATA}_bootstrap"`,
	}, "\n")

	return append([]string{"bash", "-ceu", "--", script, "-", pgdata}, args...)
}

// DedicatedSnapshotVolumeRestoreCommand returns the command for performing a pgBackRest delta restore
// into a dedicated snapshot volume. In addition to calling the pgBackRest restore command with any
// pgBackRest options provided, the script also removes the patroni.dynamic.json file if present. This
// ensures the configuration from the cluster being restored from is not utilized when bootstrapping a
// new cluster, and the configuration for the new cluster is utilized instead.
func DedicatedSnapshotVolumeRestoreCommand(pgdata string, args ...string) []string {

	// The postmaster.pid file is removed, if it exists, before attempting a restore.
	// This allows the restore to be tried more than once without the causing an
	// error due to the presence of the file in subsequent attempts.

	// Wrap pgbackrest restore command in backup_label checks. If pre/post
	// backup_labels are different, restore moved database forward, so return 0
	// so that the Job is successful and we know to proceed with snapshot.
	// Otherwise return 1, Job will fail, and we will not proceed with snapshot.
	restoreScript := `declare -r pgdata="$1" opts="$2"
BACKUP_LABEL=$([[ ! -e "${pgdata}/backup_label" ]] || md5sum "${pgdata}/backup_label")
echo "Starting pgBackRest delta restore"

install --directory --mode=0700 "${pgdata}"
rm -f "${pgdata}/postmaster.pid"
bash -xc "pgbackrest restore ${opts}"
rm -f "${pgdata}/patroni.dynamic.json"

BACKUP_LABEL_POST=$([[ ! -e "${pgdata}/backup_label" ]] || md5sum "${pgdata}/backup_label")
if [[ "${BACKUP_LABEL}" != "${BACKUP_LABEL_POST}" ]]
then
  exit 0
fi
echo Database was not advanced by restore. No snapshot will be taken.
echo Check that your last backup was successful.
exit 1`

	return append([]string{"bash", "-ceu", "--", restoreScript, "-", pgdata}, args...)
}

// populatePGInstanceConfigurationMap returns options representing the pgBackRest configuration for
// a PostgreSQL instance
func populatePGInstanceConfigurationMap(
	serviceName, serviceNamespace, repoHostName, pgdataDir,
	fetchKeyCommand, postgresVersion string,
	pgPort int32, repos []v1beta1.PGBackRestRepo,
	globalConfig map[string]string,
) iniSectionSet {

	// TODO(cbandy): pass a FQDN in already.
	repoHostFQDN := repoHostName + "-0." +
		serviceName + "." + serviceNamespace + ".svc." +
		naming.KubernetesClusterDomain(context.Background())

	global := iniMultiSet{}
	stanza := iniMultiSet{}

	// For faster and more robust WAL archiving, we turn on pgBackRest archive-async.
	global.Set("archive-async", "y")
	// pgBackRest spool-path should always be co-located with the Postgres WAL path.
	global.Set("spool-path", "/pgdata/pgbackrest-spool")
	// pgBackRest will log to the pgData volume for commands run on the PostgreSQL instance
	global.Set("log-path", naming.PGBackRestPGDataLogPath)

	for _, repo := range repos {
		global.Set(repo.Name+"-path", defaultRepo1Path+repo.Name)

		// repo volumes do not contain configuration (unlike other repo types which has actual
		// pgBackRest settings such as "bucket", "region", etc.), so only grab the name from the
		// repo if a Volume is detected, and don't attempt to get an configs
		if repo.Volume == nil {
			for option, val := range getExternalRepoConfigs(repo) {
				global.Set(option, val)
			}
		}

		// Only "volume" (i.e. PVC-based) repos should ever have a repo host configured.  This
		// means cloud-based repos (S3, GCS or Azure) should not have a repo host configured.
		if repoHostName != "" && repo.Volume != nil {
			global.Set(repo.Name+"-host", repoHostFQDN)
			global.Set(repo.Name+"-host-type", "tls")
			global.Set(repo.Name+"-host-ca-file", certAuthorityAbsolutePath)
			global.Set(repo.Name+"-host-cert-file", certClientAbsolutePath)
			global.Set(repo.Name+"-host-key-file", certClientPrivateKeyAbsolutePath)
			global.Set(repo.Name+"-host-user", "postgres")
		}
	}

	for option, val := range globalConfig {
		global.Set(option, val)
	}

	// Now add the local PG instance to the stanza section. The local PG host must always be
	// index 1: https://github.com/pgbackrest/pgbackrest/issues/1197#issuecomment-708381800
	stanza.Set("pg1-path", pgdataDir)
	stanza.Set("pg1-port", fmt.Sprint(pgPort))
	stanza.Set("pg1-socket-path", postgres.SocketDirectory)

	if fetchKeyCommand != "" {
		stanza.Set("archive-header-check", "n")
		stanza.Set("page-header-check", "n")
		stanza.Set("pg-version-force", postgresVersion)
	}

	return iniSectionSet{
		"global":          global,
		DefaultStanzaName: stanza,
	}
}

// populateRepoHostConfigurationMap returns options representing the pgBackRest configuration for
// a pgBackRest dedicated repository host
func populateRepoHostConfigurationMap(
	serviceName, serviceNamespace, pgdataDir,
	fetchKeyCommand, postgresVersion string,
	pgPort int32, pgHosts []string, repos []v1beta1.PGBackRestRepo,
	globalConfig map[string]string,
) iniSectionSet {

	global := iniMultiSet{}
	stanza := iniMultiSet{}

	var pgBackRestLogPathSet bool
	for _, repo := range repos {
		global.Set(repo.Name+"-path", defaultRepo1Path+repo.Name)

		// repo volumes do not contain configuration (unlike other repo types which has actual
		// pgBackRest settings such as "bucket", "region", etc.), so only grab the name from the
		// repo if a Volume is detected, and don't attempt to get an configs
		if repo.Volume == nil {
			for option, val := range getExternalRepoConfigs(repo) {
				global.Set(option, val)
			}
		}

		if !pgBackRestLogPathSet && repo.Volume != nil {
			// pgBackRest will log to the first configured repo volume when commands
			// are run on the pgBackRest repo host. With our previous check in
			// RepoHostVolumeDefined(), we've already validated that at least one
			// defined repo has a volume.
			global.Set("log-path", fmt.Sprintf(naming.PGBackRestRepoLogPath, repo.Name))
			pgBackRestLogPathSet = true
		}
	}

	// If no log path was set, don't log because the default path is not writable.
	if !pgBackRestLogPathSet {
		global.Set("log-level-file", "off")
	}

	for option, val := range globalConfig {
		global.Set(option, val)
	}

	// set the configs for all PG hosts
	for i, pgHost := range pgHosts {
		// TODO(cbandy): pass a FQDN in already.
		pgHostFQDN := pgHost + "-0." +
			serviceName + "." + serviceNamespace + ".svc." +
			naming.KubernetesClusterDomain(context.Background())

		stanza.Set(fmt.Sprintf("pg%d-host", i+1), pgHostFQDN)
		stanza.Set(fmt.Sprintf("pg%d-host-type", i+1), "tls")
		stanza.Set(fmt.Sprintf("pg%d-host-ca-file", i+1), certAuthorityAbsolutePath)
		stanza.Set(fmt.Sprintf("pg%d-host-cert-file", i+1), certClientAbsolutePath)
		stanza.Set(fmt.Sprintf("pg%d-host-key-file", i+1), certClientPrivateKeyAbsolutePath)

		stanza.Set(fmt.Sprintf("pg%d-path", i+1), pgdataDir)
		stanza.Set(fmt.Sprintf("pg%d-port", i+1), fmt.Sprint(pgPort))
		stanza.Set(fmt.Sprintf("pg%d-socket-path", i+1), postgres.SocketDirectory)

		if fetchKeyCommand != "" {
			stanza.Set("archive-header-check", "n")
			stanza.Set("page-header-check", "n")
			stanza.Set("pg-version-force", postgresVersion)
		}
	}

	return iniSectionSet{
		"global":          global,
		DefaultStanzaName: stanza,
	}
}

// getExternalRepoConfigs returns a map containing the configuration settings for an external
// pgBackRest repository as defined in the PostgresCluster spec
func getExternalRepoConfigs(repo v1beta1.PGBackRestRepo) map[string]string {

	repoConfigs := make(map[string]string)

	if repo.Azure != nil {
		repoConfigs[repo.Name+"-type"] = "azure"
		repoConfigs[repo.Name+"-azure-container"] = repo.Azure.Container
	} else if repo.GCS != nil {
		repoConfigs[repo.Name+"-type"] = "gcs"
		repoConfigs[repo.Name+"-gcs-bucket"] = repo.GCS.Bucket
	} else if repo.S3 != nil {
		repoConfigs[repo.Name+"-type"] = "s3"
		repoConfigs[repo.Name+"-s3-bucket"] = repo.S3.Bucket
		repoConfigs[repo.Name+"-s3-endpoint"] = repo.S3.Endpoint
		repoConfigs[repo.Name+"-s3-region"] = repo.S3.Region
	}

	return repoConfigs
}

// reloadCommand returns an entrypoint that convinces the pgBackRest TLS server
// to reload its options and certificate files when they change. The process
// will appear as name in `ps` and `top`.
func reloadCommand(name string) []string {
	// Use a Bash loop to periodically check the mtime of the mounted server
	// volume and configuration file. When either changes, signal pgBackRest
	// and print the observed timestamp.
	//
	// We send SIGHUP because this allows the TLS server configuration to be
	// reloaded starting in pgBackRest 2.37. We filter by parent process to ignore
	// the forked connection handlers. The server parent process is zero because
	// it is started by Kubernetes.
	// - https://github.com/pgbackrest/pgbackrest/commit/7b3ea883c7c010aafbeb14d150d073a113b703e4

	// Coreutils `sleep` uses a lot of memory, so the following opens a file
	// descriptor and uses the timeout of the builtin `read` to wait. That same
	// descriptor gets closed and reopened to use the builtin `[ -nt` to check
	// mtimes.
	// - https://unix.stackexchange.com/a/407383
	const script = `
exec {fd}<> <(:||:)
until read -r -t 5 -u "${fd}"; do
  if
    [[ "${filename}" -nt "/proc/self/fd/${fd}" ]] &&
    pkill -HUP --exact --parent=0 pgbackrest
  then
    exec {fd}>&- && exec {fd}<> <(:||:)
    stat --dereference --format='Loaded configuration dated %y' "${filename}"
  elif
    { [[ "${directory}" -nt "/proc/self/fd/${fd}" ]] ||
      [[ "${authority}" -nt "/proc/self/fd/${fd}" ]]
    } &&
    pkill -HUP --exact --parent=0 pgbackrest
  then
    exec {fd}>&- && exec {fd}<> <(:||:)
    stat --format='Loaded certificates dated %y' "${directory}"
  fi
done
`

	// Elide the above script from `ps` and `top` by wrapping it in a function
	// and calling that.
	wrapper := `monitor() {` + script + `};` +
		` export directory="$1" authority="$2" filename="$3"; export -f monitor;` +
		` exec -a "$0" bash -ceu monitor`

	return []string{"bash", "-ceu", "--", wrapper, name,
		serverMountPath, certAuthorityAbsolutePath, serverConfigAbsolutePath}
}

// serverConfig returns the options needed to run the TLS server for cluster.
func serverConfig(cluster *v1beta1.PostgresCluster) iniSectionSet {
	global := iniMultiSet{}
	server := iniMultiSet{}

	// IPv6 support is a relatively recent addition to Kubernetes, so listen on
	// the IPv4 wildcard address and trust that Pod DNS names will resolve to
	// IPv4 addresses for now.
	//
	// NOTE(cbandy): The unspecified IPv6 address, which ends up being the IPv6
	// wildcard address, did not work in all environments. In some cases, the
	// "server-ping" command would not connect.
	// - https://tools.ietf.org/html/rfc3493#section-3.8
	//
	// TODO(cbandy): When pgBackRest provides a way to bind to all addresses,
	// use that here and configure "server-ping" to use "localhost" which
	// Kubernetes guarantees resolves to a loopback address.
	// - https://kubernetes.io/docs/concepts/cluster-administration/networking/
	// - https://releases.k8s.io/v1.18.0/pkg/kubelet/kubelet_pods.go#L327
	// - https://releases.k8s.io/v1.23.0/pkg/kubelet/kubelet_pods.go#L345
	global.Set("tls-server-address", "0.0.0.0")

	// NOTE (dsessler7): As pointed out by Chris above, there is an issue in
	// pgBackRest (#1841), where using a wildcard address to bind all addresses
	// does not work in certain IPv6 environments. Until this is fixed, we are
	// going to workaround the issue by allowing the user to add an annotation to
	// enable IPv6. We will check for that annotation here and override the
	// "tls-server-address" setting accordingly.
	if strings.EqualFold(cluster.Annotations[naming.PGBackRestIPVersion], "ipv6") {
		global.Set("tls-server-address", "::")
	}

	// The client certificate for this cluster is allowed to connect for any stanza.
	// Without the wildcard "*", the "pgbackrest info" and "pgbackrest repo-ls"
	// commands fail with "access denied" when invoked without a "--stanza" flag.
	global.Add("tls-server-auth", clientCommonName(cluster)+"=*")

	global.Set("tls-server-ca-file", certAuthorityAbsolutePath)
	global.Set("tls-server-cert-file", certServerAbsolutePath)
	global.Set("tls-server-key-file", certServerPrivateKeyAbsolutePath)

	// Send all server logs to stderr and stdout without timestamps.
	// - stderr has ERROR messages
	// - stdout has WARN, INFO, and DETAIL messages
	//
	// The "trace" level shows when a connection is accepted, but nothing about
	// the remote address or what commands it might send.
	// - https://github.com/pgbackrest/pgbackrest/blob/release/2.38/src/command/server/server.c#L158-L159
	// - https://pgbackrest.org/configuration.html#section-log
	server.Set("log-level-console", "detail")
	server.Set("log-level-stderr", "error")
	server.Set("log-level-file", "off")
	server.Set("log-timestamp", "n")

	return iniSectionSet{
		"global":        global,
		"global:server": server,
	}
}
