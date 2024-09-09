// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"context"
	"time"

	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type ticker struct {
	time.Duration
	event.GenericEvent
	Handler   handler.EventHandler
	Immediate bool
}

// NewTicker returns a Source that emits e every d.
func NewTicker(d time.Duration, e event.GenericEvent,
	h handler.EventHandler) source.Source {
	return &ticker{Duration: d, GenericEvent: e, Handler: h}
}

// NewTickerImmediate returns a Source that emits e at start and every d.
func NewTickerImmediate(d time.Duration, e event.GenericEvent,
	h handler.EventHandler) source.Source {
	return &ticker{Duration: d, GenericEvent: e, Handler: h, Immediate: true}
}

func (t ticker) String() string { return "every " + t.Duration.String() }

// Start is called by controller-runtime Controller and returns quickly.
// It cleans up when ctx is cancelled.
func (t ticker) Start(
	ctx context.Context, q workqueue.RateLimitingInterface,
) error {
	ticker := time.NewTicker(t.Duration)

	// Pass t.GenericEvent to h when it is not filtered out by p.
	// - https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/source/internal#EventHandler
	emit := func() {
		t.Handler.Generic(ctx, t.GenericEvent, q)
	}

	if t.Immediate {
		emit()
	}

	// Repeat until ctx is cancelled.
	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				emit()
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}
