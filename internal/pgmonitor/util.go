/*
 Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
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
	"context"
	"os"

	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func GetQueriesConfigDir(ctx context.Context) string {
	log := logging.FromContext(ctx)
	// The QUERIES_CONFIG_DIR environment variable can be used to tell postgres-operator where to
	// find the setup.sql and queries.yml files when running the postgres-operator binary locally
	if queriesConfigDir := os.Getenv("QUERIES_CONFIG_DIR"); queriesConfigDir != "" {
		log.Info("Directory for setup.sql and queries files set by QUERIES_CONFIG_DIR env var. " +
			"This should only be used when running the postgres-operator binary locally.")
		return queriesConfigDir
	}

	return "/opt/crunchy/conf"
}

// ExporterEnabled returns true if the monitoring exporter is enabled
func ExporterEnabled(cluster *v1beta1.PostgresCluster) bool {
	if cluster.Spec.Monitoring == nil {
		return false
	}
	if cluster.Spec.Monitoring.PGMonitor == nil {
		return false
	}
	if cluster.Spec.Monitoring.PGMonitor.Exporter == nil {
		return false
	}
	return true
}
