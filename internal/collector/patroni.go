// Copyright 2024 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"

	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func EnablePatroniMetrics(ctx context.Context,
	inCluster *v1beta1.PostgresCluster,
	outConfig *Config,
) {
	if feature.Enabled(ctx, feature.OpenTelemetryMetrics) {
		// Add Prometheus exporter
		outConfig.Exporters[Prometheus] = map[string]any{
			"endpoint": "0.0.0.0:8889",
		}

		// Add Prometheus Receiver
		outConfig.Receivers[Prometheus] = map[string]any{
			"config": map[string]any{
				"scrape_configs": []map[string]any{
					{
						"job_name": "patroni",
						"scheme":   "https",
						"tls_config": map[string]any{
							"insecure_skip_verify": true,
						},
						"scrape_interval": "10s",
						"static_configs": []map[string]any{
							{
								"targets": []string{
									"0.0.0.0:8008",
								},
							},
						},
					},
				},
			},
		}

		// Add Metrics Pipeline
		outConfig.Pipelines[Metrics] = Pipeline{
			Receivers: []ComponentID{Prometheus},
			Exporters: []ComponentID{Prometheus},
		}
	}
}
