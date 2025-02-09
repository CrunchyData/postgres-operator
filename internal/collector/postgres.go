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
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func NewConfigForPostgresPod(ctx context.Context,
	inCluster *v1beta1.PostgresCluster,
	outParameters *postgres.Parameters,
) *Config {
	config := NewConfig(inCluster.Spec.Instrumentation)

	EnablePatroniLogging(ctx, inCluster, config)
	EnablePatroniMetrics(ctx, inCluster, config)
	EnablePostgresLogging(ctx, inCluster, config, outParameters)

	return config
}

// The contents of "postgres_logs_transforms.yaml" as JSON.
// See: https://pkg.go.dev/embed
//
//go:embed "generated/postgres_logs_transforms.json"
var postgresLogsTransforms json.RawMessage

// postgresCSVNames returns the names of fields in the CSV logs for version.
func postgresCSVNames(version int) string {
	// JSON is the preferred format, so use those names.
	// https://www.postgresql.org/docs/current/runtime-config-logging.html#RUNTIME-CONFIG-LOGGING-JSONLOG

	// https://www.postgresql.org/docs/8.3/runtime-config-logging.html#RUNTIME-CONFIG-LOGGING-CSVLOG
	names := `timestamp,user,dbname,pid` +
		`,connection_from` + // NOTE: this contains the JSON "remote_host" and "remote_port" values
		`,session_id,line_num,ps,session_start,vxid,txid` +
		`,error_severity,state_code,message,detail,hint` +
		`,internal_query,internal_position,context,statement,cursor_position` +
		`,location` // NOTE: this contains the JSON "func_name", "file_name", and "file_line_num" values

	// https://www.postgresql.org/docs/9.0/runtime-config-logging.html#RUNTIME-CONFIG-LOGGING-CSVLOG
	if version >= 9 {
		names += `,application_name`
	}

	// https://www.postgresql.org/docs/13/runtime-config-logging.html#RUNTIME-CONFIG-LOGGING-CSVLOG
	if version >= 13 {
		names += `,backend_type`
	}

	// https://www.postgresql.org/docs/14/runtime-config-logging.html#RUNTIME-CONFIG-LOGGING-CSVLOG
	if version >= 14 {
		names += `,leader_pid,query_id`
	}

	return names
}

