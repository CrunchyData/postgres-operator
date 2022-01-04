package operator

/*
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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
	"fmt"

	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Switchover performs a controlled switchover within a PostgreSQL cluster, i.e.
// demoting a primary and promoting a replica. There are two types of switchover
// methods that can be invoked.
//
// Method #1: Automatic Choice
//
// The switchover command invokves Patroni which works as such:
//
// 1. The function looks for all available replicas as well as the current
// primary. We look up the primary for convenience to avoid various API calls
//
// 2. We then search over the list to find both a primary and a suitable
// candidate for promotion. A candidate is suitable if:
//
//   - It is on the latest timeline
//   - It has the least amount of replication lag
//
// This is done to limit the risk of data loss.
//
// If either a primary or candidate is **not** found, we do not switch over.
//
// 3. If all of the above works successfully, a switchover is attempted.
//
// Method #2: Targeted Choice
//
// 1. If the "target" parameter, which should contain the name of the target
// instances (Deployment), is not empty then we will attempt to locate that
// target Pod.
//
// 2. The target Pod name, called the candidate is passed into the switchover
// command generation function, and then is ultimately used in the switchover.
func Switchover(clientset kubernetes.Interface, restConfig *rest.Config, cluster *crv1.Pgcluster, target string) error {
	// the method to get the pod is dictated by whether or not there is a target
	// specified.
	//
	// If target is specified, then we will attempt to get the Pod that
	// represents that target.
	//
	// If it is not specified, then we will attempt to get any Pod.
	//
	// If either errors, we will return an error
	candidate := ""
	pod, err := getCandidatePod(clientset, cluster, target)

	if err != nil {
		return err
	}

	if target != "" {
		candidate = pod.Name
	}

	// generate the command
	cmd := generatePostgresSwitchoverCommand(cluster.Name, candidate)

	// good to generally log which instances are being used in the switchover
	log.Infof("controlled switchover started for cluster %q", cluster.Name)

	if _, stderr, err := kubeapi.ExecToPodThroughAPI(restConfig, clientset,
		cmd, "database", pod.Name, cluster.Namespace, nil); err != nil {
		return fmt.Errorf(stderr)
	}

	log.Infof("controlled switchover completed for cluster %q", cluster.Name)

	// and that's all
	return nil
}

// generatePostgresSwitchoverCommand creates the command that is used to issue
// a switchover (demote a primary, promote a replica).
//
// There are two ways to run this command:
//
// 1. Pass in only a clusterName. Patroni will select the best candidate
// 2. Pass in a clusterName AND a target candidate name, which has to be the
//    name of a Pod
func generatePostgresSwitchoverCommand(clusterName, candidate string) []string {
	cmd := []string{"patronictl", "switchover", "--force", clusterName}

	if candidate != "" {
		cmd = append(cmd, "--candidate", candidate)
	}

	return cmd
}
