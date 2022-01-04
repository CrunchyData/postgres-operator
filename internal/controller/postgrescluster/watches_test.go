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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllertest"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestWatchPodsUpdate(t *testing.T) {
	queue := controllertest.Queue{Interface: workqueue.New()}
	reconciler := &Reconciler{}

	update := reconciler.watchPods().UpdateFunc
	assert.Assert(t, update != nil)

	// No metadata; no reconcile.
	update(event.UpdateEvent{
		ObjectOld: &corev1.Pod{},
		ObjectNew: &corev1.Pod{},
	}, queue)
	assert.Equal(t, queue.Len(), 0)

	// Cluster label, but nothing else; no reconcile.
	update(event.UpdateEvent{
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
	update(event.UpdateEvent{
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
		update(event.UpdateEvent{
			ObjectOld: base.DeepCopy(),
			ObjectNew: pending.DeepCopy(),
		}, queue)
		assert.Equal(t, queue.Len(), 1, "expected one reconcile")

		item, _ := queue.Get()
		assert.Equal(t, item, expected)
		queue.Done(item)

		// Still pending; one reconcile by label.
		update(event.UpdateEvent{
			ObjectOld: pending.DeepCopy(),
			ObjectNew: pending.DeepCopy(),
		}, queue)
		assert.Equal(t, queue.Len(), 1, "expected one reconcile")

		item, _ = queue.Get()
		assert.Equal(t, item, expected)
		queue.Done(item)

		// No longer pending; one reconcile by label.
		update(event.UpdateEvent{
			ObjectOld: pending.DeepCopy(),
			ObjectNew: base.DeepCopy(),
		}, queue)
		assert.Equal(t, queue.Len(), 1, "expected one reconcile")

		item, _ = queue.Get()
		assert.Equal(t, item, expected)
		queue.Done(item)
	})
}
