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
	"fmt"
	"sort"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// global pgBackRest default log path configuration, used by all three
	// default pod configurations
	defaultLogPath = "/tmp"

	// defaultRepo1Path stores the default pgBackRest repo path
	defaultRepo1Path = "/pgbackrest/"

	// stanza pgBackRest default configurations
	defaultStanzaName = "db"

	// default pgBackRest port and socket path
	defaultPG1Port       = "5432"
	defaultPG1SocketPath = "/tmp"

	// configmap key references
	cmJobKey     = "pgbackrest_job.conf"
	cmPrimaryKey = "pgbackrest_primary.conf"
	// CMRepoKey is the name of the configuration file for a pgBackRest  deidicated repository host
	CMRepoKey = "pgbackrest_repo.conf"

	// ConfigDir is the pgBackRest configuration directory
	ConfigDir = "/etc/pgbackrest/conf.d"
	// ConfigVol is the name of the pgBackRest configuration volume
	ConfigVol = "pgbackrest-config"
	// configPath is the pgBackRest configuration file path
	configPath = "/etc/pgbackrest/pgbackrest.conf"

	// CMNameSuffix is the suffix used with postgrescluster name for associated configmap.
	// for instance, if the cluster is named 'mycluster', the
	// configmap will be named 'mycluster-pgbackrest-config'
	CMNameSuffix = "%s-pgbackrest-config"
)

// CreatePGBackRestConfigMapIntent creates a configmap struct with pgBackRest pgbackrest.conf settings in the data field.
// The keys within the data field correspond to the use of that configuration.
// pgbackrest_job.conf is used by certain jobs, such as stanza create and backup
// pgbackrest_primary.conf is used by the primary database pod
// pgbackrest_repo.conf is used by the pgBackRest repository pod
func CreatePGBackRestConfigMapIntent(postgresCluster *v1beta1.PostgresCluster,
	repoHostName string, instanceNames []string) *v1.ConfigMap {

	meta := naming.PGBackRestConfig(postgresCluster)
	meta.Labels = naming.PGBackRestConfigLabels(postgresCluster.GetName())

	cm := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: meta,
	}

	// create an empty map for the config data
	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}

	var serviceName string
	addInstanceHosts := (postgresCluster.Spec.Archive.PGBackRest.RepoHost != nil) &&
		(postgresCluster.Spec.Archive.PGBackRest.RepoHost.Dedicated == nil)
	addDedicatedHost := (postgresCluster.Spec.Archive.PGBackRest.RepoHost != nil) &&
		(postgresCluster.Spec.Archive.PGBackRest.RepoHost.Dedicated != nil)
	if addInstanceHosts || addDedicatedHost {
		serviceName = naming.ClusterPodService(postgresCluster).Name
	}

	pgdataDir := naming.GetPGDATADirectory(postgresCluster)
	for i, name := range instanceNames {
		otherInstances := make([]string, 0)
		if addInstanceHosts {
			otherInstances = make([]string, len(instanceNames))
			copy(otherInstances, instanceNames)
			otherInstances = append(otherInstances[:i], otherInstances[i+1:]...)
		}
		cm.Data[name+".conf"] = getConfigString(
			populatePGInstanceConfigurationMap(serviceName, repoHostName, pgdataDir, otherInstances,
				postgresCluster.Spec.Archive.PGBackRest.Repos))
	}

	if addDedicatedHost && repoHostName != "" {
		cm.Data[CMRepoKey] = getConfigString(
			populateRepoHostConfigurationMap(serviceName, pgdataDir, instanceNames,
				postgresCluster.Spec.Archive.PGBackRest.Repos))
	}

	return cm
}

