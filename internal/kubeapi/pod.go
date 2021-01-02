package kubeapi

/*
 Copyright 2018 - 2021 Crunchy Data Solutions, Inc.
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
	"encoding/json"

	jsonpatch "github.com/evanphx/json-patch"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

func AddLabelToPod(clientset kubernetes.Interface, origPod *v1.Pod, key, value, namespace string) error {
	var newData, patchBytes []byte
	var err error

	//get the original data before we change it
	origData, err := json.Marshal(origPod)
	if err != nil {
		return err
	}

	origPod.ObjectMeta.Labels[key] = value

	newData, err = json.Marshal(origPod)
	if err != nil {
		return err
	}

	patchBytes, err = jsonpatch.CreateMergePatch(origData, newData)
	if err != nil {
		return err
	}

	_, err = clientset.CoreV1().Pods(namespace).Patch(origPod.Name, types.MergePatchType, patchBytes)
	if err != nil {
		log.Error(err)
		log.Errorf("error add label to Pod  %s %s=%s", origPod.Name, key, value)
	}
	log.Debugf("add label to Pod %s %s=%v", origPod.Name, key, value)
	return err
}
