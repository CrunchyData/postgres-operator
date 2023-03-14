/*
 Copyright 2021 - 2023 Crunchy Data Solutions, Inc.
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

package postgres

import (
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestSetHugePages(t *testing.T) {
	t.Run("hugepages not set at all", func(t *testing.T) {
		cluster := new(v1beta1.PostgresCluster)

		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{
			Name:     "test-instance1",
			Replicas: initialize.Int32(1),
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{},
			},
		}}

		pgParameters := NewParameters()
		SetHugePages(cluster, &pgParameters)

		assert.Equal(t, pgParameters.Default.Has("huge_pages"), true)
		assert.Equal(t, pgParameters.Default.Value("huge_pages"), "off")
	})

	t.Run("hugepages quantity not set", func(t *testing.T) {
		cluster := new(v1beta1.PostgresCluster)

		emptyQuantity, _ := resource.ParseQuantity("")
		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{
			Name:     "test-instance1",
			Replicas: initialize.Int32(1),
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceHugePagesPrefix + "2Mi": emptyQuantity,
				},
			},
		}}

		pgParameters := NewParameters()
		SetHugePages(cluster, &pgParameters)

		assert.Equal(t, pgParameters.Default.Has("huge_pages"), true)
		assert.Equal(t, pgParameters.Default.Value("huge_pages"), "off")
	})

	t.Run("hugepages set to zero", func(t *testing.T) {
		cluster := new(v1beta1.PostgresCluster)

		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{
			Name:     "test-instance1",
			Replicas: initialize.Int32(1),
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceHugePagesPrefix + "2Mi": resource.MustParse("0Mi"),
				},
			},
		}}

		pgParameters := NewParameters()
		SetHugePages(cluster, &pgParameters)

		assert.Equal(t, pgParameters.Default.Has("huge_pages"), true)
		assert.Equal(t, pgParameters.Default.Value("huge_pages"), "off")
	})

	t.Run("hugepages set correctly", func(t *testing.T) {
		cluster := new(v1beta1.PostgresCluster)

		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{
			Name:     "test-instance1",
			Replicas: initialize.Int32(1),
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceHugePagesPrefix + "2Mi": resource.MustParse("16Mi"),
				},
			},
		}}

		pgParameters := NewParameters()
		SetHugePages(cluster, &pgParameters)

		assert.Equal(t, pgParameters.Default.Has("huge_pages"), true)
		assert.Equal(t, pgParameters.Default.Value("huge_pages"), "try")
	})

}
