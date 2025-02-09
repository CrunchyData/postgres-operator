// Copyright 2024 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// The contents of "pgbouncer_metrics_queries.yaml" as JSON.
// See: https://pkg.go.dev/embed
//
//go:embed "generated/pgbouncer_metrics_queries.json"
var pgBouncerMetricsQueries json.RawMessage

// NewConfigForPgBouncerPod creates a config for the OTel collector container
// that runs as a sidecar in the pgBouncer Pod
func NewConfigForPgBouncerPod(
	ctx context.Context, cluster *v1beta1.PostgresCluster, sqlQueryUsername string,
) *Config {
	if cluster.Spec.Proxy == nil || cluster.Spec.Proxy.PGBouncer == nil {
		// pgBouncer is disabled; return nil
		return nil
	}

	config := NewConfig(cluster.Spec.Instrumentation)

	EnablePgBouncerLogging(ctx, cluster, config)
	EnablePgBouncerMetrics(ctx, config, sqlQueryUsername)

	return config
}

// EnablePgBouncerLogging adds necessary configuration to the collector config to collect
// logs from pgBouncer when the OpenTelemetryLogging feature flag is enabled.
func EnablePgBouncerLogging(ctx context.Context,
	inCluster *v1beta1.PostgresCluster,
	outConfig *Config) {
	if feature.Enabled(ctx, feature.OpenTelemetryLogs) {
		directory := naming.PGBouncerLogPath

		// Keep track of what log records and files have been processed.
		// Use a subdirectory of the logs directory to stay within the same failure domain.
		//
		// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/extension/storage/filestorage#readme
		outConfig.Extensions["file_storage/pgbouncer_logs"] = map[string]any{
			"directory":        directory + "/receiver",
			"create_directory": true,
			"fsync":            true,
		}

		// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/receiver/filelogreceiver#readme
		outConfig.Receivers["filelog/pgbouncer_log"] = map[string]any{
			// Read the log files and keep track of what has been processed.
			"include": []string{directory + "/*.log"},
			"storage": "file_storage/pgbouncer_logs",
		}

		// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/processor/resourceprocessor#readme
		outConfig.Processors["resource/pgbouncer"] = map[string]any{
			"attributes": []map[string]any{
				// Container and Namespace names need no escaping because they are DNS labels.
				// Pod names need no escaping because they are DNS subdomains.
				//
				// https://kubernetes.io/docs/concepts/overview/working-with-objects/names
				// https://github.com/open-telemetry/semantic-conventions/blob/v1.29.0/docs/resource/k8s.md
				{"action": "insert", "key": "k8s.container.name", "value": naming.ContainerPGBouncer},
				{"action": "insert", "key": "k8s.namespace.name", "value": "${env:K8S_POD_NAMESPACE}"},
				{"action": "insert", "key": "k8s.pod.name", "value": "${env:K8S_POD_NAME}"},
			},
		}

		// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/processor/transformprocessor#readme
		outConfig.Processors["transform/pgbouncer_logs"] = map[string]any{
			"log_statements": []map[string]any{{
				"context": "log",
				"statements": []string{
					// Set instrumentation scope
					`set(instrumentation_scope.name, "pgbouncer")`,

					// Extract timestamp, pid, log level, and message and store in cache.
					`merge_maps(cache, ExtractPatterns(body, ` +
						`"^(?<timestamp>\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2}\\.\\d{3} [A-Z]{3}) ` +
						`\\[(?<pid>\\d+)\\] (?<log_level>[A-Z]+) (?<msg>.*$)"), "insert")`,

					// https://opentelemetry.io/docs/specs/otel/logs/data-model/#field-severitytext
					`set(severity_text, cache["log_level"])`,

					// Map pgBouncer (libusual) "logging levels" to OpenTelemetry severity levels.
					//
					// https://github.com/libusual/libusual/blob/master/usual/logging.c
					// https://opentelemetry.io/docs/specs/otel/logs/data-model/#field-severitynumber
					// https://opentelemetry.io/docs/specs/otel/logs/data-model-appendix/#appendix-b-severitynumber-example-mappings
					// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/pkg/ottl/contexts/ottllog#enums
					`set(severity_number, SEVERITY_NUMBER_DEBUG)  where severity_text == "NOISE" or severity_text == "DEBUG"`,
					`set(severity_number, SEVERITY_NUMBER_INFO)   where severity_text == "LOG"`,
					`set(severity_number, SEVERITY_NUMBER_WARN)   where severity_text == "WARNING"`,
					`set(severity_number, SEVERITY_NUMBER_ERROR)  where severity_text == "ERROR"`,
					`set(severity_number, SEVERITY_NUMBER_FATAL)  where severity_text == "FATAL"`,

					// Parse the timestamp.
					// The format is neither RFC 3339 nor ISO 8601:
					//
					// The date and time are separated by a single space U+0020,
					// followed by a dot U+002E, milliseconds, another space U+0020,
					// then a timezone abbreviation.
					//
					// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/pkg/stanza/docs/types/timestamp.md
					`set(time, Time(cache["timestamp"], "%F %T.%L %Z"))`,

					// Keep the unparsed log record in a standard attribute, and replace
					// the log record body with the message field.
					//
					// https://github.com/open-telemetry/semantic-conventions/blob/v1.29.0/docs/general/logs.md
					`set(attributes["log.record.original"], body)`,

					// Set pid as attribute
					`set(attributes["process.pid"], cache["pid"])`,

					// Set the log message to body.
					`set(body, cache["msg"])`,
				},
			}},
		}

		// If there are exporters to be added to the logs pipelines defined in
		// the spec, add them to the pipeline. Otherwise, add the DebugExporter.
		var exporters []ComponentID
		if inCluster.Spec.Instrumentation != nil &&
			inCluster.Spec.Instrumentation.Logs != nil &&
			inCluster.Spec.Instrumentation.Logs.Exporters != nil {
			exporters = inCluster.Spec.Instrumentation.Logs.Exporters
		} else {
			exporters = []ComponentID{DebugExporter}
		}

		outConfig.Pipelines["logs/pgbouncer"] = Pipeline{
			Extensions: []ComponentID{"file_storage/pgbouncer_logs"},
			Receivers:  []ComponentID{"filelog/pgbouncer_log"},
			Processors: []ComponentID{
				"resource/pgbouncer",
				"transform/pgbouncer_logs",
				SubSecondBatchProcessor,
				CompactingProcessor,
			},
			Exporters: exporters,
		}
	}
}

// EnablePgBouncerMetrics adds necessary configuration to the collector config to scrape
// metrics from pgBouncer when the OpenTelemetryMetrics feature flag is enabled.
func EnablePgBouncerMetrics(ctx context.Context, config *Config, sqlQueryUsername string) {
	if feature.Enabled(ctx, feature.OpenTelemetryMetrics) {
		// Add Prometheus exporter
		config.Exporters[Prometheus] = map[string]any{
			"endpoint": "0.0.0.0:8889",
		}

		// Add SqlQuery Receiver
		config.Receivers[SqlQuery] = map[string]any{
			"driver": "postgres",
			"datasource": fmt.Sprintf(`host=localhost dbname=pgbouncer port=5432 user=%s password=${env:PGPASSWORD}`,
				sqlQueryUsername),
			"queries": slices.Clone(pgBouncerMetricsQueries),
		}

		// Add Metrics Pipeline
		config.Pipelines[Metrics] = Pipeline{
			Receivers: []ComponentID{SqlQuery},
			Exporters: []ComponentID{Prometheus},
		}
	}
}
