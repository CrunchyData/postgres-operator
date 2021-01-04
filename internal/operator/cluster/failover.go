// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

/*
 Copyright 2018 - 2021 Crunchy Data Solutions, Inc.
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

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var roleChangeCmd = []string{"patronictl", "edit-config", "--force",
	"--set", "tags.primary_on_role_change=null"}

// RemovePrimaryOnRoleChangeTag sets the 'primary_on_role_change' tag to null in the
// Patroni DCS, effectively removing the tag.  This is accomplished by exec'ing into
// the primary PG pod, and sending a patch request to update the appropriate data (i.e.
// the 'primary_on_role_change' tag) in the DCS.
func RemovePrimaryOnRoleChangeTag(clientset kubernetes.Interface, restconfig *rest.Config,
	clusterName, namespace string) error {
	ctx := context.TODO()

	selector := config.LABEL_PG_CLUSTER + "=" + clusterName +
		"," + config.LABEL_PGHA_ROLE + "=" + config.LABEL_PGHA_ROLE_PRIMARY

	// only consider pods that are running
	options := metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("status.phase", string(v1.PodRunning)).String(),
		LabelSelector: selector,
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, options)

	if err != nil {
		log.Error(err)
		return err
	} else if len(pods.Items) == 0 {
		return fmt.Errorf("no pods found for cluster %q", clusterName)
	} else if len(pods.Items) > 1 {
		log.Error("More than one primary found after completing the post-failover backup")
	}
	pod := pods.Items[0]

	// execute the command that will be run on the pod selected for the failover
	// in order to trigger the failover and promote that specific pod to primary
	log.Debugf("running Exec command '%s' with namespace=[%s] podname=[%s] container name=[%s]",
		roleChangeCmd, namespace, pod.Name, pod.Spec.Containers[0].Name)
	stdout, stderr, err := kubeapi.ExecToPodThroughAPI(restconfig, clientset, roleChangeCmd,
		pod.Spec.Containers[0].Name, pod.Name, namespace, nil)
	log.Debugf("stdout=[%s] stderr=[%s]", stdout, stderr)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}
