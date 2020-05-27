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
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetSecrets gets a list of Secrets by selector
func GetSecrets(clientset *kubernetes.Clientset, selector, namespace string) (*v1.SecretList, error) {

	lo := meta_v1.ListOptions{LabelSelector: selector}

	secrets, err := clientset.CoreV1().Secrets(namespace).List(lo)
	if err != nil {
		log.Error(err)
		log.Error("error getting secrets selector=[" + selector + "]")
		return secrets, err
	}

	return secrets, err
}

// GetSecret gets a Secrets by name
func GetSecret(clientset *kubernetes.Clientset, name, namespace string) (*v1.Secret, error) {
	secret, err := clientset.CoreV1().Secrets(namespace).Get(name, meta_v1.GetOptions{})

	if err != nil {
		log.Error(err)
		return nil, err
	}

	return secret, nil
}

// CreateSecret
func CreateSecret(clientset *kubernetes.Clientset, secret *v1.Secret, namespace string) error {

	_, err := clientset.CoreV1().Secrets(namespace).Create(secret)
	if err != nil {
		log.Error(err)
		log.Error("error creating secret " + secret.Name)
		return err
	}
	log.Debugf("created secret %s", secret.Name)

	return nil
}

// DeleteSecret
func DeleteSecret(clientset *kubernetes.Clientset, name, namespace string) error {

	err := clientset.CoreV1().Secrets(namespace).Delete(name, &meta_v1.DeleteOptions{})
	if err != nil {
		log.Error(err)
		log.Error("error deleting secret " + name)
	} else {
		log.Debugf("deleted secret %s", name)
	}

	return err
}

func UpdateSecret(clientset *kubernetes.Clientset, sec *v1.Secret, namespace string) error {
	_, err := clientset.CoreV1().Secrets(namespace).Update(sec)
	if err != nil {
		log.Error(err)
		log.Error("error updating secret %s", sec.Name)
	}
	return err

}
