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
	"strconv"

	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/pgmonitor"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// https://pkg.go.dev/embed
//
//go:embed "generated/postgres_5s_metrics.json"
var fiveSecondMetrics json.RawMessage

//go:embed "generated/postgres_5m_metrics.json"
var fiveMinuteMetrics json.RawMessage

//go:embed "generated/gte_pg17_metrics.json"
var gtePG17 json.RawMessage

//go:embed "generated/lt_pg17_metrics.json"
var ltPG17 json.RawMessage

//go:embed "generated/gte_pg16_metrics.json"
var gtePG16 json.RawMessage

//go:embed "generated/lt_pg16_metrics.json"
var ltPG16 json.RawMessage

type queryMetrics struct {
	Metrics []*metric `json:"metrics"`
	Query   string    `json:"sql"`
}

type metric struct {
	Aggregation      string            `json:"aggregation,omitempty"`
	AttributeColumns []string          `json:"attribute_columns,omitempty"`
	DataType         string            `json:"data_type,omitempty"`
	Description      string            `json:"description,omitempty"`
	MetricName       string            `json:"metric_name"`
	Monotonic        bool              `json:"monotonic,omitempty"`
	StartTsColumn    string            `json:"start_ts_column,omitempty"`
	StaticAttributes map[string]string `json:"static_attributes,omitempty"`
	TsColumn         string            `json:"ts_column,omitempty"`
	Unit             string            `json:"unit,omitempty"`
	ValueColumn      string            `json:"value_column"`
	ValueType        string            `json:"value_type,omitempty"`
}

func EnablePostgresMetrics(ctx context.Context, inCluster *v1beta1.PostgresCluster, config *Config) {
	if feature.Enabled(ctx, feature.OpenTelemetryMetrics) {
		log := logging.FromContext(ctx)
		var err error

		// We must create a copy of the fiveSecondMetrics variable, otherwise we
		// will continually append to it and blow up our ConfigMap
		fiveSecondMetricsClone := slices.Clone(fiveSecondMetrics)
		fiveMinuteMetricsClone := slices.Clone(fiveMinuteMetrics)

		if inCluster.Spec.PostgresVersion >= 17 {
			fiveSecondMetricsClone, err = appendToJSONArray(fiveSecondMetricsClone, gtePG17)
		} else {
			fiveSecondMetricsClone, err = appendToJSONArray(fiveSecondMetricsClone, ltPG17)
		}
		if err != nil {
			log.Error(err, "error compiling postgres metrics")
		}

		if inCluster.Spec.PostgresVersion >= 16 {
			fiveSecondMetricsClone, err = appendToJSONArray(fiveSecondMetricsClone, gtePG16)
		} else {
			fiveSecondMetricsClone, err = appendToJSONArray(fiveSecondMetricsClone, ltPG16)
		}
		if err != nil {
			log.Error(err, "error compiling postgres metrics")
		}

		// Remove any queries that user has specified in the spec
		if inCluster.Spec.Instrumentation != nil &&
			inCluster.Spec.Instrumentation.Metrics != nil &&
			inCluster.Spec.Instrumentation.Metrics.CustomQueries != nil &&
			inCluster.Spec.Instrumentation.Metrics.CustomQueries.Remove != nil {

			// Convert json to array of queryMetrics objects
			var fiveSecondMetricsArr []queryMetrics
			err := json.Unmarshal(fiveSecondMetricsClone, &fiveSecondMetricsArr)
			if err != nil {
				log.Error(err, "error compiling postgres metrics")
			}

			// Remove any specified metrics from the five second metrics
			fiveSecondMetricsArr = removeMetricsFromQueries(
				inCluster.Spec.Instrumentation.Metrics.CustomQueries.Remove, fiveSecondMetricsArr)

			// Convert json to array of queryMetrics objects
			var fiveMinuteMetricsArr []queryMetrics
			err = json.Unmarshal(fiveMinuteMetricsClone, &fiveMinuteMetricsArr)
			if err != nil {
				log.Error(err, "error compiling postgres metrics")
			}

			// Remove any specified metrics from the five minute metrics
			fiveMinuteMetricsArr = removeMetricsFromQueries(
				inCluster.Spec.Instrumentation.Metrics.CustomQueries.Remove, fiveMinuteMetricsArr)

			// Convert back to json data
			// The error return value can be ignored as the errchkjson linter
			// deems the []queryMetrics to be a safe argument:
			// https://github.com/breml/errchkjson
			fiveSecondMetricsClone, _ = json.Marshal(fiveSecondMetricsArr)
			fiveMinuteMetricsClone, _ = json.Marshal(fiveMinuteMetricsArr)
		}

		// Add Prometheus exporter
		config.Exporters[Prometheus] = map[string]any{
			"endpoint": "0.0.0.0:" + strconv.Itoa(PrometheusPort),
		}

		config.Receivers[FiveSecondSqlQuery] = map[string]any{
			"driver": "postgres",
			"datasource": fmt.Sprintf(
				`host=localhost dbname=postgres port=5432 user=%s password=${env:PGPASSWORD}`,
				pgmonitor.MonitoringUser),
			"collection_interval": "5s",
			// Give Postgres time to finish setup.
			"initial_delay": "10s",
			"queries":       slices.Clone(fiveSecondMetricsClone),
		}

		config.Receivers[FiveMinuteSqlQuery] = map[string]any{
			"driver": "postgres",
			"datasource": fmt.Sprintf(
				`host=localhost dbname=postgres port=5432 user=%s password=${env:PGPASSWORD}`,
				pgmonitor.MonitoringUser),
			"collection_interval": "300s",
			// Give Postgres time to finish setup.
			"initial_delay": "10s",
			"queries":       slices.Clone(fiveMinuteMetricsClone),
		}

		// Add Metrics Pipeline
		config.Pipelines[PostgresMetrics] = Pipeline{
			Receivers: []ComponentID{FiveSecondSqlQuery, FiveMinuteSqlQuery},
			Processors: []ComponentID{
				SubSecondBatchProcessor,
				CompactingProcessor,
			},
			Exporters: []ComponentID{Prometheus},
		}

		// Add custom queries if they are defined in the spec
		if inCluster.Spec.Instrumentation != nil &&
			inCluster.Spec.Instrumentation.Metrics != nil &&
			inCluster.Spec.Instrumentation.Metrics.CustomQueries != nil &&
			inCluster.Spec.Instrumentation.Metrics.CustomQueries.Add != nil {

			for _, querySet := range inCluster.Spec.Instrumentation.Metrics.CustomQueries.Add {
				// Create a receiver for the query set
				receiverName := "sqlquery/" + querySet.Name
				config.Receivers[receiverName] = map[string]any{
					"driver": "postgres",
					"datasource": fmt.Sprintf(
						`host=localhost dbname=postgres port=5432 user=%s password=${env:PGPASSWORD}`,
						pgmonitor.MonitoringUser),
					"collection_interval": querySet.CollectionInterval,
					// Give Postgres time to finish setup.
					"initial_delay": "10s",
					"queries": "${file:/etc/otel-collector/" +
						querySet.Name + "/" + querySet.Queries.Key + "}",
				}

				// Add the receiver to the pipeline
				pipeline := config.Pipelines[PostgresMetrics]
				pipeline.Receivers = append(pipeline.Receivers, receiverName)
				config.Pipelines[PostgresMetrics] = pipeline
			}
		}
	}
}

