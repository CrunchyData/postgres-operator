// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package runtime_test

import (
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
)

func TestConvertUnstructured(t *testing.T) {
	var cm corev1.ConfigMap
	cm.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))
	cm.Namespace = "one"
	cm.Name = "two"
	cm.Data = map[string]string{"w": "x", "y": "z"}

	t.Run("List", func(t *testing.T) {
		original := new(corev1.ConfigMapList)
		original.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMapList"))
		original.Items = []corev1.ConfigMap{*cm.DeepCopy()}

		list, err := runtime.ToUnstructuredList(original)
		assert.NilError(t, err)

		converted, err := runtime.FromUnstructuredList[corev1.ConfigMapList](list)
		assert.NilError(t, err)
		assert.DeepEqual(t, original, converted)
	})

	t.Run("Object", func(t *testing.T) {
		original := cm.DeepCopy()

		object, err := runtime.ToUnstructuredObject(original)
		assert.NilError(t, err)

		converted, err := runtime.FromUnstructuredObject[corev1.ConfigMap](object)
		assert.NilError(t, err)
		assert.DeepEqual(t, original, converted)
	})
}
