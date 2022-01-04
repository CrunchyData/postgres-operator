/*
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

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

		assert.Equal(t, outHBAs.Mandatory[0].String(), `host all "ccp_monitoring" "127.0.0.0/8" md5`)
		assert.Equal(t, outHBAs.Mandatory[1].String(), `host all "ccp_monitoring" "::1/128" md5`)
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
