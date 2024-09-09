// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/crunchydata/postgres-operator/internal/logging"
)

// CreateDatabasesInPostgreSQL calls exec to create databases that do not exist
// in PostgreSQL.
func CreateDatabasesInPostgreSQL(
	ctx context.Context, exec Executor, databases []string,
) error {
	log := logging.FromContext(ctx)

	var err error
	var sql bytes.Buffer

	// Prevent unexpected dereferences by emptying "search_path". The "pg_catalog"
	// schema is still searched, and only temporary objects can be created.
	// - https://www.postgresql.org/docs/current/runtime-config-client.html#GUC-SEARCH-PATH
	_, _ = sql.WriteString(`SET search_path TO '';`)

	// Fill a temporary table with the JSON of the database specifications.
	// "\copy" reads from subsequent lines until the special line "\.".
	// - https://www.postgresql.org/docs/current/app-psql.html#APP-PSQL-META-COMMANDS-COPY
	_, _ = sql.WriteString(`
CREATE TEMPORARY TABLE input (id serial, data json);
\copy input (data) from stdin with (format text)
`)

	encoder := json.NewEncoder(&sql)
	encoder.SetEscapeHTML(false)

	for i := range databases {
		if err == nil {
			err = encoder.Encode(map[string]any{
				"database": databases[i],
			})
		}
	}
	_, _ = sql.WriteString(`\.` + "\n")

	// Create databases that do not already exist.
	// - https://www.postgresql.org/docs/current/sql-createdatabase.html
	_, _ = sql.WriteString(`
SELECT pg_catalog.format('CREATE DATABASE %I',
       pg_catalog.json_extract_path_text(input.data, 'database'))
  FROM input
 WHERE NOT EXISTS (
       SELECT 1 FROM pg_catalog.pg_database
       WHERE datname = pg_catalog.json_extract_path_text(input.data, 'database'))
 ORDER BY input.id
\gexec
`)

	stdout, stderr, err := exec.Exec(ctx, &sql,
		map[string]string{
			"ON_ERROR_STOP": "on", // Abort when any one statement fails.
			"QUIET":         "on", // Do not print successful statements to stdout.
		})

	log.V(1).Info("created PostgreSQL databases", "stdout", stdout, "stderr", stderr)

	return err
}
