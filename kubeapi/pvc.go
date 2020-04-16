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
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

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
func GetPVC(clientset *kubernetes.Clientset, name, namespace string) (*v1.PersistentVolumeClaim, error) {
	options := meta_v1.GetOptions{}
	return clientset.CoreV1().PersistentVolumeClaims(namespace).Get(name, options)
}

// GetPVCIfExists gets a PVC by name. If the PVC does not exist, it returns nils.
func GetPVCIfExists(clientset *kubernetes.Clientset, name, namespace string) (*v1.PersistentVolumeClaim, error) {
	pvc, err := GetPVC(clientset, name, namespace)
	if err != nil {
		pvc = nil
	}
	if IsNotFound(err) {
		err = nil
	}
	return pvc, err
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

// IsPVCDeleted checks to see if a PVC has been deleted.  It will continuously check to
// see if the PVC has been deleted, only returning once the PVC is verified to have been
// deleted, or the timeout specified is reached.
func IsPVCDeleted(client *kubernetes.Clientset, timeout time.Duration, pvcName,
	namespace string) error {

	duration := time.After(timeout)
	tick := time.Tick(500 * time.Millisecond)
	for {
		select {
		case <-duration:
			return fmt.Errorf("timed out waiting for PVC to delete: %s", pvcName)
		case <-tick:
			if pvc, err := GetPVCIfExists(client, pvcName, namespace); err == nil && pvc == nil {
				return nil
			}
		}
	}
}
