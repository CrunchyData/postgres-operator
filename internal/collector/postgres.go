// Copyright 2024 - 2026 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func NewConfigForPostgresPod(ctx context.Context,
	inCluster *v1beta1.PostgresCluster,
	outParameters *postgres.ParameterSet,
) *Config {
	config := NewConfig(inCluster.Spec.Instrumentation)

	// Metrics
	EnablePostgresMetrics(ctx, inCluster, config)
	EnablePatroniMetrics(ctx, inCluster, config)

	// Logging
	EnablePostgresLogging(ctx, inCluster, config, outParameters)
	EnablePatroniLogging(ctx, inCluster, config)

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
	outParameters *postgres.ParameterSet,
) {
	if OpenTelemetryLogsEnabled(ctx, inCluster) {
		directory := postgres.LogDirectory()
		spec := inCluster.Spec.Instrumentation
		version := inCluster.Spec.PostgresVersion

		// https://www.postgresql.org/docs/current/runtime-config-logging.html
		outParameters.Add("logging_collector", "on")
		outParameters.Add("log_directory", directory)

		// PostgreSQL v8.3 adds support for CSV logging, and
		// PostgreSQL v15 adds support for JSON logging. The latter is preferred
		// because newlines are escaped as "\n", U+005C + U+006E.
		if version < 15 {
			outParameters.Add("log_destination", "csvlog")
		} else {
			outParameters.Add("log_destination", "jsonlog")
		}

		// If retentionPeriod is set in the spec, use that value; otherwise, we want
		// to use a reasonably short duration. Defaulting to 1 day.
		retentionPeriod := metav1.Duration{Duration: 24 * time.Hour}
		if spec.Logs != nil && spec.Logs.RetentionPeriod != nil {
			retentionPeriod = spec.Logs.RetentionPeriod.AsDuration()
		}
		logFilename, logRotationAge := generateLogFilenameAndRotationAge(retentionPeriod)

		// NOTE: The automated portions of log_filename are *entirely* based
		// on time. There is no spelling that is guaranteed to be unique or
		// monotonically increasing.
		//
		// TODO(logs): Limit the size/bytes of logs without losing messages;
		// probably requires another process that deletes the oldest files.
		//
		// The ".log" suffix is replaced by ".json" for JSON log files.
		outParameters.Add("log_filename", logFilename)
		outParameters.Add("log_file_mode", "0660")
		outParameters.Add("log_rotation_age", logRotationAge)
		outParameters.Add("log_rotation_size", "0")
		outParameters.Add("log_truncate_on_rotation", "on")

		// Log in a timezone that the OpenTelemetry Collector will understand.
		outParameters.Add("log_timezone", "UTC")

		// Keep track of what log records and files have been processed.
		// Use a subdirectory of the logs directory to stay within the same failure domain.
		// TODO(log-rotation): Create this directory during Collector startup.
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
			// The wildcard covers all potential log file names.
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
				{"type": "add", "field": "body.headers", "value": postgresCSVNames(version)},
			},
		}

		// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/receiver/filelogreceiver#readme
		outConfig.Receivers["filelog/postgres_jsonlog"] = map[string]any{
			// Read the JSON files and keep track of what has been processed.
			// The wildcard covers all potential log file names.
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
				{"action": "insert", "key": "process.executable.name", "value": "postgres"},

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
		exporters := []ComponentID{DebugExporter}
		if spec.Logs != nil && spec.Logs.Exporters != nil {
			exporters = slices.Clone(spec.Logs.Exporters)
		}

		postgresProcessors := []ComponentID{
			"resource/postgres",
			"transform/postgres_logs",
		}

		// We can only add the ResourceDetectionProcessor if there are detectors set,
		// otherwise it will fail. This is due to a change in the following upstream commmit:
		// https://github.com/open-telemetry/opentelemetry-collector-contrib/commit/50cd2e8433cee1e292e7b7afac9758365f3a1298
		if spec.Config != nil && spec.Config.Detectors != nil && len(spec.Config.Detectors) > 0 {
			postgresProcessors = append(postgresProcessors, ResourceDetectionProcessor)
		}

		// Order of processors matter so we add the batching and compacting processors after
		// potentially adding the resourcedetection processor
		postgresProcessors = append(postgresProcessors, LogsBatchProcessor, CompactingProcessor)

		outConfig.Pipelines["logs/postgres"] = Pipeline{
			Extensions: []ComponentID{"file_storage/postgres_logs"},
			// TODO(logs): Choose only one receiver, maybe?
			Receivers: []ComponentID{
				"filelog/postgres_csvlog",
				"filelog/postgres_jsonlog",
			},
			Processors: postgresProcessors,
			Exporters:  exporters,
		}

		// pgBackRest pipeline
		outConfig.Extensions["file_storage/pgbackrest_logs"] = map[string]any{
			"directory":        naming.PGBackRestPGDataLogPath + "/receiver",
			"create_directory": false,
			"fsync":            true,
		}

		// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/receiver/filelogreceiver#readme
		outConfig.Receivers["filelog/pgbackrest_log"] = map[string]any{
			// We use logrotate to rotate the pgbackrest logs which renames the
			// old .log file to .log.1. We want the collector to ingest logs from
			// both files as it is possible that pgbackrest will continue to write
			// a log record or two to the old file while rotation is occurring.
			// The collector knows not to create duplicate logs.
			"include": []string{
				naming.PGBackRestPGDataLogPath + "/*.log",
				naming.PGBackRestPGDataLogPath + "/*.log.1",
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
				{"action": "insert", "key": "process.executable.name", "value": "pgbackrest"},
			},
		}

		// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/processor/transformprocessor#readme
		outConfig.Processors["transform/pgbackrest_logs"] = map[string]any{
			"log_statements": slices.Clone(pgBackRestLogsTransforms),
		}

		pgbackrestProcessors := []ComponentID{
			"resource/pgbackrest",
			"transform/pgbackrest_logs",
		}

		// We can only add the ResourceDetectionProcessor if there are detectors set,
		// otherwise it will fail. This is due to a change in the following upstream commmit:
		// https://github.com/open-telemetry/opentelemetry-collector-contrib/commit/50cd2e8433cee1e292e7b7afac9758365f3a1298
		if spec.Config != nil && spec.Config.Detectors != nil && len(spec.Config.Detectors) > 0 {
			pgbackrestProcessors = append(pgbackrestProcessors, ResourceDetectionProcessor)
		}

		// Order of processors matter so we add the batching and compacting processors after
		// potentially adding the resourcedetection processor
		pgbackrestProcessors = append(pgbackrestProcessors, LogsBatchProcessor, CompactingProcessor)

		outConfig.Pipelines["logs/pgbackrest"] = Pipeline{
			Extensions: []ComponentID{"file_storage/pgbackrest_logs"},
			Receivers:  []ComponentID{"filelog/pgbackrest_log"},
			Processors: pgbackrestProcessors,
			Exporters:  exporters,
		}
	}
}

