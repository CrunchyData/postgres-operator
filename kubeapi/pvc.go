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

// CreatePVC creates a PVC
func CreatePVC(clientset *kubernetes.Clientset, pvc *v1.PersistentVolumeClaim, namespace string) error {

	result, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Create(pvc)
	if err != nil {
		log.Error("error creating pvc " + err.Error() + " in namespace " + namespace)
		return err
	}

	log.Debugf("created PVC %s", result.Name)

	return err

}

// GetPVCs gets a list of PVC by selector
func GetPVCs(clientset *kubernetes.Clientset, selector, namespace string) (*v1.PersistentVolumeClaimList, error) {

	lo := meta_v1.ListOptions{LabelSelector: selector}

	pvclist, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(lo)
	if err != nil {
		log.Error("error getting pvc list " + err.Error())
	}

	return pvclist, err

}

// GetPVC gets a PVC by name
// returns pvc, found=bool, error
func GetPVC(clientset *kubernetes.Clientset, name, namespace string) (*v1.PersistentVolumeClaim, bool, error) {

	options := meta_v1.GetOptions{}
	pvc, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(name, options)
	if kerrors.IsNotFound(err) {
		log.Debugf("PVC %s is not found", name)
		return pvc, false, err
	}

	if err != nil {
		log.Error("error getting pvc " + err.Error())
		return pvc, false, err
	}

	return pvc, true, err

}

// DeletePVC deletes a PVC by name
func DeletePVC(clientset *kubernetes.Clientset, name, namespace string) error {
	delOptions := meta_v1.DeleteOptions{}
	var delProp meta_v1.DeletionPropagation
	delProp = meta_v1.DeletePropagationForeground
	delOptions.PropagationPolicy = &delProp

	//err := clientset.CoreV1().PersistentVolumelaims(namespace).Delete(name, &meta_v1.DeleteOptions{})
	err := clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(name, &delOptions)
	if err != nil {
		log.Error("error deleting pvc " + err.Error())
		return err
	}

	log.Info("deleted PVC " + name)

	return err

}

// DeletePVCs deletes all PVCs by selector
func DeletePVCs(clientset *kubernetes.Clientset, selector, namespace string) error {
	delOptions := meta_v1.DeleteOptions{}
	var delProp meta_v1.DeletionPropagation
	delProp = meta_v1.DeletePropagationForeground
	delOptions.PropagationPolicy = &delProp

	lo := meta_v1.ListOptions{LabelSelector: selector}

	err := clientset.CoreV1().PersistentVolumeClaims(namespace).DeleteCollection(&delOptions, lo)
	if err != nil {
		log.Error("error deleting pvcs " + err.Error() + " with selector " + selector)
		return err
	}

	return err

}
