// Copyright 2024 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func testInstrumentationSpec() *v1beta1.InstrumentationSpec {
	spec := v1beta1.InstrumentationSpec{
		Config: &v1beta1.InstrumentationConfigSpec{
			Exporters: map[string]any{
				"googlecloud": map[string]any{
					"log": map[string]any{
						"default_log_name": "opentelemetry.io/collector-exported-log",
					},
					"project": "google-project-name",
				},
			},
		},
		Logs: &v1beta1.InstrumentationLogsSpec{
			Exporters: []string{"googlecloud"},
		},
		Metrics: &v1beta1.InstrumentationMetricsSpec{
			Exporters: []string{"googlecloud"},
		},
	}

	return spec.DeepCopy()
}
