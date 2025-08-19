// Copyright 2024 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// The contents of "pgbouncer_metrics_queries.yaml" as JSON.
// See: https://pkg.go.dev/embed
//
//go:embed "generated/pgbouncer_metrics_queries.json"
var pgBouncerMetricsQueries json.RawMessage

// PGBouncerPostRotateScript is the script that is run after pgBouncer's log
// files have been rotated. The pgbouncer process is sent a sighup signal.
const PGBouncerPostRotateScript = "pkill -HUP --exact pgbouncer"

// NewConfigForPgBouncerPod creates a config for the OTel collector container
// that runs as a sidecar in the pgBouncer Pod
func NewConfigForPgBouncerPod(
	ctx context.Context, cluster *v1beta1.PostgresCluster, sqlQueryUsername, logfile string,
) *Config {
	if cluster.Spec.Proxy == nil || cluster.Spec.Proxy.PGBouncer == nil {
		// pgBouncer is disabled; return nil
		return nil
	}

	config := NewConfig(cluster.Spec.Instrumentation)

	EnablePgBouncerLogging(ctx, cluster, config, logfile)
	EnablePgBouncerMetrics(ctx, cluster, config, sqlQueryUsername)

	return config
}

// EnablePgBouncerLogging adds necessary configuration to the collector config to collect
// logs from pgBouncer when the OpenTelemetryLogging feature flag is enabled.
func EnablePgBouncerLogging(ctx context.Context,
	inCluster *v1beta1.PostgresCluster,
	outConfig *Config,
	logfile string,
) {
	var spec *v1beta1.InstrumentationLogsSpec
	if inCluster != nil && inCluster.Spec.Instrumentation != nil {
		spec = inCluster.Spec.Instrumentation.Logs
	}

	if OpenTelemetryLogsEnabled(ctx, inCluster) {
		directory := filepath.Dir(logfile)

		// Keep track of what log records and files have been processed.
		// Use a subdirectory of the logs directory to stay within the same failure domain.
		//
		// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/extension/storage/filestorage#readme
		outConfig.Extensions["file_storage/pgbouncer_logs"] = map[string]any{
			"directory":        directory + "/receiver",
			"create_directory": false,
			"fsync":            true,
		}

		// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/receiver/filelogreceiver#readme
		outConfig.Receivers["filelog/pgbouncer_log"] = map[string]any{
			// Read the log files and keep track of what has been processed.
			// We want to watch the ".log.1" file as well as it is possible that
			// a log entry or two will end up there after the original ".log"
			// file is renamed to ".log.1" during rotation. OTel will not create
			// duplicate log entries.
			"include": []string{directory + "/*.log", directory + "/*.log.1"},
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
				{"action": "insert", "key": "process.executable.name", "value": "pgbouncer"},
			},
		}

		// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/processor/transformprocessor#readme
		outConfig.Processors["transform/pgbouncer_logs"] = map[string]any{
			"log_statements": []map[string]any{{
				"statements": []string{
					// Set instrumentation scope
					`set(instrumentation_scope.name, "pgbouncer")`,

					// Extract timestamp, pid, log level, and message and store in cache.
					`merge_maps(log.cache, ExtractPatterns(log.body, ` +
						`"^(?<timestamp>\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2}\\.\\d{3} [A-Z]{3}) ` +
						`\\[(?<pid>\\d+)\\] (?<log_level>[A-Z]+) (?<msg>.*$)"), "insert")`,

					// https://opentelemetry.io/docs/specs/otel/logs/data-model/#field-severitytext
					`set(log.severity_text, log.cache["log_level"])`,

					// Map pgBouncer (libusual) "logging levels" to OpenTelemetry severity levels.
					//
					// https://github.com/libusual/libusual/blob/master/usual/logging.c
					// https://opentelemetry.io/docs/specs/otel/logs/data-model/#field-severitynumber
					// https://opentelemetry.io/docs/specs/otel/logs/data-model-appendix/#appendix-b-severitynumber-example-mappings
					// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/pkg/ottl/contexts/ottllog#enums
					`set(log.severity_number, SEVERITY_NUMBER_DEBUG)  where log.severity_text == "NOISE" or log.severity_text == "DEBUG"`,
					`set(log.severity_number, SEVERITY_NUMBER_INFO)   where log.severity_text == "LOG"`,
					`set(log.severity_number, SEVERITY_NUMBER_WARN)   where log.severity_text == "WARNING"`,
					`set(log.severity_number, SEVERITY_NUMBER_ERROR)  where log.severity_text == "ERROR"`,
					`set(log.severity_number, SEVERITY_NUMBER_FATAL)  where log.severity_text == "FATAL"`,

					// Parse the timestamp.
					// The format is neither RFC 3339 nor ISO 8601:
					//
					// The date and time are separated by a single space U+0020,
					// followed by a dot U+002E, milliseconds, another space U+0020,
					// then a timezone abbreviation.
					//
					// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/pkg/stanza/docs/types/timestamp.md
					`set(log.time, Time(log.cache["timestamp"], "%F %T.%L %Z")) where IsString(log.cache["timestamp"])`,

					// Keep the unparsed log record in a standard attribute, and replace
					// the log record body with the message field.
					//
					// https://github.com/open-telemetry/semantic-conventions/blob/v1.29.0/docs/general/logs.md
					`set(log.attributes["log.record.original"], log.body)`,

					// Set pid as attribute
					`set(log.attributes["process.pid"], log.cache["pid"])`,

					// Set the log message to body.
					`set(log.body, log.cache["msg"])`,
				},
			}},
		}

		// If there are exporters to be added to the logs pipelines defined in
		// the spec, add them to the pipeline. Otherwise, add the DebugExporter.
		exporters := []ComponentID{DebugExporter}
		if spec != nil && spec.Exporters != nil {
			exporters = slices.Clone(spec.Exporters)
		}

		outConfig.Pipelines["logs/pgbouncer"] = Pipeline{
			Extensions: []ComponentID{"file_storage/pgbouncer_logs"},
			Receivers:  []ComponentID{"filelog/pgbouncer_log"},
			Processors: []ComponentID{
				"resource/pgbouncer",
				"transform/pgbouncer_logs",
				ResourceDetectionProcessor,
				LogsBatchProcessor,
				CompactingProcessor,
			},
			Exporters: exporters,
		}
	}
}

