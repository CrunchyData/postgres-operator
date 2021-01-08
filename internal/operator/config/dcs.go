package config

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

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/util"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

const (
	// PGHADCSConfigName represents the name of the DCS configuration stored in the
	// "<clustername>-pgha-config" configMap, which is "<clustername>-dcs-config"
	PGHADCSConfigName = "%s-dcs-config"
	// DCSConfigMapName represents the name of the DCS configMap created for each cluster, which
	// has the name "<clustername>-config"
	dcsConfigMapName = "%s-config"
	// dcsConfigAnnotation represents that name of the annotation used to store the cluster's DCS
	// configuration
	dcsConfigAnnotation = "config"
)

// DCS configures the DCS configuration settings for a specific PG cluster.
type DCS struct {
	kubeclientset kubernetes.Interface
	configMap     *corev1.ConfigMap
	configName    string
	clusterScope  string
}

// DCSConfig represents the cluster-wide configuration that is stored in the Distributed
// Configuration Store (DCS).
type DCSConfig struct {
	LoopWait              int                `json:"loop_wait,omitempty"`
	TTL                   int                `json:"ttl,omitempty"`
	RetryTimeout          int                `json:"retry_timeout,omitempty"`
	MaximumLagOnFailover  int                `json:"maximum_lag_on_failover,omitempty"`
	MasterStartTimeout    int                `json:"master_start_timeout,omitempty"`
	SynchronousMode       bool               `json:"synchronous_mode,omitempty"`
	SynchronousModeStrict bool               `json:"synchronous_mode_strict,omitempty"`
	PostgreSQL            *PostgresDCS       `json:"postgresql,omitempty"`
	StandbyCluster        *StandbyDCS        `json:"standby_cluster,omitempty"`
	Slots                 map[string]SlotDCS `json:"slots,omitempty"`
}

