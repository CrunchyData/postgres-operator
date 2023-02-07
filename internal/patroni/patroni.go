package patroni

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
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
)

// dbContainerName is the name of the container containing the PG database in a PG (primary or
// replica) pod
const dbContainerName = "database"

var (
	// reloadCMD is the command for reloading a specific PG instance (primary or
	// replica) within a Postgres cluster. It requires a cluster and instance name
	// to be appended to it
	reloadCMD = []string{"patronictl", "reload", "--force"}
	// restartCMD is the command for restart a specific PG database (primary or
	// replica) within a Postgres cluster. It requires a cluster and instance name
	// to be appended to it.
	restartCMD = []string{"patronictl", "restart", "--force"}

	// ErrInstanceNotFound is the error thrown when a target instance cannot be found in the cluster
	ErrInstanceNotFound = errors.New("The instance does not exist in the cluster")
)

// Client defines the various actions a Patroni client is able to perform against a specified
// PGCluster
type Client interface {
	ReloadCluster() error
	RestartCluster() ([]RestartResult, error)
	RestartInstances(instance ...string) ([]RestartResult, error)
}

// patroniClient represents a Patroni client that is able to perform various Patroni actions
// within specific PG Cluster.  The actions available correspond to the endpoints exposed by the
// Patroni REST API, as well the associated commands available via the 'patronictl' client.
type patroniClient struct {
	restConfig    *rest.Config
	kubeclientset kubernetes.Interface
	clusterName   string
	namespace     string
}

// RestartResult represents the result of a cluster restart, specifically the name of the
// an instance that was restarted within a cluster, and an error that can be populated in
// the event an instance cannot be successfully restarted.
type RestartResult struct {
	Instance string
	Error    error
}

// NewPatroniClient creates a new Patroni client
func NewPatroniClient(restConfig *rest.Config, kubeclientset kubernetes.Interface,
	clusterName, namespace string) Client {
	return &patroniClient{
		restConfig:    restConfig,
		kubeclientset: kubeclientset,
		clusterName:   clusterName,
		namespace:     namespace,
	}
}

// getClusterInstances returns a map primary
func (p *patroniClient) getClusterInstances() (map[string]corev1.Pod, error) {
	ctx := context.TODO()

	// selector in the format "pg-cluster=<cluster-name>,any role"
	selector := fmt.Sprintf("%s=%s,%s", config.LABEL_PG_CLUSTER, p.clusterName,
		config.LABEL_PG_DATABASE)
	instances, err := p.kubeclientset.CoreV1().Pods(p.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, err
	}

	instanceMap := make(map[string]corev1.Pod)

	for _, instance := range instances.Items {
		instanceMap[instance.GetObjectMeta().GetLabels()[config.LABEL_DEPLOYMENT_NAME]] = instance
	}

	return instanceMap, nil
}

// ReloadCluster reloads the configuration for a PostgreSQL cluster.  Specififcally, a Patroni
// reload (which includes a PG reload) is executed on the primary and each replica within the cluster.
func (p *patroniClient) ReloadCluster() error {
	instanceMap, err := p.getClusterInstances()
	if err != nil {
		return err
	}

	for _, instancePod := range instanceMap {
		if err := p.reload(instancePod.GetName()); err != nil {
			return err
		}
	}

	return nil
}

// ReloadCluster restarts all PostgreSQL databases within a PostgreSQL cluster.  Specififcally, a
// Patroni restart is executed on the primary and each replica within the cluster.  A slice is also
// returned containing the names of all instances restarted within the cluster.
func (p *patroniClient) RestartCluster() ([]RestartResult, error) {
	var restartResult []RestartResult

	instanceMap, err := p.getClusterInstances()
	if err != nil {
		return nil, err
	}

	for instance, instancePod := range instanceMap {
		if err := p.restart(instancePod.GetName()); err != nil {
			restartResult = append(restartResult, RestartResult{
				Instance: instance,
				Error:    err,
			})
			continue
		}
		restartResult = append(restartResult, RestartResult{Instance: instance})
	}

	return restartResult, nil
}

// RestartInstances restarts the PostgreSQL databases for the instances specified.  Specififcally, a
// Patroni restart is executed on the primary and each replica within the cluster.
func (p *patroniClient) RestartInstances(instances ...string) ([]RestartResult, error) {
	var restartResult []RestartResult

	instanceMap, err := p.getClusterInstances()
	if err != nil {
		return nil, err
	}

	targetInstanceMap := make(map[string]corev1.Pod)

	// verify the targets specified (if any are specified) actually exist in the cluster
	for _, instance := range instances {
		if _, ok := instanceMap[instance]; ok {
			targetInstanceMap[instance] = instanceMap[instance]
		} else {
			restartResult = append(restartResult, RestartResult{
				Instance: instance,
				Error:    ErrInstanceNotFound,
			})
		}
	}

	for instance, instancePod := range targetInstanceMap {
		if err := p.restart(instancePod.GetName()); err != nil {
			restartResult = append(restartResult, RestartResult{
				Instance: instance,
				Error:    err,
			})
			continue
		}
		restartResult = append(restartResult, RestartResult{Instance: instance})
	}

	return restartResult, nil
}

// reload performs a Patroni reload (which includes a PG reload) on a specific instance (primary or
// replica) within a PG cluster
func (p *patroniClient) reload(podName string) error {
	cmd := reloadCMD
	cmd = append(cmd, p.clusterName, podName)

	stdout, stderr, err := kubeapi.ExecToPodThroughAPI(p.restConfig, p.kubeclientset,
		cmd, dbContainerName, podName, p.namespace, nil)

	if err != nil {
		return fmt.Errorf(stderr)
	}

	log.Debugf("Successfully reloaded PG on pod %s: %s", podName, stdout)

	return nil
}

// restart performs a Patroni restart on a specific instance (primary or replica) within a PG
// cluster.
func (p *patroniClient) restart(podName string) error {
	cmd := restartCMD
	cmd = append(cmd, p.clusterName, podName)

	stdout, stderr, err := kubeapi.ExecToPodThroughAPI(p.restConfig, p.kubeclientset, cmd,
		dbContainerName, podName, p.namespace, nil)
	if err != nil {
		return err
	} else if stderr != "" {
		return fmt.Errorf(stderr)
	}

	log.Debugf("Successfully restarted PG on pod %s: %s", podName, stdout)

	return err
}
