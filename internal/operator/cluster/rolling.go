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
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/operator"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type deploymentType int

const (
	deploymentTypePrimary deploymentType = iota
	deploymentTypeReplica
)

const (
	rollingUpdatePeriod  = 4 * time.Second
	rollingUpdateTimeout = 60 * time.Second
)

// RollingUpdate performs a type of "rolling update" on a series of Deployments
// of a PostgreSQL cluster in an attempt to minimize downtime.
//
// The functions take a function that serves to update the contents of a
// Deployment.
//
// The rolling update is performed as such:
//
// 1. Each replica is updated. A replica is shut down and changes are applied
//    The Operator waits until the replica is back online (and/or a time period)
//    And moves on to the next one
// 2. A controlled switchover is performed. The Operator chooses the best
//    candidate replica for the switch over.
// 3. The former primary is then shut down and updated.
//
// If this is not a HA cluster, then the Deployment is just singly restarted
//
// Erroring during this process can be fun. If an error occurs within the middle
// of a rolling update, in order to avoid placing the cluster in an
// indeterminate state, most errors are just logged for later troubleshooting
func RollingUpdate(clientset kubernetes.Interface, restConfig *rest.Config, cluster *crv1.Pgcluster,
	updateFunc func(*crv1.Pgcluster, *appsv1.Deployment) error) error {
	log.Debugf("rolling update for cluster %q", cluster.Name)

	// we need to determine which deployments are replicas and which is the
	// primary. Note, that based on external factors, this can change during the
	// execution of this function, so this is our best guess at the time of the
	// rolling update being performed.
	//
	// Given the craziness of a distributed world, we may even unearth two
	// primaries, or no primaries! So we will need to gracefully handle that as
	// well
	//
	// We will get this through the Pod list as the role label is on the Pod
	instances, err := generateDeploymentTypeMap(clientset, cluster)
	// If we fail to generate the deployment type map, we just have to fail here.
	// We can't do any updates
	if err != nil {
		return err
	}

	// go through all of the replicas and perform the modifications
	for i := range instances[deploymentTypeReplica] {
		deployment := instances[deploymentTypeReplica][i]

		// Try to apply the update. If it returns an error during the process,
		// continue on to the next replica
		if err := applyUpdateToPostgresInstance(clientset, restConfig, cluster, deployment, updateFunc); err != nil {
			log.Error(err)
			continue
		}

		// Ensure that the replica comes back up and can be connected to, otherwise
		// keep moving on. This involves waiting for the Deployment to come back
		// up...
		if err := waitForDeploymentReady(clientset, deployment.Namespace, deployment.Name,
			rollingUpdatePeriod, rollingUpdateTimeout); err != nil {
			log.Warn(err)
		}

		// ...followed by wiating for the PostgreSQL instance to come back up
		if err := waitForPostgresInstance(clientset, restConfig, cluster, deployment,
			rollingUpdatePeriod, rollingUpdateTimeout); err != nil {
			log.Warn(err)
		}
	}

	// if there is at least one replica and only one primary, perform a controlled
	// switchover.
	//
	// if multiple primaries were found, we don't know how we would want to
	// properly switch over, so we will let Patroni make the decision in this case
	// as part of an uncontrolled failover. At this point, we should have eligible
	// replicas that have the updated Deployment state.
	if len(instances[deploymentTypeReplica]) > 0 && len(instances[deploymentTypePrimary]) == 1 {
		// if the switchover fails, warn that it failed but continue on
		if err := switchover(clientset, restConfig, cluster); err != nil {
			log.Warnf("switchover failed: %s", err.Error())
		}
	}

	// finally, go through the list of primaries (which should only be one...)
	// and apply the update. At this point we do not need to wait for anything,
	// as we should have either already promoted a new primary, or this is a
	// single instance cluster
	for i := range instances[deploymentTypePrimary] {
		if err := applyUpdateToPostgresInstance(clientset, restConfig, cluster,
			instances[deploymentTypePrimary][i], updateFunc); err != nil {
			log.Error(err)
		}
	}

	return nil
}

// applyUpdateToPostgresInstance performs an update on an individual PostgreSQL
// instance. It first ensures that the update can be applied. If it can, it will
// safely turn of the PostgreSQL instance before modifying the Deployment
// template.
func applyUpdateToPostgresInstance(clientset kubernetes.Interface, restConfig *rest.Config,
	cluster *crv1.Pgcluster, deployment appsv1.Deployment,
	updateFunc func(*crv1.Pgcluster, *appsv1.Deployment) error) error {
	ctx := context.TODO()

	// apply any updates, if they cannot be applied, then return an error here
	if err := updateFunc(cluster, &deployment); err != nil {
		return err
	}

	// Before applying the update, we want to explicitly stop PostgreSQL on each
	// instance. This prevents PostgreSQL from having to boot up in crash
	// recovery mode.
	//
	// If an error is returned, warn, but proceed with the function
	if err := stopPostgreSQLInstance(clientset, restConfig, deployment); err != nil {
		log.Warn(err)
	}

	// Perform the update.
	_, err := clientset.AppsV1().Deployments(deployment.Namespace).
		Update(ctx, &deployment, metav1.UpdateOptions{})

	return err
}

