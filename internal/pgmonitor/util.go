// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

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
