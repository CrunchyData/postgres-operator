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
	log "github.com/Sirupsen/logrus"
	"k8s.io/api/extensions/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DeleteDeployment deletes a deployment
func DeleteDeployment(clientset *kubernetes.Clientset, name, namespace string) error {
	delOptions := meta_v1.DeleteOptions{}
	var delProp meta_v1.DeletionPropagation
	delProp = meta_v1.DeletePropagationForeground
	delOptions.PropagationPolicy = &delProp

	err := clientset.ExtensionsV1beta1().Deployments(namespace).Delete(name, &delOptions)
	if err != nil {
		log.Error(err)
		log.Error("error deleting Deployment " + name)
	}
	log.Info("delete deployment " + name)
	return err
}

// CreateDeployment creates a deployment
func CreateDeployment(clientset *kubernetes.Clientset, deployment *v1beta1.Deployment, namespace string) error {
	deploymentResult, err := clientset.ExtensionsV1beta1().Deployments(namespace).Create(deployment)
	if err != nil {
		log.Error("error creating Deployment " + err.Error())
		return err
	}

	log.Info("created deployment " + deploymentResult.Name)
	return err

}

// GetDeployment gets a deployment by name
func GetDeployment(clientset *kubernetes.Clientset, name, namespace string) (*v1beta1.Deployment, bool, error) {
	deploymentResult, err := clientset.ExtensionsV1beta1().Deployments(namespace).Get(name, meta_v1.GetOptions{})
	if kerrors.IsNotFound(err) {
		log.Debug("deployment " + name + " not found")
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
func GetDeployments(clientset *kubernetes.Clientset, selector, namespace string) (*v1beta1.DeploymentList, error) {
	lo := meta_v1.ListOptions{LabelSelector: selector}

	deployments, err := clientset.ExtensionsV1beta1().Deployments(namespace).List(lo)
	if err != nil {
		log.Error(err)
		log.Error("error getting Deployment list selector[" + selector + "]")
	}
	return deployments, err

}