// generateDeploymentTypeMap takes a list of Deployments and determines what
// they represent: a primary (hopefully only one) or replicas
func generateDeploymentTypeMap(clientset kubernetes.Interface, cluster *crv1.Pgcluster) (map[deploymentType][]appsv1.Deployment, error) {
	ctx := context.TODO()

	// get a list of all of the instance deployments for the cluster
	deployments, err := operator.GetInstanceDeployments(clientset, cluster)
	if err != nil {
		return nil, err
	}

	options := metav1.ListOptions{
		LabelSelector: fields.AndSelectors(
			fields.OneTermEqualSelector(config.LABEL_PG_CLUSTER, cluster.Name),
			fields.OneTermEqualSelector(config.LABEL_PG_DATABASE, config.LABEL_TRUE),
		).String(),
	}

	pods, err := clientset.CoreV1().Pods(cluster.Namespace).List(ctx, options)
	// if we can't find any of the Pods, we can't make the proper determiniation
	if err != nil {
		return nil, err
	}

	// go through each Deployment and make a determination about its type. If we
	// ultimately cannot do that, treat the deployment as a "replica"
	instances := map[deploymentType][]appsv1.Deployment{
		deploymentTypePrimary: {},
		deploymentTypeReplica: {},
	}

	for i, deployment := range deployments.Items {
		for _, pod := range pods.Items {
			// if the Pod doesn't match, continue
			if deployment.Name != pod.ObjectMeta.GetLabels()[config.LABEL_DEPLOYMENT_NAME] {
				continue
			}

			// found matching Pod, determine if it's a primary or replica
			if pod.ObjectMeta.GetLabels()[config.LABEL_PGHA_ROLE] == config.LABEL_PGHA_ROLE_PRIMARY {
				instances[deploymentTypePrimary] = append(instances[deploymentTypePrimary], deployments.Items[i])
			} else {
				instances[deploymentTypeReplica] = append(instances[deploymentTypeReplica], deployments.Items[i])
			}

			// we found the (or at least a) matching Pod, so we can break the loop now
			break
		}
	}

	return instances, nil
}

// generatePostgresReadyCommand creates the command used to test if a PostgreSQL
// instance is ready
func generatePostgresReadyCommand(port string) []string {
	return []string{"pg_isready", "-p", port}
}

// generatePostgresSwitchoverCommand creates the command that is used to issue
// a switchover (demote a primary, promote a replica). Takes the name of the
// cluster; Patroni will choose the best candidate to switchover to
func generatePostgresSwitchoverCommand(clusterName string) []string {
	return []string{"patronictl", "switchover", "--force", clusterName}
}

// switchover performs a controlled switchover within a PostgreSQL cluster, i.e.
// demoting a primary and promoting a replica. The method works as such:
//
// 1. The function looks for all available replicas as well as the current
// primary. We look up the primary for convenience to avoid various API calls
//
// 2. We then search over the list to find both a primary and a suitable
// candidate for promotion. A candidate is suitable if:
//   - It is on the latest timeline
//   - It has the least amount of replication lag
//
// This is done to limit the risk of data loss.
//
// If either a primary or candidate is **not** found, we do not switch over.
//
// 3. If all of the above works successfully, a switchover is attempted.
func switchover(clientset kubernetes.Interface, restConfig *rest.Config, cluster *crv1.Pgcluster) error {
	// we want to find a Pod to execute the switchover command on, i.e. the
	// primary
	pod, err := util.GetPrimaryPod(clientset, cluster)
	if err != nil {
		return err
	}

	// good to generally log which instances are being used in the switchover
	log.Infof("controlled switchover started for cluster %q", cluster.Name)

	cmd := generatePostgresSwitchoverCommand(cluster.Name)
	if _, stderr, err := kubeapi.ExecToPodThroughAPI(restConfig, clientset,
		cmd, "database", pod.Name, cluster.Namespace, nil); err != nil {
		return fmt.Errorf(stderr)
	}

	log.Infof("controlled switchover completed for cluster %q", cluster.Name)

	// and that's all
	return nil
}

// waitForPostgresInstance waits for a PostgreSQL instance within a Pod is ready
// to accept connections
func waitForPostgresInstance(clientset kubernetes.Interface, restConfig *rest.Config,
	cluster *crv1.Pgcluster, deployment appsv1.Deployment, periodSecs, timeoutSecs time.Duration) error {
	ctx := context.TODO()

	// try to find the Pod that should be exec'd into
	options := metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("status.phase", string(v1.PodRunning)).String(),
		LabelSelector: fields.AndSelectors(
			fields.OneTermEqualSelector(config.LABEL_PG_CLUSTER, cluster.Name),
			fields.OneTermEqualSelector(config.LABEL_PG_DATABASE, config.LABEL_TRUE),
			fields.OneTermEqualSelector(config.LABEL_DEPLOYMENT_NAME, deployment.Name),
		).String(),
	}
	pods, err := clientset.CoreV1().Pods(deployment.Namespace).List(ctx, options)

	// if the Pod selection errors, we can't really proceed
	if err != nil {
		return fmt.Errorf("could not find pods to check postgres instance readiness: %w", err)
	} else if len(pods.Items) == 0 {
		return fmt.Errorf("could not find any postgres pods")
	}

	// get the first pod...we'll just have to presume this is the active primary
	// as we've done all we good to narrow it down at this point
	pod := pods.Items[0]
	cmd := generatePostgresReadyCommand(cluster.Spec.Port)

	// set up the timer and timeout
	// first, ensure that there is an available Pod
	timeout := time.After(timeoutSecs)
	tick := time.NewTicker(periodSecs)
	defer tick.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("readiness timeout reached for start up of cluster %q instance %q", cluster.Name, deployment.Name)
		case <-tick.C:
			// check to see if PostgreSQL is ready to accept connections
			s, _, _ := kubeapi.ExecToPodThroughAPI(restConfig, clientset,
				cmd, "database", pod.Name, pod.Namespace, nil)

			// really we should find a way to get the exit code in the future, but
			// in the interim...
			if strings.Contains(s, "accepting connections") {
				return nil
			}
		}
	}
}