// EnablePgBouncerMetrics adds necessary configuration to the collector config to scrape
// metrics from pgBouncer when the OpenTelemetryMetrics feature flag is enabled.
func EnablePgBouncerMetrics(ctx context.Context, inCluster *v1beta1.PostgresCluster,
	config *Config, sqlQueryUsername string) {

	if OpenTelemetryMetricsEnabled(ctx, inCluster) {
		// Add Prometheus exporter
		config.Exporters[Prometheus] = map[string]any{
			"endpoint": "0.0.0.0:" + strconv.Itoa(PrometheusPort),
		}

		// Add SqlQuery Receiver
		config.Receivers[SqlQuery] = map[string]any{
			"driver": "postgres",
			"datasource": fmt.Sprintf(
				`host=localhost dbname=pgbouncer port=5432 user=%s password=${env:PGPASSWORD}`,
				sqlQueryUsername),
			"queries": slices.Clone(pgBouncerMetricsQueries),
		}

		// If there are exporters to be added to the metrics pipelines defined
		// in the spec, add them to the pipeline.
		exporters := []ComponentID{Prometheus}
		if inCluster.Spec.Instrumentation.Metrics != nil &&
			inCluster.Spec.Instrumentation.Metrics.Exporters != nil {
			exporters = append(exporters, inCluster.Spec.Instrumentation.Metrics.Exporters...)
		}

		// Add Metrics Pipeline
		config.Pipelines[PGBouncerMetrics] = Pipeline{
			Receivers: []ComponentID{SqlQuery},
			Processors: []ComponentID{
				SubSecondBatchProcessor,
				CompactingProcessor,
			},
			Exporters: exporters,
		}
	}
}
