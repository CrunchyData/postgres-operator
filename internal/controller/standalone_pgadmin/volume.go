// Copyright 2023 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package standalone_pgadmin

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/pkg/errors"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// +kubebuilder:rbac:groups="",resources="persistentvolumeclaims",verbs={create,patch}

// reconcilePGAdminDataVolume writes the PersistentVolumeClaim for instance's
// pgAdmin data volume.
func (r *PGAdminReconciler) reconcilePGAdminDataVolume(
	ctx context.Context, pgadmin *v1beta1.PGAdmin,
) (*corev1.PersistentVolumeClaim, error) {

	pvc := pvc(pgadmin)

	err := errors.WithStack(r.setControllerReference(pgadmin, pvc))

	if err == nil {
		err = r.handlePersistentVolumeClaimError(pgadmin,
			errors.WithStack(r.apply(ctx, pvc)))
	}

	return pvc, err
}

// pvc defines the data volume for pgAdmin.
func pvc(pgadmin *v1beta1.PGAdmin) *corev1.PersistentVolumeClaim {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: naming.StandalonePGAdmin(pgadmin),
	}
	pvc.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"))

	pvc.Annotations = pgadmin.Spec.Metadata.GetAnnotationsOrNil()
	pvc.Labels = naming.Merge(
		pgadmin.Spec.Metadata.GetLabelsOrNil(),
		naming.StandalonePGAdminDataLabels(pgadmin.Name),
	)
	pvc.Spec = pgadmin.Spec.DataVolumeClaimSpec

	return pvc
}

// handlePersistentVolumeClaimError inspects err for expected Kubernetes API
// responses to writing a PVC. It turns errors it understands into conditions
// and events. When err is handled it returns nil. Otherwise it returns err.
//
// TODO(tjmoore4): This function is duplicated from a version that takes a PostgresCluster object.
func (r *PGAdminReconciler) handlePersistentVolumeClaimError(
	pgadmin *v1beta1.PGAdmin, err error,
) error {
	var status metav1.Status
	if api := apierrors.APIStatus(nil); errors.As(err, &api) {
		status = api.Status()
	}

	cannotResize := func(err error) {
		meta.SetStatusCondition(&pgadmin.Status.Conditions, metav1.Condition{
			Type:    v1beta1.PersistentVolumeResizing,
			Status:  metav1.ConditionFalse,
			Reason:  string(apierrors.ReasonForError(err)),
			Message: "One or more volumes cannot be resized",

			ObservedGeneration: pgadmin.Generation,
		})
	}

	volumeError := func(err error) {
		r.Recorder.Event(pgadmin,
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
