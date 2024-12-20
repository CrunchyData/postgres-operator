// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"gotest.tools/v3/assert"
)

func TestCheck(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tracer := trace.NewTracerProvider(
		trace.WithSpanProcessor(recorder),
	).Tracer("")

	{
		_, span := tracer.Start(context.Background(), "")
		assert.Assert(t, Check(span, nil))
		span.End()

		spans := recorder.Ended()
		assert.Equal(t, len(spans), 1)
		assert.Equal(t, len(spans[0].Events()), 0, "expected no events")
	}

	{
		_, span := tracer.Start(context.Background(), "")
		assert.Assert(t, !Check(span, errors.New("msg")))
		span.End()

		spans := recorder.Ended()
		assert.Equal(t, len(spans), 2)
		assert.Equal(t, len(spans[1].Events()), 1, "expected one event")

		event := spans[1].Events()[0]
		assert.Equal(t, event.Name, semconv.ExceptionEventName)

		attrs := event.Attributes
		assert.Equal(t, len(attrs), 2)
		assert.Equal(t, string(attrs[0].Key), "exception.type")
		assert.Equal(t, string(attrs[1].Key), "exception.message")
		assert.Equal(t, attrs[0].Value.AsInterface(), "*errors.errorString")
		assert.Equal(t, attrs[1].Value.AsInterface(), "msg")
	}
}

func TestEscape(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tracer := trace.NewTracerProvider(
		trace.WithSpanProcessor(recorder),
	).Tracer("")

	{
		_, span := tracer.Start(context.Background(), "")
		assert.NilError(t, Escape(span, nil))
		span.End()

		spans := recorder.Ended()
		assert.Equal(t, len(spans), 1)
		assert.Equal(t, len(spans[0].Events()), 0, "expected no events")
	}

	{
		_, span := tracer.Start(context.Background(), "")
		expected := errors.New("somesuch")
		assert.Assert(t, errors.Is(Escape(span, expected), expected),
			"expected to unwrap the original error")
		span.End()

		spans := recorder.Ended()
		assert.Equal(t, len(spans), 2)
		assert.Equal(t, len(spans[1].Events()), 1, "expected one event")

		event := spans[1].Events()[0]
		assert.Equal(t, event.Name, semconv.ExceptionEventName)

		attrs := event.Attributes
		assert.Equal(t, len(attrs), 3)
		assert.Equal(t, string(attrs[0].Key), "exception.escaped")
		assert.Equal(t, string(attrs[1].Key), "exception.type")
		assert.Equal(t, string(attrs[2].Key), "exception.message")
		assert.Equal(t, attrs[0].Value.AsInterface(), true)
		assert.Equal(t, attrs[1].Value.AsInterface(), "*errors.errorString")
		assert.Equal(t, attrs[2].Value.AsInterface(), "somesuch")
	}
}
