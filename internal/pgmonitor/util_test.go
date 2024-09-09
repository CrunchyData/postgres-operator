// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgmonitor

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestExporterEnabled(t *testing.T) {
	cluster := &v1beta1.PostgresCluster{}
	assert.Assert(t, !ExporterEnabled(cluster))

	cluster.Spec.Monitoring = &v1beta1.MonitoringSpec{}
	assert.Assert(t, !ExporterEnabled(cluster))

	cluster.Spec.Monitoring.PGMonitor = &v1beta1.PGMonitorSpec{}
	assert.Assert(t, !ExporterEnabled(cluster))

	cluster.Spec.Monitoring.PGMonitor.Exporter = &v1beta1.ExporterSpec{}
	assert.Assert(t, ExporterEnabled(cluster))

}
