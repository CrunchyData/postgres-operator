// Copyright 2024 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"

	"github.com/crunchydata/postgres-operator/internal/feature"
)

func NewConfigForPostgresPod(ctx context.Context) *Config {
	config := NewConfig()

	if feature.Enabled(ctx, feature.OpenTelemetryMetrics) {
		// Add Prometheus exporter
		config.Exporters[Prometheus] = map[string]any{
			"endpoint": "0.0.0.0:8889",
		}

		// Add Prometheus Receiver
		config.Receivers[Prometheus] = map[string]any{
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
		config.Pipelines[Metrics] = Pipeline{
			Receivers: []ComponentID{Prometheus},
			Exporters: []ComponentID{Prometheus},
		}
	}

	return config
}
