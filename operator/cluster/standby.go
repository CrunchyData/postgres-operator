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

	"github.com/crunchydata/postgres-operator/events"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/operator/backrest"
	"github.com/crunchydata/postgres-operator/operator/pvc"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"k8s.io/client-go/kubernetes"
)

var (
	// ErrClusterNotShutdown defines the error that is thrown when an action cannot
	// proceed because the cluster is not in standby mode
	ErrClusterNotShutdown = errors.New("Cluster not in shutdown status")
	// ErrStandbyNotEnabled defines the error that is thrown when
	// standby mode is not enabled but a standby action was attempted
	ErrStandbyNotEnabled = errors.New("Standby mode not enabled")
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
func DisableStandby(clientset *kubernetes.Clientset, cluster crv1.Pgcluster) error {

	clusterName := cluster.Name
	namespace := cluster.Namespace

	log.Debugf("Disable standby: disabling standby for cluster %s", clusterName)

	// find the "config" configMap created by Patroni
	configMapName := cluster.Labels[config.LABEL_PGHA_SCOPE] + "-config"
	configMap, found := kubeapi.GetConfigMap(clientset, configMapName, namespace)
	if !found {
		return fmt.Errorf("Unable to find configMap %s when attempting to enable standby",
			configMapName)
	}

	// return ErrMissingConfigAnnotation error if configMap is missing the "config" annotation
	if _, ok := configMap.ObjectMeta.Annotations["config"]; !ok {
		return util.ErrMissingConfigAnnotation
	}

	// grab the json stored in the config annotation
	configJSONStr := configMap.ObjectMeta.Annotations["config"]
	var configJSON map[string]interface{}
	json.Unmarshal([]byte(configJSONStr), &configJSON)

	// retrun an error if standby_cluster isnt found (meaning it isn't currently enabled)
	if _, ok := configJSON["standby_cluster"]; !ok {
		return fmt.Errorf("Unable to disable standby mode for cluster %s: %w", clusterName,
			ErrStandbyNotEnabled)
	}

	// now delete the standby cluster configuration and update the 'config' annotation
	delete(configJSON, "standby_cluster")
	configJSONFinalStr, err := json.Marshal(configJSON)
	if err != nil {
		return err
	}
	configMap.ObjectMeta.Annotations["config"] = string(configJSONFinalStr)
	err = kubeapi.UpdateConfigMap(clientset, configMap, namespace)
	if err != nil {
		return err
	}

	// ensure any repo override is removed
	pghaConfigMapName := cluster.Labels[config.LABEL_PGHA_SCOPE] + "-pgha-config"
	pghaConfigMap, found := kubeapi.GetConfigMap(clientset, pghaConfigMapName, namespace)
	if !found {
		return fmt.Errorf("Unable to find configMap %s when attempting to enable standby",
			pghaConfigMapName)
	}
	delete(pghaConfigMap.Data, operator.PGHAConfigReplicaBootstrapRepoTye)

	if err := kubeapi.UpdateConfigMap(clientset, pghaConfigMap, namespace); err != nil {
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
	} else if len(remainingPVC.Items) > 2 {
		return fmt.Errorf("Unexpected number of PVCs (%d) found while cleaning up PVCs to enable "+
			"standby mode for cluster %s", len(remainingPVC.Items), clusterName)
	}

	var primaryPVCName string
	var backrestRepoPVCName string
	for _, pvc := range remainingPVC.Items {
		if pvc.Name == fmt.Sprintf(backrest.BackrestRepoPVCName, clusterName) {
			backrestRepoPVCName = pvc.Name
			continue
		}
		primaryPVCName = pvc.Name
	}

	// try to delete both PVCs, continuing if the PVC is not found (e.g. if previously deleted)
	if err := kubeapi.DeletePVC(clientset, primaryPVCName, namespace); !kerrors.IsNotFound(err) &&
		err != nil {
		log.Error(err)
	}
	if err := kubeapi.DeletePVC(clientset, backrestRepoPVCName, namespace); !kerrors.IsNotFound(err) &&
		err != nil {
		log.Error(err)
	}

	log.Debugf("Enable standby: deleted the following PVCs for cluster %s: %v", clusterName,
		[]string{primaryPVCName, backrestRepoPVCName})

	timeout := time.Second * 60
	// Now wait for the PVCs to be deleted
	if err := kubeapi.IsPVCDeleted(clientset, timeout, primaryPVCName, namespace); err != nil {
		log.Error(err)
		return err
	}

	// Now wait for the PVCs to be deleted
	if err := kubeapi.IsPVCDeleted(clientset, timeout, backrestRepoPVCName,
		namespace); err != nil {
		log.Error(err)
		return err
	}

	// Now recreate the PVCs
	primaryStorage := cluster.Spec.PrimaryStorage
	backrestRepostorage := cluster.Spec.BackrestStorage

	if err := pvc.Create(clientset, primaryPVCName, clusterName, &primaryStorage,
		namespace); err != nil {
		log.Error(err)
		return fmt.Errorf("Unable to create primary PVC while enabling standby mode: %w", err)
	}

	if err := pvc.Create(clientset, backrestRepoPVCName, clusterName, &backrestRepostorage,
		namespace); err != nil {
		log.Error(err)
		return fmt.Errorf("Unable to create primary PVC while enabling standby mode: %w", err)
	}

	log.Debugf("Enable standby: re-created primary PVC %s and pgBackRest repo PVC %s for cluster "+
		"%s", primaryPVCName, backrestRepoPVCName, clusterName)

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

	leaderConfigMapName := clusterName + "-leader"
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
	pghaConfigMap.Data[operator.PGHAConfigReplicaBootstrapRepoTye] = "s3"

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
