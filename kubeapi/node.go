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
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetNodes gets a list of all Nodes
func GetAllNodes(clientset *kubernetes.Clientset) (*v1.NodeList, error) {
	nodes, err := clientset.CoreV1().Nodes().List(meta_v1.ListOptions{})
	if err != nil {
		log.Error(err)
	}
	return nodes, err

}

// GetNodes gets a list of Nodes by selector
func GetNodes(clientset *kubernetes.Clientset, selector, namespace string) (*v1.NodeList, error) {

	lo := meta_v1.ListOptions{LabelSelector: selector}

	nodes, err := clientset.CoreV1().Nodes().List(lo)
	if err != nil {
		log.Error(err)
		log.Error("error getting nodes selector=[" + selector + "]")
		return nodes, err
	}

	return nodes, err
}
