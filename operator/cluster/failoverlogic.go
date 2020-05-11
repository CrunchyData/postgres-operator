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
	"fmt"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/events"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func Failover(identifier string, clientset *kubernetes.Clientset, client *rest.RESTClient, clusterName string, task *crv1.Pgtask, namespace string, restconfig *rest.Config) error {

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

	updateFailoverStatus(client, task, namespace, clusterName, "deleted primary deployment "+clusterName)

	//trigger the failover to the selected replica
	if err := promote(pod, clientset, client, namespace, restconfig); err != nil {
		log.Warn(err)
	}

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
	log.Debugf("targetDepName %s", targetDepName)
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

	// update the CRD with the new current primary. If there is an error, log it
	// here, otherwise return
	if err := util.CurrentPrimaryUpdate(client, &cluster, target, namespace); err != nil {
		log.Error(err)
		return err
	}

	return nil

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
