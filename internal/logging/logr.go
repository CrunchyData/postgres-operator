/*
Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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
	"github.com/wojas/genericr"
	"go.opentelemetry.io/otel/trace"
)

var global = logr.Discard()

// Discard returns a logr.Logger that discards all messages logged to it.
func Discard() logr.Logger { return logr.DiscardLogger{} }

// SetLogFunc replaces the global logr.Logger with log that gets called when an
// entry's level is at or below verbosity. (Only the most important entries are
// passed when verbosity is zero.) Before this is called, the global logr.Logger
// is a no-op.
func SetLogFunc(verbosity int, log genericr.LogFunc) {
	global = genericr.New(log).WithCaller(true).WithVerbosity(verbosity)
}

// NewContext returns a copy of ctx containing logger. Retrieve it using FromContext.
func NewContext(ctx context.Context, logger logr.Logger) context.Context {
	return logr.NewContext(ctx, logger)
}

// FromContext returns the global logr.Logger or the one stored by a prior call
// to NewContext.
func FromContext(ctx context.Context) logr.Logger {
	var log logr.Logger

	if log = logr.FromContext(ctx); log == nil {
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
