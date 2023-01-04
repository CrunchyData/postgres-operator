package main

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

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
)

func initOpenTelemetry() (func(), error) {
	// At the time of this writing, the SDK (go.opentelemetry.io/otel@v1.2.0)
	// does not automatically initialize any exporter. We import the OTLP and
	// stdout exporters and configure them below. Much of the OTLP exporter can
	// be configured through environment variables.
	//
	// - https://github.com/open-telemetry/opentelemetry-go/issues/2310
	// - https://github.com/open-telemetry/opentelemetry-specification/blob/v1.8.0/specification/sdk-environment-variables.md

	switch os.Getenv("OTEL_TRACES_EXPORTER") {
	case "json":
		var closer io.Closer
		filename := os.Getenv("OTEL_JSON_FILE")
		options := []stdouttrace.Option{}

		if filename != "" {
			file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err != nil {
				return nil, fmt.Errorf("unable to open exporter file: %w", err)
			}
			closer = file
			options = append(options, stdouttrace.WithWriter(file))
		}

		exporter, err := stdouttrace.New(options...)
		if err != nil {
			return nil, fmt.Errorf("unable to initialize stdout exporter: %w", err)
		}

		provider := trace.NewTracerProvider(trace.WithBatcher(exporter))
		flush := func() {
			_ = provider.Shutdown(context.TODO())
			if closer != nil {
				_ = closer.Close()
			}
		}

		otel.SetTracerProvider(provider)
		return flush, nil

	case "otlp":
		client := otlptracehttp.NewClient()
		exporter, err := otlptrace.New(context.TODO(), client)
		if err != nil {
			return nil, fmt.Errorf("unable to initialize OTLP exporter: %w", err)
		}

		provider := trace.NewTracerProvider(trace.WithBatcher(exporter))
		flush := func() {
			_ = provider.Shutdown(context.TODO())
		}

		otel.SetTracerProvider(provider)
		return flush, nil
	}

	// $OTEL_TRACES_EXPORTER is unset or unknown, so no TracerProvider has been assigned.
	// The default at this time is a single "no-op" tracer.

	return func() {}, nil
}

// otelTransportWrapper creates a function that wraps the provided net/http.RoundTripper
// with one that starts a span for each request, injects context into that request,
// and ends the span when that request's response body is closed.
func otelTransportWrapper(options ...otelhttp.Option) func(http.RoundTripper) http.RoundTripper {
	return func(rt http.RoundTripper) http.RoundTripper {
		return otelhttp.NewTransport(rt, options...)
	}
}
