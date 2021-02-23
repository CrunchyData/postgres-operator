package kubeapi

import (
	apps_v1 "k8s.io/api/apps/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
)

// ListDeployments returns deployments matching labels, if any.
func (k *KubeAPI) ListDeployments(namespace string, labels map[string]string) ([]apps_v1.Deployment, error) {
	var options meta_v1.ListOptions

	if labels != nil {
		options.LabelSelector = fields.Set(labels).String()
	}

	list, err := k.Client.AppsV1().Deployments(namespace).List(options)

	if list == nil && err != nil {
		list = &apps_v1.DeploymentList{}
	}

	return list.Items, err
}

// GetDeployment returns deployment by name, if exists.
func (k *KubeAPI) GetDeployment(namespace, name string) *apps_v1.Deployment {
	deployment, err := k.Client.AppsV1().Deployments(namespace).Get(name, meta_v1.GetOptions{})
	if deployment == nil && err != nil {
		deployment = &apps_v1.Deployment{}
	}

	return deployment
}
