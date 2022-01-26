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
	"testing"

	"github.com/go-logr/logr"
	"github.com/wojas/genericr"
	"go.opentelemetry.io/otel/sdk/trace"
	"gotest.tools/v3/assert"
)

func TestDiscard(t *testing.T) {
	assert.Equal(t, Discard(), logr.DiscardLogger{})
}

func TestFromContext(t *testing.T) {
	global = logr.DiscardLogger{}

	// Defaults to global.
	log := FromContext(context.Background())
	assert.Equal(t, log, global)

	// Retrieves from NewContext.
	double := struct{ logr.Logger }{logr.DiscardLogger{}}
	log = FromContext(NewContext(context.Background(), double))
	assert.Equal(t, log, double)
}

func TestFromContextTraceContext(t *testing.T) {
	var calls []map[string]interface{}

	SetLogFunc(0, func(input genericr.Entry) {
		calls = append(calls, input.FieldsMap())
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

func TestSetLogFunc(t *testing.T) {
	var calls []string

	SetLogFunc(0, func(input genericr.Entry) {
		calls = append(calls, input.Message)
	})

	global.Info("called")
	assert.DeepEqual(t, calls, []string{"called"})
}
