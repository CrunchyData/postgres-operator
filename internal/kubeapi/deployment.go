package kubeapi

/*
 Copyright 2018 - 2022 Crunchy Data Solutions, Inc.
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
	v1 "k8s.io/api/apps/v1"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

func AddLabelToDeployment(clientset kubernetes.Interface, origDeployment *v1.Deployment, key, value, namespace string) error {
	var newData, patchBytes []byte
	var err error

	//get the original data before we change it
	origData, err := json.Marshal(origDeployment)
	if err != nil {
		return err
	}

	origDeployment.ObjectMeta.Labels[key] = value

	newData, err = json.Marshal(origDeployment)
	if err != nil {
		return err
	}

	patchBytes, err = jsonpatch.CreateMergePatch(origData, newData)
	if err != nil {
		return err
	}

	_, err = clientset.AppsV1().Deployments(namespace).Patch(origDeployment.Name, types.MergePatchType, patchBytes)
	if err != nil {
		log.Error(err)
		log.Errorf("error add label to Deployment %s=%s", key, value)
	}
	log.Debugf("add label to deployment %s=%v", key, value)
	return err
}
