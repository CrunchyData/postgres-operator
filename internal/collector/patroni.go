// Copyright 2024 - 2026 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"slices"
	"strconv"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func EnablePatroniLogging(ctx context.Context,
	inCluster *v1beta1.PostgresCluster,
	outConfig *Config,
) {
	if OpenTelemetryLogsEnabled(ctx, inCluster) {
		spec := inCluster.Spec.Instrumentation
		directory := naming.PatroniPGDataLogPath

		// Keep track of what log records and files have been processed.
		// Use a subdirectory of the logs directory to stay within the same failure domain.
		// TODO(log-rotation): Create this directory during Collector startup.
		//
		// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/extension/storage/filestorage#readme
		outConfig.Extensions["file_storage/patroni_logs"] = map[string]any{
			"directory":        directory + "/receiver",
			"create_directory": true,
			"fsync":            true,
		}

		// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/receiver/filelogreceiver#readme
		outConfig.Receivers["filelog/patroni_jsonlog"] = map[string]any{
			// Read the JSON files and keep track of what has been processed.
			// When patroni rotates its log files, it renames the old .log file
			// to .log.1. We want the collector to ingest logs from both files
			// as it is possible that patroni will continue to write a log
			// record or two to the old file while rotation is occurring. The
			// collector knows not to create duplicate logs.
			"include": []string{
				directory + "/*.log", directory + "/*.log.1",
			},
			"storage": "file_storage/patroni_logs",

			"operators": []map[string]any{
				{"type": "move", "from": "body", "to": "body.original"},
			},
		}

		// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/processor/resourceprocessor#readme
		outConfig.Processors["resource/patroni"] = map[string]any{
			"attributes": []map[string]any{
				// Container and Namespace names need no escaping because they are DNS labels.
				// Pod names need no escaping because they are DNS subdomains.
				//
				// https://kubernetes.io/docs/concepts/overview/working-with-objects/names
				// https://github.com/open-telemetry/semantic-conventions/blob/v1.29.0/docs/resource/k8s.md
				{"action": "insert", "key": "k8s.container.name", "value": naming.ContainerDatabase},
				{"action": "insert", "key": "k8s.namespace.name", "value": "${env:K8S_POD_NAMESPACE}"},
				{"action": "insert", "key": "k8s.pod.name", "value": "${env:K8S_POD_NAME}"},
				{"action": "insert", "key": "process.executable.name", "value": "patroni"},
			},
		}

		// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/processor/transformprocessor#readme
		outConfig.Processors["transform/patroni_logs"] = map[string]any{
			"log_statements": []map[string]any{{
				"statements": []string{
					`set(instrumentation_scope.name, "patroni")`,

					// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/pkg/ottl/ottlfuncs#parsejson
					`set(log.cache, ParseJSON(log.body["original"]))`,

					// The log severity is in the "levelname" field.
					// https://opentelemetry.io/docs/specs/otel/logs/data-model/#field-severitytext
					`set(log.severity_text, log.cache["levelname"])`,

					// Map Patroni (python) "logging levels" to OpenTelemetry severity levels.
					//
					// https://docs.python.org/3.6/library/logging.html#logging-levels
					// https://opentelemetry.io/docs/specs/otel/logs/data-model/#field-severitynumber
					// https://github.com/open-telemetry/opentelemetry-python/blob/v1.29.0/opentelemetry-api/src/opentelemetry/_logs/severity/__init__.py
					// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/pkg/ottl/contexts/ottllog#enums
					`set(log.severity_number, SEVERITY_NUMBER_DEBUG)  where log.severity_text == "DEBUG"`,
					`set(log.severity_number, SEVERITY_NUMBER_INFO)   where log.severity_text == "INFO"`,
					`set(log.severity_number, SEVERITY_NUMBER_WARN)   where log.severity_text == "WARNING"`,
					`set(log.severity_number, SEVERITY_NUMBER_ERROR)  where log.severity_text == "ERROR"`,
					`set(log.severity_number, SEVERITY_NUMBER_FATAL)  where log.severity_text == "CRITICAL"`,

					// Parse the "asctime" field into the record timestamp.
					// The format is neither RFC 3339 nor ISO 8601:
					//
					// The date and time are separated by a single space U+0020,
					// followed by a comma U+002C, then milliseconds.
					//
					// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/pkg/stanza/docs/types/timestamp.md
					// https://docs.python.org/3.6/library/logging.html#logging.LogRecord
					`set(log.time, Time(log.cache["asctime"], "%F %T,%L")) where IsString(log.cache["asctime"])`,

					// Keep the unparsed log record in a standard attribute, and replace
					// the log record body with the message field.
					//
					// https://github.com/open-telemetry/semantic-conventions/blob/v1.29.0/docs/general/logs.md
					`set(log.attributes["log.record.original"], log.body["original"])`,
					`set(log.body, log.cache["message"])`,
				},
			}},
		}

		// If there are exporters to be added to the logs pipelines defined in
		// the spec, add them to the pipeline. Otherwise, add the DebugExporter.
		exporters := []ComponentID{DebugExporter}
		if spec.Logs != nil && spec.Logs.Exporters != nil {
			exporters = slices.Clone(spec.Logs.Exporters)
		}

		patroniProcessors := []ComponentID{
			"resource/patroni",
			"transform/patroni_logs",
		}

		// We can only add the ResourceDetectionProcessor if there are detectors set,
		// otherwise it will fail. This is due to a change in the following upstream commmit:
		// https://github.com/open-telemetry/opentelemetry-collector-contrib/commit/50cd2e8433cee1e292e7b7afac9758365f3a1298
		if spec.Config != nil && spec.Config.Detectors != nil && len(spec.Config.Detectors) > 0 {
			patroniProcessors = append(patroniProcessors, ResourceDetectionProcessor)
		}

		// Order of processors matter so we add the batching and compacting processors after
		// potentially adding the resourcedetection processor
		patroniProcessors = append(patroniProcessors, LogsBatchProcessor, CompactingProcessor)

		outConfig.Pipelines["logs/patroni"] = Pipeline{
			Extensions: []ComponentID{"file_storage/patroni_logs"},
			Receivers:  []ComponentID{"filelog/patroni_jsonlog"},
			Processors: patroniProcessors,
			Exporters:  exporters,
		}
	}
}

