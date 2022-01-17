//go:build envtest
// +build envtest

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
package postgrescluster

import (
	"testing"

	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestGeneratePodDisruptionBudget(t *testing.T) {
	_, cc := setupKubernetes(t)
	r := &Reconciler{Client: cc}
	require.ParallelCapacity(t, 0)

	var (
		minAvailable *intstr.IntOrString
		selector     metav1.LabelSelector
	)

	t.Run("empty", func(t *testing.T) {
		// If empty values are passed into the function does it blow up
		_, err := r.generatePodDisruptionBudget(
			&v1beta1.PostgresCluster{},
			metav1.ObjectMeta{},
			minAvailable,
			selector,
		)
		assert.NilError(t, err)
	})

	t.Run("valid", func(t *testing.T) {
		cluster := testCluster()
		meta := metav1.ObjectMeta{
			Name:      "test-pdb",
			Namespace: "test-ns",
			Labels: map[string]string{
				"label-key": "label-value",
			},
			Annotations: map[string]string{
				"anno-key": "anno-value",
			},
		}
		minAvailable = initialize.IntOrStringInt32(1)
		selector := metav1.LabelSelector{
			MatchLabels: map[string]string{
				"key": "value",
			},
		}
		pdb, err := r.generatePodDisruptionBudget(
			cluster,
			meta,
			minAvailable,
			selector,
		)
		assert.NilError(t, err)
		assert.Equal(t, pdb.Name, meta.Name)
		assert.Equal(t, pdb.Namespace, meta.Namespace)
		assert.Assert(t, labels.Set(pdb.Labels).Has("label-key"))
		assert.Assert(t, labels.Set(pdb.Annotations).Has("anno-key"))
		assert.Equal(t, pdb.Spec.MinAvailable, minAvailable)
		assert.DeepEqual(t, pdb.Spec.Selector.MatchLabels, map[string]string{
			"key": "value",
		})
		assert.Assert(t, metav1.IsControlledBy(pdb, cluster))
	})
}

func TestGetMinAvailable(t *testing.T) {
	t.Run("minAvailable provided", func(t *testing.T) {
		// minAvailable is defined so use that value
		ma := initialize.IntOrStringInt32(0)
		expect := getMinAvailable(ma, 1)
		assert.Equal(t, *expect, intstr.FromInt(0))

		ma = initialize.IntOrStringInt32(1)
		expect = getMinAvailable(ma, 2)
		assert.Equal(t, *expect, intstr.FromInt(1))

		ma = initialize.IntOrStringString("50%")
		expect = getMinAvailable(ma, 3)
		assert.Equal(t, *expect, intstr.FromString("50%"))

		ma = initialize.IntOrStringString("200%")
		expect = getMinAvailable(ma, 2147483647)
		assert.Equal(t, *expect, intstr.FromString("200%"))
	})

	// When minAvailable is not defined we need to decide what value to use
	t.Run("defaulting logic", func(t *testing.T) {
		// When we have one replica minAvailable should be 0
		expect := getMinAvailable(nil, 1)
		assert.Equal(t, *expect, intstr.FromInt(0))
		// When we have more than one replica minAvailable should be 1
		expect = getMinAvailable(nil, 2)
		assert.Equal(t, *expect, intstr.FromInt(1))
	})
}