func EnablePostgresLogging(
	ctx context.Context,
	inCluster *v1beta1.PostgresCluster,
	outConfig *Config,
	outParameters *postgres.Parameters,
) {
	if feature.Enabled(ctx, feature.OpenTelemetryLogs) {
		directory := postgres.LogDirectory()

		// https://www.postgresql.org/docs/current/runtime-config-logging.html
		outParameters.Mandatory.Add("logging_collector", "on")
		outParameters.Mandatory.Add("log_directory", directory)

		// PostgreSQL v8.3 adds support for CSV logging, and
		// PostgreSQL v15 adds support for JSON logging. The latter is preferred
		// because newlines are escaped as "\n", U+005C + U+006E.
		if inCluster.Spec.PostgresVersion < 15 {
			outParameters.Mandatory.Add("log_destination", "csvlog")
		} else {
			outParameters.Mandatory.Add("log_destination", "jsonlog")
		}

		// Keep seven days of logs named for the day of the week;
		// this has been the default produced by `initdb` for some time now.
		// NOTE: The automated portions of log_filename are *entirely* based
		// on time. There is no spelling that is guaranteed to be unique or
		// monotonically increasing.
		//
		// TODO(logs): Limit the size/bytes of logs without losing messages;
		// probably requires another process that deletes the oldest files.
		//
		// The ".log" suffix is replaced by ".json" for JSON log files.
		outParameters.Mandatory.Add("log_filename", "postgresql-%a.log")
		outParameters.Mandatory.Add("log_file_mode", "0660")
		outParameters.Mandatory.Add("log_rotation_age", "1d")
		outParameters.Mandatory.Add("log_rotation_size", "0")
		outParameters.Mandatory.Add("log_truncate_on_rotation", "on")

		// Log in a timezone that the OpenTelemetry Collector will understand.
		outParameters.Mandatory.Add("log_timezone", "UTC")

		// Keep track of what log records and files have been processed.
		// Use a subdirectory of the logs directory to stay within the same failure domain.
		//
		// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/extension/storage/filestorage#readme
		outConfig.Extensions["file_storage/postgres_logs"] = map[string]any{
			"directory":        directory + "/receiver",
			"create_directory": true,
			"fsync":            true,
		}

		// TODO(postgres-14): We can stop parsing CSV logs when 14 is EOL.
		// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/receiver/filelogreceiver#readme
		outConfig.Receivers["filelog/postgres_csvlog"] = map[string]any{
			// Read the CSV files and keep track of what has been processed.
			"include": []string{directory + "/*.csv"},
			"storage": "file_storage/postgres_logs",

			// Postgres does not escape newlines in its CSV log format. Search for
			// the beginning of every record, starting with an unquoted timestamp.
			// The 2nd through 5th fields are optional, so match through to the 7th field.
			// This should do a decent job of not matching the middle of some SQL statement.
			//
			// The number of fields has changed over the years, but the first few
			// are always formatted the same way.
			//
			// NOTE: This regexp is invoked in multi-line mode. https://go.dev/s/re2syntax
			"multiline": map[string]string{
				"line_start_pattern": `^\d{4}-\d\d-\d\d \d\d:\d\d:\d\d.\d{3} UTC` + // 1st: timestamp
					`,(?:"[_\D](?:[^"]|"")*")?` + //  2nd: user name
					`,(?:"[_\D](?:[^"]|"")*")?` + //  3rd: database name
					`,\d*,(?:"(?:[^"]|"")+")?` + //   4–5th: process id, connection
					`,[0-9a-f]+[.][0-9a-f]+,\d+,`, // 6–7th: session id, session line
			},

			// Differentiate these from the JSON ones below.
			"operators": []map[string]any{
				{"type": "move", "from": "body", "to": "body.original"},
				{"type": "add", "field": "body.format", "value": "csv"},
				{"type": "add", "field": "body.headers", "value": postgresCSVNames(inCluster.Spec.PostgresVersion)},
			},
		}

		// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/receiver/filelogreceiver#readme
		outConfig.Receivers["filelog/postgres_jsonlog"] = map[string]any{
			// Read the JSON files and keep track of what has been processed.
			"include": []string{directory + "/*.json"},
			"storage": "file_storage/postgres_logs",

			// Differentiate these from the CSV ones above.
			// TODO(postgres-14): We can stop parsing CSV logs when 14 is EOL.
			"operators": []map[string]any{
				{"type": "move", "from": "body", "to": "body.original"},
				{"type": "add", "field": "body.format", "value": "json"},
			},
		}

		// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/processor/resourceprocessor#readme
		outConfig.Processors["resource/postgres"] = map[string]any{
			"attributes": []map[string]any{
				// Container and Namespace names need no escaping because they are DNS labels.
				// Pod names need no escaping because they are DNS subdomains.
				//
				// https://kubernetes.io/docs/concepts/overview/working-with-objects/names
				// https://github.com/open-telemetry/semantic-conventions/blob/v1.29.0/docs/resource/k8s.md
				{"action": "insert", "key": "k8s.container.name", "value": naming.ContainerDatabase},
				{"action": "insert", "key": "k8s.namespace.name", "value": "${env:K8S_POD_NAMESPACE}"},
				{"action": "insert", "key": "k8s.pod.name", "value": "${env:K8S_POD_NAME}"},

				// https://github.com/open-telemetry/semantic-conventions/blob/v1.29.0/docs/database#readme
				{"action": "insert", "key": "db.system", "value": "postgresql"},
				{"action": "insert", "key": "db.version", "value": fmt.Sprint(inCluster.Spec.PostgresVersion)},
			},
		}

		// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/processor/transformprocessor#readme
		outConfig.Processors["transform/postgres_logs"] = map[string]any{
			"log_statements": slices.Clone(postgresLogsTransforms),
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

		outConfig.Pipelines["logs/postgres"] = Pipeline{
			Extensions: []ComponentID{"file_storage/postgres_logs"},
			// TODO(logs): Choose only one receiver, maybe?
			Receivers: []ComponentID{
				"filelog/postgres_csvlog",
				"filelog/postgres_jsonlog",
			},
			Processors: []ComponentID{
				"resource/postgres",
				"transform/postgres_logs",
				SubSecondBatchProcessor,
				CompactingProcessor,
			},
			Exporters: exporters,
		}

		// pgBackRest pipeline
		outConfig.Extensions["file_storage/pgbackrest_logs"] = map[string]any{
			"directory":        naming.PGBackRestPGDataLogPath + "/receiver",
			"create_directory": true,
			"fsync":            true,
		}

		outConfig.Receivers["filelog/pgbackrest_log"] = map[string]any{
			"include": []string{naming.PGBackRestPGDataLogPath + "/*.log"},
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

		// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/processor/resourceprocessor#readme
		outConfig.Processors["resource/pgbackrest"] = map[string]any{
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
		outConfig.Processors["transform/pgbackrest_logs"] = map[string]any{
			"log_statements": slices.Clone(pgBackRestLogsTransforms),
		}

		outConfig.Pipelines["logs/pgbackrest"] = Pipeline{
			Extensions: []ComponentID{"file_storage/pgbackrest_logs"},
			Receivers:  []ComponentID{"filelog/pgbackrest_log"},
			Processors: []ComponentID{
				"resource/pgbackrest",
				"transform/pgbackrest_logs",
				SubSecondBatchProcessor,
				CompactingProcessor,
			},
			Exporters: exporters,
		}
	}
}
