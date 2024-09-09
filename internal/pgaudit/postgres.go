// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgaudit

import (
	"context"

	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/postgres"
)

// When the pgAudit shared library is not loaded, the extension cannot be
// installed. The "CREATE EXTENSION" command fails with an error, "pgaudit must
// be loaded…".
//
// When the pgAudit shared library is loaded but the extension is not installed,
// AUDIT messages are logged according to the various levels and settings
// (including both SESSION and OBJECT events) but the messages contain fewer
// details than normal. DDL messages, for example, lack the affected object name
// and type.
//
// When the pgAudit extension is installed but the shared library is not loaded,
//  1. No AUDIT messages are logged.
//  2. DDL commands fail with error "pgaudit must be loaded…".
//  3. DML commands and SELECT queries succeed and return results.
//  4. Databases can be created and dropped.
//  5. Roles and privileges can be created, dropped, granted, and revoked, but
//     the "DROP OWNED" command fails.

// EnableInPostgreSQL installs pgAudit triggers into every database.
func EnableInPostgreSQL(ctx context.Context, exec postgres.Executor) error {
	log := logging.FromContext(ctx)

	stdout, stderr, err := exec.ExecInAllDatabases(ctx,
		// Quiet the NOTICE from IF EXISTS, and install the pgAudit event triggers.
		// - https://www.postgresql.org/docs/current/runtime-config-client.html
		// - https://github.com/pgaudit/pgaudit#settings
		`SET client_min_messages = WARNING; CREATE EXTENSION IF NOT EXISTS pgaudit;`,
		map[string]string{
			"ON_ERROR_STOP": "on", // Abort when any one command fails.
			"QUIET":         "on", // Do not print successful commands to stdout.
		})

	log.V(1).Info("enabled pgAudit", "stdout", stdout, "stderr", stderr)

	return err
}

// PostgreSQLParameters sets the parameters required by pgAudit.
func PostgreSQLParameters(outParameters *postgres.Parameters) {

	// Load the shared library when PostgreSQL starts.
	// PostgreSQL must be restarted when changing this value.
	// - https://github.com/pgaudit/pgaudit#settings
	// - https://www.postgresql.org/docs/current/runtime-config-client.html
	outParameters.Mandatory.AppendToList("shared_preload_libraries", "pgaudit")
}
