/*
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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
