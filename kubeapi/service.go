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
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DeleteService deletes a Service
func DeleteService(clientset *kubernetes.Clientset, name, namespace string) error {
	err := clientset.Core().Services(namespace).Delete(name, &meta_v1.DeleteOptions{})
	if err != nil {
		log.Error(err)
		log.Error("error deleting Service " + name)
	}
	log.Info("delete service " + name)
	return err
}

// GetServices gets a list of Services by selector
func GetServices(clientset *kubernetes.Clientset, selector, namespace string) (*v1.ServiceList, error) {

	lo := meta_v1.ListOptions{LabelSelector: selector}

	services, err := clientset.CoreV1().Services(namespace).List(lo)
	if err != nil {
		log.Error(err)
		log.Error("error getting services selector=[" + selector + "]")
		return services, err
	}

	return services, err
}

// GetService gets a Service by name
func GetService(clientset *kubernetes.Clientset, name, namespace string) (*v1.Service, bool, error) {
	svc, err := clientset.CoreV1().Services(namespace).Get(name, meta_v1.GetOptions{})
	if kerrors.IsNotFound(err) {
		return svc, false, err
	}
	if err != nil {
		log.Error(err)
		return svc, false, err
	}

	return svc, true, err
}

// CreateService creates a Service
func CreateService(clientset *kubernetes.Clientset, svc *v1.Service, namespace string) (*v1.Service, error) {
	result, err := clientset.Core().Services(namespace).Create(svc)
	if err != nil {
		log.Error(err)
		log.Error("error creating service " + svc.Name)
		return result, err
	}

	log.Info("created service " + result.Name)
	return result, err
}

func UpdateService(clientset *kubernetes.Clientset, svc *v1.Service, namespace string) error {
	_, err := clientset.Core().Services(namespace).Update(svc)
	if err != nil {
		log.Error(err)
		log.Error("error updating service %s", svc.Name)
	}
	return err

}
