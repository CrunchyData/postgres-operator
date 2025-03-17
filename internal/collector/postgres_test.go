// Copyright 2024 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestEnablePostgresLogging(t *testing.T) {
	t.Run("NilInstrumentationSpec", func(t *testing.T) {
		gate := feature.NewGate()
		assert.NilError(t, gate.SetFromMap(map[string]bool{
			feature.OpenTelemetryLogs: true,
		}))
		ctx := feature.NewContext(context.Background(), gate)

		cluster := new(v1beta1.PostgresCluster)
		cluster.Spec.PostgresVersion = 99
		require.UnmarshalInto(t, &cluster.Spec, `{
			instrumentation: {
				logs: { retentionPeriod: 5h },
			},
		}`)

		config := NewConfig(nil)
		params := postgres.NewParameterSet()

		EnablePostgresLogging(ctx, cluster, config, params)

		result, err := config.ToYAML()
		assert.NilError(t, err)
		assert.DeepEqual(t, result, `# Generated by postgres-operator. DO NOT EDIT.
# Your changes will not be saved.
exporters:
  debug:
    verbosity: detailed
extensions:
  file_storage/pgbackrest_logs:
    create_directory: false
    directory: /pgdata/pgbackrest/log/receiver
    fsync: true
  file_storage/postgres_logs:
    create_directory: true
    directory: /pgdata/logs/postgres/receiver
    fsync: true
processors:
  batch/1s:
    timeout: 1s
  batch/200ms:
    timeout: 200ms
  batch/logs:
    send_batch_size: 8192
    timeout: 200ms
  groupbyattrs/compact: {}
  resource/pgbackrest:
    attributes:
    - action: insert
      key: k8s.container.name
      value: database
    - action: insert
      key: k8s.namespace.name
      value: ${env:K8S_POD_NAMESPACE}
    - action: insert
      key: k8s.pod.name
      value: ${env:K8S_POD_NAME}
  resource/postgres:
    attributes:
    - action: insert
      key: k8s.container.name
      value: database
    - action: insert
      key: k8s.namespace.name
      value: ${env:K8S_POD_NAMESPACE}
    - action: insert
      key: k8s.pod.name
      value: ${env:K8S_POD_NAME}
    - action: insert
      key: db.system
      value: postgresql
    - action: insert
      key: db.version
      value: "99"
  resourcedetection:
    detectors: []
    override: false
    timeout: 30s
  transform/pgbackrest_logs:
    log_statements:
    - context: log
      statements:
      - set(instrumentation_scope.name, "pgbackrest")
      - set(instrumentation_scope.schema_url, "https://opentelemetry.io/schemas/1.29.0")
      - 'merge_maps(log.cache, ExtractPatterns(log.body, "^(?<timestamp>\\d{4}-\\d{2}-\\d{2}
        \\d{2}:\\d{2}:\\d{2}\\.\\d{3}) (?<process_id>P\\d{2,3})\\s*(?<error_severity>\\S*):
        (?<message>(?s).*)$"), "insert") where Len(log.body) > 0'
      - set(log.severity_text, log.cache["error_severity"]) where IsString(log.cache["error_severity"])
      - set(log.severity_number, SEVERITY_NUMBER_TRACE) where log.severity_text ==
        "TRACE"
      - set(log.severity_number, SEVERITY_NUMBER_DEBUG) where log.severity_text ==
        "DEBUG"
      - set(log.severity_number, SEVERITY_NUMBER_DEBUG2) where log.severity_text ==
        "DETAIL"
      - set(log.severity_number, SEVERITY_NUMBER_INFO) where log.severity_text ==
        "INFO"
      - set(log.severity_number, SEVERITY_NUMBER_WARN) where log.severity_text ==
        "WARN"
      - set(log.severity_number, SEVERITY_NUMBER_ERROR) where log.severity_text ==
        "ERROR"
      - set(log.time, Time(log.cache["timestamp"], "%Y-%m-%d %H:%M:%S.%L")) where
        IsString(log.cache["timestamp"])
      - set(log.attributes["process.pid"], log.cache["process_id"])
      - set(log.attributes["log.record.original"], log.body)
      - set(log.body, log.cache["message"])
  transform/postgres_logs:
    log_statements:
    - conditions:
      - body["format"] == "csv"
      context: log
      statements:
      - set(log.cache, ParseCSV(log.body["original"], log.body["headers"], delimiter=",",
        mode="strict"))
      - merge_maps(log.cache, ExtractPatterns(log.cache["connection_from"], "(?:^[[]local[]]:(?<remote_port>.+)|:(?<remote_port>[^:]+))$"),
        "insert") where Len(log.cache["connection_from"]) > 0
      - set(log.cache["remote_host"], Substring(log.cache["connection_from"], 0, Len(log.cache["connection_from"])
        - Len(log.cache["remote_port"]) - 1)) where Len(log.cache["connection_from"])
        > 0 and IsString(log.cache["remote_port"])
      - set(log.cache["remote_host"], log.cache["connection_from"]) where Len(log.cache["connection_from"])
        > 0 and not IsString(log.cache["remote_host"])
      - merge_maps(log.cache, ExtractPatterns(log.cache["location"], "^(?:(?<func_name>[^,]+),
        )?(?<file_name>[^:]+):(?<file_line_num>\\d+)$"), "insert") where Len(log.cache["location"])
        > 0
      - set(log.cache["cursor_position"], Double(log.cache["cursor_position"])) where
        IsMatch(log.cache["cursor_position"], "^[0-9.]+$")
      - set(log.cache["file_line_num"], Double(log.cache["file_line_num"])) where
        IsMatch(log.cache["file_line_num"], "^[0-9.]+$")
      - set(log.cache["internal_position"], Double(log.cache["internal_position"]))
        where IsMatch(log.cache["internal_position"], "^[0-9.]+$")
      - set(log.cache["leader_pid"], Double(log.cache["leader_pid"])) where IsMatch(log.cache["leader_pid"],
        "^[0-9.]+$")
      - set(log.cache["line_num"], Double(log.cache["line_num"])) where IsMatch(log.cache["line_num"],
        "^[0-9.]+$")
      - set(log.cache["pid"], Double(log.cache["pid"])) where IsMatch(log.cache["pid"],
        "^[0-9.]+$")
      - set(log.cache["query_id"], Double(log.cache["query_id"])) where IsMatch(log.cache["query_id"],
        "^[0-9.]+$")
      - set(log.cache["remote_port"], Double(log.cache["remote_port"])) where IsMatch(log.cache["remote_port"],
        "^[0-9.]+$")
      - set(log.body["parsed"], log.cache)
    - context: log
      statements:
      - set(instrumentation_scope.name, "postgres")
      - set(instrumentation_scope.version, resource.attributes["db.version"])
      - set(log.cache, log.body["parsed"]) where log.body["format"] == "csv"
      - set(log.cache, ParseJSON(log.body["original"])) where log.body["format"] ==
        "json"
      - set(log.severity_text, log.cache["error_severity"])
      - set(log.severity_number, SEVERITY_NUMBER_TRACE)  where log.severity_text ==
        "DEBUG5"
      - set(log.severity_number, SEVERITY_NUMBER_TRACE2) where log.severity_text ==
        "DEBUG4"
      - set(log.severity_number, SEVERITY_NUMBER_TRACE3) where log.severity_text ==
        "DEBUG3"
      - set(log.severity_number, SEVERITY_NUMBER_TRACE4) where log.severity_text ==
        "DEBUG2"
      - set(log.severity_number, SEVERITY_NUMBER_DEBUG)  where log.severity_text ==
        "DEBUG1"
      - set(log.severity_number, SEVERITY_NUMBER_INFO)   where log.severity_text ==
        "INFO" or log.severity_text == "LOG"
      - set(log.severity_number, SEVERITY_NUMBER_INFO2)  where log.severity_text ==
        "NOTICE"
      - set(log.severity_number, SEVERITY_NUMBER_WARN)   where log.severity_text ==
        "WARNING"
      - set(log.severity_number, SEVERITY_NUMBER_ERROR)  where log.severity_text ==
        "ERROR"
      - set(log.severity_number, SEVERITY_NUMBER_FATAL)  where log.severity_text ==
        "FATAL"
      - set(log.severity_number, SEVERITY_NUMBER_FATAL2) where log.severity_text ==
        "PANIC"
      - set(log.time, Time(log.cache["timestamp"], "%F %T.%L %Z")) where IsString(log.cache["timestamp"])
      - set(instrumentation_scope.schema_url, "https://opentelemetry.io/schemas/1.29.0")
      - set(resource.attributes["db.system"], "postgresql")
      - set(log.attributes["log.record.original"], log.body["original"])
      - set(log.body, log.cache)
      - set(log.attributes["client.address"],  log.body["remote_host"])  where IsString(log.body["remote_host"])
      - set(log.attributes["client.port"], Int(log.body["remote_port"])) where IsDouble(log.body["remote_port"])
      - set(log.attributes["code.filepath"], log.body["file_name"]) where IsString(log.body["file_name"])
      - set(log.attributes["code.function"], log.body["func_name"]) where IsString(log.body["func_name"])
      - set(log.attributes["code.lineno"], Int(log.body["file_line_num"])) where IsDouble(log.body["file_line_num"])
      - set(log.attributes["db.namespace"], log.body["dbname"]) where IsString(log.body["dbname"])
      - set(log.attributes["db.response.status_code"], log.body["state_code"]) where
        IsString(log.body["state_code"])
      - set(log.attributes["process.creation.time"], Concat([ Substring(log.body["session_start"],
        0, 10), "T", Substring(log.body["session_start"], 11, 8), "Z"], "")) where
        IsMatch(log.body["session_start"], "^[^ ]{10} [^ ]{8} UTC$")
      - set(log.attributes["process.pid"], Int(log.body["pid"])) where IsDouble(log.body["pid"])
      - set(log.attributes["process.title"], log.body["ps"]) where IsString(log.body["ps"])
      - set(log.attributes["user.name"], log.body["user"]) where IsString(log.body["user"])
    - conditions:
      - 'Len(body["message"]) > 7 and Substring(body["message"], 0, 7) == "AUDIT:
        "'
      context: log
      statements:
      - set(log.body["pgaudit"], ParseCSV(Substring(log.body["message"], 7, Len(log.body["message"])
        - 7), "audit_type,statement_id,substatement_id,class,command,object_type,object_name,statement,parameter",
        delimiter=",", mode="strict"))
      - set(instrumentation_scope.name, "pgaudit") where Len(log.body["pgaudit"])
        > 0
receivers:
  filelog/pgbackrest_log:
    include:
    - /pgdata/pgbackrest/log/*.log
    multiline:
      line_start_pattern: ^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3}|^-{19}
    storage: file_storage/pgbackrest_logs
  filelog/postgres_csvlog:
    include:
    - /pgdata/logs/postgres/*.csv
    multiline:
      line_start_pattern: ^\d{4}-\d\d-\d\d \d\d:\d\d:\d\d.\d{3} UTC,(?:"[_\D](?:[^"]|"")*")?,(?:"[_\D](?:[^"]|"")*")?,\d*,(?:"(?:[^"]|"")+")?,[0-9a-f]+[.][0-9a-f]+,\d+,
    operators:
    - from: body
      to: body.original
      type: move
    - field: body.format
      type: add
      value: csv
    - field: body.headers
      type: add
      value: timestamp,user,dbname,pid,connection_from,session_id,line_num,ps,session_start,vxid,txid,error_severity,state_code,message,detail,hint,internal_query,internal_position,context,statement,cursor_position,location,application_name,backend_type,leader_pid,query_id
    storage: file_storage/postgres_logs
  filelog/postgres_jsonlog:
    include:
    - /pgdata/logs/postgres/*.json
    operators:
    - from: body
      to: body.original
      type: move
    - field: body.format
      type: add
      value: json
    storage: file_storage/postgres_logs
service:
  extensions:
  - file_storage/pgbackrest_logs
  - file_storage/postgres_logs
  pipelines:
    logs/pgbackrest:
      exporters:
      - debug
      processors:
      - resource/pgbackrest
      - transform/pgbackrest_logs
      - resourcedetection
      - batch/logs
      - groupbyattrs/compact
      receivers:
      - filelog/pgbackrest_log
    logs/postgres:
      exporters:
      - debug
      processors:
      - resource/postgres
      - transform/postgres_logs
      - resourcedetection
      - batch/logs
      - groupbyattrs/compact
      receivers:
      - filelog/postgres_csvlog
      - filelog/postgres_jsonlog
`)
	})

	t.Run("InstrumentationSpecDefined", func(t *testing.T) {
		gate := feature.NewGate()
		assert.NilError(t, gate.SetFromMap(map[string]bool{
			feature.OpenTelemetryLogs: true,
		}))
		ctx := feature.NewContext(context.Background(), gate)

		cluster := new(v1beta1.PostgresCluster)
		cluster.Spec.PostgresVersion = 99
		cluster.Spec.Instrumentation = testInstrumentationSpec()

		config := NewConfig(cluster.Spec.Instrumentation)
		params := postgres.NewParameterSet()

		EnablePostgresLogging(ctx, cluster, config, params)

		result, err := config.ToYAML()
		assert.NilError(t, err)
		assert.DeepEqual(t, result, `# Generated by postgres-operator. DO NOT EDIT.
# Your changes will not be saved.
exporters:
  debug:
    verbosity: detailed
  googlecloud:
    log:
      default_log_name: opentelemetry.io/collector-exported-log
    project: google-project-name
extensions:
  file_storage/pgbackrest_logs:
    create_directory: false
    directory: /pgdata/pgbackrest/log/receiver
    fsync: true
  file_storage/postgres_logs:
    create_directory: true
    directory: /pgdata/logs/postgres/receiver
    fsync: true
processors:
  batch/1s:
    timeout: 1s
  batch/200ms:
    timeout: 200ms
  batch/logs:
    send_batch_size: 8192
    timeout: 200ms
  groupbyattrs/compact: {}
  resource/pgbackrest:
    attributes:
    - action: insert
      key: k8s.container.name
      value: database
    - action: insert
      key: k8s.namespace.name
      value: ${env:K8S_POD_NAMESPACE}
    - action: insert
      key: k8s.pod.name
      value: ${env:K8S_POD_NAME}
  resource/postgres:
    attributes:
    - action: insert
      key: k8s.container.name
      value: database
    - action: insert
      key: k8s.namespace.name
      value: ${env:K8S_POD_NAMESPACE}
    - action: insert
      key: k8s.pod.name
      value: ${env:K8S_POD_NAME}
    - action: insert
      key: db.system
      value: postgresql
    - action: insert
      key: db.version
      value: "99"
  resourcedetection:
    detectors: []
    override: false
    timeout: 30s
  transform/pgbackrest_logs:
    log_statements:
    - context: log
      statements:
      - set(instrumentation_scope.name, "pgbackrest")
      - set(instrumentation_scope.schema_url, "https://opentelemetry.io/schemas/1.29.0")
      - 'merge_maps(log.cache, ExtractPatterns(log.body, "^(?<timestamp>\\d{4}-\\d{2}-\\d{2}
        \\d{2}:\\d{2}:\\d{2}\\.\\d{3}) (?<process_id>P\\d{2,3})\\s*(?<error_severity>\\S*):
        (?<message>(?s).*)$"), "insert") where Len(log.body) > 0'
      - set(log.severity_text, log.cache["error_severity"]) where IsString(log.cache["error_severity"])
      - set(log.severity_number, SEVERITY_NUMBER_TRACE) where log.severity_text ==
        "TRACE"
      - set(log.severity_number, SEVERITY_NUMBER_DEBUG) where log.severity_text ==
        "DEBUG"
      - set(log.severity_number, SEVERITY_NUMBER_DEBUG2) where log.severity_text ==
        "DETAIL"
      - set(log.severity_number, SEVERITY_NUMBER_INFO) where log.severity_text ==
        "INFO"
      - set(log.severity_number, SEVERITY_NUMBER_WARN) where log.severity_text ==
        "WARN"
      - set(log.severity_number, SEVERITY_NUMBER_ERROR) where log.severity_text ==
        "ERROR"
      - set(log.time, Time(log.cache["timestamp"], "%Y-%m-%d %H:%M:%S.%L")) where
        IsString(log.cache["timestamp"])
      - set(log.attributes["process.pid"], log.cache["process_id"])
      - set(log.attributes["log.record.original"], log.body)
      - set(log.body, log.cache["message"])
  transform/postgres_logs:
    log_statements:
    - conditions:
      - body["format"] == "csv"
      context: log
      statements:
      - set(log.cache, ParseCSV(log.body["original"], log.body["headers"], delimiter=",",
        mode="strict"))
      - merge_maps(log.cache, ExtractPatterns(log.cache["connection_from"], "(?:^[[]local[]]:(?<remote_port>.+)|:(?<remote_port>[^:]+))$"),
        "insert") where Len(log.cache["connection_from"]) > 0
      - set(log.cache["remote_host"], Substring(log.cache["connection_from"], 0, Len(log.cache["connection_from"])
        - Len(log.cache["remote_port"]) - 1)) where Len(log.cache["connection_from"])
        > 0 and IsString(log.cache["remote_port"])
      - set(log.cache["remote_host"], log.cache["connection_from"]) where Len(log.cache["connection_from"])
        > 0 and not IsString(log.cache["remote_host"])
      - merge_maps(log.cache, ExtractPatterns(log.cache["location"], "^(?:(?<func_name>[^,]+),
        )?(?<file_name>[^:]+):(?<file_line_num>\\d+)$"), "insert") where Len(log.cache["location"])
        > 0
      - set(log.cache["cursor_position"], Double(log.cache["cursor_position"])) where
        IsMatch(log.cache["cursor_position"], "^[0-9.]+$")
      - set(log.cache["file_line_num"], Double(log.cache["file_line_num"])) where
        IsMatch(log.cache["file_line_num"], "^[0-9.]+$")
      - set(log.cache["internal_position"], Double(log.cache["internal_position"]))
        where IsMatch(log.cache["internal_position"], "^[0-9.]+$")
      - set(log.cache["leader_pid"], Double(log.cache["leader_pid"])) where IsMatch(log.cache["leader_pid"],
        "^[0-9.]+$")
      - set(log.cache["line_num"], Double(log.cache["line_num"])) where IsMatch(log.cache["line_num"],
        "^[0-9.]+$")
      - set(log.cache["pid"], Double(log.cache["pid"])) where IsMatch(log.cache["pid"],
        "^[0-9.]+$")
      - set(log.cache["query_id"], Double(log.cache["query_id"])) where IsMatch(log.cache["query_id"],
        "^[0-9.]+$")
      - set(log.cache["remote_port"], Double(log.cache["remote_port"])) where IsMatch(log.cache["remote_port"],
        "^[0-9.]+$")
      - set(log.body["parsed"], log.cache)
    - context: log
      statements:
      - set(instrumentation_scope.name, "postgres")
      - set(instrumentation_scope.version, resource.attributes["db.version"])
      - set(log.cache, log.body["parsed"]) where log.body["format"] == "csv"
      - set(log.cache, ParseJSON(log.body["original"])) where log.body["format"] ==
        "json"
      - set(log.severity_text, log.cache["error_severity"])
      - set(log.severity_number, SEVERITY_NUMBER_TRACE)  where log.severity_text ==
        "DEBUG5"
      - set(log.severity_number, SEVERITY_NUMBER_TRACE2) where log.severity_text ==
        "DEBUG4"
      - set(log.severity_number, SEVERITY_NUMBER_TRACE3) where log.severity_text ==
        "DEBUG3"
      - set(log.severity_number, SEVERITY_NUMBER_TRACE4) where log.severity_text ==
        "DEBUG2"
      - set(log.severity_number, SEVERITY_NUMBER_DEBUG)  where log.severity_text ==
        "DEBUG1"
      - set(log.severity_number, SEVERITY_NUMBER_INFO)   where log.severity_text ==
        "INFO" or log.severity_text == "LOG"
      - set(log.severity_number, SEVERITY_NUMBER_INFO2)  where log.severity_text ==
        "NOTICE"
      - set(log.severity_number, SEVERITY_NUMBER_WARN)   where log.severity_text ==
        "WARNING"
      - set(log.severity_number, SEVERITY_NUMBER_ERROR)  where log.severity_text ==
        "ERROR"
      - set(log.severity_number, SEVERITY_NUMBER_FATAL)  where log.severity_text ==
        "FATAL"
      - set(log.severity_number, SEVERITY_NUMBER_FATAL2) where log.severity_text ==
        "PANIC"
      - set(log.time, Time(log.cache["timestamp"], "%F %T.%L %Z")) where IsString(log.cache["timestamp"])
      - set(instrumentation_scope.schema_url, "https://opentelemetry.io/schemas/1.29.0")
      - set(resource.attributes["db.system"], "postgresql")
      - set(log.attributes["log.record.original"], log.body["original"])
      - set(log.body, log.cache)
      - set(log.attributes["client.address"],  log.body["remote_host"])  where IsString(log.body["remote_host"])
      - set(log.attributes["client.port"], Int(log.body["remote_port"])) where IsDouble(log.body["remote_port"])
      - set(log.attributes["code.filepath"], log.body["file_name"]) where IsString(log.body["file_name"])
      - set(log.attributes["code.function"], log.body["func_name"]) where IsString(log.body["func_name"])
      - set(log.attributes["code.lineno"], Int(log.body["file_line_num"])) where IsDouble(log.body["file_line_num"])
      - set(log.attributes["db.namespace"], log.body["dbname"]) where IsString(log.body["dbname"])
      - set(log.attributes["db.response.status_code"], log.body["state_code"]) where
        IsString(log.body["state_code"])
      - set(log.attributes["process.creation.time"], Concat([ Substring(log.body["session_start"],
        0, 10), "T", Substring(log.body["session_start"], 11, 8), "Z"], "")) where
        IsMatch(log.body["session_start"], "^[^ ]{10} [^ ]{8} UTC$")
      - set(log.attributes["process.pid"], Int(log.body["pid"])) where IsDouble(log.body["pid"])
      - set(log.attributes["process.title"], log.body["ps"]) where IsString(log.body["ps"])
      - set(log.attributes["user.name"], log.body["user"]) where IsString(log.body["user"])
    - conditions:
      - 'Len(body["message"]) > 7 and Substring(body["message"], 0, 7) == "AUDIT:
        "'
      context: log
      statements:
      - set(log.body["pgaudit"], ParseCSV(Substring(log.body["message"], 7, Len(log.body["message"])
        - 7), "audit_type,statement_id,substatement_id,class,command,object_type,object_name,statement,parameter",
        delimiter=",", mode="strict"))
      - set(instrumentation_scope.name, "pgaudit") where Len(log.body["pgaudit"])
        > 0
receivers:
  filelog/pgbackrest_log:
    include:
    - /pgdata/pgbackrest/log/*.log
    multiline:
      line_start_pattern: ^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3}|^-{19}
    storage: file_storage/pgbackrest_logs
  filelog/postgres_csvlog:
    include:
    - /pgdata/logs/postgres/*.csv
    multiline:
      line_start_pattern: ^\d{4}-\d\d-\d\d \d\d:\d\d:\d\d.\d{3} UTC,(?:"[_\D](?:[^"]|"")*")?,(?:"[_\D](?:[^"]|"")*")?,\d*,(?:"(?:[^"]|"")+")?,[0-9a-f]+[.][0-9a-f]+,\d+,
    operators:
    - from: body
      to: body.original
      type: move
    - field: body.format
      type: add
      value: csv
    - field: body.headers
      type: add
      value: timestamp,user,dbname,pid,connection_from,session_id,line_num,ps,session_start,vxid,txid,error_severity,state_code,message,detail,hint,internal_query,internal_position,context,statement,cursor_position,location,application_name,backend_type,leader_pid,query_id
    storage: file_storage/postgres_logs
  filelog/postgres_jsonlog:
    include:
    - /pgdata/logs/postgres/*.json
    operators:
    - from: body
      to: body.original
      type: move
    - field: body.format
      type: add
      value: json
    storage: file_storage/postgres_logs
service:
  extensions:
  - file_storage/pgbackrest_logs
  - file_storage/postgres_logs
  pipelines:
    logs/pgbackrest:
      exporters:
      - googlecloud
      processors:
      - resource/pgbackrest
      - transform/pgbackrest_logs
      - resourcedetection
      - batch/logs
      - groupbyattrs/compact
      receivers:
      - filelog/pgbackrest_log
    logs/postgres:
      exporters:
      - googlecloud
      processors:
      - resource/postgres
      - transform/postgres_logs
      - resourcedetection
      - batch/logs
      - groupbyattrs/compact
      receivers:
      - filelog/postgres_csvlog
      - filelog/postgres_jsonlog
`)
	})
}
