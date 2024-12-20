// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/sdk/trace"
	"gotest.tools/v3/assert"
)

func TestDiscard(t *testing.T) {
	assert.Equal(t, Discard(), logr.Discard())
}

func TestFromContext(t *testing.T) {
	global = logr.Discard()

	// Defaults to global.
	log := FromContext(context.Background())
	assert.Equal(t, log, global)

	// Retrieves from NewContext.
	double := logr.New(&sink{})
	log = FromContext(NewContext(context.Background(), double))
	assert.Equal(t, log, double)
}

func TestFromContextTraceContext(t *testing.T) {
	var calls []map[string]any

	SetLogSink(&sink{
		fnInfo: func(_ int, _ string, kv ...any) {
			m := make(map[string]any)
			for i := 0; i < len(kv); i += 2 {
				m[kv[i].(string)] = kv[i+1]
			}
			calls = append(calls, m)
		},
	})

	ctx := context.Background()

	// Nothing when there's no trace.
	FromContext(ctx).Info("")
	assert.Equal(t, calls[0]["span_id"], nil)
	assert.Equal(t, calls[0]["trace_id"], nil)

	ctx, span := trace.NewTracerProvider().Tracer("").Start(ctx, "test-span")
	defer span.End()

	// OpenTelemetry trace context when there is.
	FromContext(ctx).Info("")
	assert.Equal(t, calls[1]["span_id"], span.SpanContext().SpanID())
	assert.Equal(t, calls[1]["trace_id"], span.SpanContext().TraceID())
}

func TestSetLogSink(t *testing.T) {
	var calls []string

	SetLogSink(&sink{
		fnInfo: func(_ int, m string, _ ...any) {
			calls = append(calls, m)
		},
	})

	global.Info("called")
	assert.DeepEqual(t, calls, []string{"called"})
}
