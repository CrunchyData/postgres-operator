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
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetServiceAccount gets a ServiceAccount by name
func GetServiceAccount(clientset *kubernetes.Clientset, name, namespace string) (*v1.ServiceAccount, bool, error) {

	sa, err := clientset.Core().ServiceAccounts(namespace).Get(name, meta_v1.GetOptions{})
	if kerrors.IsNotFound(err) {
		return sa, false, err
	}

	if err != nil {
		log.Error(err)
		return sa, false, err
	}

	return sa, true, err
}

// DeleteServiceAccount
func DeleteServiceAccount(clientset *kubernetes.Clientset, name, namespace string) error {

	err := clientset.Core().ServiceAccounts(namespace).Delete(name, &meta_v1.DeleteOptions{})
	if err != nil {
		log.Error(err)
		log.Error("error deleting sa " + name)
	} else {
		log.Debugf("deleted sa %s", name)
	}

	return err
}

func UpdateServiceAccount(clientset *kubernetes.Clientset, sec *v1.ServiceAccount, namespace string) error {
	_, err := clientset.Core().ServiceAccounts(namespace).Update(sec)
	if err != nil {
		log.Error(err)
		log.Error("error updating sa %s", sec.Name)
	}
	return err

}

// CreateServiceAccount
func CreateServiceAccount(clientset *kubernetes.Clientset, sa *v1.ServiceAccount, namespace string) error {

	_, err := clientset.Core().ServiceAccounts(namespace).Create(sa)
	if err != nil {
		log.Error(err)
		log.Error("error creating sa " + sa.Name)
	}
	log.Debugf("created sa %s", sa.Name)

	return err
}
