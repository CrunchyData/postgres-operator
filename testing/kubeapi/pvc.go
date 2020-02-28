package kubeapi

import (
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
)

// IsPVCBound returns true if pvc is bound.
func IsPVCBound(pvc core_v1.PersistentVolumeClaim) bool {
	return pvc.Status.Phase == core_v1.ClaimBound
}

// ListPVCs returns persistent volume claims matching labels, if any.
func (k *KubeAPI) ListPVCs(namespace string, labels map[string]string) ([]core_v1.PersistentVolumeClaim, error) {
	var options meta_v1.ListOptions

	if labels != nil {
		options.LabelSelector = fields.Set(labels).String()
	}

	list, err := k.Client.CoreV1().PersistentVolumeClaims(namespace).List(options)

	if list == nil && err != nil {
		list = &core_v1.PersistentVolumeClaimList{}
	}

	return list.Items, err
}
