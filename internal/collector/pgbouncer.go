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
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// The contents of "pgbouncer_metrics_queries.yaml" as JSON.
// See: https://pkg.go.dev/embed
//
//go:embed "generated/pgbouncer_metrics_queries.json"
var pgBouncerMetricsQueries json.RawMessage

// NewConfigForPgBouncerPod creates a config for the OTel collector container
// that runs as a sidecar in the pgBouncer Pod
func NewConfigForPgBouncerPod(
	ctx context.Context, cluster *v1beta1.PostgresCluster, sqlQueryUsername string,
) *Config {
	if cluster.Spec.Proxy == nil || cluster.Spec.Proxy.PGBouncer == nil {
		// pgBouncer is disabled; return nil
		return nil
	}

	config := NewConfig()

	EnablePgBouncerMetrics(ctx, config, sqlQueryUsername)

	return config
}

func EnablePgBouncerMetrics(ctx context.Context, config *Config, sqlQueryUsername string) {
	if feature.Enabled(ctx, feature.OpenTelemetryMetrics) {
		// Add Prometheus exporter
		config.Exporters[Prometheus] = map[string]any{
			"endpoint": "0.0.0.0:8889",
		}

		// Add SqlQuery Receiver
		config.Receivers[SqlQuery] = map[string]any{
			"driver": "postgres",
			"datasource": fmt.Sprintf(`host=localhost dbname=pgbouncer port=5432 user=%s password=${env:PGPASSWORD}`,
				sqlQueryUsername),
			"queries": slices.Clone(pgBouncerMetricsQueries),
		}

		// Add Metrics Pipeline
		config.Pipelines[Metrics] = Pipeline{
			Receivers: []ComponentID{SqlQuery},
			Exporters: []ComponentID{Prometheus},
		}
	}
}
