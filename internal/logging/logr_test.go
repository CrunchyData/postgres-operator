// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
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
	var calls []map[string]interface{}

	SetLogSink(&sink{
		fnInfo: func(_ int, _ string, kv ...interface{}) {
			m := make(map[string]interface{})
			for i := 0; i < len(kv); i += 2 {
				m[kv[i].(string)] = kv[i+1]
			}
			calls = append(calls, m)
		},
	})

	ctx := context.Background()

	// Nothing when there's no trace.
	FromContext(ctx).Info("")
	assert.Equal(t, calls[0]["spanid"], nil)
	assert.Equal(t, calls[0]["traceid"], nil)

	ctx, span := trace.NewTracerProvider().Tracer("").Start(ctx, "test-span")
	defer span.End()

	// OpenTelemetry trace context when there is.
	FromContext(ctx).Info("")
	assert.Equal(t, calls[1]["spanid"], span.SpanContext().SpanID())
	assert.Equal(t, calls[1]["traceid"], span.SpanContext().TraceID())
}

func TestSetLogSink(t *testing.T) {
	var calls []string

	SetLogSink(&sink{
		fnInfo: func(_ int, m string, _ ...interface{}) {
			calls = append(calls, m)
		},
	})

	global.Info("called")
	assert.DeepEqual(t, calls, []string{"called"})
}
