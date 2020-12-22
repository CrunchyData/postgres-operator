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

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// global pgBackRest default log path configuration, used by all three
	// default pod configurations
	defaultLogPath = "/tmp"

	// defaultRepo1Host stores the default pgBackRest shared repo hostname,
	// minus the postgrescluster name. For a cluster 'mycluster', the
	// defaultRepo1Host name would be 'mycluster-backrest-shared-repo'
	defaultRepo1HostPostfix = "-backrest-shared-repo"

	// defaultRepo1PathPrefix stores the default pgBackRest repo path
	// before the host name value. For a cluster 'mycluster', the
	// defaultRepo1Path name would be '/backrestrepo/mycluster-backrest-shared-repo'
	defaultRepo1PathPrefix = "/backrestrepo/"

	// stanza pgBackRest default configurations
	defaultStanzaName = "db"

	// default Postgres path, port and socket path
	defaultPG1PathPrefix = "/pgdata/"
	defaultPG1Port       = "5432"
	defaultPG1SocketPath = "/tmp"

	// configmap key references
	cmJobKey     = "pgbackrest_job.conf"
	cmPrimaryKey = "pgbackrest_primary.conf"
	cmRepoKey    = "pgbackrest_repo.conf"

	// pgBackRest configuration directory,
	// configuration file path and
	// configuration volume name
	configDir  = "/etc/pgbackrest"
	configPath = "/etc/pgbackrest/pgbackrest.conf"
	configVol  = "pgbackrest-config"

	// suffix used with postgrescluster name for associated configmap.
	// for instance, if the cluster is named 'mycluster', the
	// configmap will be named 'mycluster-pgbackrest-config'
	cmNameSuffix = "%s-pgbackrest-config"
)

// CreatePGBackRestConfigMapStruct creates a configmap struct with pgBackRest pgbackrest.conf settings in the data field.
// The keys within the data field correspond to the use of that configuration.
// pgbackrest_job.conf is used by certain jobs, such as stanza create and backup
// pgbackrest_primary.conf is used by the primary database pod
// pgbackrest_repo.conf is used by the pgBackRest repository pod
func CreatePGBackRestConfigMapStruct(postgresCluster *v1alpha1.PostgresCluster, pghosts []string) v1.ConfigMap {

	cm := v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf(cmNameSuffix, postgresCluster.GetName()),
			Namespace: postgresCluster.GetNamespace(),
		},
	}

	// create an empty map for the config data
	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}

	// if the default data map is not ok, populate with the configuration string
	if _, ok := cm.Data[cmJobKey]; !ok {
		cm.Data[cmJobKey] = getConfigString(populatePGBackrestConfigurationMap(postgresCluster, pghosts, cmJobKey))
	}

	// if the primary data map is not ok, populate with the configuration string
	if _, ok := cm.Data[cmPrimaryKey]; !ok {
		cm.Data[cmPrimaryKey] = getConfigString(populatePGBackrestConfigurationMap(postgresCluster, pghosts, cmPrimaryKey))
	}

	// if the repo data map is not ok, populate with the configuration string
	if _, ok := cm.Data[cmRepoKey]; !ok {
		cm.Data[cmRepoKey] = getConfigString(populatePGBackrestConfigurationMap(postgresCluster, pghosts, cmRepoKey))
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

	volume := v1.Volume{Name: configVol}
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
		MountPath: configDir,
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
	configVolumeAndMount(pgBackRestConfigMap, pod, containerName, cmRepoKey)
}

// JobConfigVolumeAndMount creates a volume and mount configuration from the pgBackRest configmap to be used by the
// postgrescluster's job pods
func JobConfigVolumeAndMount(pgBackRestConfigMap *v1.ConfigMap, pod *v1.PodSpec, containerName string) {
	configVolumeAndMount(pgBackRestConfigMap, pod, containerName, cmJobKey)
}

// populatePGBackrestConfigurationMap constructs our default pgBackRest configuration map,
// fills in any updated values passed in from elsewhere, then returns the completed configuration map
func populatePGBackrestConfigurationMap(postgresCluster *v1alpha1.PostgresCluster, pghosts []string, configmapKey string) map[string]map[string]string {

	// Create a map for our pgBackRest configuration values
	// there are three basic configuration defaults, used
	// by the database pod, pgBackRest repo pod and 'job' pods.
	// Below is the map structure with all possible default key
	// values listed.
	/*
		"global": {
			"log-path":          "",
			"repo1-host":        "",
			"repo1-path":        "",
		},
		"stanza": {
			"name":            "",
			"pg1-host":        "",
			"pg1-path":        "",
			"pg1-port":        "",
			"pg1-socket-path": "",
		},
	*/
	pgBackRestConfig := map[string]map[string]string{

		// will hold the [global] configs
		"global": {},
		// will hold the [stanza-name] configs
		"stanza": {},
	}

	// for any value set in BackrestConfig, update the map
	// global config

	// for all initial pgBackRest functions
	pgBackRestConfig["global"]["log-path"] = defaultLogPath

	// for primary database pod
	if configmapKey == "pgbackrest_primary.conf" {
		pgBackRestConfig["global"]["repo1-host"] = postgresCluster.GetName() + defaultRepo1HostPostfix
	}

	// for primary database pod and repo pod
	if configmapKey == "pgbackrest_primary.conf" || configmapKey == "pgbackrest_repo.conf" {
		pgBackRestConfig["global"]["repo1-path"] = defaultRepo1PathPrefix + postgresCluster.GetName() + defaultRepo1HostPostfix
	}

	// stanza config

	// for primary database pod and repo pod
	if configmapKey == "pgbackrest_primary.conf" || configmapKey == "pgbackrest_repo.conf" {
		pgBackRestConfig["stanza"]["name"] = defaultStanzaName
	}

	// iterate through the provided hostnames and set the configuration blocks for each
	for i, hostname := range pghosts {

		// for repo pod
		if configmapKey == "pgbackrest_repo.conf" {
			pgBackRestConfig["stanza"][fmt.Sprintf("pg%d-host", i+1)] = hostname
		}

		// for primary database pod and repo pod
		if configmapKey == "pgbackrest_primary.conf" || configmapKey == "pgbackrest_repo.conf" {
			pgBackRestConfig["stanza"][fmt.Sprintf("pg%d-path", i+1)] = defaultPG1PathPrefix + hostname
		}

		// for primary database pod and repo pod
		if configmapKey == "pgbackrest_primary.conf" || configmapKey == "pgbackrest_repo.conf" {
			pgBackRestConfig["stanza"][fmt.Sprintf("pg%d-port", i+1)] = defaultPG1Port
		}

		// for primary database pod and repo pod
		if configmapKey == "pgbackrest_primary.conf" || configmapKey == "pgbackrest_repo.conf" {
			pgBackRestConfig["stanza"][fmt.Sprintf("pg%d-socket-path", i+1)] = defaultPG1SocketPath
		}
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
