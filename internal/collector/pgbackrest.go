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

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// The contents of "pgbackrest_logs_transforms.yaml" as JSON.
// See: https://pkg.go.dev/embed
//
//go:embed "generated/pgbackrest_logs_transforms.json"
var pgBackRestLogsTransforms json.RawMessage

func NewConfigForPgBackrestRepoHostPod(
	ctx context.Context,
	spec *v1beta1.InstrumentationSpec,
	repos []v1beta1.PGBackRestRepo,
) *Config {
	config := NewConfig(spec)

	if OpenTelemetryLogsEnabled(ctx, spec) {

		var directory string
		for _, repo := range repos {
			if repo.Volume != nil {
				directory = fmt.Sprintf(naming.PGBackRestRepoLogPath, repo.Name)
				break
			}
		}

		// We should only enter this function if a PVC is assigned for a dedicated repohost
		// but if we don't have one, exit early.
		if directory == "" {
			return config
		}

		// Keep track of what log records and files have been processed.
		// Use a subdirectory of the logs directory to stay within the same failure domain.
		config.Extensions["file_storage/pgbackrest_logs"] = map[string]any{
			"directory":        directory + "/receiver",
			"create_directory": false,
			"fsync":            true,
		}

		// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/receiver/filelogreceiver#readme
		config.Receivers["filelog/pgbackrest_log"] = map[string]any{
			// Read the files and keep track of what has been processed.
			// We use logrotate to rotate the pgbackrest logs which renames the
			// old .log file to .log.1. We want the collector to ingest logs from
			// both files as it is possible that pgbackrest will continue to write
			// a log record or two to the old file while rotation is occurring.
			// The collector knows not to create duplicate logs.
			"include": []string{
				directory + "/*.log", directory + "/*.log.1",
			},
			"storage": "file_storage/pgbackrest_logs",
			// pgBackRest prints logs with a log prefix, which includes a timestamp
			// as long as the timestamp is not turned off in the configuration.
			// When pgBackRest starts a process, it also will print a newline
			// (if the file has already been written to) and a process "banner"
			// which looks like "-------------------PROCESS START-------------------\n".
			// Therefore we break multiline on the timestamp or the 19 dashes that start the banner.
			// - https://github.com/pgbackrest/pgbackrest/blob/main/src/common/log.c#L451
			"multiline": map[string]string{
				"line_start_pattern": `^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3}|^-{19}`,
			},
		}

		config.Processors["resource/pgbackrest"] = map[string]any{
			"attributes": []map[string]any{
				// Container and Namespace names need no escaping because they are DNS labels.
				// Pod names need no escaping because they are DNS subdomains.
				//
				// https://kubernetes.io/docs/concepts/overview/working-with-objects/names
				// https://github.com/open-telemetry/semantic-conventions/blob/v1.29.0/docs/resource/k8s.md
				// https://github.com/open-telemetry/semantic-conventions/blob/v1.29.0/docs/general/logs.md
				{"action": "insert", "key": "k8s.container.name", "value": naming.PGBackRestRepoContainerName},
				{"action": "insert", "key": "k8s.namespace.name", "value": "${env:K8S_POD_NAMESPACE}"},
				{"action": "insert", "key": "k8s.pod.name", "value": "${env:K8S_POD_NAME}"},
				{"action": "insert", "key": "process.executable.name", "value": "pgbackrest"},
			},
		}

		// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/processor/transformprocessor#readme
		config.Processors["transform/pgbackrest_logs"] = map[string]any{
			"log_statements": slices.Clone(pgBackRestLogsTransforms),
		}

		// If there are exporters to be added to the logs pipelines defined in
		// the spec, add them to the pipeline. Otherwise, add the DebugExporter.
		exporters := []ComponentID{DebugExporter}
		if spec != nil && spec.Logs != nil && spec.Logs.Exporters != nil {
			exporters = slices.Clone(spec.Logs.Exporters)
		}

		config.Pipelines["logs/pgbackrest"] = Pipeline{
			Extensions: []ComponentID{"file_storage/pgbackrest_logs"},
			Receivers:  []ComponentID{"filelog/pgbackrest_log"},
			Processors: []ComponentID{
				"resource/pgbackrest",
				"transform/pgbackrest_logs",
				ResourceDetectionProcessor,
				LogsBatchProcessor,
				CompactingProcessor,
			},
			Exporters: exporters,
		}
	}
	return config
}
