// Copyright 2023 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package standalone_pgadmin

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pkg/errors"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/events"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestReconcilePGAdminDataVolume(t *testing.T) {
	ctx := context.Background()
	cc := setupKubernetes(t)
	require.ParallelCapacity(t, 1)

	reconciler := &PGAdminReconciler{
		Client: cc,
		Owner:  client.FieldOwner(t.Name()),
	}

	ns := setupNamespace(t, cc)
	pgadmin := &v1beta1.PGAdmin{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-standalone-pgadmin",
			Namespace: ns.Name,
		},
		Spec: v1beta1.PGAdminSpec{
			DataVolumeClaimSpec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceStorage: resource.MustParse("1Gi")}},
				StorageClassName: initialize.String("storage-class-for-data"),
			}}}

	assert.NilError(t, cc.Create(ctx, pgadmin))
	t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, pgadmin)) })

	t.Run("DataVolume", func(t *testing.T) {
		pvc, err := reconciler.reconcilePGAdminDataVolume(ctx, pgadmin)
		assert.NilError(t, err)

		assert.Assert(t, metav1.IsControlledBy(pvc, pgadmin))

		assert.Equal(t, pvc.Labels[naming.LabelStandalonePGAdmin], pgadmin.Name)
		assert.Equal(t, pvc.Labels[naming.LabelRole], naming.RolePGAdmin)
		assert.Equal(t, pvc.Labels[naming.LabelData], naming.DataPGAdmin)

		assert.Assert(t, cmp.MarshalMatches(pvc.Spec, `
accessModes:
- ReadWriteOnce
resources:
  requests:
    storage: 1Gi
storageClassName: storage-class-for-data
volumeMode: Filesystem
		`))
	})
}

