// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

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
