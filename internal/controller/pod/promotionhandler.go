package pod

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
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/controller"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/operator/backrest"
	clusteroperator "github.com/crunchydata/postgres-operator/internal/operator/cluster"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	// isInRecoveryCommand is the command run to determine if postgres is in recovery
	isInRecoveryCMD []string = []string{"psql", "-t", "-c", "'SELECT pg_is_in_recovery();'", "-p"}

	// leaderStatusCMD is the command run to get the Patroni status for the primary
	leaderStatusCMD []string = []string{"curl", fmt.Sprintf("localhost:%s/master",
		config.DEFAULT_PATRONI_PORT)}

	// isStandbyDisabledTick is the duration of the tick used when waiting for standby mode to
	// be disabled
	isStandbyDisabledTick time.Duration = time.Millisecond * 500

	// isStandbyDisabledTimeout is the amount of time to wait before timing out when waitig for
	// standby mode to be disabled
	isStandbyDisabledTimeout time.Duration = time.Minute * 5
)

// handlePostgresPodPromotion is responsible for handling updates to PG pods the occur as a result
// of a failover.  Specifically, this handler is triggered when a replica has been promoted, and
// it now has either the "promoted" or "primary" role label.
func (c *Controller) handlePostgresPodPromotion(newPod *apiv1.Pod, cluster crv1.Pgcluster) error {

	if cluster.Status.State == crv1.PgclusterStateShutdown {
		if err := c.handleStartupInit(cluster); err != nil {
			return err
		}
	}

	// create a post-failover backup if not a standby cluster
	if !cluster.Spec.Standby && cluster.Status.State == crv1.PgclusterStateInitialized {
		if err := cleanAndCreatePostFailoverBackup(c.Client,
			cluster.Name, newPod.Namespace); err != nil {
			log.Error(err)
			return err
		}
	}

	return nil
}

// handleStartupInit is resposible for handling cluster initilization for a cluster that has been
// restarted (after it was previously shutdown)
func (c *Controller) handleStartupInit(cluster crv1.Pgcluster) error {

	// since the cluster is just being restarted, it can just be set to initialized once the
	// primary is ready
	if err := controller.SetClusterInitializedStatus(c.Client, cluster.Name,
		cluster.Namespace); err != nil {
		log.Error(err)
		return err
	}

	// now scale any replicas deployments to 1
	clusteroperator.ScaleClusterDeployments(c.Client, cluster, 1, false, true, false, false)

	return nil
}

// handleStandbyPodPromotion is responsible for handling updates to PG pods the occur as a result
// of disabling standby mode.  Specifically, this handler is triggered when a standby leader
// is turned into a regular leader.
func (c *Controller) handleStandbyPromotion(newPod *apiv1.Pod, cluster crv1.Pgcluster) error {

	clusterName := cluster.Name
	namespace := cluster.Namespace

	if err := waitForStandbyPromotion(c.Client.Config, c.Client, *newPod, cluster); err != nil {
		return err
	}

	// rotate the exporter password if the metrics sidecar is enabled
	if cluster.Spec.Exporter {
		if err := clusteroperator.RotateExporterPassword(c.Client, c.Client.Config, &cluster); err != nil {
			log.Error(err)
			return err
		}
	}

	// rotate the pgBouncer passwords if pgbouncer is enabled within the cluster
	if cluster.Spec.PgBouncer.Enabled() {
		if err := clusteroperator.RotatePgBouncerPassword(c.Client, c.Client.Config, &cluster); err != nil {
			log.Error(err)
			return err
		}
	}

	if err := cleanAndCreatePostFailoverBackup(c.Client, clusterName, namespace); err != nil {
		log.Error(err)
		return err
	}

	return nil
}

// waitForStandbyPromotion waits for standby mode to be disabled for a specific cluster and has
// been promoted.  This is done by verifying that recovery is no longer enabled in the database,
// while also ensuring there are not any pending restarts for the database.
// done by confirming
func waitForStandbyPromotion(restConfig *rest.Config, clientset kubernetes.Interface, newPod apiv1.Pod,
	cluster crv1.Pgcluster) error {

	var recoveryDisabled bool

	// wait for the server to accept writes to ensure standby has truly been disabled before
	// proceeding
	if err := wait.Poll(isStandbyDisabledTick, isStandbyDisabledTimeout, func() (bool, error) {
		if !recoveryDisabled {
			cmd := isInRecoveryCMD
			cmd = append(cmd, cluster.Spec.Port)

			isInRecoveryStr, _, _ := kubeapi.ExecToPodThroughAPI(restConfig, clientset,
				cmd, "database", newPod.Name, newPod.Namespace, nil)

			recoveryDisabled = strings.Contains(isInRecoveryStr, "f")

			if !recoveryDisabled {
				return false, nil
			}
		}

		primaryJSONStr, _, _ := kubeapi.ExecToPodThroughAPI(restConfig, clientset,
			leaderStatusCMD, newPod.Spec.Containers[0].Name, newPod.Name,
			newPod.Namespace, nil)

		primaryJSON := map[string]interface{}{}
		_ = json.Unmarshal([]byte(primaryJSONStr), &primaryJSON)

		return (primaryJSON["state"] == "running" && (primaryJSON["pending_restart"] == nil ||
			!primaryJSON["pending_restart"].(bool))), nil
	}); err != nil {
		return fmt.Errorf("timed out waiting for cluster %s to accept writes after disabling "+
			"standby mode", cluster.Name)
	}

	return nil
}

// cleanAndCreatePostFailoverBackup cleans up any existing backup resources and then creates
// a pgtask to trigger the creation of a post-failover backup
func cleanAndCreatePostFailoverBackup(clientset kubeapi.Interface, clusterName, namespace string) error {
	ctx := context.TODO()

	//look up the backrest-repo pod name
	selector := fmt.Sprintf("%s=%s,%s=true", config.LABEL_PG_CLUSTER,
		clusterName, config.LABEL_PGO_BACKREST_REPO)
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if len(pods.Items) != 1 {
		return fmt.Errorf("pods len != 1 for cluster %s", clusterName)
	} else if err != nil {
		return err
	}

	if err := backrest.CleanBackupResources(clientset, namespace,
		clusterName); err != nil {
		log.Error(err)
		return err
	}
	if _, err := backrest.CreatePostFailoverBackup(clientset, namespace,
		clusterName, pods.Items[0].Name); err != nil {
		log.Error(err)
		return err
	}

	return nil
}
