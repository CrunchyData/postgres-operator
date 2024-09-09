// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgmonitor

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestPostgreSQLHBA(t *testing.T) {
	t.Run("ExporterDisabled", func(t *testing.T) {
		inCluster := &v1beta1.PostgresCluster{}
		outHBAs := postgres.HBAs{}
		PostgreSQLHBAs(inCluster, &outHBAs)
		assert.Equal(t, len(outHBAs.Mandatory), 0)
	})

	t.Run("ExporterEnabled", func(t *testing.T) {
		inCluster := &v1beta1.PostgresCluster{}
		inCluster.Spec.Monitoring = &v1beta1.MonitoringSpec{
			PGMonitor: &v1beta1.PGMonitorSpec{
				Exporter: &v1beta1.ExporterSpec{
					Image: "image",
				},
			},
		}

		outHBAs := postgres.HBAs{}
		PostgreSQLHBAs(inCluster, &outHBAs)

		assert.Equal(t, len(outHBAs.Mandatory), 3)
		assert.Equal(t, outHBAs.Mandatory[0].String(), `host all "ccp_monitoring" "127.0.0.0/8" scram-sha-256`)
		assert.Equal(t, outHBAs.Mandatory[1].String(), `host all "ccp_monitoring" "::1/128" scram-sha-256`)
		assert.Equal(t, outHBAs.Mandatory[2].String(), `host all "ccp_monitoring" all reject`)
	})
}

func TestPostgreSQLParameters(t *testing.T) {
	t.Run("ExporterDisabled", func(t *testing.T) {
		inCluster := &v1beta1.PostgresCluster{}
		outParameters := postgres.NewParameters()
		PostgreSQLParameters(inCluster, &outParameters)
		assert.Assert(t, !outParameters.Mandatory.Has("shared_preload_libraries"))
	})

	t.Run("ExporterEnabled", func(t *testing.T) {
		inCluster := &v1beta1.PostgresCluster{}
		inCluster.Spec.Monitoring = &v1beta1.MonitoringSpec{
			PGMonitor: &v1beta1.PGMonitorSpec{
				Exporter: &v1beta1.ExporterSpec{
					Image: "image",
				},
			},
		}
		outParameters := postgres.NewParameters()

		PostgreSQLParameters(inCluster, &outParameters)
		libs, found := outParameters.Mandatory.Get("shared_preload_libraries")
		assert.Assert(t, found)
		assert.Assert(t, strings.Contains(libs, "pg_stat_statements"))
		assert.Assert(t, strings.Contains(libs, "pgnodemx"))
	})

	t.Run("SharedPreloadLibraries Defined", func(t *testing.T) {
		inCluster := &v1beta1.PostgresCluster{}
		inCluster.Spec.Monitoring = &v1beta1.MonitoringSpec{
			PGMonitor: &v1beta1.PGMonitorSpec{
				Exporter: &v1beta1.ExporterSpec{
					Image: "image",
				},
			},
		}
		outParameters := postgres.NewParameters()
		outParameters.Mandatory.Add("shared_preload_libraries", "daisy")

		PostgreSQLParameters(inCluster, &outParameters)
		libs, found := outParameters.Mandatory.Get("shared_preload_libraries")
		assert.Assert(t, found)
		assert.Assert(t, strings.Contains(libs, "pg_stat_statements"))
		assert.Assert(t, strings.Contains(libs, "pgnodemx"))
		assert.Assert(t, strings.Contains(libs, "daisy"))
	})
}
