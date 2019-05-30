// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

/*
 Copyright 2019 Crunchy Data Solutions, Inc.
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
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"time"
)

func Failover(clientset *kubernetes.Clientset, client *rest.RESTClient, clusterName string, task *crv1.Pgtask, namespace string, restconfig *rest.Config) error {

	var pod *v1.Pod
	var err error
	target := task.ObjectMeta.Labels[config.LABEL_TARGET]

	log.Info("strategy 1 Failover called on " + clusterName + " target is " + target)

	pod, err = util.GetPod(clientset, target, namespace)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Debugf("best pod to failover to is %s", pod.Name)

	//delete the primary deployment if it exists

	//in the autofail scenario, some user might accidentally remove
	//the primary deployment, this would cause an autofail to occur
	//so the deployment needs to be checked to be present before
	//we attempt to remove it...in a manual failover case, the
	//deployment should be found, and then you would proceed to remove
	//it

	selector := config.LABEL_PG_CLUSTER + "=" + clusterName + "," + config.LABEL_SERVICE_NAME + "=" + clusterName
	log.Debugf("selector in failover get deployments is %s", selector)
	var depList *appsv1.DeploymentList
	depList, err = kubeapi.GetDeployments(clientset, selector, namespace)
	if len(depList.Items) > 0 {
		log.Debug("in failover, the primary deployment is found before removal")
		err = deletePrimary(clientset, namespace, clusterName)
		if err != nil {
			log.Error(err)
			return err
		}
	} else {
		log.Debug("in failover, the primary deployment is NOT found so we will not attempt to remove it")
	}

	updateFailoverStatus(client, task, namespace, clusterName, "deleted primary deployment "+clusterName)

	//trigger the failover on the replica
	err = promote(pod, clientset, client, namespace, restconfig)
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

func deletePrimary(clientset *kubernetes.Clientset, namespace, clusterName string) error {

	//the primary will be the one with a pod that has a label
	//that looks like service-name=clustername and is not a backrest job
	selector := config.LABEL_SERVICE_NAME + "=" + clusterName + "," + config.LABEL_BACKREST_RESTORE + "!=true," + config.LABEL_BACKREST_JOB + "!=true"
	pods, err := kubeapi.GetPods(clientset, selector, namespace)
	if len(pods.Items) == 0 {
		log.Errorf("no primary pod found when trying to delete primary %s", selector)
		return errors.New("could not find primary pod")
	}
	if len(pods.Items) > 1 {
		log.Errorf("more than 1 primary pod found when trying to delete primary %s", selector)
		return errors.New("more than 1 primary pod found in delete primary logic")
	}

	//update the label to 'fenced' on the pod to fence off traffic from
	//any client or replica using the primary, this effectively
	//stops traffic from the Primary service to the primary pod
	//we are about to delete
	pod := pods.Items[0]

	deploymentToDelete := pod.ObjectMeta.Labels[config.LABEL_DEPLOYMENT_NAME]

	//delete the deployment with pg-cluster=clusterName,primary=true
	log.Debugf("deleting deployment %s", deploymentToDelete)
	err = kubeapi.DeleteDeployment(clientset, deploymentToDelete, namespace)

	err = waitForDelete(deploymentToDelete, pod.Name, clientset, namespace)

	return err
}

func promote(
	pod *v1.Pod,
	clientset *kubernetes.Clientset,
	client *rest.RESTClient, namespace string, restconfig *rest.Config) error {

	//get the target pod that matches the replica-name=target

	command := make([]string, 1)
	command[0] = "/opt/cpm/bin/promote.sh"

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
		log.Debugf("waitinf for %s to delete", pod.Name)
		time.Sleep(time.Second * time.Duration(9))
	}

	return errors.New(fmt.Sprintf("timeout waiting for %s %s to delete", deploymentToDelete, podName))

}
