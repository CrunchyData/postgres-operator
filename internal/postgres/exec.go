// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"bytes"
	"context"
	"io"
	"sort"
	"strings"
)

// Executor provides methods for calling "psql".
type Executor func(
	ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
) error

// Exec uses "psql" to execute sql. The sql statement(s) are passed via stdin
// and may contain psql variables that are assigned from the variables map.
// - https://www.postgresql.org/docs/current/app-psql.html#APP-PSQL-VARIABLES
func (exec Executor) Exec(
	ctx context.Context, sql io.Reader, variables map[string]string,
) (string, string, error) {
	// Convert variables into `psql` arguments.
	args := make([]string, 0, len(variables))
	for k, v := range variables {
		args = append(args, "--set="+k+"="+v)
	}

	// The map iteration above is nondeterministic. Sort the arguments so that
	// calls to exec are deterministic.
	// - https://golang.org/ref/spec#For_range
	sort.Strings(args)

	// Execute `psql` without reading config files nor prompting for a password.
	var stdout, stderr bytes.Buffer
	err := exec(ctx, sql, &stdout, &stderr,
		append([]string{"psql", "-Xw", "--file=-"}, args...)...)
	return stdout.String(), stderr.String(), err
}

// ExecInAllDatabases uses "bash" and "psql" to execute sql in every database
// that allows connections, including templates. The sql command(s) may contain
// psql variables that are assigned from the variables map.
// - https://www.postgresql.org/docs/current/app-psql.html#APP-PSQL-VARIABLES
func (exec Executor) ExecInAllDatabases(
	ctx context.Context, sql string, variables map[string]string,
) (string, string, error) {
	const databases = "" +
		// Prevent unexpected dereferences by emptying "search_path".
		// The "pg_catalog" schema is still searched.
		// - https://www.postgresql.org/docs/current/runtime-config-client.html#GUC-SEARCH-PATH
		`SET search_path = '';` +

		// Return the names of databases that allow connections, including
		// "template1". Exclude "template0" to ensure it is never manipulated.
		// - https://www.postgresql.org/docs/current/managing-databases.html
		`SELECT datname FROM pg_catalog.pg_database` +
		` WHERE datallowconn AND datname NOT IN ('template0')`

	return exec.ExecInDatabasesFromQuery(ctx, databases, sql, variables)
}

// ExecInDatabasesFromQuery uses "bash" and "psql" to execute sql in every
// database returned by the databases query. The sql statement(s) may contain
// psql variables that are assigned from the variables map.
// - https://www.postgresql.org/docs/current/app-psql.html#APP-PSQL-VARIABLES
func (exec Executor) ExecInDatabasesFromQuery(
	ctx context.Context, databases, sql string, variables map[string]string,
) (string, string, error) {
	// Use a Bash loop to call `psql` multiple times. The query to run in every
	// database is passed via standard input while the database query is passed
	// as the first argument. Remaining arguments are passed through to `psql`.
	stdin := strings.NewReader(sql)
	args := []string{databases}
	for k, v := range variables {
		args = append(args, "--set="+k+"="+v)
	}

	// The map iteration above is nondeterministic. Sort the variable arguments
	// so that calls to exec are deterministic.
	// - https://golang.org/ref/spec#For_range
	sort.Strings(args[1:])

	const script = `
sql_target=$(< /dev/stdin)
sql_databases="$1"
shift 1

databases=$(psql "$@" -Xw -Aqt --file=- <<< "${sql_databases}")
while IFS= read -r database; do
	PGDATABASE="${database}" psql "$@" -Xw --file=- <<< "${sql_target}"
done <<< "${databases}"
`

	// Execute the script with some error handling enabled.
	var stdout, stderr bytes.Buffer
	err := exec(ctx, stdin, &stdout, &stderr,
		append([]string{"bash", "-ceu", "--", script, "-"}, args...)...)
	return stdout.String(), stderr.String(), err
}
