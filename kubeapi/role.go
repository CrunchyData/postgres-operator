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

// GetRole gets a Role by name
func GetRole(clientset *kubernetes.Clientset, name, namespace string) (*v1.Role, bool, error) {

	role, err := clientset.Rbac().Roles(namespace).Get(name, meta_v1.GetOptions{})
	if kerrors.IsNotFound(err) {
		return role, false, err
	}

	if err != nil {
		log.Error(err)
		return role, false, err
	}

	return role, true, err
}

// DeleteRole
func DeleteRole(clientset *kubernetes.Clientset, name, namespace string) error {

	err := clientset.Rbac().Roles(namespace).Delete(name, &meta_v1.DeleteOptions{})
	if err != nil {
		log.Error(err)
		log.Error("error deleting role " + name)
	} else {
		log.Debugf("deleted role %s", name)
	}

	return err
}

func UpdateRole(clientset *kubernetes.Clientset, sec *v1.Role, namespace string) error {
	_, err := clientset.Rbac().Roles(namespace).Update(sec)
	if err != nil {
		log.Error(err)
		log.Error("error updating role %s", sec.Name)
	}
	return err

}

// CreateRole
func CreateRole(clientset *kubernetes.Clientset, role *v1.Role, namespace string) error {

	_, err := clientset.Rbac().Roles(namespace).Create(role)
	if err != nil {
		log.Error(err)
		log.Error("error creating role " + role.Name)
	}
	log.Debugf("created role %s", role.Name)

	return err
}
