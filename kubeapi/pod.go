package kubeapi

/*
 Copyright 2017-2019 Crunchy Data Solutions, Inc.
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
	"io"

	jsonpatch "github.com/evanphx/json-patch"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// DeletePod deletes a Pod
func DeletePod(clientset *kubernetes.Clientset, name, namespace string) error {
	err := clientset.Core().Pods(namespace).Delete(name, &meta_v1.DeleteOptions{})
	if err != nil {
		log.Error(err)
		log.Error("error deleting Pod " + name)
	}
	log.Info("delete pod " + name)
	return err
}

// GetPods gets a list of Pods by selector
func GetPods(clientset *kubernetes.Clientset, selector, namespace string) (*v1.PodList, error) {

	lo := meta_v1.ListOptions{LabelSelector: selector}

	pods, err := clientset.CoreV1().Pods(namespace).List(lo)
	if err != nil {
		log.Error(err)
		log.Error("error getting pods selector=[" + selector + "]")
		return pods, err
	}

	return pods, err
}

// GetPodsWithBothSelectors gets a list of Pods by selector and field selector
func GetPodsWithBothSelectors(clientset *kubernetes.Clientset, selector, fieldselector, namespace string) (*v1.PodList, error) {

	lo := meta_v1.ListOptions{LabelSelector: selector, FieldSelector: fieldselector}

	pods, err := clientset.CoreV1().Pods(namespace).List(lo)
	if err != nil {
		log.Error(err)
		log.Error("error getting pods selector=[" + selector + "]")
		return pods, err
	}

	return pods, err
}

// GetPod gets a Pod by name
func GetPod(clientset *kubernetes.Clientset, name, namespace string) (*v1.Pod, bool, error) {
	svc, err := clientset.CoreV1().Pods(namespace).Get(name, meta_v1.GetOptions{})
	if kerrors.IsNotFound(err) {
		return svc, false, err
	}
	if err != nil {
		return svc, false, err
	}

	return svc, true, err
}

// CreatePod creates a Pod
func CreatePod(clientset *kubernetes.Clientset, svc *v1.Pod, namespace string) (*v1.Pod, error) {
	result, err := clientset.Core().Pods(namespace).Create(svc)
	if err != nil {
		log.Error(err)
		log.Error("error creating pod " + svc.Name)
		return result, err
	}

	log.Info("created pod " + result.Name)
	return result, err
}

func UpdatePod(clientset *kubernetes.Clientset, pod *v1.Pod, namespace string) error {
	_, err := clientset.Core().Pods(namespace).Update(pod)
	if err != nil {
		log.Error(err)
		log.Error("error updating pod %s", pod.Name)
	}
	return err

}

func AddLabelToPod(clientset *kubernetes.Clientset, origPod *v1.Pod, key, value, namespace string) error {
	var newData, patchBytes []byte
	var err error

	//get the original data before we change it
	origData, err := json.Marshal(origPod)
	if err != nil {
		return err
	}

	origPod.ObjectMeta.Labels[key] = value

	newData, err = json.Marshal(origPod)
	if err != nil {
		return err
	}

	patchBytes, err = jsonpatch.CreateMergePatch(origData, newData)
	if err != nil {
		return err
	}

	_, err = clientset.Core().Pods(namespace).Patch(origPod.Name, types.MergePatchType, patchBytes)
	if err != nil {
		log.Error(err)
		log.Errorf("error add label to Pod  %s %s=%s", origPod.Name, key, value)
	}
	log.Debugf("add label to Pod %s %s=%v", origPod.Name, key, value)
	return err
}

func GetLogs(client *kubernetes.Clientset, logOpts v1.PodLogOptions, out io.Writer, podName, ns string) error {
	req := client.CoreV1().Pods(ns).GetLogs(podName, &logOpts)

	readCloser, err := req.Stream()
	if err != nil {
		return err
	}

	defer readCloser.Close()
	_, err = io.Copy(out, readCloser)
	return err
}

//TODO include GetLogs as used in pvcimpl.go
