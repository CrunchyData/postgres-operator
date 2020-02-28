package kubeapi

import (
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
)

// IsPodReady returns true if all containers of pod are ready.
func IsPodReady(pod core_v1.Pod) bool {
	for _, status := range pod.Status.ContainerStatuses {
		if !status.Ready {
			return false
		}
	}
	return true
}

// GetPod returns a pod from the specified namespace.
func (k *KubeAPI) GetPod(namespace, name string) (*core_v1.Pod, error) {
	return k.Client.CoreV1().Pods(namespace).Get(name, meta_v1.GetOptions{})
}

// ListPods returns pods matching labels, if any.
func (k *KubeAPI) ListPods(namespace string, labels map[string]string) ([]core_v1.Pod, error) {
	var options meta_v1.ListOptions

	if labels != nil {
		options.LabelSelector = fields.Set(labels).String()
	}

	list, err := k.Client.CoreV1().Pods(namespace).List(options)

	if list == nil && err != nil {
		list = &core_v1.PodList{}
	}

	return list.Items, err
}
