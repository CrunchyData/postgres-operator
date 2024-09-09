// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v5"

	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

var RESERVED_SCHEMA_NAMES = map[string]bool{
	"public":    true, // This is here for documentation; Postgres will reject a role named `public` as reserved
	"pgbouncer": true,
	"monitor":   true,
}

func sanitizeAlterRoleOptions(options string) string {
	const AlterRolePrefix = `ALTER ROLE "any" WITH `

	// Parse the options and discard them completely when incoherent.
	parsed, err := pg_query.Parse(AlterRolePrefix + options)
	if err != nil || len(parsed.GetStmts()) != 1 {
		return ""
	}

	// Rebuild the options list without invalid options. TODO(go1.21) TODO(slices)
	orig := parsed.GetStmts()[0].GetStmt().GetAlterRoleStmt().GetOptions()
	next := make([]*pg_query.Node, 0, len(orig))
	for i, option := range orig {
		if strings.EqualFold(option.GetDefElem().GetDefname(), "password") {
			continue
		}
		next = append(next, orig[i])
	}
	if len(next) > 0 {
		parsed.GetStmts()[0].GetStmt().GetAlterRoleStmt().Options = next
	} else {
		return ""
	}

	// Turn the modified statement back into SQL and remove the ALTER ROLE portion.
	sql, _ := pg_query.Deparse(parsed)
	return strings.TrimPrefix(sql, AlterRolePrefix)
}

// WriteUsersInPostgreSQL calls exec to create users that do not exist in
// PostgreSQL. Once they exist, it updates their options and passwords and
// grants them access to their specified databases. The databases must already
// exist.
func WriteUsersInPostgreSQL(
	ctx context.Context, cluster *v1beta1.PostgresCluster, exec Executor,
	users []v1beta1.PostgresUserSpec, verifiers map[string]string,
) error {
	log := logging.FromContext(ctx)

	var err error
	var sql bytes.Buffer

	// Prevent unexpected dereferences by emptying "search_path". The "pg_catalog"
	// schema is still searched, and only temporary objects can be created.
	// - https://www.postgresql.org/docs/current/runtime-config-client.html#GUC-SEARCH-PATH
	_, _ = sql.WriteString(`SET search_path TO '';`)

	// Fill a temporary table with the JSON of the user specifications.
	// "\copy" reads from subsequent lines until the special line "\.".
	// - https://www.postgresql.org/docs/current/app-psql.html#APP-PSQL-META-COMMANDS-COPY
	_, _ = sql.WriteString(`
CREATE TEMPORARY TABLE input (id serial, data json);
\copy input (data) from stdin with (format text)
`)
	encoder := json.NewEncoder(&sql)
	encoder.SetEscapeHTML(false)

	for i := range users {
		spec := users[i]

		databases := spec.Databases
		options := sanitizeAlterRoleOptions(spec.Options)

		// The "postgres" user must always be a superuser that can login to
		// the "postgres" database.
		if spec.Name == "postgres" {
			databases = append(databases[:0:0], "postgres")
			options = `LOGIN SUPERUSER`
		}

		if err == nil {
			err = encoder.Encode(map[string]any{
				"databases": databases,
				"options":   options,
				"username":  spec.Name,
				"verifier":  verifiers[string(spec.Name)],
			})
		}
	}
	_, _ = sql.WriteString(`\.` + "\n")

	// Create the following objects in a transaction so that permissions are
	// correct before any other session sees them.
	// - https://www.postgresql.org/docs/current/ddl-priv.html
	_, _ = sql.WriteString(`BEGIN;`)

	// Create users that do not already exist. Permissions are granted later.
	// Roles created this way automatically have the LOGIN option.
	// - https://www.postgresql.org/docs/current/sql-createuser.html
	_, _ = sql.WriteString(`
SELECT pg_catalog.format('CREATE USER %I',
       pg_catalog.json_extract_path_text(input.data, 'username'))
  FROM input
 WHERE NOT EXISTS (
       SELECT 1 FROM pg_catalog.pg_roles
       WHERE rolname = pg_catalog.json_extract_path_text(input.data, 'username'))
 ORDER BY input.id
\gexec
`)

	// Set any options from the specification. Validation ensures that the value
	// does not contain semicolons.
	// - https://www.postgresql.org/docs/current/sql-alterrole.html
	_, _ = sql.WriteString(`
SELECT pg_catalog.format('ALTER ROLE %I WITH %s PASSWORD %L',
       pg_catalog.json_extract_path_text(input.data, 'username'),
       pg_catalog.json_extract_path_text(input.data, 'options'),
       pg_catalog.json_extract_path_text(input.data, 'verifier'))
  FROM input ORDER BY input.id
\gexec
`)

	// Grant access to any specified databases.
	// - https://www.postgresql.org/docs/current/sql-grant.html
	_, _ = sql.WriteString(`
SELECT pg_catalog.format('GRANT ALL PRIVILEGES ON DATABASE %I TO %I',
       pg_catalog.json_array_elements_text(
       pg_catalog.json_extract_path(
       pg_catalog.json_strip_nulls(input.data), 'databases')),
       pg_catalog.json_extract_path_text(input.data, 'username'))
  FROM input ORDER BY input.id
\gexec
`)

	// Commit (finish) the transaction.
	_, _ = sql.WriteString(`COMMIT;`)

	stdout, stderr, err := exec.Exec(ctx, &sql,
		map[string]string{
			"ON_ERROR_STOP": "on", // Abort when any one statement fails.
			"QUIET":         "on", // Do not print successful statements to stdout.
		})

	log.V(1).Info("wrote PostgreSQL users", "stdout", stdout, "stderr", stderr)

	// The operator will attempt to write schemas for the users in the spec if
	// 	* the feature gate is enabled and
	// 	* the cluster is annotated.
	if feature.Enabled(ctx, feature.AutoCreateUserSchema) {
		autoCreateUserSchemaAnnotationValue, annotationExists := cluster.Annotations[naming.AutoCreateUserSchemaAnnotation]
		if annotationExists && strings.EqualFold(autoCreateUserSchemaAnnotationValue, "true") {
			log.V(1).Info("Writing schemas for users.")
			err = WriteUsersSchemasInPostgreSQL(ctx, exec, users)
		}
	}

	return err
}

