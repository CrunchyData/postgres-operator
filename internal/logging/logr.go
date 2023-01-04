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

package logging

import (
	"context"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/trace"
)

var global = logr.Discard()

// Discard returns a logr.Logger that discards all messages logged to it.
func Discard() logr.Logger { return logr.Discard() }

// SetLogSink replaces the global logr.Logger with sink. Before this is called,
// the global logr.Logger is a no-op.
func SetLogSink(sink logr.LogSink) { global = logr.New(sink) }

// NewContext returns a copy of ctx containing logger. Retrieve it using FromContext.
func NewContext(ctx context.Context, logger logr.Logger) context.Context {
	return logr.NewContext(ctx, logger)
}

// FromContext returns the global logr.Logger or the one stored by a prior call
// to NewContext.
func FromContext(ctx context.Context) logr.Logger {
	log, err := logr.FromContext(ctx)
	if err != nil {
		log = global
	}

	// Add trace context, if any, according to OpenTelemetry recommendations.
	// Omit trace flags for now because they don't seem relevant.
	// - https://github.com/open-telemetry/opentelemetry-specification/blob/v0.7.0/specification/logs/overview.md
	if sc := trace.SpanFromContext(ctx).SpanContext(); sc.IsValid() {
		log = log.WithValues("spanid", sc.SpanID(), "traceid", sc.TraceID())
	}

	return log
}

// sink implements logr.LogSink using two function pointers.
type sink struct {
	depth     int
	verbosity int
	names     []string
	values    []interface{}

	// TODO(cbandy): add names or frame to the functions below.

	fnError func(error, string, ...interface{})
	fnInfo  func(int, string, ...interface{})
}

var _ logr.LogSink = (*sink)(nil)

func (s *sink) Enabled(level int) bool     { return level <= s.verbosity }
func (s *sink) Init(info logr.RuntimeInfo) { s.depth = info.CallDepth }

func (s sink) combineValues(kv ...interface{}) []interface{} {
	if len(kv) == 0 {
		return s.values
	}
	if n := len(s.values); n > 0 {
		return append(s.values[:n:n], kv...)
	}
	return kv
}

func (s *sink) Error(err error, msg string, kv ...interface{}) {
	s.fnError(err, msg, s.combineValues(kv...)...)
}

func (s *sink) Info(level int, msg string, kv ...interface{}) {
	s.fnInfo(level, msg, s.combineValues(kv...)...)
}

func (s *sink) WithName(name string) logr.LogSink {
	n := len(s.names)
	out := *s
	out.names = append(out.names[:n:n], name)
	return &out
}

func (s *sink) WithValues(kv ...interface{}) logr.LogSink {
	n := len(s.values)
	out := *s
	out.values = append(out.values[:n:n], kv...)
	return &out
}
