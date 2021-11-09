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

package pgbackrest

import (
	"context"
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

const (
	// global pgBackRest default log path configuration, used by all three
	// default pod configurations
	defaultLogPath = "/tmp"

	// defaultRepo1Path stores the default pgBackRest repo path
	defaultRepo1Path = "/pgbackrest/"

	// DefaultStanzaName is the name of the default pgBackRest stanza
	DefaultStanzaName = "db"

	// configmap key references
	cmJobKey     = "pgbackrest_job.conf"
	cmPrimaryKey = "pgbackrest_primary.conf"

	// CMInstanceKey is the name of the pgBackRest configuration file for a PostgreSQL instance
	CMInstanceKey = "pgbackrest_instance.conf"

	// CMRepoKey is the name of the pgBackRest configuration file for a pgBackRest dedicated
	// repository host
	CMRepoKey = "pgbackrest_repo.conf"

	// ConfigDir is the pgBackRest configuration directory
	ConfigDir = "/etc/pgbackrest/conf.d"
	// ConfigHashKey is the name of the file storing the pgBackRest config hash
	ConfigHashKey = "config-hash"
	// ConfigVol is the name of the pgBackRest configuration volume
	ConfigVol = "pgbackrest-config"
	// configPath is the pgBackRest configuration file path
	configPath = "/etc/pgbackrest/pgbackrest.conf"

	// CMNameSuffix is the suffix used with postgrescluster name for associated configmap.
	// for instance, if the cluster is named 'mycluster', the
	// configmap will be named 'mycluster-pgbackrest-config'
	CMNameSuffix = "%s-pgbackrest-config"

	// repoMountPath is where to mount the pgBackRest repo volume.
	repoMountPath = "/pgbackrest"
)

// CreatePGBackRestConfigMapIntent creates a configmap struct with pgBackRest pgbackrest.conf settings in the data field.
// The keys within the data field correspond to the use of that configuration.
// pgbackrest_job.conf is used by certain jobs, such as stanza create and backup
// pgbackrest_primary.conf is used by the primary database pod
// pgbackrest_repo.conf is used by the pgBackRest repository pod
func CreatePGBackRestConfigMapIntent(postgresCluster *v1beta1.PostgresCluster,
	repoHostName, configHash, serviceName, serviceNamespace string,
	instanceNames []string) *corev1.ConfigMap {

	meta := naming.PGBackRestConfig(postgresCluster)
	meta.Annotations = naming.Merge(
		postgresCluster.Spec.Metadata.GetAnnotationsOrNil(),
		postgresCluster.Spec.Backups.PGBackRest.Metadata.GetAnnotationsOrNil())
	meta.Labels = naming.Merge(postgresCluster.Spec.Metadata.GetLabelsOrNil(),
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
	initialize.StringMap(&cm.Data)

	addDedicatedHost := DedicatedRepoHostEnabled(postgresCluster)
	pgdataDir := postgres.DataDirectory(postgresCluster)
	// Port will always be populated, since the API will set a default of 5432 if not provided
	pgPort := *postgresCluster.Spec.Port
	cm.Data[CMInstanceKey] = getConfigString(
		populatePGInstanceConfigurationMap(serviceName, serviceNamespace, repoHostName,
			pgdataDir, pgPort, postgresCluster.Spec.Backups.PGBackRest.Repos,
			postgresCluster.Spec.Backups.PGBackRest.Global))

	if addDedicatedHost && repoHostName != "" {
		cm.Data[CMRepoKey] = getConfigString(
			populateRepoHostConfigurationMap(serviceName, serviceNamespace,
				pgdataDir, pgPort, instanceNames,
				postgresCluster.Spec.Backups.PGBackRest.Repos,
				postgresCluster.Spec.Backups.PGBackRest.Global))
	}

	cm.Data[ConfigHashKey] = configHash

	return cm
}

// configVolumeAndMount creates a volume and mount configuration from the pgBackRest configmap to be used by the postgrescluster
func configVolumeAndMount(pgBackRestConfigMap *corev1.ConfigMap, pod *corev1.PodSpec, containerName, configKey string) {
	// Note: the 'container' string will be 'database' for the PostgreSQL database container,
	// otherwise it will be 'backrest'
	var (
		pgBackRestConfig []corev1.VolumeProjection
	)

	volume := corev1.Volume{Name: ConfigVol}
	volume.Projected = &corev1.ProjectedVolumeSource{}

	// Add our projections after those specified in the CR. Items later in the
	// list take precedence over earlier items (that is, last write wins).
	// - https://docs.openshift.com/container-platform/latest/nodes/containers/nodes-containers-projected-volumes.html
	// - https://kubernetes.io/docs/concepts/storage/volumes/#projected
	volume.Projected.Sources = append(
		pgBackRestConfig,
		corev1.VolumeProjection{
			ConfigMap: &corev1.ConfigMapProjection{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: pgBackRestConfigMap.Name,
				},
				Items: []corev1.KeyToPath{{
					Key:  configKey,
					Path: configPath,
				}},
			},
		},
	)

	mount := corev1.VolumeMount{
		Name:      volume.Name,
		MountPath: ConfigDir,
		ReadOnly:  true,
	}

	pod.Volumes = mergeVolumes(pod.Volumes, volume)

	container := findOrAppendContainer(&pod.Containers, containerName)

	container.VolumeMounts = mergeVolumeMounts(container.VolumeMounts, mount)
}

// PostgreSQLConfigVolumeAndMount creates a volume and mount configuration from the pgBackRest configmap to be used by the
// postgrescluster's PostgreSQL pod
func PostgreSQLConfigVolumeAndMount(pgBackRestConfigMap *corev1.ConfigMap, pod *corev1.PodSpec, containerName string) {
	configVolumeAndMount(pgBackRestConfigMap, pod, containerName, cmPrimaryKey)
}

// RepositoryConfigVolumeAndMount creates a volume and mount configuration from the pgBackRest configmap to be used by the
// postgrescluster's pgBackRest repo pod
func RepositoryConfigVolumeAndMount(pgBackRestConfigMap *corev1.ConfigMap, pod *corev1.PodSpec, containerName string) {
	configVolumeAndMount(pgBackRestConfigMap, pod, containerName, CMRepoKey)
}

// JobConfigVolumeAndMount creates a volume and mount configuration from the pgBackRest configmap to be used by the
// postgrescluster's job pods
func JobConfigVolumeAndMount(pgBackRestConfigMap *corev1.ConfigMap, pod *corev1.PodSpec, containerName string) {
	configVolumeAndMount(pgBackRestConfigMap, pod, containerName, cmJobKey)
}

// RestoreCommand returns the command for performing a pgBackRest restore.  In addition to calling
// the pgBackRest restore command with any pgBackRest options provided, the script also does the
// following:
// - Removes the patroni.dynamic.json file if present.  This ensures the configuration from the
//   cluster being restored from is not utilized when bootstrapping a new cluster, and the
//   configuration for the new cluster is utilized instead.
// - Starts the database and allows recovery to complete.  A temporary postgresql.conf file
//   with the minimum settings needed to safely start the database is created and utilized.
// - Renames the data directory as needed to bootstrap the cluster using the restored database.
//   This ensures compatibility with the "existing" bootstrap method that is included in the
//   Patroni config when bootstrapping a cluster using an existing data directory.
func RestoreCommand(pgdata string, args ...string) []string {

	// After pgBackRest restores files, PostgreSQL starts in recovery to finish
	// replaying WAL files. "hot_standby" is "on" (by default) so we can detect
	// when recovery has finished. In that mode, some parameters cannot be
	// smaller than they were when PostgreSQL was backed up. Configure them to
	// match the values reported by "pg_controldata". Those parameters are also
	// written to WAL files and may change during recovery. When they increase,
	// PostgreSQL exits and we reconfigure and restart it.
	// For PG14, when some parameters from WAL require a restart, the behavior is
	// to pause unless a restart is requested. For this edge case, we run a CASE
	// query to check
	// (a) if the instance is in recovery;
	// (b) if so, if the WAL replay is paused;
	// (c) if so, to unpause WAL replay, allowing our expected behavior to resume.
	// A note on the PostgreSQL code: we cast `pg_catalog.pg_wal_replay_resume()` as text
	// because that method returns a void (which is a non-NULL but empty result). When
	// that void is cast as a string, it is an ''
	// - https://www.postgresql.org/docs/current/hot-standby.html
	// - https://www.postgresql.org/docs/current/app-pgcontroldata.html

	const restoreScript = `declare -r pgdata="$1" opts="$2"
install --directory --mode=0700 "${pgdata}"
eval "pgbackrest restore ${opts}"
rm -f "${pgdata}/patroni.dynamic.json"
export PGDATA="${pgdata}" PGHOST='/tmp'

until [ "${recovery=}" = 'f' ]; do
if [ -z "${recovery}" ]; then
control=$(pg_controldata)
read -r max_conn <<< "${control##*max_connections setting:}"
read -r max_lock <<< "${control##*max_locks_per_xact setting:}"
read -r max_ptxn <<< "${control##*max_prepared_xacts setting:}"
read -r max_work <<< "${control##*max_worker_processes setting:}"
echo > /tmp/pg_hba.restore.conf 'local all "postgres" peer'
cat > /tmp/postgres.restore.conf <<EOF
archive_command = 'false'
archive_mode = 'on'
hba_file = '/tmp/pg_hba.restore.conf'
max_connections = '${max_conn}'
max_locks_per_transaction = '${max_lock}'
max_prepared_transactions = '${max_ptxn}'
max_worker_processes = '${max_work}'
unix_socket_directories = '/tmp'
EOF
if [ "$(< "${pgdata}/PG_VERSION")" -ge 12 ]; then
read -r max_wals <<< "${control##*max_wal_senders setting:}"
echo >> /tmp/postgres.restore.conf "max_wal_senders = '${max_wals}'"
fi

pg_ctl start --silent --wait --options='--config-file=/tmp/postgres.restore.conf'
fi

recovery=$(psql -Atc "SELECT CASE
  WHEN NOT pg_catalog.pg_is_in_recovery() THEN false
  WHEN NOT pg_catalog.pg_is_wal_replay_paused() THEN true
  ELSE pg_catalog.pg_wal_replay_resume()::text = ''
END recovery" && sleep 1) || true
done

pg_ctl stop --silent --wait
mv "${pgdata}" "${pgdata}_bootstrap"`

	return append([]string{"bash", "-ceu", "--", restoreScript, "-", pgdata}, args...)
}

// populatePGInstanceConfigurationMap returns a map representing the pgBackRest configuration for
// a PostgreSQL instance
func populatePGInstanceConfigurationMap(serviceName, serviceNamespace, repoHostName, pgdataDir string,
	pgPort int32, repos []v1beta1.PGBackRestRepo,
	globalConfig map[string]string) map[string]map[string]string {

	pgBackRestConfig := map[string]map[string]string{

		// will hold the [global] configs
		"global": {},
		// will hold the [stanza-name] configs
		"stanza": {},
	}

	// set the default stanza name
	pgBackRestConfig["stanza"]["name"] = DefaultStanzaName

	// set global settings, which includes all repos
	pgBackRestConfig["global"]["log-path"] = defaultLogPath
	for _, repo := range repos {

		repoConfigs := make(map[string]string)

		// repo volumes do not contain configuration (unlike other repo types which has actual
		// pgBackRest settings such as "bucket", "region", etc.), so only grab the name from the
		// repo if a Volume is detected, and don't attempt to get an configs
		if repo.Volume == nil {
			repoConfigs = getExternalRepoConfigs(repo)
		}

		// Only "volume" (i.e. PVC-based) repos should ever have a repo host configured.  This
		// means cloud-based repos (S3, GCS or Azure) should not have a repo host configured.
		if repoHostName != "" && repo.Volume != nil {
			pgBackRestConfig["global"][repo.Name+"-host"] = repoHostName + "-0." + serviceName +
				"." + serviceNamespace + ".svc." +
				naming.KubernetesClusterDomain(context.Background())
			pgBackRestConfig["global"][repo.Name+"-host-user"] = "postgres"
		}
		pgBackRestConfig["global"][repo.Name+"-path"] = defaultRepo1Path + repo.Name

		for option, val := range repoConfigs {
			pgBackRestConfig["global"][option] = val
		}
	}

	for option, val := range globalConfig {
		pgBackRestConfig["global"][option] = val
	}

	// Now add the local PG instance to the stanza section. The local PG host must always be
	// index 1: https://github.com/pgbackrest/pgbackrest/issues/1197#issuecomment-708381800
	pgBackRestConfig["stanza"]["pg1-path"] = pgdataDir
	pgBackRestConfig["stanza"]["pg1-port"] = fmt.Sprint(pgPort)
	pgBackRestConfig["stanza"]["pg1-socket-path"] = postgres.SocketDirectory

	return pgBackRestConfig
}

// populateRepoHostConfigurationMap returns a map representing the pgBackRest configuration for
// a pgBackRest dedicated repository host
func populateRepoHostConfigurationMap(serviceName, serviceNamespace, pgdataDir string,
	pgPort int32, pgHosts []string, repos []v1beta1.PGBackRestRepo,
	globalConfig map[string]string) map[string]map[string]string {

	pgBackRestConfig := map[string]map[string]string{

		// will hold the [global] configs
		"global": {},
		// will hold the [stanza-name] configs
		"stanza": {},
	}

	// set the default stanza name
	pgBackRestConfig["stanza"]["name"] = DefaultStanzaName

	// set the config for the local repo host
	pgBackRestConfig["global"]["log-path"] = defaultLogPath
	for _, repo := range repos {
		var repoConfigs map[string]string

		// repo volumes do not contain configuration (unlike other repo types which has actual
		// pgBackRest settings such as "bucket", "region", etc.), so only grab the name from the
		// repo if a Volume is detected, and don't attempt to get an configs
		if repo.Volume == nil {
			repoConfigs = getExternalRepoConfigs(repo)
		}
		pgBackRestConfig["global"][repo.Name+"-path"] = defaultRepo1Path + repo.Name

		for option, val := range repoConfigs {
			pgBackRestConfig["global"][option] = val
		}
	}

	for option, val := range globalConfig {
		pgBackRestConfig["global"][option] = val
	}

	// set the configs for all PG hosts
	for i, pgHost := range pgHosts {
		pgBackRestConfig["stanza"][fmt.Sprintf("pg%d-host", i+1)] = pgHost + "-0." + serviceName +
			"." + serviceNamespace + ".svc." +
			naming.KubernetesClusterDomain(context.Background())
		pgBackRestConfig["stanza"][fmt.Sprintf("pg%d-path", i+1)] = pgdataDir
		pgBackRestConfig["stanza"][fmt.Sprintf("pg%d-port", i+1)] = fmt.Sprint(pgPort)
		pgBackRestConfig["stanza"][fmt.Sprintf("pg%d-socket-path", i+1)] = postgres.SocketDirectory
	}

	return pgBackRestConfig
}

// getConfigString provides a formatted string of the desired
// pgBackRest configuration for insertion into the relevant
// configmap
func getConfigString(c map[string]map[string]string) string {

	configString := fmt.Sprintln("[global]")
	for _, k := range sortedKeys(c["global"]) {
		configString += fmt.Sprintf("%s=%s\n", k, c["global"][k])
	}

	if c["stanza"]["name"] != "" {
		configString += fmt.Sprintf("\n[%s]\n", c["stanza"]["name"])

		for _, k := range sortedKeys(c["stanza"]) {
			if k != "name" {
				configString += fmt.Sprintf("%s=%s\n", k, c["stanza"][k])
			}
		}
	}
	return configString
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

// sortedKeys sorts and returns the keys from a given map
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return keys
}