func EnablePatroniMetrics(ctx context.Context,
	inCluster *v1beta1.PostgresCluster,
	outConfig *Config,
) {
	if OpenTelemetryMetricsEnabled(ctx, inCluster) {
		// Add Prometheus exporter
		outConfig.Exporters[Prometheus] = map[string]any{
			"endpoint": "0.0.0.0:" + strconv.Itoa(PrometheusPort),
		}

		// Add Prometheus Receiver
		outConfig.Receivers[Prometheus] = map[string]any{
			"config": map[string]any{
				"scrape_configs": []map[string]any{
					{
						"job_name": "patroni",
						"scheme":   "https",
						"tls_config": map[string]any{
							"insecure_skip_verify": true,
						},
						"scrape_interval": "10s",
						"static_configs": []map[string]any{
							{
								"targets": []string{
									"0.0.0.0:8008",
								},
							},
						},
					},
				},
			},
		}

		// If there are exporters to be added to the metrics pipelines defined
		// in the spec, add them to the pipeline.
		exporters := []ComponentID{Prometheus}
		if inCluster.Spec.Instrumentation.Metrics != nil &&
			inCluster.Spec.Instrumentation.Metrics.Exporters != nil {
			exporters = append(exporters, inCluster.Spec.Instrumentation.Metrics.Exporters...)
		}

		// Add Metrics Pipeline
		outConfig.Pipelines[PatroniMetrics] = Pipeline{
			Receivers: []ComponentID{Prometheus},
			Processors: []ComponentID{
				SubSecondBatchProcessor,
				CompactingProcessor,
			},
			Exporters: exporters,
		}
	}
}
