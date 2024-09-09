// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package initialize_test

import (
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/internal/initialize"
)

func TestAnnotations(t *testing.T) {
	// Ignores nil interface.
	initialize.Annotations(nil)

	pod := new(corev1.Pod)

	// Starts nil.
	assert.Assert(t, pod.Annotations == nil)

	// Gets initialized.
	initialize.Annotations(pod)
	assert.DeepEqual(t, pod.Annotations, map[string]string{})

	// Now writable.
	pod.Annotations["x"] = "y"

	// Doesn't overwrite.
	initialize.Annotations(pod)
	assert.DeepEqual(t, pod.Annotations, map[string]string{"x": "y"})

	// Works with PodTemplate, too.
	template := new(corev1.PodTemplate)
	initialize.Annotations(template)
	assert.DeepEqual(t, template.Annotations, map[string]string{})
}

func TestLabels(t *testing.T) {
	// Ignores nil interface.
	initialize.Labels(nil)

	pod := new(corev1.Pod)

	// Starts nil.
	assert.Assert(t, pod.Labels == nil)

	// Gets initialized.
	initialize.Labels(pod)
	assert.DeepEqual(t, pod.Labels, map[string]string{})

	// Now writable.
	pod.Labels["x"] = "y"

	// Doesn't overwrite.
	initialize.Labels(pod)
	assert.DeepEqual(t, pod.Labels, map[string]string{"x": "y"})

	// Works with PodTemplate, too.
	template := new(corev1.PodTemplate)
	initialize.Labels(template)
	assert.DeepEqual(t, template.Labels, map[string]string{})
}
