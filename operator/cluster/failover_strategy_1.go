// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

/*
 Copyright 2017-2018 Crunchy Data Solutions, Inc.
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
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	jsonpatch "github.com/evanphx/json-patch"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// AddCluster ...
func (r Strategy1) Failover(clientset *kubernetes.Clientset, client *rest.RESTClient, clusterName string, task *crv1.Pgtask, namespace string, restconfig *rest.Config) error {

	var pod *v1.Pod
	var err error
	target := task.ObjectMeta.Labels[util.LABEL_TARGET]

	log.Info("strategy 1 Failover called on " + clusterName + " target is " + target)

	pod, err = util.GetPod(clientset, target, namespace)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Debug("best pod to failover to is " + pod.Name)

	//delete the primary deployment
	err = deletePrimary(clientset, namespace, clusterName)
	if err != nil {
		log.Error(err)
		return err
	}
	updateFailoverStatus(client, task, namespace, clusterName, "deleting primary deployment "+clusterName)

	//trigger the failover on the replica
	err = promote(pod, clientset, client, namespace, restconfig)
	updateFailoverStatus(client, task, namespace, clusterName, "promoting pod "+pod.Name+" target "+target)

	//drain the deployment, this will shutdown the database pod
	err = kubeapi.PatchReplicas(clientset, target, namespace, "/spec/replicas", 0)
	if err != nil {
		log.Error(err)
		return err
	}

	//relabel the deployment with primary labels
	err = relabel(pod, clientset, namespace, clusterName, target)
	updateFailoverStatus(client, task, namespace, clusterName, "re-labeling deployment...pod "+pod.Name+"was the failover target...failover completed")

	//enable the deployment by making replicas equal to 1
	err = kubeapi.PatchReplicas(clientset, target, namespace, "/spec/replicas", 1)
	if err != nil {
		log.Error(err)
		return err
	}

	return err

}

func updateFailoverStatus(client *rest.RESTClient, task *crv1.Pgtask, namespace, clusterName, message string) {

	log.Debug("updateFailoverStatus namespace=[" + namespace + "] taskName=[" + task.Name + "] message=[" + message + "]")

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

	//delete the deployment with pg-cluster=clusterName,primary=true
	//should only be 1 primary with this name!
	deps, err := kubeapi.GetDeployments(clientset, util.LABEL_PG_CLUSTER+"="+clusterName+",primary=true", namespace)
	for _, d := range deps.Items {
		log.Debugf("deleting deployment %s\n", d.Name)
		kubeapi.DeleteDeployment(clientset, d.Name, namespace)
	}

	return err
}

func promote(
	pod *v1.Pod,
	clientset *kubernetes.Clientset,
	client *rest.RESTClient, namespace string, restconfig *rest.Config) error {

	//get the target pod that matches the replica-name=target

	command := make([]string, 1)
	command[0] = "/opt/cpm/bin/promote.sh"

	log.Debug("running Exec with namespace=[" + namespace + "] podname=[" + pod.Name + "] container name=[" + pod.Spec.Containers[0].Name + "]")
	stdout, stderr, err := kubeapi.ExecToPodThroughAPI(restconfig, clientset, command, pod.Spec.Containers[0].Name, pod.Name, namespace, nil)
	log.Debug("stdout=[" + stdout + "] stderr=[" + stderr + "]")
	if err != nil {
		log.Error(err)
	}

	return err
}

func relabel(pod *v1.Pod, clientset *kubernetes.Clientset, namespace, clusterName, target string) error {
	var err error

	targetDeployment, found, err := kubeapi.GetDeployment(clientset, target, namespace)
	if !found {
		return err
	}

	//set primary=true on the deployment
	//set name=clustername on the deployment
	newLabels := make(map[string]string)
	newLabels[util.LABEL_NAME] = clusterName
	newLabels[util.LABEL_PRIMARY] = "true"

	err = updateLabels(namespace, clientset, targetDeployment, target, newLabels)
	if err != nil {
		log.Error(err)
	}

	err = kubeapi.MergePatchDeployment(clientset, targetDeployment, clusterName, namespace)
	if err != nil {
		log.Error(err)
	}

	return err
}

// TODO this code came mostly from util/util.go...refactor to merge
func updateLabels(namespace string, clientset *kubernetes.Clientset, deployment *v1beta1.Deployment, clusterName string, newLabels map[string]string) error {

	var err error

	log.Debugf("%v is the labels to apply\n", newLabels)

	var patchBytes, newData, origData []byte
	origData, err = json.Marshal(deployment)
	if err != nil {
		return err
	}

	accessor, err2 := meta.Accessor(deployment)
	if err2 != nil {
		return err2
	}

	objLabels := accessor.GetLabels()
	if objLabels == nil {
		objLabels = make(map[string]string)
	}
	log.Debugf("current labels are %v\n", objLabels)

	//update the deployment labels
	for key, value := range newLabels {
		objLabels[key] = value
	}
	log.Debugf("updated labels are %v\n", objLabels)

	accessor.SetLabels(objLabels)

	newData, err = json.Marshal(deployment)
	if err != nil {
		return err
	}
	patchBytes, err = jsonpatch.CreateMergePatch(origData, newData)
	if err != nil {
		return err
	}

	_, err = clientset.ExtensionsV1beta1().Deployments(namespace).Patch(clusterName, types.MergePatchType, patchBytes, "")
	if err != nil {
		log.Debug("error patching deployment " + err.Error())
	}
	return err

}

func updatePodLabels(namespace string, clientset *kubernetes.Clientset, pod *v1.Pod, clusterName string, newLabels map[string]string) error {

	var err error

	log.Debugf("%v is the labels to apply\n", newLabels)

	var patchBytes, newData, origData []byte
	origData, err = json.Marshal(pod)
	if err != nil {
		return err
	}

	accessor, err2 := meta.Accessor(pod)
	if err2 != nil {
		return err2
	}

	objLabels := accessor.GetLabels()
	if objLabels == nil {
		objLabels = make(map[string]string)
	}
	log.Debugf("current labels are %v\n", objLabels)

	//update the pod labels
	for key, value := range newLabels {
		objLabels[key] = value
	}
	log.Debugf("updated labels are %v\n", objLabels)

	accessor.SetLabels(objLabels)

	newData, err = json.Marshal(pod)
	if err != nil {
		return err
	}
	patchBytes, err = jsonpatch.CreateMergePatch(origData, newData)
	if err != nil {
		return err
	}

	_, err = clientset.CoreV1().Pods(namespace).Patch(pod.Name, types.MergePatchType, patchBytes, "")
	if err != nil {
		log.Debug("error patching deployment " + err.Error())
	}
	return err

}

func validateDBContainer(pod *v1.Pod) bool {
	found := false

	for _, c := range pod.Spec.Containers {
		if c.Name == "database" {
			return true
		}
	}
	return found

}