// PostgresDCS represents the PostgreSQL settings that can be applied cluster-wide to a
// PostgreSQL cluster via the DCS.
type PostgresDCS struct {
	UsePGRewind  bool                   `json:"use_pg_rewind,omitempty"`
	UseSlots     bool                   `json:"use_slots,omitempty"`
	RecoveryConf map[string]interface{} `json:"recovery_conf,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
}

// StandbyDCS represents standby cluster settings that can be applied cluster-wide via the DCS.
type StandbyDCS struct {
	Host                  string                 `json:"host,omitempty"`
	Port                  int                    `json:"port,omitempty"`
	PrimarySlotName       map[string]interface{} `json:"primary_slot_name,omitempty"`
	CreateReplicaMethods  []string               `json:"create_replica_methods,omitempty"`
	RestoreCommand        string                 `json:"restore_command,omitempty"`
	ArchiveCleanupCommand string                 `json:"archive_cleanup_command,omitempty"`
	RecoveryMinApplyDelay int                    `json:"recovery_min_apply_delay,omitempty"`
}

// SlotDCS represents slot settings that can be applied cluster-wide via the DCS.
type SlotDCS struct {
	Type     string `json:"type,omitempty"`
	Database string `json:"database,omitempty"`
	Plugin   string `json:"plugin,omitempty"`
}

// NewDCS creates a new DCS config struct using the configMap provided.  The DCSConfig will
// include a configMap that will be used to configure the DCS for a specific cluster.
func NewDCS(configMap *corev1.ConfigMap, kubeclientset kubernetes.Interface,
	clusterScope string) *DCS {
	clusterName := configMap.GetLabels()[config.LABEL_PG_CLUSTER]

	return &DCS{
		kubeclientset: kubeclientset,
		configMap:     configMap,
		configName:    fmt.Sprintf(PGHADCSConfigName, clusterName),
		clusterScope:  clusterScope,
	}
}

// Sync attempts to apply all configuration in the the DCSConfig's configMap.  If the DCS
// configuration is missing from the configMap, then and attempt is made to add it by refreshing
// the DCS configuration.
func (d *DCS) Sync() error {
	clusterName := d.configMap.GetObjectMeta().GetLabels()[config.LABEL_PG_CLUSTER]
	namespace := d.configMap.GetObjectMeta().GetNamespace()

	log.Debugf("Cluster Config: syncing DCS config for cluster %s (namespace %s)", clusterName,
		namespace)

	if err := d.apply(); err != nil &&
		errors.Is(err, ErrMissingClusterConfig) {
		if err := d.refresh(); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	log.Debugf("Cluster Config: finished syncing DCS config for cluster %s (namespace %s)",
		clusterName, namespace)

	return nil
}

// Update updates the contents of the DCS configuration stored within the configMap included
// in the DCS.
func (d *DCS) Update(dcsConfig *DCSConfig) error {
	clusterName := d.configMap.GetObjectMeta().GetLabels()[config.LABEL_PG_CLUSTER]
	namespace := d.configMap.GetObjectMeta().GetNamespace()

	log.Debugf("Cluster Config: updating DCS config for cluster %s (namespace %s)", clusterName,
		namespace)

	content, err := yaml.Marshal(dcsConfig)
	if err != nil {
		return err
	}

	if err := patchConfigMapData(d.kubeclientset, d.configMap, d.configName, content); err != nil {
		return err
	}

	log.Debugf("Cluster Config: successfully updated DCS config for cluster %s (namespace %s)",
		clusterName, namespace)

	return nil
}

// apply applies the DCS configuration stored in the ClusterConfig's configMap to the cluster's
// DCS.  Specicially, it updates the cluster's DCS, i.e. the the "config" annotation of the
// "<clustername>-config" configMap, with the contents of the "<clustername-dcs-config>"
// configuration included in the DCS's configMap.
func (d *DCS) apply() error {
	clusterName := d.configMap.GetLabels()[config.LABEL_PG_CLUSTER]
	namespace := d.configMap.GetObjectMeta().GetNamespace()

	log.Debugf("Cluster Config: applying DCS config to cluster %s in namespace %s", clusterName,
		namespace)

	// first grab the DCS config from the PGHA config map
	dcsConfig, rawDCS, err := d.GetDCSConfig()
	if err != nil {
		return err
	}

	// next grab the current/live DCS from the "config" annotation of the Patroni configMap
	clusterDCS, rawClusterDCS, err := d.getClusterDCSConfig()
	if err != nil {
		return err
	}

	// if the DCS contents are equal then no further action is needed
	if reflect.DeepEqual(dcsConfig, clusterDCS) {
		log.Debugf("Cluster Config: DCS config for cluster %s in namespace %s is up-to-date, "+
			"nothing to apply", clusterName, namespace)
		return nil
	}

	// ensure the current "pause" setting is not overridden if currently set for the cluster
	if _, ok := rawClusterDCS["pause"]; ok {
		rawDCS["pause"] = rawClusterDCS["pause"]
	}

	// proceed with updating the DCS with the contents of the configMap
	dcsConfigJSON, err := json.Marshal(rawDCS)
	if err != nil {
		return err
	}

	if err := d.patchDCSAnnotation(string(dcsConfigJSON)); err != nil {
		return err
	}

	log.Debugf("Cluster Config: successfully applied DCS to cluster %s in namespace %s",
		clusterName, namespace)

	return nil
}

// getClusterDCSConfig obtains the configuration that is currently stored in the cluster's DCS.
// Specifically, it obtains the configuration stored in the "config" annotation of the
// "<clustername>-config" configMap.
func (d *DCS) getClusterDCSConfig() (*DCSConfig, map[string]json.RawMessage, error) {
	ctx := context.TODO()
	clusterDCS := &DCSConfig{}

	namespace := d.configMap.GetObjectMeta().GetNamespace()

	dcsCM, err := d.kubeclientset.CoreV1().ConfigMaps(namespace).
		Get(ctx, fmt.Sprintf(dcsConfigMapName, d.clusterScope), metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}

	config, ok := dcsCM.GetObjectMeta().GetAnnotations()[dcsConfigAnnotation]
	if !ok {
		return nil, nil, util.ErrMissingConfigAnnotation
	}

	if err := json.Unmarshal([]byte(config), clusterDCS); err != nil {
		return nil, nil, err
	}

	var rawJSON map[string]json.RawMessage
	if err := json.Unmarshal([]byte(config), &rawJSON); err != nil {
		return nil, nil, err
	}

	return clusterDCS, rawJSON, nil
}

// GetDCSConfig returns the current DCS configuration included in the ClusterConfig's
// configMap, i.e. the contents of the "<clustername-dcs-config>" configuration unmarshalled
// into a DCSConfig struct.
func (d *DCS) GetDCSConfig() (*DCSConfig, map[string]json.RawMessage, error) {
	dcsYAML, ok := d.configMap.Data[d.configName]
	if !ok {
		return nil, nil, ErrMissingClusterConfig
	}

	dcsConfig := &DCSConfig{}

	if err := yaml.Unmarshal([]byte(dcsYAML), dcsConfig); err != nil {
		return nil, nil, err
	}

	var rawJSON map[string]json.RawMessage
	if err := yaml.Unmarshal([]byte(dcsYAML), &rawJSON); err != nil {
		return nil, nil, err
	}

	return dcsConfig, rawJSON, nil
}

// patchDCSAnnotation patches the "config" annotation within the DCS configMap with the
// content provided.
func (d *DCS) patchDCSAnnotation(content string) error {
	ctx := context.TODO()
	jsonOpBytes, err := kubeapi.NewJSONPatch().Replace("metadata", "annotations", dcsConfigAnnotation)(content).Bytes()
	if err != nil {
		return err
	}

	name := fmt.Sprintf(dcsConfigMapName, d.clusterScope)
	log.Debugf("patching configmap %s: %s", name, jsonOpBytes)
	_, err = d.kubeclientset.CoreV1().ConfigMaps(d.configMap.GetNamespace()).
		Patch(ctx, name, types.JSONPatchType, jsonOpBytes, metav1.PatchOptions{})

	return err
}

// refresh updates the DCS configuration stored in the "<clustername>-pgha-config"
// configMap with the current DCS configuration for the cluster.  Specifically, it is updated with
// the configuration stored in the "config" annotation of the "<clustername>-config" configMap.
func (d *DCS) refresh() error {
	clusterName := d.configMap.Labels[config.LABEL_PG_CLUSTER]
	namespace := d.configMap.GetObjectMeta().GetNamespace()

	log.Debugf("Cluster Config: refreshing DCS config for cluster %s (namespace %s)", clusterName,
		namespace)

	clusterDCS, _, err := d.getClusterDCSConfig()
	if err != nil {
		return err
	}

	clusterDCSBytes, err := yaml.Marshal(clusterDCS)
	if err != nil {
		return err
	}

	if err := patchConfigMapData(d.kubeclientset, d.configMap, d.configName,
		clusterDCSBytes); err != nil {
		return err
	}

	log.Debugf("Cluster Config: successfully refreshed DCS config for cluster %s (namespace %s)",
		clusterName, namespace)

	return nil
}
