// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

/*
 Copyright 2019 - 2021 Crunchy Data Solutions, Inc.
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
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	pgo "github.com/crunchydata/postgres-operator/pkg/generated/clientset/versioned"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func Failover(identifier string, clientset kubeapi.Interface, clusterName string, task *crv1.Pgtask, namespace string, restconfig *rest.Config) error {
	ctx := context.TODO()

	var pod *v1.Pod
	var err error
	target := task.ObjectMeta.Labels[config.LABEL_TARGET]

	log.Infof("Failover called on [%s] target [%s]", clusterName, target)

	pod, err = util.GetPod(clientset, target, namespace)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Debugf("pod selected to failover to is %s", pod.Name)

	updateFailoverStatus(clientset, task, namespace, "deleted primary deployment "+clusterName)

	// trigger the failover to the selected replica
	if err := promote(pod, clientset, namespace, restconfig); err != nil {
		log.Warn(err)
	}

	updateFailoverStatus(clientset, task, namespace, "promoting pod "+pod.Name+" target "+target)

	// relabel the deployment with primary labels
	// by setting service-name=clustername
	upod, err := clientset.CoreV1().Pods(namespace).Get(ctx, pod.Name, metav1.GetOptions{})
	if err != nil {
		log.Error(err)
		log.Error("error in getting pod during failover relabel")
		return err
	}

	// set the service-name label to the cluster name to match
	// the primary service selector
	log.Debugf("setting label on pod %s=%s", config.LABEL_SERVICE_NAME, clusterName)

	patch, err := kubeapi.NewMergePatch().Add("metadata", "labels", config.LABEL_SERVICE_NAME)(clusterName).Bytes()
	if err == nil {
		log.Debugf("patching pod %s: %s", upod.Name, patch)
		_, err = clientset.CoreV1().Pods(namespace).
			Patch(ctx, upod.Name, types.MergePatchType, patch, metav1.PatchOptions{})
	}
	if err != nil {
		log.Error(err)
		log.Error("error in updating pod during failover relabel")
		return err
	}

	targetDepName := upod.ObjectMeta.Labels[config.LABEL_DEPLOYMENT_NAME]
	log.Debugf("patching deployment %s: %s", targetDepName, patch)
	_, err = clientset.AppsV1().Deployments(namespace).
		Patch(ctx, targetDepName, types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		log.Error(err)
		log.Error("error in updating deployment during failover relabel")
		return err
	}

	updateFailoverStatus(clientset, task, namespace, "updating label deployment...pod "+pod.Name+"was the failover target...failover completed")

	// update the pgcluster current-primary to new deployment name
	cluster, err := clientset.CrunchydataV1().Pgclusters(namespace).Get(ctx, clusterName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("could not find pgcluster %s with labels", clusterName)
		return err
	}

	// update the CRD with the new current primary. If there is an error, log it
	// here, otherwise return
	if err := util.CurrentPrimaryUpdate(clientset, cluster, target, namespace); err != nil {
		log.Error(err)
		return err
	}

	return nil
}

func updateFailoverStatus(clientset pgo.Interface, task *crv1.Pgtask, namespace, message string) {
	ctx := context.TODO()

	log.Debugf("updateFailoverStatus namespace=[%s] taskName=[%s] message=[%s]", namespace, task.Name, message)

	// update the task
	t, err := clientset.CrunchydataV1().Pgtasks(task.Namespace).Get(ctx, task.Name, metav1.GetOptions{})
	if err != nil {
		return
	}
	*task = *t

	task.Status.Message = message

	t, err = clientset.CrunchydataV1().Pgtasks(task.Namespace).Update(ctx, task, metav1.UpdateOptions{})
	if err != nil {
		return
	}
	*task = *t
}

func promote(
	pod *v1.Pod,
	clientset kubernetes.Interface,
	namespace string, restconfig *rest.Config) error {
	// generate the curl command that will be run on the pod selected for the failover in order
	// to trigger the failover and promote that specific pod to primary
	command := make([]string, 3)
	command[0] = "/bin/bash"
	command[1] = "-c"
	command[2] = fmt.Sprintf("curl -s http://127.0.0.1:%s/failover -XPOST "+
		"-d '{\"candidate\":\"%s\"}'", config.DEFAULT_PATRONI_PORT, pod.Name)

	log.Debugf("running Exec with namespace=[%s] podname=[%s] container name=[%s]", namespace, pod.Name, pod.Spec.Containers[0].Name)
	stdout, stderr, err := kubeapi.ExecToPodThroughAPI(restconfig, clientset, command, pod.Spec.Containers[0].Name, pod.Name, namespace, nil)
	log.Debugf("stdout=[%s] stderr=[%s]", stdout, stderr)
	if err != nil {
		log.Error(err)
	}

	return err
}

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
	} else if len(pods.Items) > 1 {
		log.Error("More than one primary found after completing the post-failover backup")
	}
	pod := pods.Items[0]

	// generate the curl command that will be run on the pod selected for the failover in order
	// to trigger the failover and promote that specific pod to primary
	command := make([]string, 3)
	command[0] = "/bin/bash"
	command[1] = "-c"
	command[2] = fmt.Sprintf("curl -s 127.0.0.1:%s/config -XPATCH -d "+
		"'{\"tags\":{\"primary_on_role_change\":null}}'", config.DEFAULT_PATRONI_PORT)

	log.Debugf("running Exec command '%s' with namespace=[%s] podname=[%s] container name=[%s]",
		command, namespace, pod.Name, pod.Spec.Containers[0].Name)
	stdout, stderr, err := kubeapi.ExecToPodThroughAPI(restconfig, clientset, command,
		pod.Spec.Containers[0].Name, pod.Name, namespace, nil)
	log.Debugf("stdout=[%s] stderr=[%s]", stdout, stderr)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}
