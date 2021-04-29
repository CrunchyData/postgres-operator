package cluster

/*
Copyright 2020 - 2021 Crunchy Data Solutions, Inc.
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
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/operator"
	"github.com/crunchydata/postgres-operator/internal/operator/pvc"
	"github.com/crunchydata/postgres-operator/internal/util"
	"github.com/crunchydata/postgres-operator/pkg/events"
	log "github.com/sirupsen/logrus"

	"github.com/crunchydata/postgres-operator/internal/config"
	cfg "github.com/crunchydata/postgres-operator/internal/operator/config"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
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
	"restore_command": "source /opt/crunchy/bin/postgres-ha/pgbackrest/pgbackrest-set-env.sh && pgbackrest archive-get %f \"%p\""
}`
)

// DisableStandby disables standby mode for the cluster
func DisableStandby(clientset kubernetes.Interface, cluster crv1.Pgcluster) error {
	ctx := context.TODO()
	clusterName := cluster.Name
	namespace := cluster.Namespace

	log.Debugf("Disable standby: disabling standby for cluster %s", clusterName)

	configMapName := fmt.Sprintf("%s-pgha-config", cluster.Name)
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).
		Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	dcs := cfg.NewDCS(configMap, clientset, cluster.Name)
	dcsConfig, _, err := dcs.GetDCSConfig()
	if err != nil {
		return err
	}
	dcsConfig.StandbyCluster = nil
	if err := dcs.Update(dcsConfig); err != nil {
		return err
	}

	// ensure any repo override is removed
	pghaConfigMapName := fmt.Sprintf("%s-pgha-config", cluster.Name)
	jsonOpBytes, err := kubeapi.NewJSONPatch().Remove("data", operator.PGHAConfigReplicaBootstrapRepoType).Bytes()
	if err != nil {
		return err
	}

	log.Debugf("patching configmap %s: %s", pghaConfigMapName, jsonOpBytes)
	if _, err := clientset.CoreV1().ConfigMaps(namespace).
		Patch(ctx, pghaConfigMapName, types.JSONPatchType, jsonOpBytes, metav1.PatchOptions{}); err != nil {
		return err
	}

	if err := publishStandbyEnabled(&cluster); err != nil {
		log.Error(err)
	}

	log.Debugf("Disable standby: finished disabling standby mode for cluster %s", clusterName)

	return nil
}

// EnableStandby enables standby mode for the cluster
func EnableStandby(clientset kubernetes.Interface, cluster crv1.Pgcluster) error {
	ctx := context.TODO()
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
	remainingPVC, err := clientset.
		CoreV1().PersistentVolumeClaims(namespace).
		List(ctx, metav1.ListOptions{LabelSelector: remainingPVCSelector})
	if err != nil {
		log.Error(err)
		return fmt.Errorf("Unable to get remaining PVCs while enabling standby mode: %w", err)
	}

	for _, currPVC := range remainingPVC.Items {

		// delete the original PVC and wait for it to be removed
		deletePropagation := metav1.DeletePropagationForeground
		err := clientset.
			CoreV1().PersistentVolumeClaims(namespace).
			Delete(ctx, currPVC.Name, metav1.DeleteOptions{PropagationPolicy: &deletePropagation})
		if err == nil {
			err = wait.Poll(time.Second/2, time.Minute, func() (bool, error) {
				_, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, currPVC.Name, metav1.GetOptions{})
				return false, err
			})
		}
		if !kerrors.IsNotFound(err) {
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
			namespace, util.GetCustomLabels(&cluster)); err != nil {
			log.Error(err)
			return fmt.Errorf("Unable to create primary PVC while enabling standby mode: %w", err)
		}
	}

	log.Debugf("Enable standby: re-created PVC's %v for cluster %s", remainingPVC.Items,
		clusterName)

	// find the "config" configMap created by Patroni
	dcsConfigMapName := cluster.Name + "-config"
	dcsConfigMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, dcsConfigMapName, metav1.GetOptions{})
	if err != nil {
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
	_ = json.Unmarshal([]byte(configJSONStr), &configJSON)

	var standbyJSON map[string]interface{}
	_ = json.Unmarshal([]byte(standbyClusterConfigJSON), &standbyJSON)

	// set standby_cluster to default config unless already set
	if _, ok := configJSON["standby_cluster"]; !ok {
		configJSON["standby_cluster"] = standbyJSON
	}

	configJSONFinalStr, err := json.Marshal(configJSON)
	if err != nil {
		return err
	}
	dcsConfigMap.ObjectMeta.Annotations["config"] = string(configJSONFinalStr)
	_, err = clientset.CoreV1().ConfigMaps(namespace).Update(ctx, dcsConfigMap, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	leaderConfigMapName := cluster.Name + "-leader"
	// Delete the "leader" configMap
	if err = clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, leaderConfigMapName, metav1.DeleteOptions{}); err != nil &&
		!kerrors.IsNotFound(err) {
		log.Error("Unable to delete configMap %s while enabling standby mode for cluster "+
			"%s: %v", leaderConfigMapName, clusterName, err)
		return err
	}

	// override to the repo type to ensure s3/gcs is utilized for standby creation
	pghaConfigMapName := cluster.Name + "-pgha-config"
	pghaConfigMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, pghaConfigMapName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Unable to find configMap %s when attempting to enable standby",
			pghaConfigMapName)
	}

	repoType := crv1.BackrestStorageTypeS3

	for _, r := range cluster.Spec.BackrestStorageTypes {
		if r == crv1.BackrestStorageTypeGCS {
			repoType = crv1.BackrestStorageTypeGCS
		}
	}

	pghaConfigMap.Data[operator.PGHAConfigReplicaBootstrapRepoType] = string(repoType)

	// delete the DCS config so that it will refresh with the included standby settings
	delete(pghaConfigMap.Data, fmt.Sprintf(cfg.PGHADCSConfigName, clusterName))

	if _, err := clientset.CoreV1().ConfigMaps(namespace).Update(ctx, pghaConfigMap, metav1.UpdateOptions{}); err != nil {
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

	// capture the cluster creation event
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
