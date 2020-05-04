package cluster

/*
Copyright 2020 Crunchy Data Solutions, Inc.
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
	"encoding/json"
	"errors"
	"fmt"
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/crunchydata/postgres-operator/events"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/operator/pvc"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"

	crv1 "github.com/crunchydata/postgres-operator/internal/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	cfg "github.com/crunchydata/postgres-operator/operator/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	// ErrStandbyNotAllowed contains the error message returned when an API call is not
	// permitted because it involves a cluster that is in standby mode
	ErrStandbyNotAllowed = errors.New("Action not permitted because standby mode is enabled")
	// ErrStandbyNotEnabled defines the error that is thrown when
	// standby mode is not enabled but a standby action was attempted
	ErrStandbyNotEnabled = errors.New("Standby mode not enabled")
	// ErrClusterNotShutdown defines the error that is thrown when an action cannot
	// proceed because the cluster is not in standby mode
	ErrClusterNotShutdown = errors.New("Cluster not in shutdown status")
)

const (
	standbyClusterConfigJSON = `
{
    "create_replica_methods": [
	    "pgbackrest_standby"
    ],
	"restore_command": "source /opt/cpm/bin/pgbackrest/pgbackrest-set-env.sh && pgbackrest archive-get %f \"%p\""
}`
)

// DisableStandby disables standby mode for the cluster
func DisableStandby(clientset kubernetes.Interface, cluster crv1.Pgcluster) error {

	clusterName := cluster.Name
	namespace := cluster.Namespace

	log.Debugf("Disable standby: disabling standby for cluster %s", clusterName)

	configMapName := fmt.Sprintf("%s-pgha-config", cluster.Labels[config.LABEL_PGHA_SCOPE])
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(configMapName,
		metav1.GetOptions{})
	if err != nil {
		return err
	}
	dcs := cfg.NewDCS(configMap, clientset)
	dcsConfig, _, err := dcs.GetDCSConfig()
	if err != nil {
		return err
	}
	dcsConfig.StandbyCluster = nil
	if err := dcs.Update(dcsConfig); err != nil {
		return err
	}

	// ensure any repo override is removed
	pghaConfigMapName := fmt.Sprintf("%s-pgha-config", cluster.Labels[config.LABEL_PGHA_SCOPE])
	jsonOp := []util.JSONPatchOperation{{
		Op:   "remove",
		Path: fmt.Sprintf("/data/%s", operator.PGHAConfigReplicaBootstrapRepoType),
	}}

	jsonOpBytes, err := json.Marshal(jsonOp)
	if err != nil {
		return err
	}

	if _, err := clientset.CoreV1().ConfigMaps(namespace).Patch(pghaConfigMapName,
		types.JSONPatchType, jsonOpBytes); err != nil {
		return err
	}

	if err := publishStandbyEnabled(&cluster); err != nil {
		log.Error(err)
	}

	log.Debugf("Disable standby: finished disabling standby mode for cluster %s", clusterName)

	return nil
}

// EnableStandby enables standby mode for the cluster
func EnableStandby(clientset *kubernetes.Clientset, cluster crv1.Pgcluster) error {

	clusterName := cluster.Name
	namespace := cluster.Namespace

	log.Debugf("Enable standby: attempting to enable standby for cluster %s", clusterName)

	// First verify that the cluster is in a shut down status.  If not then return an
	// error
	if cluster.Status.State != crv1.PgclusterStateShutdown {
		return fmt.Errorf("Unable to enable standby mode: %w", ErrClusterNotShutdown)
	}

	// Now find the existing PVCs for the primary and backrest repo and delete them.
	// These should be the only remaining PVCs for the cluster since all replica PVCs
	// were deleted when scaling down the cluster in order to shut down the database.
	remainingPVCSelector := fmt.Sprintf("%s=%s", config.LABEL_PG_CLUSTER, clusterName)
	remainingPVC, err := kubeapi.GetPVCs(clientset, remainingPVCSelector, namespace)
	if err != nil {
		log.Error(err)
		return fmt.Errorf("Unable to get remaining PVCs while enabling standby mode: %w", err)
	}

	for _, currPVC := range remainingPVC.Items {

		// delete the original PVC and wait for it to be removed
		if err := kubeapi.DeletePVC(clientset, currPVC.Name, namespace); !kerrors.IsNotFound(err) &&
			err != nil {
			log.Error(err)
		}
		if err := kubeapi.IsPVCDeleted(clientset, time.Second*60, currPVC.Name,
			namespace); err != nil {
			log.Error(err)
			return err
		}

		// determine whether the PVC is a backrest repo, primary or replica, and then re-create
		// using the proper storage spec as defined in pgo.yaml
		storageSpec := crv1.PgStorageSpec{}
		if currPVC.Name == cluster.Labels[config.ANNOTATION_PRIMARY_DEPLOYMENT] {
			storageSpec = cluster.Spec.PrimaryStorage
		} else if currPVC.Name == fmt.Sprintf(util.BackrestRepoPVCName, clusterName) {
			storageSpec = cluster.Spec.BackrestStorage
		} else {
			storageSpec = cluster.Spec.ReplicaStorage
		}
		if err := pvc.Create(clientset, currPVC.Name, clusterName, &storageSpec,
			namespace); err != nil {
			log.Error(err)
			return fmt.Errorf("Unable to create primary PVC while enabling standby mode: %w", err)
		}
	}

	log.Debugf("Enable standby: re-created PVC's %v for cluster %s", remainingPVC.Items,
		clusterName)

	// find the "config" configMap created by Patroni
	dcsConfigMapName := cluster.Labels[config.LABEL_PGHA_SCOPE] + "-config"
	dcsConfigMap, found := kubeapi.GetConfigMap(clientset, dcsConfigMapName, namespace)
	if !found {
		return fmt.Errorf("Unable to find configMap %s when attempting to enable standby",
			dcsConfigMapName)
	}

	// return ErrMissingConfigAnnotation error if configMap is missing the "config" annotation
	if _, ok := dcsConfigMap.ObjectMeta.Annotations["config"]; !ok {
		return util.ErrMissingConfigAnnotation
	}

	// grab the json stored in the config annotation
	configJSONStr := dcsConfigMap.ObjectMeta.Annotations["config"]
	var configJSON map[string]interface{}
	json.Unmarshal([]byte(configJSONStr), &configJSON)

	var standbyJSON map[string]interface{}
	json.Unmarshal([]byte(standbyClusterConfigJSON), &standbyJSON)

	// set standby_cluster to default config unless already set
	if _, ok := configJSON["standby_cluster"]; !ok {
		configJSON["standby_cluster"] = standbyJSON
	}

	configJSONFinalStr, err := json.Marshal(configJSON)
	if err != nil {
		return err
	}
	dcsConfigMap.ObjectMeta.Annotations["config"] = string(configJSONFinalStr)
	err = kubeapi.UpdateConfigMap(clientset, dcsConfigMap, namespace)
	if err != nil {
		return err
	}

	leaderConfigMapName := cluster.Labels[config.LABEL_PGHA_SCOPE] + "-leader"
	// Delete the "leader" configMap
	if err = kubeapi.DeleteConfigMap(clientset, leaderConfigMapName, namespace); err != nil &&
		!kerrors.IsNotFound(err) {
		log.Error("Unable to delete configMap %s while enabling standby mode for cluster "+
			"%s: %w", leaderConfigMapName, clusterName, err)
		return err
	}

	// override to the repo type to ensure s3 is utilized for standby creation
	pghaConfigMapName := cluster.Labels[config.LABEL_PGHA_SCOPE] + "-pgha-config"
	pghaConfigMap, found := kubeapi.GetConfigMap(clientset, pghaConfigMapName, namespace)
	if !found {
		return fmt.Errorf("Unable to find configMap %s when attempting to enable standby",
			pghaConfigMapName)
	}
	pghaConfigMap.Data[operator.PGHAConfigReplicaBootstrapRepoType] = "s3"

	// delete the DCS config so that it will refresh with the included standby settings
	delete(pghaConfigMap.Data, fmt.Sprintf(cfg.PGHADCSConfigName, clusterName))

	if err := kubeapi.UpdateConfigMap(clientset, pghaConfigMap, namespace); err != nil {
		return err
	}

	if err := publishStandbyEnabled(&cluster); err != nil {
		log.Error(err)
	}

	log.Debugf("Enable standby: finished enabling standby mode for cluster %s", clusterName)

	return nil
}

func publishStandbyEnabled(cluster *crv1.Pgcluster) error {

	clusterName := cluster.Name

	//capture the cluster creation event
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventStandbyEnabledFormat{
		EventHeader: events.EventHeader{
			Namespace: cluster.Namespace,
			Username:  cluster.Spec.UserLabels[config.LABEL_PGOUSER],
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventStandbyEnabled,
		},
		Clustername: clusterName,
	}

	if err := events.Publish(f); err != nil {
		log.Error(err.Error())
		return err
	}

	return nil
}

func publishStandbyDisabled(cluster *crv1.Pgcluster) error {

	clusterName := cluster.Name

	//capture the cluster creation event
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventStandbyDisabledFormat{
		EventHeader: events.EventHeader{
			Namespace: cluster.Namespace,
			Username:  cluster.Spec.UserLabels[config.LABEL_PGOUSER],
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventStandbyDisabled,
		},
		Clustername: clusterName,
	}

	if err := events.Publish(f); err != nil {
		log.Error(err.Error())
		return err
	}

	return nil
}
