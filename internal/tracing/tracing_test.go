// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"gotest.tools/v3/assert"
)

func TestDefaultTracer(t *testing.T) {
	ctx := context.Background()

	t.Run("no-op", func(t *testing.T) {
		tracer := FromContext(ctx)
		_, s1 := tracer.Start(ctx, "asdf")
		defer s1.End()
		assert.Assert(t, !s1.IsRecording())

		_, s2 := Start(ctx, "doot")
		defer s2.End()
		assert.Assert(t, !s2.IsRecording())
	})

	t.Run("set", func(t *testing.T) {
		prior := global
		t.Cleanup(func() { SetDefaultTracer(prior) })

		recorder := tracetest.NewSpanRecorder()
		SetDefaultTracer(trace.NewTracerProvider(
			trace.WithSpanProcessor(recorder),
		).Tracer("myst"))

		_, span := Start(ctx, "zork")
		span.End()

		spans := recorder.Ended()
		assert.Equal(t, len(spans), 1)
		assert.Equal(t, spans[0].InstrumentationScope().Name, "myst")
		assert.Equal(t, spans[0].Name(), "zork")
	})
}

func TestNew(t *testing.T) {
	prior := otel.GetTracerProvider()
	t.Cleanup(func() { otel.SetTracerProvider(prior) })

	recorder := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(trace.NewTracerProvider(
		trace.WithSpanProcessor(recorder),
	))

	_, span := New("onetwo").Start(context.Background(), "three")
	span.End()

	spans := recorder.Ended()
	assert.Equal(t, len(spans), 1)
	assert.Equal(t, spans[0].InstrumentationScope().Name, "onetwo")
	assert.Equal(t, spans[0].InstrumentationScope().SchemaURL, semconv.SchemaURL)
	assert.Equal(t, spans[0].Name(), "three")
}

func TestFromContext(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()

	ctx := NewContext(context.Background(), trace.NewTracerProvider(
		trace.WithSpanProcessor(recorder),
	).Tracer("something"))

	_, span := Start(ctx, "spanspan")
	span.End()

	spans := recorder.Ended()
	assert.Equal(t, len(spans), 1)
	assert.Equal(t, spans[0].InstrumentationScope().Name, "something")
	assert.Equal(t, spans[0].Name(), "spanspan")
}

func TestAttributes(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()

	ctx := NewContext(context.Background(), trace.NewTracerProvider(
		trace.WithSpanProcessor(recorder),
	).Tracer(""))

	_, span := Start(ctx, "")
	Bool(span, "aa", true)
	Int(span, "abc", 99)
	String(span, "xyz", "copy pasta")
	span.End()

	spans := recorder.Ended()
	assert.Equal(t, len(spans), 1)
	assert.Equal(t, len(spans[0].Attributes()), 3)

	attrs := spans[0].Attributes()
	assert.Equal(t, string(attrs[0].Key), "aa")
	assert.Equal(t, string(attrs[1].Key), "abc")
	assert.Equal(t, string(attrs[2].Key), "xyz")
	assert.Equal(t, attrs[0].Value.AsInterface(), true)
	assert.Equal(t, attrs[1].Value.AsInterface(), int64(99))
	assert.Equal(t, attrs[2].Value.AsInterface(), "copy pasta")
}