// WriteUsersSchemasInPostgreSQL will create a schema for each user in each database that user has access to
func WriteUsersSchemasInPostgreSQL(ctx context.Context, exec Executor,
	users []v1beta1.PostgresUserSpec) error {

	log := logging.FromContext(ctx)

	var err error
	var stdout string
	var stderr string

	for i := range users {
		spec := users[i]

		// We skip if the user has the name of a reserved schema
		if RESERVED_SCHEMA_NAMES[string(spec.Name)] {
			log.V(1).Info("Skipping schema creation for user with reserved name",
				"name", string(spec.Name))
			continue
		}

		// We skip if the user has no databases
		if len(spec.Databases) == 0 {
			continue
		}

		var sql bytes.Buffer

		// Prevent unexpected dereferences by emptying "search_path". The "pg_catalog"
		// schema is still searched, and only temporary objects can be created.
		// - https://www.postgresql.org/docs/current/runtime-config-client.html#GUC-SEARCH-PATH
		_, _ = sql.WriteString(`SET search_path TO '';`)

		_, _ = sql.WriteString(`SELECT * FROM json_array_elements_text(:'databases');`)

		databases, _ := json.Marshal(spec.Databases)

		stdout, stderr, err = exec.ExecInDatabasesFromQuery(ctx,
			sql.String(),
			strings.Join([]string{
				// Quiet NOTICE messages from IF EXISTS statements.
				// - https://www.postgresql.org/docs/current/runtime-config-client.html
				`SET client_min_messages = WARNING;`,

				// Creates a schema named after and owned by the user
				// - https://www.postgresql.org/docs/current/ddl-schemas.html
				// - https://www.postgresql.org/docs/current/sql-createschema.html

				// We create a schema named after the user because
				// the PG search_path does not need to be updated,
				// since search_path defaults to "$user", public.
				// - https://www.postgresql.org/docs/current/ddl-schemas.html#DDL-SCHEMAS-PATH
				`CREATE SCHEMA IF NOT EXISTS :"username" AUTHORIZATION :"username";`,
			}, "\n"),
			map[string]string{
				"databases": string(databases),
				"username":  string(spec.Name),

				"ON_ERROR_STOP": "on", // Abort when any one statement fails.
				"QUIET":         "on", // Do not print successful commands to stdout.
			},
		)

		log.V(1).Info("wrote PostgreSQL schemas", "stdout", stdout, "stderr", stderr)
	}
	return err
}
