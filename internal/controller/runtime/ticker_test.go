// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

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
	th := handler.Funcs{GenericFunc: func(ctx context.Context, e event.GenericEvent, q workqueue.RateLimitingInterface) {
		called = append(called, e)

		assert.Equal(t, q, tq, "should be called with the queue passed in Start")
	}}

	t.Run("NotImmediate", func(t *testing.T) {
		called = nil

		ticker := NewTicker(100*time.Millisecond, expected, th)
		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		t.Cleanup(cancel)

		// Start the ticker and wait for the deadline to pass.
		assert.NilError(t, ticker.Start(ctx, tq))
		<-ctx.Done()

		assert.Equal(t, len(called), 2)
		assert.Equal(t, called[0], expected, "expected at 100ms")
		assert.Equal(t, called[1], expected, "expected at 200ms")
	})

	t.Run("Immediate", func(t *testing.T) {
		called = nil

		ticker := NewTickerImmediate(100*time.Millisecond, expected, th)
		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		t.Cleanup(cancel)

		// Start the ticker and wait for the deadline to pass.
		assert.NilError(t, ticker.Start(ctx, tq))
		<-ctx.Done()

		assert.Assert(t, len(called) > 2)
		assert.Equal(t, called[0], expected, "expected at 0ms")
		assert.Equal(t, called[1], expected, "expected at 100ms")
		assert.Equal(t, called[2], expected, "expected at 200ms")
	})
}
