// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgis

import (
	"context"
	"strings"

	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/postgres"
)

// EnableInPostgreSQL installs triggers for the following extensions into every database:
//   - postgis
//   - postgis_topology
//   - fuzzystrmatch
//   - postgis_tiger_geocoder
func EnableInPostgreSQL(ctx context.Context, exec postgres.Executor) error {
	log := logging.FromContext(ctx)

	stdout, stderr, err := exec.ExecInAllDatabases(ctx,
		strings.Join([]string{
			// Quiet NOTICE messages from IF NOT EXISTS statements.
			// - https://www.postgresql.org/docs/current/runtime-config-client.html
			`SET client_min_messages = WARNING;`,

			`CREATE EXTENSION IF NOT EXISTS postgis;`,
			`CREATE EXTENSION IF NOT EXISTS postgis_topology;`,
			`CREATE EXTENSION IF NOT EXISTS fuzzystrmatch;`,
			`CREATE EXTENSION IF NOT EXISTS postgis_tiger_geocoder;`,
		}, "\n"),
		map[string]string{
			"ON_ERROR_STOP": "on", // Abort when any one statement fails.
			"QUIET":         "on", // Do not print successful statements to stdout.
		})

	log.V(1).Info("enabled PostGIS and related extensions", "stdout", stdout, "stderr", stderr)

	return err
}
