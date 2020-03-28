package kubeapi

/*
 Copyright 2018 - 2020 Crunchy Data Solutions, Inc.
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
	err := clientset.CoreV1().Pods(namespace).Delete(name, &meta_v1.DeleteOptions{})
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

	_, err = clientset.CoreV1().Pods(namespace).Patch(origPod.Name, types.MergePatchType, patchBytes)
	if err != nil {
		log.Error(err)
		log.Errorf("error add label to Pod  %s %s=%s", origPod.Name, key, value)
	}
	log.Debugf("add label to Pod %s %s=%v", origPod.Name, key, value)
	return err
}
