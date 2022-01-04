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

package postgis

import (
	"context"
	"strings"

	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/postgres"
)

// EnableInPostgreSQL installs triggers for the following extensions into every database:
//  - postgis
//  - postgis_topology
//  - fuzzystrmatch
//  - postgis_tiger_geocoder
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
