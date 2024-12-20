// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
)

// Check returns true when err is nil. Otherwise, it adds err as an exception
// event on s and returns false. If you intend to return err, consider using
// [Escape] instead.
//
// See: https://opentelemetry.io/docs/specs/semconv/exceptions/exceptions-spans
func Check(s Span, err error) bool {
	if err == nil {
		return true
	}
	if s.IsRecording() {
		s.RecordError(err)
	}
	return false
}

// Escape adds non-nil err as an escaped exception event on s and returns err.
// See: https://opentelemetry.io/docs/specs/semconv/exceptions/exceptions-spans
func Escape(s Span, err error) error {
	if err != nil && s.IsRecording() {
		s.RecordError(err, trace.WithAttributes(semconv.ExceptionEscaped(true)))
	}
	return err
}
