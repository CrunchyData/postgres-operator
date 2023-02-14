package cluster

/*
 Copyright 2020 - 2023 Crunchy Data Solutions, Inc.
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
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
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

const rollingUpdateMaxRetries = 5

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
// If "rescale" is selected, then each Deployment is scaled to 0 after the
// Postgres cluster is shut down, but before the changes are applied. This is
// normally not needed, but invoked on operations where an object must be
// completely unconsumed by a resource (e.g. during a change to a PVC).
//
// Erroring during this process can be fun. If an error occurs within the middle
// of a rolling update, in order to avoid placing the cluster in an
// indeterminate state, most errors are just logged for later troubleshooting
func RollingUpdate(clientset kubeapi.Interface, restConfig *rest.Config,
	cluster *crv1.Pgcluster, rescale bool,
	updateFunc func(kubeapi.Interface, *crv1.Pgcluster, *appsv1.Deployment) error) error {
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

		if err := applyUpdateToPostgresInstanceWithRetries(clientset, restConfig, cluster,
			deployment, rescale, updateFunc); err != nil {
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
		if err := waitForPostgresInstanceReady(clientset, restConfig, cluster, deployment,
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
		if err := operator.Switchover(clientset, restConfig, cluster, ""); err != nil {
			log.Warnf("switchover failed: %s", err.Error())
		}
	}

	// finally, go through the list of primaries (which should only be one...)
	// and apply the update. At this point we do not need to wait for anything,
	// as we should have either already promoted a new primary, or this is a
	// single instance cluster
	for i := range instances[deploymentTypePrimary] {
		if err := applyUpdateToPostgresInstanceWithRetries(clientset, restConfig, cluster,
			instances[deploymentTypePrimary][i], rescale, updateFunc); err != nil {
			log.Error(err)
			continue
		}
	}

	return nil
}

// applyUpdateToPostgresInstance performs an update on an individual PostgreSQL
// instance. It first ensures that the update can be applied. If it can, it will
// safely turn of the PostgreSQL instance before modifying the Deployment
// template.
func applyUpdateToPostgresInstance(clientset kubeapi.Interface, restConfig *rest.Config,
	cluster *crv1.Pgcluster, deployment *appsv1.Deployment, rescale bool,
	updateFunc func(kubeapi.Interface, *crv1.Pgcluster, *appsv1.Deployment) error) error {
	var err error
	replicas := new(int32)
	ctx := context.TODO()

	// apply any updates, if they cannot be applied, then return an error here
	if err := updateFunc(clientset, cluster, deployment); err != nil {
		return err
	}

	// Before applying the update, we want to explicitly stop PostgreSQL on each
	// instance. This prevents PostgreSQL from having to boot up in crash
	// recovery mode.
	//
	// If an error is returned, warn, but proceed with the function
	if err := StopPostgreSQLInstance(clientset, restConfig, *deployment); err != nil {
		log.Warn(err)
	}

	// if "rescale" is selected, scale the deployment down to 0.
	if rescale {
		// store the original total of replicas required for scaling back up
		replicas = deployment.Spec.Replicas
		deployment.Spec.Replicas = new(int32)
	}

	// Perform the update.
	deployment, err = clientset.AppsV1().Deployments(deployment.Namespace).
		Update(ctx, deployment, metav1.UpdateOptions{})

	// if the update fails, return an error
	if err != nil {
		return err
	}

	// if we're rescaling, ensure that the scale down finished and then scale back
	// up. if something goes wrong, warn that it did but proceed onward as we may
	// not know exactly why the scaling failed.
	if rescale {
		if err := waitForPostgresInstanceTermination(clientset, cluster, deployment,
			rollingUpdatePeriod, rollingUpdateTimeout); err != nil {
			log.Warn(err)
		}

		if err := operator.ScaleDeployment(clientset, deployment, replicas); err != nil {
			log.Warn(err)
		}
	}

	return nil
}

// applyUpdateToPostgresInstanceWithRetries calls the
// applyUpdateToPostgresInstance function, but allows for it to retry if there
// are any failures
func applyUpdateToPostgresInstanceWithRetries(clientset kubeapi.Interface, restConfig *rest.Config,
	cluster *crv1.Pgcluster, deployment *appsv1.Deployment, rescale bool,
	updateFunc func(kubeapi.Interface, *crv1.Pgcluster, *appsv1.Deployment) error) error {
	ctx := context.TODO()

	// Try to apply the update. If it returns an error during the process,
	// determine if the error is a conflict. If it is, try again for a few
	// times.
	//
	// If not, try again
	for i := 0; i < rollingUpdateMaxRetries; i++ {
		err := applyUpdateToPostgresInstance(clientset, restConfig, cluster,
			deployment, rescale, updateFunc)

		if err == nil {
			break
		}

		// if the error is anything other than a conflict, log the error and
		// continue through the loop
		if !kerrors.IsConflict(err) {
			return err
		}

		// if the error is a conflict and the next time through the loop is the
		// max number of retries, log that we are giving up.
		if i+1 >= rollingUpdateMaxRetries {
			log.Error(err)
			return fmt.Errorf("abandoning updating instance %s", deployment.Name)
		}

		// because this is a conflict, reload the deployment
		// if the reload errors, let's go through the retry loop again with the
		// same deployment object
		if d, err := clientset.AppsV1().Deployments(deployment.Namespace).Get(ctx,
			deployment.Name, metav1.GetOptions{}); err == nil {
			deployment = d
		}
	}

	return nil
}

// generateDeploymentTypeMap takes a list of Deployments and determines what
// they represent: a primary (hopefully only one) or replicas
func generateDeploymentTypeMap(clientset kubernetes.Interface, cluster *crv1.Pgcluster) (map[deploymentType][]*appsv1.Deployment, error) {
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
	instances := map[deploymentType][]*appsv1.Deployment{
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
				instances[deploymentTypePrimary] = append(instances[deploymentTypePrimary], &deployments.Items[i])
			} else {
				instances[deploymentTypeReplica] = append(instances[deploymentTypeReplica], &deployments.Items[i])
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

// getPostgresPodsForDeployment attempts to get all of the running Pods (which
// should only be 0 or 1) that are in a Deployment
func getPostgresPodsForDeployment(clientset kubernetes.Interface, cluster *crv1.Pgcluster, deployment *appsv1.Deployment) (*v1.PodList, error) {
	ctx := context.TODO()
	options := metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("status.phase", string(v1.PodRunning)).String(),
		LabelSelector: fields.AndSelectors(
			fields.OneTermEqualSelector(config.LABEL_PG_CLUSTER, cluster.Name),
			fields.OneTermEqualSelector(config.LABEL_PG_DATABASE, config.LABEL_TRUE),
			fields.OneTermEqualSelector(config.LABEL_DEPLOYMENT_NAME, deployment.Name),
		).String(),
	}
	return clientset.CoreV1().Pods(deployment.Namespace).List(ctx, options)
}

// waitForPostgresInstanceReady waits for a PostgreSQL instance within a Pod is ready
// to accept connections
func waitForPostgresInstanceReady(clientset kubernetes.Interface, restConfig *rest.Config,
	cluster *crv1.Pgcluster, deployment *appsv1.Deployment, periodSecs, timeoutSecs time.Duration) error {
	// try to find the Pod that should be exec'd into
	pods, err := getPostgresPodsForDeployment(clientset, cluster, deployment)

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

	// start polling to test if the Postgres instance is available to accept
	// connections
	if err := wait.Poll(periodSecs, timeoutSecs, func() (bool, error) {
		// check to see if PostgreSQL is ready to accept connections
		s, _, _ := kubeapi.ExecToPodThroughAPI(restConfig, clientset,
			cmd, "database", pod.Name, pod.Namespace, nil)

		// really we should find a way to get the exit code in the future, but
		// in the interim, we know that we can accept connections if the below
		// string is present
		return strings.Contains(s, "accepting connections"), nil
	}); err != nil {
		return fmt.Errorf("readiness timeout reached for start up of cluster %q instance %q",
			cluster.Name, deployment.Name)
	}

	return nil
}

// waitForPostgresInstanceTermination waits for the Pod of a Postgres instance
// to be terminated...i.e. there are no Pods that are running that match the
// Postgres instance.
func waitForPostgresInstanceTermination(clientset kubernetes.Interface,
	cluster *crv1.Pgcluster, deployment *appsv1.Deployment, periodSecs, timeoutSecs time.Duration) error {
	// start polling to test if the Postgres instance is terminated
	if err := wait.PollImmediate(periodSecs, timeoutSecs, func() (bool, error) {
		// determine if there are any active Pods in the cluster. If there are more
		// than 1, then this instance has not yet terminated
		pods, err := getPostgresPodsForDeployment(clientset, cluster, deployment)

		return err == nil && len(pods.Items) == 0, nil
	}); err != nil {
		return fmt.Errorf("readiness timeout reached for termination of cluster %q instance %q",
			cluster.Name, deployment.Name)
	}

	return nil
}
