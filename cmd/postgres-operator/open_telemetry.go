package main

/*
Copyright 2020 - 2021 Crunchy Data
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
	"fmt"
	"io"
	"net/http"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/exporters/stdout"
	"go.opentelemetry.io/otel/exporters/trace/jaeger"
)

func initOpenTelemetry() (func(), error) {
	// At the time of this writing, the SDK (go.opentelemetry.io/otel@v0.13.0)
	// does not automatically initialize any trace or metric exporter. An upcoming
	// specification details environment variables that should facilitate this in
	// the future.
	//
	// - https://github.com/open-telemetry/opentelemetry-specification/blob/f5519f2b/specification/sdk-environment-variables.md

	switch os.Getenv("OTEL_EXPORTER") {
	case "jaeger":
		var endpoint jaeger.EndpointOption
		agent := os.Getenv("JAEGER_AGENT_ENDPOINT")
		collector := jaeger.CollectorEndpointFromEnv()

		if agent != "" {
			endpoint = jaeger.WithAgentEndpoint(agent)
		}
		if collector != "" {
			endpoint = jaeger.WithCollectorEndpoint(collector)
		}

		provider, flush, err := jaeger.NewExportPipeline(endpoint)
		if err != nil {
			return nil, fmt.Errorf("unable to initialize Jaeger exporter: %w", err)
		}

		global.SetTracerProvider(provider)
		return flush, nil

	case "json":
		var closer io.Closer
		filename := os.Getenv("OTEL_JSON_FILE")
		options := []stdout.Option{stdout.WithoutMetricExport()}

		if filename != "" {
			file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err != nil {
				return nil, fmt.Errorf("unable to open exporter file: %w", err)
			}
			closer = file
			options = append(options, stdout.WithWriter(file))
		}

		provider, pusher, err := stdout.NewExportPipeline(options, nil)
		if err != nil {
			return nil, fmt.Errorf("unable to initialize stdout exporter: %w", err)
		}
		flush := func() {
			pusher.Stop()
			if closer != nil {
				_ = closer.Close()
			}
		}

		global.SetTracerProvider(provider)
		return flush, nil
	}

	// $OTEL_EXPORTER is unset or unknown, so no TracerProvider has been assigned.
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