// generateLogFilenameAndRotationAge takes a retentionPeriod and returns a
// log_filename and log_rotation_age to be used to configure postgres logging
func generateLogFilenameAndRotationAge(
	retentionPeriod metav1.Duration,
) (logFilename, logRotationAge string) {
	// Given how postgres does its log rotation with the truncate feature, we
	// will always need to make up the total retention period with multiple log
	// files that hold subunits of the total time (e.g. if the retentionPeriod
	// is an hour, there will be 60 1-minute long files; if the retentionPeriod
	// is a day, there will be 24 1-hour long files, etc)

	hours := math.Ceil(retentionPeriod.Hours())

	switch true {
	case hours <= 1: // One hour's worth of logs in 60 minute long log files
		logFilename = "postgresql-%M.log"
		logRotationAge = "1min"
	case hours <= 24: // One day's worth of logs in 24 hour long log files
		logFilename = "postgresql-%H.log"
		logRotationAge = "1h"
	case hours <= 24*7: // One week's worth of logs in 7 day long log files
		logFilename = "postgresql-%a.log"
		logRotationAge = "1d"
	case hours <= 24*28: // One month's worth of logs in 28-31 day long log files
		logFilename = "postgresql-%d.log"
		logRotationAge = "1d"
	default: // One year's worth of logs in 365 day long log files
		logFilename = "postgresql-%j.log"
		logRotationAge = "1d"
	}

	return
}
