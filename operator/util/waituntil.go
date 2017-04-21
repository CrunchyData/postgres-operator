/*
 Copyright 2017 Crunchy Data Solutions, Inc.
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

package util

import (
	log "github.com/Sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/watch"
	"time"
)

//lo := v1.ListOptions{LabelSelector: "pg-database=" + "testpod"}
//podPhase is v1.PodRunning
//timeout := time.Minute
func WaitUntilPod(clientset *kubernetes.Clientset, lo v1.ListOptions, podPhase v1.PodPhase, timeout time.Duration, namespace string) error {

	var err error
	var fw watch.Interface

	fw, err = clientset.Core().Pods(namespace).Watch(lo)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	conditions := []watch.ConditionFunc{
		func(event watch.Event) (bool, error) {
			log.Info("watch Modified called")
			gotpod2 := event.Object.(*v1.Pod)
			log.Info("pod2 phase=" + gotpod2.Status.Phase)
			if gotpod2.Status.Phase == podPhase {
				return true, nil
			}
			//return event.Type == watch.Modified, nil
			return false, nil
		},
	}

	log.Info("before watch.Until")

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
	log.Info("after watch.Until")
	return err

}

//timeout := time.Minute
func WaitUntilPodIsDeleted(clientset *kubernetes.Clientset, podname string, timeout time.Duration, namespace string) error {

	var err error
	var fw watch.Interface

	lo := v1.ListOptions{LabelSelector: "pg-database=" + podname}
	fw, err = clientset.Core().Pods(namespace).Watch(lo)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	conditions := []watch.ConditionFunc{
		func(event watch.Event) (bool, error) {
			if event.Type == watch.Deleted {
				log.Info("pod delete event received in WaitUntilPodIsDeleted")
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

//timeout := time.Minute
func WaitUntilDeploymentIsDeleted(clientset *kubernetes.Clientset, depname string, timeout time.Duration, namespace string) error {

	var err error
	var fw watch.Interface

	lo := v1.ListOptions{LabelSelector: "name=" + depname}
	fw, err = clientset.Deployments(namespace).Watch(lo)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	conditions := []watch.ConditionFunc{
		func(event watch.Event) (bool, error) {
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
		log.Error("timeout waiting for Running " + err.Error())
		return err
	}
	if lastEvent == nil {
		log.Error("expected event")
		return err
	}
	return err

}

//timeout := time.Minute
func WaitUntilReplicasetIsDeleted(clientset *kubernetes.Clientset, rcname string, timeout time.Duration, namespace string) error {

	var err error
	var fw watch.Interface

	lo := v1.ListOptions{LabelSelector: "name=" + rcname}
	fw, err = clientset.ReplicaSets(namespace).Watch(lo)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	conditions := []watch.ConditionFunc{
		func(event watch.Event) (bool, error) {
			if event.Type == watch.Deleted {
				log.Info("ReplicaSets delete event received in WaitUntilReplicasetIsDeleted")
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
