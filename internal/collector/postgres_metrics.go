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

func EnablePostgresMetrics(ctx context.Context, inCluster *v1beta1.PostgresCluster, config *Config) {
	if feature.Enabled(ctx, feature.OpenTelemetryMetrics) {
		if inCluster.Spec.PostgresVersion >= 17 {
			fiveSecondMetrics, _ = appendToJSONArray(fiveSecondMetrics, gtePG17)
		} else {
			fiveSecondMetrics, _ = appendToJSONArray(fiveSecondMetrics, ltPG17)
		}

		if inCluster.Spec.PostgresVersion >= 16 {
			fiveSecondMetrics, _ = appendToJSONArray(fiveSecondMetrics, gtePG16)
		} else {
			fiveSecondMetrics, _ = appendToJSONArray(fiveSecondMetrics, ltPG16)
		}
		// Add Prometheus exporter
		config.Exporters[Prometheus] = map[string]any{
			"endpoint": "0.0.0.0:8889",
		}

		config.Receivers[FiveSecondSqlQuery] = map[string]any{
			"driver":              "postgres",
			"datasource":          fmt.Sprintf(`host=localhost dbname=postgres port=5432 user=%s password=${env:PGPASSWORD}`, pgmonitor.MonitoringUser),
			"collection_interval": "5s",
			// Give Postgres time to finish setup.
			"initial_delay": "10s",
			"queries":       slices.Clone(fiveSecondMetrics),
		}

		config.Receivers[FiveMinuteSqlQuery] = map[string]any{
			"driver":              "postgres",
			"datasource":          fmt.Sprintf(`host=localhost dbname=postgres port=5432 user=%s password=${env:PGPASSWORD}`, pgmonitor.MonitoringUser),
			"collection_interval": "300s",
			// Give Postgres time to finish setup.
			"initial_delay": "10s",
			"queries":       slices.Clone(fiveMinuteMetrics),
		}
		// Add Metrics Pipeline
		config.Pipelines[PostgresMetrics] = Pipeline{
			Receivers: []ComponentID{FiveSecondSqlQuery, FiveMinuteSqlQuery},
			Exporters: []ComponentID{Prometheus},
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
