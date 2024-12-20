// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"net/http"
	"os"

	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/contrib/propagators/autoprop"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"

	"github.com/crunchydata/postgres-operator/internal/logging"
)

func initOpenTelemetry(ctx context.Context) (func(context.Context) error, error) {
	var started []interface{ Shutdown(context.Context) error }

	// shutdown returns the results of calling all the Shutdown methods in started.
	var shutdown = func(ctx context.Context) error {
		var err error
		for _, s := range started {
			err = errors.Join(err, s.Shutdown(ctx))
		}
		started = nil
		return err
	}

	// The default for OTEL_PROPAGATORS is "tracecontext,baggage".
	otel.SetTextMapPropagator(autoprop.NewTextMapPropagator())

	// Skip any remaining setup when OTEL_SDK_DISABLED is exactly "true".
	// - https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables
	if os.Getenv("OTEL_SDK_DISABLED") == "true" {
		return shutdown, nil
	}

	log := logging.FromContext(ctx).WithName("open-telemetry")
	otel.SetLogger(log)
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		// TODO(events): Emit this as an event instead.
		log.V(1).Info(semconv.ExceptionEventName,
			string(semconv.ExceptionMessageKey), err)
	}))

	// Build a resource from the OTEL_RESOURCE_ATTRIBUTES and OTEL_SERVICE_NAME environment variables.
	// - https://opentelemetry.io/docs/languages/go/resources
	self, _ := resource.Merge(resource.NewSchemaless(
		semconv.ServiceVersion(versionString),
	), resource.Default())

	// Provide defaults for some other detectable attributes.
	if r, err := resource.New(ctx,
		resource.WithProcessRuntimeName(),
		resource.WithProcessRuntimeVersion(),
		resource.WithProcessRuntimeDescription(),
	); err == nil {
		self, _ = resource.Merge(r, self)
	}
	if r, err := resource.New(ctx,
		resource.WithHost(),
		resource.WithOS(),
	); err == nil {
		self, _ = resource.Merge(r, self)
	}

	// The default for OTEL_TRACES_EXPORTER is "otlp" but we prefer "none".
	// Only assign an exporter when the environment variable is set.
	if os.Getenv("OTEL_TRACES_EXPORTER") != "" {
		exporter, err := autoexport.NewSpanExporter(ctx)
		if err != nil {
			return nil, errors.Join(err, shutdown(ctx))
		}

		// The defaults for this batch processor come from the OTEL_BSP_* environment variables.
		// - https://pkg.go.dev/go.opentelemetry.io/otel/sdk/internal/env
		provider := trace.NewTracerProvider(
			trace.WithBatcher(exporter),
			trace.WithResource(self),
		)
		started = append(started, provider)
		otel.SetTracerProvider(provider)
	}

	return shutdown, nil
}

// otelTransportWrapper creates a function that wraps the provided net/http.RoundTripper
// with one that starts a span for each request, injects context into that request,
// and ends the span when that request's response body is closed.
func otelTransportWrapper(options ...otelhttp.Option) func(http.RoundTripper) http.RoundTripper {
	return func(rt http.RoundTripper) http.RoundTripper {
		return otelhttp.NewTransport(rt, options...)
	}
}
