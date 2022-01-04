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
