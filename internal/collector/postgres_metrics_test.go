// Copyright 2024 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"encoding/json"
	"testing"

	"gotest.tools/v3/assert"
)

func TestRemoveMetricsFromQueries(t *testing.T) {
	// Convert json to map
	var fiveMinuteMetricsArr []queryMetrics
	err := json.Unmarshal(fiveMinuteMetrics, &fiveMinuteMetricsArr)
	assert.NilError(t, err)

	assert.Equal(t, len(fiveMinuteMetricsArr), 3)
	newArr := removeMetricsFromQueries([]string{"ccp_database_size_bytes"}, fiveMinuteMetricsArr)
	assert.Equal(t, len(newArr), 2)

	t.Run("DeleteOneMetric", func(t *testing.T) {
		sqlMetricsData := `[
  {
    "metrics": [
      {
        "description": "Count of sequences that have reached greater than or equal to 75% of their max available numbers.\nFunction monitor.sequence_status() can provide more details if run directly on system.\n",
        "metric_name": "ccp_sequence_exhaustion_count",
        "static_attributes": { "server": "localhost:5432" },
        "value_column": "count"
      }
    ],
    "sql": "SELECT count(*) AS count FROM (\n     SELECT CEIL((s.max_value-min_value::NUMERIC+1)/s.increment_by::NUMERIC) AS slots\n        , CEIL((COALESCE(s.last_value,s.min_value)-s.min_value::NUMERIC+1)/s.increment_by::NUMERIC) AS used\n    FROM pg_catalog.pg_sequences s\n) x WHERE (ROUND(used/slots*100)::int) \u003e 75;\n"
  },
  {
    "metrics": [
      {
        "attribute_columns": ["dbname"],
        "description": "Number of times disk blocks were found already in the buffer cache, so that a read was not necessary",
        "metric_name": "ccp_stat_database_blks_hit",
        "static_attributes": { "server": "localhost:5432" },
        "value_column": "blks_hit"
      },
      {
        "attribute_columns": ["dbname"],
        "description": "Number of disk blocks read in this database",
        "metric_name": "ccp_stat_database_blks_read",
        "static_attributes": { "server": "localhost:5432" },
        "value_column": "blks_read"
      }
    ],
    "sql": "SELECT s.datname AS dbname , s.xact_commit , s.xact_rollback , s.blks_read , s.blks_hit , s.tup_returned , s.tup_fetched , s.tup_inserted , s.tup_updated , s.tup_deleted , s.conflicts , s.temp_files , s.temp_bytes , s.deadlocks FROM pg_catalog.pg_stat_database s JOIN pg_catalog.pg_database d ON d.datname = s.datname WHERE d.datistemplate = false;\n"
  }
]`
		var sqlMetricsArr []queryMetrics
		err := json.Unmarshal([]byte(sqlMetricsData), &sqlMetricsArr)
		assert.NilError(t, err)

		assert.Equal(t, len(sqlMetricsArr), 2)
		metricsArr := sqlMetricsArr[1].Metrics
		assert.Equal(t, len(metricsArr), 2)

		refinedSqlMetricsArr := removeMetricsFromQueries([]string{"ccp_stat_database_blks_hit"}, sqlMetricsArr)
		assert.Equal(t, len(refinedSqlMetricsArr), 2)
		metricsArr = refinedSqlMetricsArr[1].Metrics
		assert.Equal(t, len(metricsArr), 1)
		remainingMetric := metricsArr[0]
		assert.Equal(t, remainingMetric.MetricName, "ccp_stat_database_blks_read")
	})

	t.Run("DeleteQueryMetricSet", func(t *testing.T) {
		sqlMetricsData := `[
  {
    "metrics": [
      {
        "description": "Count of sequences that have reached greater than or equal to 75% of their max available numbers.\nFunction monitor.sequence_status() can provide more details if run directly on system.\n",
        "metric_name": "ccp_sequence_exhaustion_count",
        "static_attributes": { "server": "localhost:5432" },
        "value_column": "count"
      }
    ],
    "sql": "SELECT count(*) AS count FROM (\n     SELECT CEIL((s.max_value-min_value::NUMERIC+1)/s.increment_by::NUMERIC) AS slots\n        , CEIL((COALESCE(s.last_value,s.min_value)-s.min_value::NUMERIC+1)/s.increment_by::NUMERIC) AS used\n    FROM pg_catalog.pg_sequences s\n) x WHERE (ROUND(used/slots*100)::int) \u003e 75;\n"
  },
  {
    "metrics": [
      {
        "attribute_columns": ["dbname"],
        "description": "Number of times disk blocks were found already in the buffer cache, so that a read was not necessary",
        "metric_name": "ccp_stat_database_blks_hit",
        "static_attributes": { "server": "localhost:5432" },
        "value_column": "blks_hit"
      },
      {
        "attribute_columns": ["dbname"],
        "description": "Number of disk blocks read in this database",
        "metric_name": "ccp_stat_database_blks_read",
        "static_attributes": { "server": "localhost:5432" },
        "value_column": "blks_read"
      }
    ],
    "sql": "SELECT s.datname AS dbname , s.xact_commit , s.xact_rollback , s.blks_read , s.blks_hit , s.tup_returned , s.tup_fetched , s.tup_inserted , s.tup_updated , s.tup_deleted , s.conflicts , s.temp_files , s.temp_bytes , s.deadlocks FROM pg_catalog.pg_stat_database s JOIN pg_catalog.pg_database d ON d.datname = s.datname WHERE d.datistemplate = false;\n"
  }
]`
		var sqlMetricsArr []queryMetrics
		err := json.Unmarshal([]byte(sqlMetricsData), &sqlMetricsArr)
		assert.NilError(t, err)

		assert.Equal(t, len(sqlMetricsArr), 2)
		metricsArr := sqlMetricsArr[1].Metrics
		assert.Equal(t, len(metricsArr), 2)

		refinedSqlMetricsArr := removeMetricsFromQueries([]string{"ccp_stat_database_blks_hit",
			"ccp_stat_database_blks_read"}, sqlMetricsArr)
		assert.Equal(t, len(refinedSqlMetricsArr), 1)
		metricsArr = sqlMetricsArr[0].Metrics
		assert.Equal(t, len(metricsArr), 1)
	})

}
