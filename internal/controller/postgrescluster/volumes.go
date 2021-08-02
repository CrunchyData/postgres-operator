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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation/field"
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
			// NOTE(cbandy): This clears the condition, but it may immediately
			// return with a new LastTransitionTime when a PVC spec is invalid.
			meta.RemoveStatusCondition(&cluster.Status.Conditions, resizing.Type)
		}
	}

	return volumes.Items, err
}

// handlePersistentVolumeClaimError inspects err for expected Kubernetes API
// responses to writing a PVC. It turns errors it understands into conditions
// and events. When err is handled it returns nil. Otherwise it returns err.
func (r *Reconciler) handlePersistentVolumeClaimError(
	cluster *v1beta1.PostgresCluster, err error,
) error {
	var status metav1.Status
	if api := apierrors.APIStatus(nil); errors.As(err, &api) {
		status = api.Status()
	}

	cannotResize := func(err error) {
		meta.SetStatusCondition(&cluster.Status.Conditions, metav1.Condition{
			Type:    v1beta1.PersistentVolumeResizing,
			Status:  metav1.ConditionFalse,
			Reason:  string(apierrors.ReasonForError(err)),
			Message: "One or more volumes cannot be resized",

			ObservedGeneration: cluster.Generation,
			LastTransitionTime: metav1.Now(),
		})
	}

	volumeError := func(err error) {
		r.Recorder.Event(cluster,
			corev1.EventTypeWarning, "PersistentVolumeError", err.Error())
	}

	// Forbidden means (RBAC is broken or) the API request was rejected by an
	// admission controller. Assume it is the latter and raise the issue as a
	// condition and event.
	// - https://releases.k8s.io/v1.21.0/plugin/pkg/admission/storage/persistentvolume/resize/admission.go
	if apierrors.IsForbidden(err) {
		cannotResize(err)
		volumeError(err)
		return nil
	}

	if apierrors.IsInvalid(err) && status.Details != nil {
		unknownCause := false
		for _, cause := range status.Details.Causes {
			switch {
			// Forbidden "spec" happens when the PVC is waiting to be bound.
			// It should resolve on its own and trigger another reconcile. Raise
			// the issue as an event.
			// - https://releases.k8s.io/v1.21.0/pkg/apis/core/validation/validation.go#L2028
			//
			// TODO(cbandy): This can also happen when changing a field other
			// than requests within the spec (access modes, storage class, etc).
			// That case needs a condition or should be prevented via a webhook.
			case
				cause.Type == metav1.CauseType(field.ErrorTypeForbidden) &&
					cause.Field == "spec":
				volumeError(err)

			// Forbidden "storage" happens when the change is not allowed. Raise
			// the issue as a condition and event.
			// - https://releases.k8s.io/v1.21.0/pkg/apis/core/validation/validation.go#L2028
			case
				cause.Type == metav1.CauseType(field.ErrorTypeForbidden) &&
					cause.Field == "spec.resources.requests.storage":
				cannotResize(err)
				volumeError(err)

			default:
				unknownCause = true
			}
		}

		if len(status.Details.Causes) > 0 && !unknownCause {
			// All the causes were identified and handled.
			return nil
		}
	}

	return err
}

// getRepoPVCNames returns a map containing the names of repo PVCs that have
// the appropriate labels for each defined pgBackRest repo, if found.
func getRepoPVCNames(
	cluster *v1beta1.PostgresCluster,
	currentRepoPVCs []*corev1.PersistentVolumeClaim,
) map[string]string {

	repoPVCs := make(map[string]string)
	for _, repo := range cluster.Spec.Backups.PGBackRest.Repos {
		for _, pvc := range currentRepoPVCs {
			if pvc.Labels[naming.LabelPGBackRestRepo] == repo.Name {
				repoPVCs[repo.Name] = pvc.GetName()
				break
			}
		}
	}

	return repoPVCs
}

// getPGPVCName returns the name of a PVC that has the provided labels, if found.
func getPGPVCName(labelMap map[string]string,
	clusterVolumes []corev1.PersistentVolumeClaim,
) (string, error) {

	selector, err := naming.AsSelector(metav1.LabelSelector{
		MatchLabels: labelMap,
	})
	if err != nil {
		return "", errors.WithStack(err)
	}

	for _, pvc := range clusterVolumes {
		if selector.Matches(labels.Set(pvc.GetLabels())) {
			return pvc.GetName(), nil
		}
	}

	return "", nil
}
