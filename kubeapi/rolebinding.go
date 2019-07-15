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
	"k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetRoleBinding gets a RoleBinding by name
func GetRoleBinding(clientset *kubernetes.Clientset, name, namespace string) (*v1.RoleBinding, bool, error) {

	roleBinding, err := clientset.Rbac().RoleBindings(namespace).Get(name, meta_v1.GetOptions{})
	if kerrors.IsNotFound(err) {
		return roleBinding, false, err
	}

	if err != nil {
		log.Error(err)
		return roleBinding, false, err
	}

	return roleBinding, true, err
}

// DeleteRoleBinding
func DeleteRoleBinding(clientset *kubernetes.Clientset, name, namespace string) error {

	err := clientset.Rbac().RoleBindings(namespace).Delete(name, &meta_v1.DeleteOptions{})
	if err != nil {
		log.Error(err)
		log.Error("error deleting roleBinding " + name)
	} else {
		log.Debugf("deleted roleBinding %s", name)
	}

	return err
}

func UpdateRoleBinding(clientset *kubernetes.Clientset, sec *v1.RoleBinding, namespace string) error {
	_, err := clientset.Rbac().RoleBindings(namespace).Update(sec)
	if err != nil {
		log.Error(err)
		log.Error("error updating roleBinding %s", sec.Name)
	}
	return err

}

// CreateRoleBinding
func CreateRoleBinding(clientset *kubernetes.Clientset, roleBinding *v1.RoleBinding, namespace string) error {

	_, err := clientset.Rbac().RoleBindings(namespace).Create(roleBinding)
	if err != nil {
		log.Error(err)
		log.Error("error creating rolebinding " + roleBinding.Name)
	}
	log.Debugf("created rolebinding %s", roleBinding.Name)

	return err
}
