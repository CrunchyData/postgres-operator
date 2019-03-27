package util

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
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"time"
)

// WaitUntilPod ...
//podPhase is v1.PodRunning
//timeout := time.Minute
func WaitUntilPod(clientset *kubernetes.Clientset, lo meta_v1.ListOptions, podPhase v1.PodPhase, timeout time.Duration, namespace string) error {

	var err error
	var fw watch.Interface

	fw, err = clientset.CoreV1().Pods(namespace).Watch(lo)
	if err != nil {
		log.Error("error watching pods " + err.Error())
		return err
	}

	conditions := []watch.ConditionFunc{
		func(event watch.Event) (bool, error) {
			log.Debug("watch Modified called")
			gotpod2 := event.Object.(*v1.Pod)
			log.Debugf("pod2 phase=%s", gotpod2.Status.Phase)
			if gotpod2.Status.Phase == podPhase {
				return true, nil
			}
			return false, nil
		},
	}

	log.Debug("before watch.Until")

	var lastEvent *watch.Event
	lastEvent, err = watch.Until(timeout, fw, conditions...)
	if err != nil {
		log.Errorf("timeout waiting for %v error=%s", podPhase, err.Error())
		return err
	}
	if lastEvent == nil {
		log.Error("expected event")
		return err
	}
	log.Debug("after watch.Until")
	return err

}

// WaitUntilPodIsDeleted timeout := time.Minute
func WaitUntilPodIsDeleted(clientset *kubernetes.Clientset, podname string, timeout time.Duration, namespace string) error {

	var err error
	var fw watch.Interface

	lo := meta_v1.ListOptions{LabelSelector: "pg-cluster=" + podname}
	fw, err = clientset.CoreV1().Pods(namespace).Watch(lo)
	if err != nil {
		log.Error("error watching pods 2 " + err.Error())
		return err
	}

	conditions := []watch.ConditionFunc{
		func(event watch.Event) (bool, error) {
			if event.Type == watch.Deleted {
				log.Debug("pod delete event received in WaitUntilPodIsDeleted")
				return true, nil
			}
			return false, nil
		},
	}

	var lastEvent *watch.Event
	lastEvent, err = watch.Until(timeout, fw, conditions...)
	if err != nil {
		log.Error("timeout waiting for Running " + err.Error())
		return err
	}
	if lastEvent == nil {
		log.Error("expected event")
		return err
	}
	return err

}

// WaitUntilDeploymentIsDeleted timeout := time.Minute
func WaitUntilDeploymentIsDeleted(clientset *kubernetes.Clientset, depname string, timeout time.Duration, namespace string) error {

	var err error
	var fw watch.Interface

	lo := meta_v1.ListOptions{LabelSelector: "name=" + depname}
	fw, err = clientset.ExtensionsV1beta1().Deployments(namespace).Watch(lo)
	if err != nil {
		log.Error("error watching deployments " + err.Error())
		return err
	}

	conditions := []watch.ConditionFunc{
		func(event watch.Event) (bool, error) {
			log.Infof("waiting for deployment to be deleted got event=%v\n", event.Type)
			if event.Type == watch.Deleted {
				log.Info("deployment delete event received in WaitUntilDeploymentIsDeleted")
				return true, nil
			}
			return false, nil
		},
	}

	var lastEvent *watch.Event
	lastEvent, err = watch.Until(timeout, fw, conditions...)
	if err != nil {
		log.Error("timeout waiting for deployment to be deleted " + depname + err.Error())
		return err
	}
	if lastEvent == nil {
		log.Error("expected event")
		return err
	}
	return err

}