// appendToJSONArray appends elements of a json.RawMessage containing an array
// to another json.RawMessage containing an array.
func appendToJSONArray(a1, a2 json.RawMessage) (json.RawMessage, error) {
	var slc1 []json.RawMessage
	if err := json.Unmarshal(a1, &slc1); err != nil {
		return nil, err
	}

	var slc2 []json.RawMessage
	if err := json.Unmarshal(a2, &slc2); err != nil {
		return nil, err
	}

	mergedSlice := append(slc1, slc2...)

	merged, err := json.Marshal(mergedSlice)
	if err != nil {
		return nil, err
	}

	return merged, nil
}

func removeMetricsFromQueries(metricsToRemove []string,
	queryMetricsArr []queryMetrics,
) []queryMetrics {
	// Iterate over the metrics that should be removed
Outer:
	for _, metricToRemove := range metricsToRemove {
		// Iterate over array of query/metrics objects
		for j, queryAndMetrics := range queryMetricsArr {
			// Iterate over the metrics array
			metricsArr := queryAndMetrics.Metrics
			for k, metric := range metricsArr {
				// Check to see if the metric_name matches the metricToRemove
				if metric.MetricName == metricToRemove {
					// Remove the metric. Since there won't ever be any
					// duplicates, we will be exiting this loop early and
					// therefore don't care about the order of the metrics
					// array.
					metricsArr[len(metricsArr)-1], metricsArr[k] = nil, metricsArr[len(metricsArr)-1]
					metricsArr = metricsArr[:len(metricsArr)-1]
					queryMetricsArr[j].Metrics = metricsArr

					// If the metrics array is empty, remove the query/metrics
					// map entirely. Again, we don't care about order.
					if len(metricsArr) == 0 {
						queryMetricsArr[j] = queryMetricsArr[len(queryMetricsArr)-1]
						queryMetricsArr = queryMetricsArr[:len(queryMetricsArr)-1]
					}

					// We found and deleted the metric, so we can continue
					// to the next iteration of the Outer loop.
					continue Outer
				}
			}
		}
	}

	return queryMetricsArr
}
