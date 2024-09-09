// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgupgrade

import (
	"fmt"
	"testing"

	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestPopulateCluster(t *testing.T) {
	t.Run("Found", func(t *testing.T) {
		cluster := v1beta1.NewPostgresCluster()
		cluster.SetName("cluster")

		world := NewWorld()
		err := world.populateCluster(cluster, nil)

		assert.NilError(t, err)
		assert.Equal(t, world.Cluster, cluster)
		assert.Assert(t, world.ClusterNotFound == nil)
	})

	t.Run("NotFound", func(t *testing.T) {
		cluster := v1beta1.NewPostgresCluster()
		expected := apierrors.NewNotFound(schema.GroupResource{}, "name")

		world := NewWorld()
		err := world.populateCluster(cluster, expected)

		assert.NilError(t, err, "NotFound is handled")
		assert.Assert(t, world.Cluster == nil)
		assert.Equal(t, world.ClusterNotFound, expected)
	})

	t.Run("Error", func(t *testing.T) {
		cluster := v1beta1.NewPostgresCluster()
		expected := fmt.Errorf("danger")

		world := NewWorld()
		err := world.populateCluster(cluster, expected)

		assert.Equal(t, err, expected)
		assert.Assert(t, world.Cluster == nil)
		assert.Assert(t, world.ClusterNotFound == nil)
	})
}

func TestPopulatePatroniEndpoint(t *testing.T) {
	endpoints := []corev1.Endpoints{
		{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					LabelPatroni: "west",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					LabelPatroni: "east",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"different-label": "north",
				},
			},
		},
	}

	world := NewWorld()
	world.populatePatroniEndpoints(endpoints)

	// The first two have the correct labels.
	assert.DeepEqual(t, world.PatroniEndpoints, []*corev1.Endpoints{
		&endpoints[0],
		&endpoints[1],
	})
}

func TestPopulateShutdown(t *testing.T) {
	t.Run("NoCluster", func(t *testing.T) {
		world := NewWorld()

		world.populateShutdown()
		assert.Assert(t, !world.ClusterShutdown)
	})

	t.Run("NotShutdown", func(t *testing.T) {
		cluster := v1beta1.NewPostgresCluster()
		cluster.Spec.Shutdown = initialize.Bool(false)

		world := NewWorld()
		world.Cluster = cluster

		world.populateShutdown()
		assert.Assert(t, !world.ClusterShutdown)
	})

	t.Run("OldStatus", func(t *testing.T) {
		cluster := v1beta1.NewPostgresCluster()
		cluster.SetGeneration(99)
		cluster.Spec.Shutdown = initialize.Bool(true)
		cluster.Status.ObservedGeneration = 21

		world := NewWorld()
		world.Cluster = cluster

		world.populateShutdown()
		assert.Assert(t, !world.ClusterShutdown)
	})

	t.Run("InstancesRunning", func(t *testing.T) {
		cluster := v1beta1.NewPostgresCluster()
		cluster.SetGeneration(99)
		cluster.Spec.Shutdown = initialize.Bool(true)
		cluster.Status.ObservedGeneration = 99
		cluster.Status.InstanceSets = []v1beta1.PostgresInstanceSetStatus{{Replicas: 2}}

		world := NewWorld()
		world.Cluster = cluster

		world.populateShutdown()
		assert.Assert(t, !world.ClusterShutdown)
	})

	t.Run("InstancesStopped", func(t *testing.T) {
		cluster := v1beta1.NewPostgresCluster()
		cluster.SetGeneration(99)
		cluster.Spec.Shutdown = initialize.Bool(true)
		cluster.Status.ObservedGeneration = 99
		cluster.Status.InstanceSets = []v1beta1.PostgresInstanceSetStatus{{Replicas: 0}}

		world := NewWorld()
		world.Cluster = cluster

		world.populateShutdown()
		assert.Assert(t, world.ClusterShutdown)
	})
}

func TestPopulateStatefulSets(t *testing.T) {
	t.Run("NoPopulatesWithoutStartupGiven", func(t *testing.T) {
		cluster := v1beta1.NewPostgresCluster()
		world := NewWorld()
		world.Cluster = cluster

		primary := appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "the-one",
				Labels: map[string]string{
					LabelInstance: "whatever",
				},
			},
		}
		replica := appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "something-else",
				Labels: map[string]string{
					LabelInstance: "whatever",
				},
			},
		}
		other := appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "repo-host",
				Labels: map[string]string{
					"other-label": "other",
				},
			},
		}
		world.populateStatefulSets([]appsv1.StatefulSet{primary, replica, other})

		assert.Assert(t, world.ClusterPrimary == nil)
		assert.Assert(t, world.ClusterReplicas == nil)
		assert.Assert(t, world.ReplicasExpected == 1)
	})

	t.Run("PopulatesWithStartupGiven", func(t *testing.T) {
		cluster := v1beta1.NewPostgresCluster()
		cluster.Status.StartupInstance = "the-one"

		world := NewWorld()
		world.Cluster = cluster

		primary := appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "the-one",
				Labels: map[string]string{
					LabelInstance: "whatever",
				},
			},
		}
		replica := appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "something-else",
				Labels: map[string]string{
					LabelInstance: "whatever",
				},
			},
		}
		other := appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "repo-host",
				Labels: map[string]string{
					"other-label": "other",
				},
			},
		}
		world.populateStatefulSets([]appsv1.StatefulSet{primary, replica, other})

		assert.DeepEqual(t, world.ClusterPrimary, &primary)
		assert.DeepEqual(t, world.ClusterReplicas, []*appsv1.StatefulSet{&replica})
		assert.Assert(t, world.ReplicasExpected == 1)
	})
}
