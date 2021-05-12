/*
 Copyright 2021 Crunchy Data Solutions, Inc.
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

package postgrescluster

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=list

// observePersistentVolumeClaims reads all PVCs for cluster from the Kubernetes
// API and sets the PersistentVolumeResizing condition as appropriate.
func (r *Reconciler) observePersistentVolumeClaims(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
) ([]corev1.PersistentVolumeClaim, error) {
	volumes := &corev1.PersistentVolumeClaimList{}

	selector, err := naming.AsSelector(naming.Cluster(cluster.Name))
	if err == nil {
		err = errors.WithStack(
			r.Client.List(ctx, volumes,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selector},
			))
	}

	resizing := metav1.Condition{
		Type:    v1beta1.PersistentVolumeResizing,
		Message: "One or more volumes are changing size",

		ObservedGeneration: cluster.Generation,
	}

	minNotZero := func(a, b metav1.Time) metav1.Time {
		if b.IsZero() || (a.Before(&b) && !a.IsZero()) {
			return a
		}
		return b
	}

	for _, pvc := range volumes.Items {
		for _, condition := range pvc.Status.Conditions {
			switch condition.Type {
			case
				corev1.PersistentVolumeClaimResizing,
				corev1.PersistentVolumeClaimFileSystemResizePending:

				// Initialize from the first condition.
				if resizing.Status == "" {
					resizing.Status = metav1.ConditionStatus(condition.Status)
					resizing.Reason = condition.Reason
					resizing.LastTransitionTime = condition.LastTransitionTime

					// corev1.PersistentVolumeClaimCondition.Reason is optional
					// while metav1.Condition.Reason is required.
					if resizing.Reason == "" {
						resizing.Reason = string(condition.Type)
					}
				}

				// Use most things from an adverse condition.
				if condition.Status != corev1.ConditionTrue {
					resizing.Status = metav1.ConditionStatus(condition.Status)
					resizing.Reason = condition.Reason
					resizing.Message = condition.Message
					resizing.LastTransitionTime = condition.LastTransitionTime

					// corev1.PersistentVolumeClaimCondition.Reason is optional
					// while metav1.Condition.Reason is required.
					if resizing.Reason == "" {
						resizing.Reason = string(condition.Type)
					}
				}

				// Use the oldest transition time of healthy conditions.
				if resizing.Status == metav1.ConditionTrue &&
					condition.Status == corev1.ConditionTrue {
					resizing.LastTransitionTime = minNotZero(
						resizing.LastTransitionTime, condition.LastTransitionTime)
				}
			}
		}
	}

	if resizing.Status != "" {
		meta.SetStatusCondition(&cluster.Status.Conditions, resizing)
	} else {
		// Avoid a panic! Fixed in Kubernetes v1.21.0 and controller-runtime v0.9.0-alpha.0.
		// - https://issue.k8s.io/99714
		if len(cluster.Status.Conditions) > 0 {
			meta.RemoveStatusCondition(&cluster.Status.Conditions, resizing.Type)
		}
	}

	return volumes.Items, err
}
