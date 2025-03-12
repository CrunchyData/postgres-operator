// Copyright 2024 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package collector_test

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/internal/collector"
	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestEnablePgAdminLogging(t *testing.T) {
	t.Run("NilInstrumentationSpec", func(t *testing.T) {
		gate := feature.NewGate()
		assert.NilError(t, gate.SetFromMap(map[string]bool{
			feature.OpenTelemetryLogs: true,
		}))

		ctx := feature.NewContext(context.Background(), gate)

		configmap := new(corev1.ConfigMap)
		initialize.Map(&configmap.Data)
		var instrumentation *v1beta1.InstrumentationSpec
		require.UnmarshalInto(t, &instrumentation, `{
			logs: { retentionPeriod: 12h },
		}`)
		err := collector.EnablePgAdminLogging(ctx, instrumentation, configmap)
		assert.NilError(t, err)

		assert.Assert(t, cmp.MarshalMatches(configmap.Data, `
collector.yaml: |
  # Generated by postgres-operator. DO NOT EDIT.
  # Your changes will not be saved.
  exporters:
    debug:
      verbosity: detailed
  extensions:
    file_storage/pgadmin_data_logs:
      create_directory: false
      directory: /var/lib/pgadmin/logs/receiver
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
    resource/pgadmin:
      attributes:
      - action: insert
        key: k8s.container.name
        value: pgadmin
      - action: insert
        key: k8s.namespace.name
        value: ${env:K8S_POD_NAMESPACE}
      - action: insert
        key: k8s.pod.name
        value: ${env:K8S_POD_NAME}
    resourcedetection:
      detectors: []
      override: false
      timeout: 30s
    transform/pgadmin_log:
      log_statements:
      - context: log
        statements:
        - set(attributes["log.record.original"], body)
        - set(cache, ParseJSON(body))
        - merge_maps(attributes, ExtractPatterns(cache["message"], "(?P<webrequest>[A-Z]{3}.*?[\\d]{3})"),
          "insert")
        - set(body, cache["message"])
        - set(instrumentation_scope.name, cache["name"])
        - set(severity_text, cache["level"])
        - set(time_unix_nano, Int(cache["time"]*1000000000))
        - set(severity_number, SEVERITY_NUMBER_DEBUG)  where severity_text == "DEBUG"
        - set(severity_number, SEVERITY_NUMBER_INFO)   where severity_text == "INFO"
        - set(severity_number, SEVERITY_NUMBER_WARN)   where severity_text == "WARNING"
        - set(severity_number, SEVERITY_NUMBER_ERROR)  where severity_text == "ERROR"
        - set(severity_number, SEVERITY_NUMBER_FATAL)  where severity_text == "CRITICAL"
  receivers:
    filelog/gunicorn:
      include:
      - /var/lib/pgadmin/logs/gunicorn.log
      storage: file_storage/pgadmin_data_logs
    filelog/pgadmin:
      include:
      - /var/lib/pgadmin/logs/pgadmin.log
      storage: file_storage/pgadmin_data_logs
  service:
    extensions:
    - file_storage/pgadmin_data_logs
    pipelines:
      logs/gunicorn:
        exporters:
        - debug
        processors:
        - resource/pgadmin
        - transform/pgadmin_log
        - resourcedetection
        - batch/logs
        - groupbyattrs/compact
        receivers:
        - filelog/gunicorn
      logs/pgadmin:
        exporters:
        - debug
        processors:
        - resource/pgadmin
        - transform/pgadmin_log
        - resourcedetection
        - batch/logs
        - groupbyattrs/compact
        receivers:
        - filelog/pgadmin
`))
	})

	t.Run("InstrumentationSpecDefined", func(t *testing.T) {
		gate := feature.NewGate()
		assert.NilError(t, gate.SetFromMap(map[string]bool{
			feature.OpenTelemetryLogs: true,
		}))

		ctx := feature.NewContext(context.Background(), gate)

		var spec v1beta1.InstrumentationSpec
		require.UnmarshalInto(t, &spec, `{
			config: {
				exporters: {
					googlecloud: {
						log: { default_log_name: opentelemetry.io/collector-exported-log },
						project: google-project-name,
					},
				},
			},
			logs: { exporters: [googlecloud] },
		}`)

		configmap := new(corev1.ConfigMap)
		initialize.Map(&configmap.Data)
		err := collector.EnablePgAdminLogging(ctx, &spec, configmap)
		assert.NilError(t, err)

		assert.Assert(t, cmp.MarshalMatches(configmap.Data, `
collector.yaml: |
  # Generated by postgres-operator. DO NOT EDIT.
  # Your changes will not be saved.
  exporters:
    debug:
      verbosity: detailed
    googlecloud:
      log:
        default_log_name: opentelemetry.io/collector-exported-log
      project: google-project-name
  extensions:
    file_storage/pgadmin_data_logs:
      create_directory: false
      directory: /var/lib/pgadmin/logs/receiver
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
    resource/pgadmin:
      attributes:
      - action: insert
        key: k8s.container.name
        value: pgadmin
      - action: insert
        key: k8s.namespace.name
        value: ${env:K8S_POD_NAMESPACE}
      - action: insert
        key: k8s.pod.name
        value: ${env:K8S_POD_NAME}
    resourcedetection:
      detectors: []
      override: false
      timeout: 30s
    transform/pgadmin_log:
      log_statements:
      - context: log
        statements:
        - set(attributes["log.record.original"], body)
        - set(cache, ParseJSON(body))
        - merge_maps(attributes, ExtractPatterns(cache["message"], "(?P<webrequest>[A-Z]{3}.*?[\\d]{3})"),
          "insert")
        - set(body, cache["message"])
        - set(instrumentation_scope.name, cache["name"])
        - set(severity_text, cache["level"])
        - set(time_unix_nano, Int(cache["time"]*1000000000))
        - set(severity_number, SEVERITY_NUMBER_DEBUG)  where severity_text == "DEBUG"
        - set(severity_number, SEVERITY_NUMBER_INFO)   where severity_text == "INFO"
        - set(severity_number, SEVERITY_NUMBER_WARN)   where severity_text == "WARNING"
        - set(severity_number, SEVERITY_NUMBER_ERROR)  where severity_text == "ERROR"
        - set(severity_number, SEVERITY_NUMBER_FATAL)  where severity_text == "CRITICAL"
  receivers:
    filelog/gunicorn:
      include:
      - /var/lib/pgadmin/logs/gunicorn.log
      storage: file_storage/pgadmin_data_logs
    filelog/pgadmin:
      include:
      - /var/lib/pgadmin/logs/pgadmin.log
      storage: file_storage/pgadmin_data_logs
  service:
    extensions:
    - file_storage/pgadmin_data_logs
    pipelines:
      logs/gunicorn:
        exporters:
        - googlecloud
        processors:
        - resource/pgadmin
        - transform/pgadmin_log
        - resourcedetection
        - batch/logs
        - groupbyattrs/compact
        receivers:
        - filelog/gunicorn
      logs/pgadmin:
        exporters:
        - googlecloud
        processors:
        - resource/pgadmin
        - transform/pgadmin_log
        - resourcedetection
        - batch/logs
        - groupbyattrs/compact
        receivers:
        - filelog/pgadmin
`))
	})
}