// configVolumeAndMount creates a volume and mount configuration from the pgBackRest configmap to be used by the postgrescluster
func configVolumeAndMount(pgBackRestConfigMap *v1.ConfigMap, pod *v1.PodSpec, containerName, configKey string) {
	// Note: the 'container' string will be 'database' for the PostgreSQL database container,
	// otherwise it will be 'backrest'
	var (
		pgBackRestConfig []v1.VolumeProjection
	)

	volume := v1.Volume{Name: ConfigVol}
	volume.Projected = &v1.ProjectedVolumeSource{}

	// Add our projections after those specified in the CR. Items later in the
	// list take precedence over earlier items (that is, last write wins).
	// - https://docs.openshift.com/container-platform/latest/nodes/containers/nodes-containers-projected-volumes.html
	// - https://kubernetes.io/docs/concepts/storage/volumes/#projected
	volume.Projected.Sources = append(
		pgBackRestConfig,
		v1.VolumeProjection{
			ConfigMap: &v1.ConfigMapProjection{
				LocalObjectReference: v1.LocalObjectReference{
					Name: pgBackRestConfigMap.Name,
				},
				Items: []v1.KeyToPath{{
					Key:  configKey,
					Path: configPath,
				}},
			},
		},
	)

	mount := v1.VolumeMount{
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
func PostgreSQLConfigVolumeAndMount(pgBackRestConfigMap *v1.ConfigMap, pod *v1.PodSpec, containerName string) {
	configVolumeAndMount(pgBackRestConfigMap, pod, containerName, cmPrimaryKey)
}

// RepositoryConfigVolumeAndMount creates a volume and mount configuration from the pgBackRest configmap to be used by the
// postgrescluster's pgBackRest repo pod
func RepositoryConfigVolumeAndMount(pgBackRestConfigMap *v1.ConfigMap, pod *v1.PodSpec, containerName string) {
	configVolumeAndMount(pgBackRestConfigMap, pod, containerName, CMRepoKey)
}

// JobConfigVolumeAndMount creates a volume and mount configuration from the pgBackRest configmap to be used by the
// postgrescluster's job pods
func JobConfigVolumeAndMount(pgBackRestConfigMap *v1.ConfigMap, pod *v1.PodSpec, containerName string) {
	configVolumeAndMount(pgBackRestConfigMap, pod, containerName, cmJobKey)
}

// populateRepoHostConfigurationMap returns a map representing the pgBackRest configuration for
// a PostgreSQL instance
func populatePGInstanceConfigurationMap(serviceName, repoHostName, pgdataDir string,
	otherPGHostNames []string, repos []v1beta1.RepoVolume) map[string]map[string]string {

	pgBackRestConfig := map[string]map[string]string{

		// will hold the [global] configs
		"global": {},
		// will hold the [stanza-name] configs
		"stanza": {},
	}

	// set the default stanza name
	pgBackRestConfig["stanza"]["name"] = defaultStanzaName

	// set global settings, which includes all repos
	pgBackRestConfig["global"]["log-path"] = defaultLogPath
	for _, repoVol := range repos {
		if repoHostName != "" && serviceName != "" {
			pgBackRestConfig["global"][repoVol.Name+"-host"] = repoHostName + "-0." + serviceName
			pgBackRestConfig["global"][repoVol.Name+"-host-user"] = "postgres"
		}
		pgBackRestConfig["global"][repoVol.Name+"-path"] = defaultRepo1Path + repoVol.Name
	}

	i := 1
	// Now add all PG instances to the stanza section. Make sure the local PG host is always
	// index 1: https://github.com/pgbackrest/pgbackrest/issues/1197#issuecomment-708381800
	pgBackRestConfig["stanza"][fmt.Sprintf("pg%d-path", i)] = pgdataDir
	pgBackRestConfig["stanza"][fmt.Sprintf("pg%d-port", i)] = defaultPG1Port
	pgBackRestConfig["stanza"][fmt.Sprintf("pg%d-socket-path", i)] = defaultPG1SocketPath
	i++

	if len(otherPGHostNames) == 0 {
		return pgBackRestConfig
	}

	for _, name := range otherPGHostNames {
		pgBackRestConfig["stanza"][fmt.Sprintf("pg%d-host", i)] = name + "-0." + serviceName
		pgBackRestConfig["stanza"][fmt.Sprintf("pg%d-path", i)] = pgdataDir
		pgBackRestConfig["stanza"][fmt.Sprintf("pg%d-port", i)] = defaultPG1Port
		pgBackRestConfig["stanza"][fmt.Sprintf("pg%d-socket-path", i)] = defaultPG1SocketPath
		i++
	}

	return pgBackRestConfig
}

// populateRepoHostConfigurationMap returns a map representing the pgBackRest configuration for
// a pgBackRest dedicated repository host
func populateRepoHostConfigurationMap(serviceName, pgdataDir string,
	pgHosts []string, repos []v1beta1.RepoVolume) map[string]map[string]string {

	pgBackRestConfig := map[string]map[string]string{

		// will hold the [global] configs
		"global": {},
		// will hold the [stanza-name] configs
		"stanza": {},
	}

	// set the default stanza name
	pgBackRestConfig["stanza"]["name"] = defaultStanzaName

	// set the config for the local repo host
	pgBackRestConfig["global"]["log-path"] = defaultLogPath
	for _, repoVol := range repos {
		pgBackRestConfig["global"][repoVol.Name+"-path"] = defaultRepo1Path + repoVol.Name
	}

	// set the configs for all PG hosts
	for i, pgHost := range pgHosts {
		pgBackRestConfig["stanza"][fmt.Sprintf("pg%d-host", i+1)] = pgHost + "-0." + serviceName
		pgBackRestConfig["stanza"][fmt.Sprintf("pg%d-path", i+1)] = pgdataDir
		pgBackRestConfig["stanza"][fmt.Sprintf("pg%d-port", i+1)] = defaultPG1Port
		pgBackRestConfig["stanza"][fmt.Sprintf("pg%d-socket-path", i+1)] = defaultPG1SocketPath
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

// sortedKeys sorts and returns the keys from a given map
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return keys
}