func TestHandlePersistentVolumeClaimError(t *testing.T) {
	recorder := events.NewRecorder(t, runtime.Scheme)
	reconciler := &PGAdminReconciler{
		Recorder: recorder,
	}

	pgadmin := new(v1beta1.PGAdmin)
	pgadmin.Namespace = "ns1"
	pgadmin.Name = "pg2"

	reset := func() {
		pgadmin.Status.Conditions = pgadmin.Status.Conditions[:0]
		recorder.Events = recorder.Events[:0]
	}

	// It returns any error it does not recognize completely.
	t.Run("Unexpected", func(t *testing.T) {
		t.Cleanup(reset)

		err := errors.New("whomp")

		assert.Equal(t, err, reconciler.handlePersistentVolumeClaimError(pgadmin, err))
		assert.Assert(t, len(pgadmin.Status.Conditions) == 0)
		assert.Assert(t, len(recorder.Events) == 0)

		err = apierrors.NewInvalid(
			corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim").GroupKind(),
			"some-pvc",
			field.ErrorList{
				field.Forbidden(field.NewPath("metadata"), "dunno"),
			})

		assert.Equal(t, err, reconciler.handlePersistentVolumeClaimError(pgadmin, err))
		assert.Assert(t, len(pgadmin.Status.Conditions) == 0)
		assert.Assert(t, len(recorder.Events) == 0)
	})

	// Neither statically nor dynamically provisioned claims can be resized
	// before they are bound to a persistent volume. Kubernetes rejects such
	// changes during PVC validation.
	//
	// A static PVC is one with a present-and-blank storage class. It is
	// pending until a PV exists that matches its selector, requests, etc.
	// - https://docs.k8s.io/concepts/storage/persistent-volumes/#static
	// - https://docs.k8s.io/concepts/storage/persistent-volumes/#class-1
	//
	// A dynamic PVC is associated with a storage class. Storage classes that
	// "WaitForFirstConsumer" do not bind a PV until there is a pod.
	// - https://docs.k8s.io/concepts/storage/persistent-volumes/#dynamic
	t.Run("Pending", func(t *testing.T) {
		t.Run("Grow", func(t *testing.T) {
			t.Cleanup(reset)

			err := apierrors.NewInvalid(
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim").GroupKind(),
				"my-pending-pvc",
				field.ErrorList{
					// - https://releases.k8s.io/v1.24.0/pkg/apis/core/validation/validation.go#L2184
					field.Forbidden(field.NewPath("spec"), "… immutable … bound claim …"),
				})

			// PVCs will bind eventually. This error should become an event without a condition.
			assert.NilError(t, reconciler.handlePersistentVolumeClaimError(pgadmin, err))

			assert.Check(t, len(pgadmin.Status.Conditions) == 0)
			assert.Check(t, len(recorder.Events) > 0)

			for _, event := range recorder.Events {
				assert.Equal(t, event.Type, "Warning")
				assert.Equal(t, event.Reason, "PersistentVolumeError")
				assert.Assert(t, cmp.Contains(event.Note, "PersistentVolumeClaim"))
				assert.Assert(t, cmp.Contains(event.Note, "my-pending-pvc"))
				assert.Assert(t, cmp.Contains(event.Note, "bound claim"))
				assert.DeepEqual(t, event.Regarding, corev1.ObjectReference{
					APIVersion: v1beta1.GroupVersion.Identifier(),
					Kind:       "PGAdmin",
					Namespace:  "ns1", Name: "pg2",
				})
			}
		})

		t.Run("Shrink", func(t *testing.T) {
			t.Cleanup(reset)

			// Requests to make a pending PVC smaller fail for multiple reasons.
			err := apierrors.NewInvalid(
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim").GroupKind(),
				"my-pending-pvc",
				field.ErrorList{
					// - https://releases.k8s.io/v1.24.0/pkg/apis/core/validation/validation.go#L2184
					field.Forbidden(field.NewPath("spec"), "… immutable … bound claim …"),

					// - https://releases.k8s.io/v1.24.0/pkg/apis/core/validation/validation.go#L2188
					field.Forbidden(field.NewPath("spec", "resources", "requests", "storage"), "… not be less …"),
				})

			// PVCs will bind eventually, but the size is rejected.
			assert.NilError(t, reconciler.handlePersistentVolumeClaimError(pgadmin, err))

			assert.Check(t, len(pgadmin.Status.Conditions) > 0)
			assert.Check(t, len(recorder.Events) > 0)

			for _, condition := range pgadmin.Status.Conditions {
				assert.Equal(t, condition.Type, "PersistentVolumeResizing")
				assert.Equal(t, condition.Status, metav1.ConditionFalse)
				assert.Equal(t, condition.Reason, "Invalid")
				assert.Assert(t, cmp.Contains(condition.Message, "cannot be resized"))
			}

			for _, event := range recorder.Events {
				assert.Equal(t, event.Type, "Warning")
				assert.Equal(t, event.Reason, "PersistentVolumeError")
				assert.Assert(t, cmp.Contains(event.Note, "PersistentVolumeClaim"))
				assert.Assert(t, cmp.Contains(event.Note, "my-pending-pvc"))
				assert.Assert(t, cmp.Contains(event.Note, "bound claim"))
				assert.Assert(t, cmp.Contains(event.Note, "not be less"))
				assert.DeepEqual(t, event.Regarding, corev1.ObjectReference{
					APIVersion: v1beta1.GroupVersion.Identifier(),
					Kind:       "PGAdmin",
					Namespace:  "ns1", Name: "pg2",
				})
			}
		})
	})

	// Statically provisioned claims cannot be resized. Kubernetes responds
	// differently based on the size growing or shrinking.
	//
	// Dynamically provisioned claims of storage classes that do *not*
	// "allowVolumeExpansion" behave the same way.
	t.Run("NoExpansion", func(t *testing.T) {
		t.Run("Grow", func(t *testing.T) {
			t.Cleanup(reset)

			// - https://releases.k8s.io/v1.24.0/plugin/pkg/admission/storage/persistentvolume/resize/admission.go#L108
			err := apierrors.NewForbidden(
				corev1.Resource("persistentvolumeclaims"), "my-static-pvc",
				errors.New("… only dynamically provisioned …"))

			// This PVC cannot resize. The error should become an event and condition.
			assert.NilError(t, reconciler.handlePersistentVolumeClaimError(pgadmin, err))

			assert.Check(t, len(pgadmin.Status.Conditions) > 0)
			assert.Check(t, len(recorder.Events) > 0)

			for _, condition := range pgadmin.Status.Conditions {
				assert.Equal(t, condition.Type, "PersistentVolumeResizing")
				assert.Equal(t, condition.Status, metav1.ConditionFalse)
				assert.Equal(t, condition.Reason, "Forbidden")
				assert.Assert(t, cmp.Contains(condition.Message, "cannot be resized"))
			}

			for _, event := range recorder.Events {
				assert.Equal(t, event.Type, "Warning")
				assert.Equal(t, event.Reason, "PersistentVolumeError")
				assert.Assert(t, cmp.Contains(event.Note, "persistentvolumeclaim"))
				assert.Assert(t, cmp.Contains(event.Note, "my-static-pvc"))
				assert.Assert(t, cmp.Contains(event.Note, "only dynamic"))
				assert.DeepEqual(t, event.Regarding, corev1.ObjectReference{
					APIVersion: v1beta1.GroupVersion.Identifier(),
					Kind:       "PGAdmin",
					Namespace:  "ns1", Name: "pg2",
				})
			}
		})

		// Dynamically provisioned claims of storage classes that *do*
		// "allowVolumeExpansion" can grow but cannot shrink. Kubernetes
		// rejects such changes during PVC validation, just like static claims.
		//
		// A future version of Kubernetes will allow `spec.resources` to shrink
		// so long as it is greater than `status.capacity`.
		// - https://git.k8s.io/enhancements/keps/sig-storage/1790-recover-resize-failure
		t.Run("Shrink", func(t *testing.T) {
			t.Cleanup(reset)

			err := apierrors.NewInvalid(
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim").GroupKind(),
				"my-static-pvc",
				field.ErrorList{
					// - https://releases.k8s.io/v1.24.0/pkg/apis/core/validation/validation.go#L2188
					field.Forbidden(field.NewPath("spec", "resources", "requests", "storage"), "… not be less …"),
				})

			// The PVC size is rejected. This error should become an event and condition.
			assert.NilError(t, reconciler.handlePersistentVolumeClaimError(pgadmin, err))

			assert.Check(t, len(pgadmin.Status.Conditions) > 0)
			assert.Check(t, len(recorder.Events) > 0)

			for _, condition := range pgadmin.Status.Conditions {
				assert.Equal(t, condition.Type, "PersistentVolumeResizing")
				assert.Equal(t, condition.Status, metav1.ConditionFalse)
				assert.Equal(t, condition.Reason, "Invalid")
				assert.Assert(t, cmp.Contains(condition.Message, "cannot be resized"))
			}

			for _, event := range recorder.Events {
				assert.Equal(t, event.Type, "Warning")
				assert.Equal(t, event.Reason, "PersistentVolumeError")
				assert.Assert(t, cmp.Contains(event.Note, "PersistentVolumeClaim"))
				assert.Assert(t, cmp.Contains(event.Note, "my-static-pvc"))
				assert.Assert(t, cmp.Contains(event.Note, "not be less"))
				assert.DeepEqual(t, event.Regarding, corev1.ObjectReference{
					APIVersion: v1beta1.GroupVersion.Identifier(),
					Kind:       "PGAdmin",
					Namespace:  "ns1", Name: "pg2",
				})
			}
		})
	})
}
