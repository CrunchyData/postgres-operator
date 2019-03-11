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
	//"k8s.io/api/core/v1"
	"k8s.io/api/storage/v1"
	//kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetStorageClasses gets a list of StorageClasses
func GetStorageClasses(clientset *kubernetes.Clientset) (*v1.StorageClassList, error) {

	lo := meta_v1.ListOptions{}

	scs, err := clientset.StorageV1().StorageClasses().List(lo)
	if err != nil {
		log.Error(err)
		return scs, err
	}

	return scs, err
}
