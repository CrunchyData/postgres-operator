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

package runtime

import (
	"context"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func TestTickerString(t *testing.T) {
	assert.Equal(t, ticker{Duration: time.Millisecond}.String(), "every 1ms")
	assert.Equal(t, ticker{Duration: 10 * time.Second}.String(), "every 10s")
	assert.Equal(t, ticker{Duration: time.Hour}.String(), "every 1h0m0s")
}

func TestTicker(t *testing.T) {
	t.Parallel()

	var called []event.GenericEvent
	expected := event.GenericEvent{Object: new(corev1.ConfigMap)}

	tq := workqueue.NewRateLimitingQueue(workqueue.DefaultItemBasedRateLimiter())
	th := handler.Funcs{GenericFunc: func(e event.GenericEvent, q workqueue.RateLimitingInterface) {
		called = append(called, e)

		assert.Equal(t, q, tq, "should be called with the queue passed in Start")
	}}

	t.Run("WithoutPredicates", func(t *testing.T) {
		called = nil

		ticker := NewTicker(100*time.Millisecond, expected)
		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		t.Cleanup(cancel)

		// Start the ticker and wait for the deadline to pass.
		assert.NilError(t, ticker.Start(ctx, th, tq))
		<-ctx.Done()

		assert.Equal(t, len(called), 2)
		assert.Equal(t, called[0], expected, "expected at 100ms")
		assert.Equal(t, called[1], expected, "expected at 200ms")
	})

	t.Run("WithPredicates", func(t *testing.T) {
		called = nil

		// Predicates that exclude events after a fixed number have passed.
		pLength := predicate.Funcs{GenericFunc: func(event.GenericEvent) bool { return len(called) < 3 }}
		pTrue := predicate.Funcs{GenericFunc: func(event.GenericEvent) bool { return true }}

		ticker := NewTicker(50*time.Millisecond, expected)
		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		t.Cleanup(cancel)

		// Start the ticker and wait for the deadline to pass.
		assert.NilError(t, ticker.Start(ctx, th, tq, pTrue, pLength))
		<-ctx.Done()

		assert.Equal(t, len(called), 3)
		assert.Equal(t, called[0], expected)
		assert.Equal(t, called[1], expected)
		assert.Equal(t, called[2], expected)
	})

	t.Run("Immediate", func(t *testing.T) {
		called = nil

		ticker := NewTickerImmediate(100*time.Millisecond, expected)
		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		t.Cleanup(cancel)

		// Start the ticker and wait for the deadline to pass.
		assert.NilError(t, ticker.Start(ctx, th, tq))
		<-ctx.Done()

		assert.Equal(t, len(called), 3)
		assert.Equal(t, called[0], expected, "expected at 0ms")
		assert.Equal(t, called[1], expected, "expected at 100ms")
		assert.Equal(t, called[2], expected, "expected at 200ms")
	})
}
