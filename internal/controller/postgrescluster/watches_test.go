// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllertest"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestWatchPodsUpdate(t *testing.T) {
	ctx := context.Background()
	queue := &controllertest.Queue{Interface: workqueue.New()}
	reconciler := &Reconciler{}

	update := reconciler.watchPods().UpdateFunc
	assert.Assert(t, update != nil)

	// No metadata; no reconcile.
	update(ctx, event.UpdateEvent{
		ObjectOld: &corev1.Pod{},
		ObjectNew: &corev1.Pod{},
	}, queue)
	assert.Equal(t, queue.Len(), 0)

	// Cluster label, but nothing else; no reconcile.
	update(ctx, event.UpdateEvent{
		ObjectOld: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"postgres-operator.crunchydata.com/cluster": "starfish",
				},
			},
		},
		ObjectNew: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"postgres-operator.crunchydata.com/cluster": "starfish",
				},
			},
		},
	}, queue)
	assert.Equal(t, queue.Len(), 0)

	// Cluster standby leader changed; one reconcile by label.
	update(ctx, event.UpdateEvent{
		ObjectOld: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"status": `{"role":"standby_leader"}`,
				},
				Labels: map[string]string{
					"postgres-operator.crunchydata.com/cluster": "starfish",
				},
			},
		},
		ObjectNew: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "some-ns",
				Labels: map[string]string{
					"postgres-operator.crunchydata.com/cluster": "starfish",
					"postgres-operator.crunchydata.com/role":    "master",
				},
			},
		},
	}, queue)
	assert.Equal(t, queue.Len(), 1)

	item, _ := queue.Get()
	expected := reconcile.Request{}
	expected.Namespace = "some-ns"
	expected.Name = "starfish"
	assert.Equal(t, item, expected)
	queue.Done(item)

	t.Run("PendingRestart", func(t *testing.T) {
		expected := reconcile.Request{}
		expected.Namespace = "some-ns"
		expected.Name = "starfish"

		base := &corev1.Pod{}
		base.Namespace = "some-ns"
		base.Labels = map[string]string{
			"postgres-operator.crunchydata.com/cluster": "starfish",
		}

		pending := base.DeepCopy()
		pending.Annotations = map[string]string{
			"status": `{"pending_restart":true}`,
		}

		// Newly pending; one reconcile by label.
		update(ctx, event.UpdateEvent{
			ObjectOld: base.DeepCopy(),
			ObjectNew: pending.DeepCopy(),
		}, queue)
		assert.Equal(t, queue.Len(), 1, "expected one reconcile")

		item, _ := queue.Get()
		assert.Equal(t, item, expected)
		queue.Done(item)

		// Still pending; one reconcile by label.
		update(ctx, event.UpdateEvent{
			ObjectOld: pending.DeepCopy(),
			ObjectNew: pending.DeepCopy(),
		}, queue)
		assert.Equal(t, queue.Len(), 1, "expected one reconcile")

		item, _ = queue.Get()
		assert.Equal(t, item, expected)
		queue.Done(item)

		// No longer pending; one reconcile by label.
		update(ctx, event.UpdateEvent{
			ObjectOld: pending.DeepCopy(),
			ObjectNew: base.DeepCopy(),
		}, queue)
		assert.Equal(t, queue.Len(), 1, "expected one reconcile")

		item, _ = queue.Get()
		assert.Equal(t, item, expected)
		queue.Done(item)
	})

	// Pod annotation with arbitrary key; no reconcile.
	update(ctx, event.UpdateEvent{
		ObjectOld: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"clortho": "vince",
				},
				Labels: map[string]string{
					"postgres-operator.crunchydata.com/cluster": "starfish",
				},
			},
		},
		ObjectNew: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"clortho": "vin",
				},
				Labels: map[string]string{
					"postgres-operator.crunchydata.com/cluster": "starfish",
				},
			},
		},
	}, queue)
	assert.Equal(t, queue.Len(), 0)

	// Pod annotation with suggested-pgdata-pvc-size; reconcile.
	update(ctx, event.UpdateEvent{
		ObjectOld: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"suggested-pgdata-pvc-size": "5000Mi",
				},
				Labels: map[string]string{
					"postgres-operator.crunchydata.com/cluster": "starfish",
				},
			},
		},
		ObjectNew: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"suggested-pgdata-pvc-size": "8000Mi",
				},
				Labels: map[string]string{
					"postgres-operator.crunchydata.com/cluster": "starfish",
				},
			},
		},
	}, queue)
	assert.Equal(t, queue.Len(), 1)
}
