package util

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
	//crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"k8s.io/api/core/v1"
	//"k8s.io/apimachinery/pkg/api/errors"
	"errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/rest"
	//"math/rand"
	//"strings"
	//"time"
)

// GetBestTarget
func GetBestTarget(clientset *kubernetes.Clientset, clusterName, namespace string) (*v1.Pod, error) {

	var err error
	var pod *v1.Pod
	return pod, err
}

// GetPodName from a deployment name
func GetPod(clientset *kubernetes.Clientset, deploymentName, namespace string) (*v1.Pod, error) {

	var err error

	var pod *v1.Pod
	var pods *v1.PodList
	lo := meta_v1.ListOptions{LabelSelector: "replica-name=" + deploymentName}
	pods, err = clientset.CoreV1().Pods(namespace).List(lo)
	if err != nil {
		log.Error(err)
		return pod, err
	}
	if len(pods.Items) != 1 {
		return pod, errors.New("could not determine which pod to failover to")
	}

	for _, v := range pods.Items {
		pod = &v
	}
	if len(pod.Spec.Containers) != 1 {
		return pod, errors.New("could not find a container in the pod")
	}

	return pod, err
}
