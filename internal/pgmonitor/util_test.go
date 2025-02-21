// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgmonitor

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestExporterEnabled(t *testing.T) {
	cluster := &v1beta1.PostgresCluster{}
	ctx := context.Background()
	assert.Assert(t, !ExporterEnabled(ctx, cluster))

	cluster.Spec.Monitoring = &v1beta1.MonitoringSpec{}
	assert.Assert(t, !ExporterEnabled(ctx, cluster))

	cluster.Spec.Monitoring.PGMonitor = &v1beta1.PGMonitorSpec{}
	assert.Assert(t, !ExporterEnabled(ctx, cluster))

	cluster.Spec.Monitoring.PGMonitor.Exporter = &v1beta1.ExporterSpec{}
	assert.Assert(t, ExporterEnabled(ctx, cluster))

	gate := feature.NewGate()
	assert.NilError(t, gate.SetFromMap(map[string]bool{
		feature.OpenTelemetryMetrics: true,
	}))
	ctx = feature.NewContext(ctx, gate)
	cluster.Spec.Monitoring.PGMonitor.Exporter = &v1beta1.ExporterSpec{}
	assert.Assert(t, !ExporterEnabled(ctx, cluster))
}
