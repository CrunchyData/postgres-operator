package kubeapi

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
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetNamespaces gets a list of Namespaces
func GetNamespaces(clientset *kubernetes.Clientset) (*v1.NamespaceList, error) {

	lo := meta_v1.ListOptions{}

	namespaces, err := clientset.CoreV1().Namespaces().List(lo)
	if err != nil {
		log.Error(err)
		return namespaces, err
	}

	return namespaces, err
}

// GetNamespace gets a Namespace by name
func GetNamespace(clientset *kubernetes.Clientset, name string) (*v1.Namespace, bool, error) {
	ns, err := clientset.CoreV1().Namespaces().Get(name, meta_v1.GetOptions{})
	if kerrors.IsNotFound(err) {
		return ns, false, err
	}
	if err != nil {
		return ns, false, err
	}

	return ns, true, err
}
