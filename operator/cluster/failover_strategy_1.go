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
	"bytes"
	"encoding/json"
	"errors"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	jsonpatch "github.com/evanphx/json-patch"
	//remotecommandconsts "k8s.io/apimachinery/pkg/util/remotecommand"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	//"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/util"
	"k8s.io/api/extensions/v1beta1"
	//kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	//"text/template"
)

// AddCluster ...
func (r Strategy1) Failover(clientset *kubernetes.Clientset, client *rest.RESTClient, clusterName string, task *crv1.Pgtask, namespace string, restconfig *rest.Config) error {

	var pod *v1.Pod
	var err error
	target := task.ObjectMeta.Labels["target"]

	log.Info("strategy 1 Failover called on " + clusterName + " target is " + target)

	if target == "" {
		log.Debug("failover target not set, will use best estimate")
		pod, err = util.GetBestTarget(clientset, clusterName, namespace)
	} else {
		pod, err = util.GetPod(clientset, target, namespace)
	}
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
	err = promote(pod, clientset, client, namespace, target, restconfig)
	//if err != nil {
	//log.Error(err)
	//return err
	//}
	updateFailoverStatus(client, task, namespace, clusterName, "promoting replica"+target)

	//relabel the deployment with primary labels
	err = relabel(pod, clientset, namespace, clusterName, target)
	//if err != nil {
	//log.Error(err)
	////return err
	//}
	updateFailoverStatus(client, task, namespace, clusterName, "re-labeling replica")

	return err

}

func updateFailoverStatus(client *rest.RESTClient, task *crv1.Pgtask, namespace, clusterName, message string) {

	taskName := clusterName + "-failover"
	log.Debug("updateFailoverStatus namespace=[" + namespace + "] taskName=[" + task.Name + "] message=[" + message + "]")

	//update the task

	err := client.Get().
		Name(task.ObjectMeta.Name).
		Namespace(task.ObjectMeta.Namespace).
		Resource(crv1.PgtaskResourcePlural).
		Do().
		Into(task)
	if err != nil {
		log.Error("error getting pgtask for update " + taskName)
		log.Error(err)
		return
	}

	task.Status.Message = message

	err = client.Put().
		Name(task.ObjectMeta.Name).
		Namespace(task.ObjectMeta.Namespace).
		Resource(crv1.PgtaskResourcePlural).
		Body(task).
		Do().
		Error()
	if err != nil {
		log.Error("error updating pgtask message " + taskName)
		log.Error(err)
		return
	}

}

func deletePrimary(clientset *kubernetes.Clientset, namespace, clusterName string) error {
	var err error

	//delete the deployments
	delOptions := meta_v1.DeleteOptions{}
	var delProp meta_v1.DeletionPropagation
	delProp = meta_v1.DeletePropagationForeground
	delOptions.PropagationPolicy = &delProp

	log.Debug("deleting deployment " + clusterName)
	err = clientset.ExtensionsV1beta1().Deployments(namespace).Delete(clusterName, &delOptions)
	if err != nil {
		log.Error("error deleting primary Deployment " + err.Error())
	}

	return err
}

func promote(pod *v1.Pod, clientset *kubernetes.Clientset, client *rest.RESTClient, namespace, target string, restconfig *rest.Config) error {
	var err error

	//get the target pod that matches the replica-name=target

	command := make([]string, 1)
	command[0] = "/opt/cpm/bin/promote.sh"

	log.Debug("running Exec with namespace=[" + namespace + "] podname=[" + pod.Name + "] container name=[" + pod.Spec.Containers[0].Name + "]")
	err = util.Exec(restconfig, namespace, pod.Name, pod.Spec.Containers[0].Name, command)
	if err != nil {
		log.Error(err)
	}

	return err
}

func relabel(pod *v1.Pod, clientset *kubernetes.Clientset, namespace, clusterName, target string) error {
	var err error

	var targetDeployment *v1beta1.Deployment

	targetDeployment, err = clientset.ExtensionsV1beta1().Deployments(namespace).Get(target, meta_v1.GetOptions{})
	if err != nil {
		log.Error("error getting list of deployments" + err.Error())
		return err
	}

	//set replica=false on the deployment
	//set name=clustername on the deployment
	newLabels := make(map[string]string)
	newLabels["replica"] = "false"
	newLabels["name"] = clusterName

	err = updateLabels(namespace, clientset, targetDeployment, target, newLabels)
	if err != nil {
		log.Error(err)
	}

	err = updatePodLabels(namespace, clientset, pod, target, newLabels)
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

func promoteExperimental(clientset *kubernetes.Clientset, client *rest.RESTClient, namespace, target string, restconfig *rest.Config) error {
	var err error
	var execOut bytes.Buffer
	var execErr bytes.Buffer

	//get the target pod that matches the replica-name=target

	var pod v1.Pod
	var pods *v1.PodList
	lo := meta_v1.ListOptions{LabelSelector: "replica-name=" + target}
	pods, err = clientset.CoreV1().Pods(namespace).List(lo)
	if err != nil {
		log.Error(err)
		return err
	}
	if len(pods.Items) != 1 {
		return errors.New("could not determine which pod to failover to")
	}

	for _, v := range pods.Items {
		pod = v
	}
	if len(pod.Spec.Containers) != 1 {
		return errors.New("could not find a container in the pod")
	}

	command := make([]string, 1)
	command[0] = "/opt/cpm/bin/promote.sh"

	req := client.Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(namespace).
		SubResource("exec")
	req.VersionedParams(&v1.PodExecOptions{
		Container: pod.Spec.Containers[0].Name,
		Command:   command,
		Stdout:    true,
		Stderr:    true,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(restconfig, "POST", req.URL())
	if err != nil {
		log.Error("failed to init executor: %v", err)
		return err
	}

	err = exec.Stream(remotecommand.StreamOptions{
		//SupportedProtocols: remotecommandconsts.SupportedStreamingProtocols,
		Stdout: &execOut,
		Stderr: &execErr,
	})

	if err != nil {
		log.Error("could not execute: %v", err)
		return err
	}

	if execErr.Len() > 0 {
		log.Error("promote error stderr: %v", execErr.String())
		return errors.New("promote error stderr: " + execErr.String())
	}

	log.Debug("promote output [" + execOut.String() + "]")

	return err
}
