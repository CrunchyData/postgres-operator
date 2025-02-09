// Copyright 2024 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"

	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func EnablePatroniLogging(ctx context.Context,
	inCluster *v1beta1.PostgresCluster,
	outConfig *Config,
) {
	if feature.Enabled(ctx, feature.OpenTelemetryLogs) {
		directory := naming.PatroniPGDataLogPath

		// Keep track of what log records and files have been processed.
		// Use a subdirectory of the logs directory to stay within the same failure domain.
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
			"include": []string{directory + "/*.log"},
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
			},
		}

		// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/processor/transformprocessor#readme
		outConfig.Processors["transform/patroni_logs"] = map[string]any{
			"log_statements": []map[string]any{{
				"context": "log",
				"statements": []string{
					`set(instrumentation_scope.name, "patroni")`,

					// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/pkg/ottl/ottlfuncs#parsejson
					`set(cache, ParseJSON(body["original"]))`,

					// The log severity is in the "levelname" field.
					// https://opentelemetry.io/docs/specs/otel/logs/data-model/#field-severitytext
					`set(severity_text, cache["levelname"])`,

					// Map Patroni (python) "logging levels" to OpenTelemetry severity levels.
					//
					// https://docs.python.org/3.6/library/logging.html#logging-levels
					// https://opentelemetry.io/docs/specs/otel/logs/data-model/#field-severitynumber
					// https://github.com/open-telemetry/opentelemetry-python/blob/v1.29.0/opentelemetry-api/src/opentelemetry/_logs/severity/__init__.py
					// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/pkg/ottl/contexts/ottllog#enums
					`set(severity_number, SEVERITY_NUMBER_DEBUG)  where severity_text == "DEBUG"`,
					`set(severity_number, SEVERITY_NUMBER_INFO)   where severity_text == "INFO"`,
					`set(severity_number, SEVERITY_NUMBER_WARN)   where severity_text == "WARNING"`,
					`set(severity_number, SEVERITY_NUMBER_ERROR)  where severity_text == "ERROR"`,
					`set(severity_number, SEVERITY_NUMBER_FATAL)  where severity_text == "CRITICAL"`,

					// Parse the "asctime" field into the record timestamp.
					// The format is neither RFC 3339 nor ISO 8601:
					//
					// The date and time are separated by a single space U+0020,
					// followed by a comma U+002C, then milliseconds.
					//
					// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/pkg/stanza/docs/types/timestamp.md
					// https://docs.python.org/3.6/library/logging.html#logging.LogRecord
					`set(time, Time(cache["asctime"], "%F %T,%L"))`,

					// Keep the unparsed log record in a standard attribute, and replace
					// the log record body with the message field.
					//
					// https://github.com/open-telemetry/semantic-conventions/blob/v1.29.0/docs/general/logs.md
					`set(attributes["log.record.original"], body["original"])`,
					`set(body, cache["message"])`,
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

		outConfig.Pipelines["logs/patroni"] = Pipeline{
			Extensions: []ComponentID{"file_storage/patroni_logs"},
			Receivers:  []ComponentID{"filelog/patroni_jsonlog"},
			Processors: []ComponentID{
				"resource/patroni",
				"transform/patroni_logs",
				SubSecondBatchProcessor,
				CompactingProcessor,
			},
			Exporters: exporters,
		}
	}
}

func EnablePatroniMetrics(ctx context.Context,
	inCluster *v1beta1.PostgresCluster,
	outConfig *Config,
) {
	if feature.Enabled(ctx, feature.OpenTelemetryMetrics) {
		// Add Prometheus exporter
		outConfig.Exporters[Prometheus] = map[string]any{
			"endpoint": "0.0.0.0:8889",
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

		// Add Metrics Pipeline
		outConfig.Pipelines[Metrics] = Pipeline{
			Receivers: []ComponentID{Prometheus},
			Exporters: []ComponentID{Prometheus},
		}
	}
}
