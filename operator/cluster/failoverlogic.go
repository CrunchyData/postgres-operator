// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

/*
 Copyright 2019 - 2020 Crunchy Data Solutions, Inc.
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
	"errors"
	"fmt"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/events"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func Failover(identifier string, clientset *kubernetes.Clientset, client *rest.RESTClient, clusterName string, task *crv1.Pgtask, namespace string, restconfig *rest.Config) error {

	var pod *v1.Pod
	var err error
	target := task.ObjectMeta.Labels[config.LABEL_TARGET]

	log.Info("strategy 1 Failover called on " + clusterName + " target is " + target)

	pod, err = util.GetPod(clientset, target, namespace)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Debugf("pod selected to failover to is %s", pod.Name)

	updateFailoverStatus(client, task, namespace, clusterName, "deleted primary deployment "+clusterName)

	//trigger the failover to the selected replica
	err = promote(pod, clientset, client, namespace, restconfig)

	publishPromoteEvent(identifier, namespace, task.ObjectMeta.Labels[config.LABEL_PGOUSER], clusterName, target)

	updateFailoverStatus(client, task, namespace, clusterName, "promoting pod "+pod.Name+" target "+target)

	//relabel the deployment with primary labels
	//by setting service-name=clustername
	var upod *v1.Pod
	upod, _, err = kubeapi.GetPod(clientset, pod.Name, namespace)
	if err != nil {
		log.Error(err)
		log.Error("error in getting pod during failover relabel")
		return err
	}

	//set the service-name label to the cluster name to match
	//the primary service selector
	log.Debugf("setting label on pod %s=%s", config.LABEL_SERVICE_NAME, clusterName)

	err = kubeapi.AddLabelToPod(clientset, upod, config.LABEL_SERVICE_NAME, clusterName, namespace)
	if err != nil {
		log.Error(err)
		log.Error("error in updating pod during failover relabel")
		return err
	}

	targetDepName := upod.ObjectMeta.Labels[config.LABEL_DEPLOYMENT_NAME]
	log.Debug("targetDepName %s", targetDepName)
	var targetDep *appsv1.Deployment
	targetDep, _, err = kubeapi.GetDeployment(clientset, targetDepName, namespace)
	if err != nil {
		log.Error(err)
		log.Errorf("not found error in getting Deployment during failover relabel %s", targetDepName)
		return err
	}

	err = kubeapi.AddLabelToDeployment(clientset, targetDep, config.LABEL_SERVICE_NAME, clusterName, namespace)
	if err != nil {
		log.Error(err)
		log.Error("error in updating deployment during failover relabel")
		return err
	}

	updateFailoverStatus(client, task, namespace, clusterName, "updating label deployment...pod "+pod.Name+"was the failover target...failover completed")

	//update the pgcluster current-primary to new deployment name
	var found bool
	cluster := crv1.Pgcluster{}
	found, err = kubeapi.Getpgcluster(client, &cluster, clusterName, namespace)
	if !found {
		log.Errorf("could not find pgcluster %s with labels", clusterName)
		return err
	}
	cluster.Spec.UserLabels[config.LABEL_CURRENT_PRIMARY] = targetDepName
	err = util.PatchClusterCRD(client, cluster.Spec.UserLabels, &cluster, namespace)
	if err != nil {
		log.Errorf("failoverlogic: could not patch pgcluster %s with labels", clusterName)
		return err
	}

	return err

}

func updateFailoverStatus(client *rest.RESTClient, task *crv1.Pgtask, namespace, clusterName, message string) {

	log.Debugf("updateFailoverStatus namespace=[%s] taskName=[%s] message=[%s]", namespace, task.Name, message)

	//update the task
	_, err := kubeapi.Getpgtask(client, task, task.ObjectMeta.Name,
		task.ObjectMeta.Namespace)
	if err != nil {
		return
	}

	task.Status.Message = message

	err = kubeapi.Updatepgtask(client,
		task,
		task.ObjectMeta.Name,
		task.ObjectMeta.Namespace)
	if err != nil {
		return
	}

}

func deletePrimary(clientset *kubernetes.Clientset, namespace, clusterName, pgouser string) error {

	//the primary will be the one with a pod that has a label
	//that looks like service-name=clustername and is not a backrest job
	selector := config.LABEL_SERVICE_NAME + "=" + clusterName + "," + config.LABEL_BACKREST_RESTORE + "!=true," + config.LABEL_BACKREST_JOB + "!=true"

	// wait for single primary pod to exist.
	pods, success := waitForSinglePrimary(clientset, selector, namespace)

	if !success {
		log.Errorf("Received false while waiting for single primary, count: ", len(pods.Items))
		return errors.New("Couldn't isolate single primary pod")
	}

	//update the label to 'fenced' on the pod to fence off traffic from
	//any client or replica using the primary, this effectively
	//stops traffic from the Primary service to the primary pod
	//we are about to delete
	pod := pods.Items[0]

	deploymentToDelete := pod.ObjectMeta.Labels[config.LABEL_DEPLOYMENT_NAME]

	publishPrimaryDeleted(pod.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER], clusterName, deploymentToDelete, pgouser, namespace)

	//delete the deployment with pg-cluster=clusterName,primary=true
	log.Debugf("deleting deployment %s", deploymentToDelete)
	err := kubeapi.DeleteDeployment(clientset, deploymentToDelete, namespace)

	err = waitForDelete(deploymentToDelete, pod.Name, clientset, namespace)

	return err
}

func promote(
	pod *v1.Pod,
	clientset *kubernetes.Clientset,
	client *rest.RESTClient, namespace string, restconfig *rest.Config) error {

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

func waitForDelete(deploymentToDelete, podName string, clientset *kubernetes.Clientset, namespace string) error {
	var tries = 10

	for i := 0; i < tries; i++ {
		pod, _, err := kubeapi.GetPod(clientset, podName, namespace)
		if kerrors.IsNotFound(err) {
			log.Debugf("%s deployment %s pod not found so its safe to proceed on failover", deploymentToDelete, podName)
			return nil
		} else if err != nil {
			log.Error(err)
			log.Error("error getting pod when evaluating old primary in failover %s %s", deploymentToDelete, podName)
			return err
		}
		log.Debugf("waiting for %s to delete", pod.Name)
		time.Sleep(time.Second * time.Duration(9))
	}

	return errors.New(fmt.Sprintf("timeout waiting for %s %s to delete", deploymentToDelete, podName))

}

// waitForSinglePrimary - during failover, there can exist the possibility that while one pod is in the process of
// terminating, the Deployment will be spinning up another pod - both will appear to be a primary even though the
// terminating pod will not be accessible via the service. This method gets the primary and if both exists, it waits to
// give the terminating pod a chance to complete. If a single primary is never isolated, it returns false with the count
// of primaries found when it gave up. The number of tries and duration can be increased if needed - max wait time is
// tries * duration.
func waitForSinglePrimary(clientset *kubernetes.Clientset, selector, namespace string) (*v1.PodList, bool) {

	var tries = 5
	var duration = 2 // seconds
	var pods *v1.PodList

	for i := 0; i < tries; i++ {

		pods, _ = kubeapi.GetPods(clientset, selector, namespace)

		if len(pods.Items) > 1 {
			log.Errorf("more than 1 primary pod found when looking for primary %s", selector)
			log.Debug("Waiting in case a pod is terminating...")
			// return errors.New("more than 1 primary pod found in delete primary logic")
			time.Sleep(time.Second * time.Duration(duration))
		} else if len(pods.Items) == 0 {
			log.Errorf("No pods found for primary deployment")
			return pods, false
		} else {
			log.Debug("Found single pod for primary deployment")
			return pods, true
		}
	}

	return pods, false
}

func publishPromoteEvent(identifier, namespace, username, clusterName, target string) {
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventFailoverClusterFormat{
		EventHeader: events.EventHeader{
			Namespace: namespace,
			Username:  username,
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventFailoverCluster,
		},
		Clustername: clusterName,
		Target:      target,
	}

	err := events.Publish(f)
	if err != nil {
		log.Error(err.Error())
	}

}
func publishPrimaryDeleted(identifier, clusterName, deploymentToDelete, username, namespace string) {
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventPrimaryDeletedFormat{
		EventHeader: events.EventHeader{
			Namespace: namespace,
			Username:  username,
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventPrimaryDeleted,
		},
		Clustername:    clusterName,
		Deploymentname: deploymentToDelete,
	}

	err := events.Publish(f)
	if err != nil {
		log.Error(err.Error())
	}
}

// RemovePrimaryOnRoleChangeTag sets the 'primary_on_role_change' tag to null in the
// Patroni DCS, effectively removing the tag.  This is accomplished by exec'ing into
// the primary PG pod, and sending a patch request to update the appropriate data (i.e.
// the 'primary_on_role_change' tag) in the DCS.
func RemovePrimaryOnRoleChangeTag(clientset *kubernetes.Clientset, restconfig *rest.Config,
	clusterName, namespace string) error {

	selector := config.LABEL_PG_CLUSTER + "=" + clusterName +
		"," + config.LABEL_PGHA_ROLE + "=" + "master"
	pods, err := kubeapi.GetPods(clientset, selector, namespace)
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
