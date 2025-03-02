// Copyright 2024 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"slices"

	corev1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func EnablePgAdminLogging(ctx context.Context, spec *v1beta1.InstrumentationSpec,
	configmap *corev1.ConfigMap,
) error {
	if !feature.Enabled(ctx, feature.OpenTelemetryLogs) {
		return nil
	}
	otelConfig := NewConfig(spec)

	otelConfig.Extensions["file_storage/pgadmin_data_logs"] = map[string]any{
		"directory":        "/var/lib/pgadmin/logs/receiver",
		"create_directory": false,
		"fsync":            true,
	}

	otelConfig.Receivers["filelog/pgadmin"] = map[string]any{
		"include": []string{"/var/lib/pgadmin/logs/pgadmin.log"},
		"storage": "file_storage/pgadmin_data_logs",
	}
	otelConfig.Receivers["filelog/gunicorn"] = map[string]any{
		"include": []string{"/var/lib/pgadmin/logs/gunicorn.log"},
		"storage": "file_storage/pgadmin_data_logs",
	}

	otelConfig.Processors["resource/pgadmin"] = map[string]any{
		"attributes": []map[string]any{
			// Container and Namespace names need no escaping because they are DNS labels.
			// Pod names need no escaping because they are DNS subdomains.
			//
			// https://kubernetes.io/docs/concepts/overview/working-with-objects/names
			// https://github.com/open-telemetry/semantic-conventions/blob/v1.29.0/docs/resource/k8s.md
			// https://github.com/open-telemetry/semantic-conventions/blob/v1.29.0/docs/general/logs.md
			{"action": "insert", "key": "k8s.container.name", "value": naming.ContainerPGAdmin},
			{"action": "insert", "key": "k8s.namespace.name", "value": "${env:K8S_POD_NAMESPACE}"},
			{"action": "insert", "key": "k8s.pod.name", "value": "${env:K8S_POD_NAME}"},
		},
	}

	otelConfig.Processors["transform/pgadmin_log"] = map[string]any{
		"log_statements": []map[string]any{
			{
				"context": "log",
				"statements": []string{
					// Keep the unparsed log record in a standard attribute, and replace
					// the log record body with the message field.
					//
					// https://github.com/open-telemetry/semantic-conventions/blob/v1.29.0/docs/general/logs.md
					`set(attributes["log.record.original"], body)`,
					`set(cache, ParseJSON(body))`,
					`merge_maps(attributes, ExtractPatterns(cache["message"], "(?P<webrequest>[A-Z]{3}.*?[\\d]{3})"), "insert")`,
					`set(body, cache["message"])`,

					// Set instrumentation scope to the "name" from each log record.
					`set(instrumentation_scope.name, cache["name"])`,

					// https://opentelemetry.io/docs/specs/otel/logs/data-model/#field-severitytext
					`set(severity_text, cache["level"])`,
					`set(time_unix_nano, Int(cache["time"]*1000000000))`,

					// Map pgAdmin "logging levels" to OpenTelemetry severity levels.
					//
					// https://opentelemetry.io/docs/specs/otel/logs/data-model/#field-severitynumber
					// https://opentelemetry.io/docs/specs/otel/logs/data-model-appendix/#appendix-b-severitynumber-example-mappings
					// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/pkg/ottl/contexts/ottllog#enums
					`set(severity_number, SEVERITY_NUMBER_DEBUG)  where severity_text == "DEBUG"`,
					`set(severity_number, SEVERITY_NUMBER_INFO)   where severity_text == "INFO"`,
					`set(severity_number, SEVERITY_NUMBER_WARN)   where severity_text == "WARNING"`,
					`set(severity_number, SEVERITY_NUMBER_ERROR)  where severity_text == "ERROR"`,
					`set(severity_number, SEVERITY_NUMBER_FATAL)  where severity_text == "CRITICAL"`,
				},
			},
		},
	}

	// If there are exporters to be added to the logs pipelines defined in
	// the spec, add them to the pipeline. Otherwise, add the DebugExporter.
	exporters := []ComponentID{DebugExporter}
	if spec != nil && spec.Logs != nil && spec.Logs.Exporters != nil {
		exporters = slices.Clone(spec.Logs.Exporters)
	}

	otelConfig.Pipelines["logs/pgadmin"] = Pipeline{
		Extensions: []ComponentID{"file_storage/pgadmin_data_logs"},
		Receivers:  []ComponentID{"filelog/pgadmin"},
		Processors: []ComponentID{
			"resource/pgadmin",
			"transform/pgadmin_log",
			LogsBatchProcessor,
			CompactingProcessor,
		},
		Exporters: exporters,
	}

	otelConfig.Pipelines["logs/gunicorn"] = Pipeline{
		Extensions: []ComponentID{"file_storage/pgadmin_data_logs"},
		Receivers:  []ComponentID{"filelog/gunicorn"},
		Processors: []ComponentID{
			"resource/pgadmin",
			"transform/pgadmin_log",
			LogsBatchProcessor,
			CompactingProcessor,
		},
		Exporters: exporters,
	}

	otelYAML, err := otelConfig.ToYAML()
	if err == nil {
		configmap.Data["collector.yaml"] = otelYAML
	}
	return err
}
