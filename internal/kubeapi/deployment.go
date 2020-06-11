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
	"encoding/json"

	jsonpatch "github.com/evanphx/json-patch"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"

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

// UpdateDeployment enables the update of a Kubernetes deployment. This expects
// to have the full manifest required for the udpate
func UpdateDeployment(clientset *kubernetes.Clientset, deployment *v1.Deployment) error {
	if _, err := clientset.AppsV1().Deployments(deployment.Namespace).Update(deployment); err != nil {
		log.Error(err)
		log.Errorf("error updating deployment %s", deployment.Name)
		return err
	}

	return nil
}

// ScaleDeployment provides the ability to scale a Kubernetes deployment.  The deployment provided
// is scaled to the number of replicas specfied via the 'replicas' parameter.
func ScaleDeployment(clientset *kubernetes.Clientset, deployment v1.Deployment,
	replicas int) error {

	replicaCount := int32(replicas)
	deployment.Spec.Replicas = &replicaCount

	if err := UpdateDeployment(clientset, &deployment); err != nil {
		log.Error(err)
		log.Errorf("unable to update replica count to %d in order to scale deployment %s",
			deployment.Name)
		return err
	}

	return nil
}
