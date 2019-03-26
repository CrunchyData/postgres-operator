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
	"encoding/json"
	jsonpatch "github.com/evanphx/json-patch"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/apps/v1"
	//	"k8s.io/api/extensions/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// DeleteDeployment deletes a deployment
func DeleteDeployment(clientset *kubernetes.Clientset, name, namespace string) error {
	delOptions := meta_v1.DeleteOptions{}
	var delProp meta_v1.DeletionPropagation
	delProp = meta_v1.DeletePropagationForeground
	delOptions.PropagationPolicy = &delProp

	err := clientset.AppsV1().Deployments(namespace).Delete(name, &delOptions)
	if err != nil {
		log.Error(err)
		log.Error("error deleting Deployment " + name)
	}
	log.Info("delete deployment " + name)
	return err
}

// CreateDeployment creates a deployment
func CreateDeployment(clientset *kubernetes.Clientset, deployment *v1.Deployment, namespace string) error {
	deploymentResult, err := clientset.AppsV1().Deployments(namespace).Create(deployment)
	if err != nil {
		log.Error("error creating Deployment " + err.Error())
		return err
	}

	log.Info("created deployment " + deploymentResult.Name)
	return err

}

// CreateDeployment creates a deployment
func CreateDeploymentV1(clientset *kubernetes.Clientset, deployment *v1.Deployment, namespace string) error {
	deploymentResult, err := clientset.AppsV1().Deployments(namespace).Create(deployment)
	if err != nil {
		log.Error("error creating Deployment " + err.Error())
		return err
	}

	log.Info("created deployment " + deploymentResult.Name)
	return err

}

// GetDeployment gets a deployment by name
func GetDeployment(clientset *kubernetes.Clientset, name, namespace string) (*v1.Deployment, bool, error) {
	deploymentResult, err := clientset.AppsV1().Deployments(namespace).Get(name, meta_v1.GetOptions{})
	if kerrors.IsNotFound(err) {
		log.Debugf("deployment %s not found", name)
		return deploymentResult, false, err
	}
	if err != nil {
		log.Error(err)
		log.Error("error getting Deployment " + name)
		return deploymentResult, false, err
	}

	return deploymentResult, true, err

}

// GetDeployments gets a list of deployments using a label selector
func GetDeployments(clientset *kubernetes.Clientset, selector, namespace string) (*v1.DeploymentList, error) {
	lo := meta_v1.ListOptions{LabelSelector: selector}

	deployments, err := clientset.AppsV1().Deployments(namespace).List(lo)
	if err != nil {
		log.Error(err)
		log.Error("error getting Deployment list selector[" + selector + "]")
	}
	return deployments, err

}

type ThingSpec struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

// PatchDeployment patches a deployment
func PatchDeployment(clientset *kubernetes.Clientset, name, namespace, jsonpath, patchvalue string) error {
	var patchBytes []byte
	var err error

	things := make([]ThingSpec, 1)
	things[0].Op = "replace"
	things[0].Path = jsonpath
	things[0].Value = patchvalue

	patchBytes, err = json.Marshal(things)
	if err != nil {
		log.Error("error in converting patch " + err.Error())
		return err
	}

	_, err = clientset.AppsV1().Deployments(namespace).Patch(name, types.JSONPatchType, patchBytes)
	if err != nil {
		log.Error(err)
		log.Error("error patching Deployment " + name)
	}
	log.Info("patch deployment " + name)
	return err
}

type IntThingSpec struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value int    `json:"value"`
}

// PatchDeployment patches a deployment
func PatchReplicas(clientset *kubernetes.Clientset, name, namespace, jsonpath string, patchvalue int) error {
	var patchBytes []byte
	var err error

	things := make([]IntThingSpec, 1)
	things[0].Op = "replace"
	things[0].Path = jsonpath
	things[0].Value = patchvalue

	patchBytes, err = json.Marshal(things)
	if err != nil {
		log.Error("error in converting patch " + err.Error())
		return err
	}

	_, err = clientset.AppsV1().Deployments(namespace).Patch(name, types.JSONPatchType, patchBytes)
	if err != nil {
		log.Error(err)
		log.Error("error patching Deployment " + name)
	}
	log.Info("patch deployment " + name)
	return err
}

// MergePatchDeployment patches a deployment for failover only at this point
func MergePatchDeployment(clientset *kubernetes.Clientset, origDeployment *v1.Deployment, newname, namespace string) error {
	var newData, patchBytes []byte
	var err error

	//get the original data before we change it
	origData, err := json.Marshal(origDeployment)
	if err != nil {
		return err
	}

	//change the deployment template for new pods to be created
	origDeployment.Spec.Selector.MatchLabels["name"] = newname
	origDeployment.Spec.Selector.MatchLabels["primary"] = "true"
	origDeployment.Spec.Selector.MatchLabels["replica"] = "false"

	origDeployment.Spec.Template.ObjectMeta.Labels["name"] = newname
	origDeployment.Spec.Template.ObjectMeta.Labels["primary"] = "true"
	origDeployment.Spec.Template.ObjectMeta.Labels["replica"] = "false"

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
		log.Error("error merge patching Deployment " + newname)
	}
	log.Info("merge patch deployment " + newname)
	return err
}

func AddLabelToDeployment(clientset *kubernetes.Clientset, origDeployment *v1.Deployment, key, value, namespace string) error {
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

func UpdateDeployment(clientset *kubernetes.Clientset, deployment *v1.Deployment, namespace string) error {
	var err error

	_, err = clientset.AppsV1().Deployments(namespace).Update(deployment)
	if err != nil {
		log.Error(err)
		log.Errorf("error updating deployment %s", deployment.Name)
		return err
	}
	return err

}
