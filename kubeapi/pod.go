package kubeapi

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
	log "github.com/Sirupsen/logrus"
	"k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

//TODO include GetLogs as used in pvcimpl.go
