// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// https://pkg.go.dev/go.opentelemetry.io/otel/trace
type (
	Span   = trace.Span
	Tracer = trace.Tracer
)

var global = noop.NewTracerProvider().Tracer("")

// SetDefaultTracer replaces the default Tracer with t. Before this is called,
// the default Tracer is a no-op.
func SetDefaultTracer(t Tracer) { global = t }

type tracerKey struct{}

// FromContext returns the Tracer stored by a prior call to [WithTracer] or [SetDefaultTracer].
func FromContext(ctx context.Context) Tracer {
	if t, ok := ctx.Value(tracerKey{}).(Tracer); ok {
		return t
	}
	return global
}

// NewContext returns a copy of ctx containing t. Retrieve it using [FromContext].
func NewContext(ctx context.Context, t Tracer) context.Context {
	return context.WithValue(ctx, tracerKey{}, t)
}

// New returns a Tracer produced by [otel.GetTracerProvider].
func New(name string, opts ...trace.TracerOption) Tracer {
	opts = append([]trace.TracerOption{
		trace.WithSchemaURL(semconv.SchemaURL),
	}, opts...)

	return otel.GetTracerProvider().Tracer(name, opts...)
}

// Start creates a Span and a Context containing it. It uses the Tracer returned by [FromContext].
func Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, Span) {
	return FromContext(ctx).Start(ctx, name, opts...)
}

// Bool sets the k attribute of s to v.
func Bool(s Span, k string, v bool) { s.SetAttributes(attribute.Bool(k, v)) }

// Int sets the k attribute of s to v.
func Int(s Span, k string, v int) { s.SetAttributes(attribute.Int(k, v)) }

// String sets the k attribute of s to v.
func String(s Span, k, v string) { s.SetAttributes(attribute.String(k, v)) }
