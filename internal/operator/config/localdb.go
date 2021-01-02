package config

/*
 Copyright 2020 - 2021 Crunchy Data Solutions, Inl.
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

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"
)

var (
	// readConfigCMD is the command used to read local cluster configuration in a database
	// container
	readConfigCMD []string = []string{
		"bash", "-c",
		"/opt/crunchy/bin/yq r /tmp/postgres-ha-bootstrap.yaml postgresql | " +
			"/opt/crunchy/bin/yq p - postgresql",
	}
	// applyAndReloadConfigCMD is the command for calling the script to apply and reload the local
	// configuration for a database container.  The required arguments are appended to this command
	// when the script is called.
	applyAndReloadConfigCMD []string = []string{"/opt/crunchy/bin/postgres-ha/common/pgha-reload-local.sh"}

	// pghaLocalConfigName represents the name of the local configuration stored for each database
	// server in the "<clustername>-pgha-config" configMap, which is "<clusterName>-local-config"
	pghaLocalConfigName = "%s-local-config"
	// pghaLocalConfigSuffix is the suffix for a local server configuration
	pghaLocalConfigSuffix = "-local-config"
)

// LocalDB configures the local configuration settings for a specific database server within a
// PG cluster.
type LocalDB struct {
	kubeclientset kubernetes.Interface
	configMap     *corev1.ConfigMap
	configNames   []string
	restConfig    *rest.Config
}

// LocalDBConfig represents the local configuration for a specific PostgreSQL database server
// within a PostgreSQL cluster.  Only user-facing configuration is exposed via this struct,
// and not any configuration that is controlled/managed by the Operator itself.
type LocalDBConfig struct {
	PostgreSQL PostgresLocalDB `json:"postgresql,omitempty"`
}

// PostgresLocalDB represents the PostgreSQL settings that can be applied to an individual
// PostgreSQL server within a PostgreSQL cluster.
type PostgresLocalDB struct {
	// Authentication is the block for managing the Patroni managed accounts
	// (superuser, replication, rewind). While the PostgreSQL Operator manages
	// these overall, one may want to override them. We allow for this, but the
	// deployer should take care when overriding this value
	Authentication                         map[string]interface{} `json:"authentication,omitempty"`
	Callbacks                              *Callbacks             `json:"callbacks,omitempty"`
	CreateReplicaMethods                   []string               `json:"create_replica_methods,omitempty"`
	ConfigDir                              string                 `json:"config_dir,omitempty"`
	UseUnixSocket                          bool                   `json:"use_unix_socket,omitempty"`
	PGPass                                 string                 `json:"pgpass,omitempty"`
	RecoveryConf                           map[string]interface{} `json:"recovery_conf,omitempty"`
	CustomConf                             map[string]interface{} `json:"custom_conf,omitempty"`
	Parameters                             map[string]interface{} `json:"parameters,omitempty"`
	PGHBA                                  []string               `json:"pg_hba,omitempty"`
	PGIdent                                []string               `json:"pg_ident,omitempty"`
	PGCTLTimeout                           int                    `json:"pg_ctl_timeout,omitempty"`
	UsePGRewind                            bool                   `json:"use_pg_rewind,omitempty"`
	RemoveDataDirectoryOnRewindFailure     bool                   `json:"remove_data_directory_on_rewind_failure,omitempty"`
	RemoveDataDirectoryOnDivergedTimelines bool                   `json:"remove_data_directory_on_diverged_timelines,omitempty"`
	PGBackRest                             *CreateReplicaMethod   `json:"pgbackrest,omitempty"`
	PGBackRestStandby                      *CreateReplicaMethod   `json:"pgbackrest_standby,omitempty"`
}

// Callbacks defines the various Patroni callbacks
type Callbacks struct {
	OnReload     string `json:"on_reload,omitempty"`
	OnRestart    string `json:"on_restart,omitempty"`
	OnRoleChange string `json:"on_role_change,omitempty"`
	OnStart      string `json:"on_start,omitempty"`
	OnStop       string `json:"on_stop,omitempty"`
}

// CreateReplicaMethod represents a Patroni replica creation method
type CreateReplicaMethod struct {
	Command  string `json:"command,omitempty"`
	KeepData bool   `json:"keep_data,omitempty"`
	NoParams bool   `json:"no_params,omitempty"`
	NoMaster int    `json:"no_master,omitempty"`
}

// NewLocalDB creates a new LocalDB, which includes a configMap that contains the local
// configuration settings for the database servers within a specific PG cluster.  Additionally
// the LocalDB includes the client(s) and other applicable resources needed to access and modify
// various resources within the Kubernetes cluster in support of configuring the included database
// servers.
func NewLocalDB(configMap *corev1.ConfigMap, restConfig *rest.Config,
	kubeclientset kubernetes.Interface) (*LocalDB, error) {
	clusterName := configMap.GetLabels()[config.LABEL_PG_CLUSTER]
	namespace := configMap.GetObjectMeta().GetNamespace()

	configNames, err := GetLocalDBConfigNames(kubeclientset, clusterName, namespace)
	if err != nil {
		return nil, err
	}

	return &LocalDB{
		kubeclientset: kubeclientset,
		restConfig:    restConfig,
		configMap:     configMap,
		configNames:   configNames,
	}, nil
}

// Sync attempts to apply all local database server configuration settings in the the LocalDB's configMap
// to the various servers included in the LocalDB.  If the configuration for a server is missing from the
// configMap, then and attempt is made to add it by refreshing that specific configuration.  Also, any
// configurations within the configMap associated with servers that no longer exist are removed.
func (l *LocalDB) Sync() error {
	clusterName := l.configMap.GetObjectMeta().GetLabels()[config.LABEL_PG_CLUSTER]
	namespace := l.configMap.GetObjectMeta().GetNamespace()

	log.Debugf("Cluster Config: syncing local config for cluster %s (namespace %s)", clusterName,
		namespace)

	var wg sync.WaitGroup

	wg.Add(1)

	// delete any configs that are in the configMap but don't have an associated DB server in the
	// cluster
	go func() {
		_ = l.clean()
		wg.Done()
	}()

	// attempt to apply local config
	for _, configName := range l.configNames {

		wg.Add(1)

		go func(config string) {
			// attempt to apply DCS config
			if err := l.apply(config); err != nil &&
				errors.Is(err, ErrMissingClusterConfig) {
				if err := l.refresh(config); err != nil {
					// log the error and move on
					log.Error(err)
				}
			} else if err != nil {
				// log the error and move on
				log.Error(err)
			}

			wg.Done()
		}(configName)
	}

	wg.Wait()

	log.Debugf("Cluster Config: finished syncing config for cluster %s (namespace %s)",
		clusterName, namespace)

	return nil
}

// Update updates the contents of the configuration for a specific database server in
// the PG cluster, specifically within the configMap included in the LocalDB.
func (l *LocalDB) Update(configName string, localDBConfig LocalDBConfig) error {
	clusterName := l.configMap.GetObjectMeta().GetLabels()[config.LABEL_PG_CLUSTER]
	namespace := l.configMap.GetObjectMeta().GetNamespace()

	log.Debugf("Cluster Config: updating local config %s in cluster %s "+
		"(namespace %s)", configName, clusterName, namespace)

	content, err := yaml.Marshal(localDBConfig)
	if err != nil {
		return err
	}

	if err := patchConfigMapData(l.kubeclientset, l.configMap, configName, content); err != nil {
		return err
	}

	log.Debugf("Cluster Config: successfully updated local config %s in cluster %s "+
		"(namespace %s)", configName, clusterName, namespace)

	return nil
}

// apply applies the configuration stored in the cluster ConfigMap for a specific database server
// to that server.  This is done by updating the contents of that database server's local
// configuration with the configuration for that cluster stored in the LocalDB's configMap, and
// then issuing a Patroni "reload" for that specific server.
func (l *LocalDB) apply(configName string) error {
	ctx := context.TODO()
	clusterName := l.configMap.GetObjectMeta().GetLabels()[config.LABEL_PG_CLUSTER]
	namespace := l.configMap.GetObjectMeta().GetNamespace()

	log.Debugf("Cluster Config: applying local config %s in cluster %s "+
		"(namespace %s)", configName, clusterName, namespace)

	localConfig, err := l.getLocalConfig(configName)
	if err != nil {
		return err
	}

	// selector in the format "pg-cluster=<cluster-name>,deployment-name=<server-name>"
	selector := fmt.Sprintf("%s=%s,%s=%s", config.LABEL_PG_CLUSTER, clusterName,
		config.LABEL_DEPLOYMENT_NAME, strings.TrimSuffix(configName, pghLocalConfigSuffix))
	dbPodList, err := l.kubeclientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return err
	}
	// if the pod list is empty, also return an error
	if len(dbPodList.Items) == 0 {
		return fmt.Errorf("no pod found for %q", clusterName)
	}

	dbPod := &dbPodList.Items[0]

	// add the config name and patroni port as params for the call to the apply & reload script
	applyCommand := append(applyAndReloadConfigCMD, localConfig, config.DEFAULT_PATRONI_PORT)

	stdout, stderr, err := kubeapi.ExecToPodThroughAPI(l.restConfig, l.kubeclientset, applyCommand,
		dbPod.Spec.Containers[0].Name, dbPod.GetName(), namespace, nil)
	if err != nil {
		log.Error(stderr, stdout)
		return err
	}

	log.Debugf("Cluster Config: successfully applied local config %s in cluster %s "+
		"(namespace %s)", configName, clusterName, namespace)

	return nil
}

// clean removes any local database server configurations from the configMap included in the
// LocalDB if the database server they are associated with no longer exists
func (l *LocalDB) clean() error {
	ctx := context.TODO()
	patch := kubeapi.NewJSONPatch()
	var cmlocalConfigs []string

	// first grab all current local configs from the configMap
	for configName := range l.configMap.Data {
		if strings.HasSuffix(configName, pghaLocalConfigSuffix) {
			cmlocalConfigs = append(cmlocalConfigs, configName)
		}
	}

	// now see if any need to be deleted
	for _, cmLocalConfig := range cmlocalConfigs {
		deleteConfig := true
		for _, managedConfigName := range l.configNames {
			if cmLocalConfig == managedConfigName {
				deleteConfig = false
				break
			}
		}
		if deleteConfig {
			patch.Remove("data", cmLocalConfig)
		}
	}

	jsonOpBytes, err := patch.Bytes()
	if err != nil {
		return err
	}

	log.Debugf("patching configmap %s: %s", l.configMap.GetName(), jsonOpBytes)
	_, err = l.kubeclientset.CoreV1().ConfigMaps(l.configMap.GetNamespace()).
		Patch(ctx, l.configMap.GetName(), types.JSONPatchType, jsonOpBytes, metav1.PatchOptions{})

	return err
}

// getLocalConfigFromCluster obtains the local configuration for a specific database server in the
// cluster.  It also returns the Pod that is currently running that specific server.
func (l *LocalDB) getLocalConfigFromCluster(configName string) (*LocalDBConfig, error) {
	ctx := context.TODO()
	clusterName := l.configMap.GetObjectMeta().GetLabels()[config.LABEL_PG_CLUSTER]
	namespace := l.configMap.GetObjectMeta().GetNamespace()

	// selector in the format "pg-cluster=<cluster-name>,deployment-name=<server-name>"
	selector := fmt.Sprintf("%s=%s,%s=%s", config.LABEL_PG_CLUSTER, clusterName,
		config.LABEL_DEPLOYMENT_NAME, strings.TrimSuffix(configName, pghLocalConfigSuffix))
	dbPodList, err := l.kubeclientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, err
	}

	// if the pod list is empty, also return an error
	if len(dbPodList.Items) == 0 {
		return nil, fmt.Errorf("no pod found for %q", clusterName)
	}

	dbPod := &dbPodList.Items[0]

	stdout, stderr, err := kubeapi.ExecToPodThroughAPI(l.restConfig, l.kubeclientset, readConfigCMD,
		dbPod.Spec.Containers[0].Name, dbPod.GetName(), namespace, nil)
	if err != nil {
		log.Errorf(stderr)
		return nil, err
	}

	// we unmarshall to ensure the configMap only contains the settings that we want to expose
	// to the end-user
	localDBConfig := &LocalDBConfig{}
	if err := yaml.Unmarshal([]byte(stdout), localDBConfig); err != nil {
		return nil, err
	}

	return localDBConfig, nil
}

// getLocalConfig returns the current local configuration included in the ClusterConfig's
// configMap for a specific database server, i.e. the contents of the "<servername-local-config>"
// configuration unmarshalled into a LocalConfig struct.
func (l *LocalDB) getLocalConfig(configName string) (string, error) {
	localYAML, ok := l.configMap.Data[configName]
	if !ok {
		return "", ErrMissingClusterConfig
	}

	jsonConfig, err := yaml.YAMLToJSON([]byte(localYAML))
	if err != nil {
		return "", err
	}

	// decode just to ensure no disallowed fields in the config
	dec := json.NewDecoder(strings.NewReader(string(jsonConfig)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&LocalDBConfig{}); err != nil {
		return "", err
	}

	return localYAML, nil
}

// refresh updates the local configuration for a specific database server in the Refresh's
// configMap with the current local configuration for that server.  Specifically, it is updated
// with the contents of the Patroni YAML configuration file stored in the container running the
// server.
func (l *LocalDB) refresh(configName string) error {
	clusterName := l.configMap.GetObjectMeta().GetLabels()[config.LABEL_PG_CLUSTER]
	namespace := l.configMap.GetObjectMeta().GetNamespace()

	log.Debugf("Cluster Config: refreshing local config %s in cluster %s "+
		"(namespace %s)", configName, clusterName, namespace)

	localConfig, err := l.getLocalConfigFromCluster(configName)
	if err != nil {
		return err
	}

	localConfigYAML, err := yaml.Marshal(localConfig)
	if err != nil {
		return err
	}

	if err := patchConfigMapData(l.kubeclientset, l.configMap, configName,
		localConfigYAML); err != nil {
		return err
	}

	log.Debugf("Cluster Config: successfully refreshed local %s in cluster %s "+
		"(namespace %s)", configName, clusterName, namespace)

	return nil
}

// GetLocalDBConfigNames returns the names of the local configuration for each database server in
// the cluster as stored in the <clusterName>-pgha-config configMap per naming conventions.
func GetLocalDBConfigNames(kubeclientset kubernetes.Interface, clusterName,
	namespace string) ([]string, error) {
	ctx := context.TODO()

	// selector in the format "pg-cluster=<cluster-name>,pgo-pg-database"
	// to get all db Deployments
	selector := fmt.Sprintf("%s=%s,%s", config.LABEL_PG_CLUSTER, clusterName,
		config.LABEL_PG_DATABASE)
	dbDeploymentList, err := kubeclientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, err
	}

	localConfigNames := make([]string, len(dbDeploymentList.Items))
	for i, deployment := range dbDeploymentList.Items {
		localConfigNames[i] = fmt.Sprintf(pghaLocalConfigName, deployment.GetName())
	}

	return localConfigNames, nil
}
