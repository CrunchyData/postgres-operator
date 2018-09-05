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

// CreateConfigMap creates a ConfigMap
func CreateConfigMap(clientset *kubernetes.Clientset, configMap *v1.ConfigMap, namespace string) error {

	result, err := clientset.CoreV1().ConfigMaps(namespace).Create(configMap)
	if err != nil {
		log.Error("error creating configMap " + err.Error() + " in namespace " + namespace)
		return err
	}

	log.Debug("created ConfigMap " + result.Name)

	return err

}

// GetConfigMap gets a ConfigMap by name
func GetConfigMap(clientset *kubernetes.Clientset, name, namespace string) (*v1.ConfigMap, bool) {
	cfg, err := clientset.CoreV1().ConfigMaps(namespace).Get(name, meta_v1.GetOptions{})
	if kerrors.IsNotFound(err) {
		log.Error(err)
		log.Error(name + " configmap not found ")
		return cfg, false
	}
	if err != nil {
		log.Error(err)
		return cfg, false
	}

	return cfg, true
}

// DeleteConfigMap deletes a ConfigMap by name
func DeleteConfigMap(clientset *kubernetes.Clientset, name, namespace string) error {

	err := clientset.CoreV1().ConfigMaps(namespace).Delete(name, &meta_v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting ConfigMap " + err.Error())
		return err
	}

	log.Info("deleted ConfigMap " + name)

	return err

}
