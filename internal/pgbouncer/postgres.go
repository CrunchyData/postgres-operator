// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgbouncer

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/internal/postgres/password"
	"github.com/crunchydata/postgres-operator/internal/util"
)

const (
	postgresqlSchema = "pgbouncer"

	// NOTE(cbandy): The "pgbouncer" database is special in PgBouncer and seems
	// to also be related to the "auth_user".
	// - https://github.com/pgbouncer/pgbouncer/issues/568
	// - https://github.com/pgbouncer/pgbouncer/issues/302#issuecomment-815097248
	postgresqlUser = "_crunchypgbouncer"
)

// sqlAuthenticationQuery returns the SECURITY DEFINER function that allows
// PgBouncer to access non-privileged and non-system user credentials.
func sqlAuthenticationQuery(sqlFunctionName string) string {
	// Only a subset of authorization identifiers should be accessible to
	// PgBouncer.
	// - https://www.postgresql.org/docs/current/catalog-pg-authid.html
	sqlAuthorizationConditions := strings.Join([]string{
		// Only those with permission to login.
		`pg_authid.rolcanlogin`,
		// No superusers. This is important: allowing superusers would make the
		// PgBouncer user a de facto superuser.
		`NOT pg_authid.rolsuper`,
		// No replicators.
		`NOT pg_authid.rolreplication`,
		// Not the PgBouncer role itself.
		`pg_authid.rolname <> ` + util.SQLQuoteLiteral(postgresqlUser),
		// Those without a password expiration or an expiration in the future.
		`(pg_authid.rolvaliduntil IS NULL OR pg_authid.rolvaliduntil >= CURRENT_TIMESTAMP)`,
	}, "\n    AND ")

	return strings.TrimSpace(`
CREATE OR REPLACE FUNCTION ` + sqlFunctionName + `(username TEXT)
RETURNS TABLE(username TEXT, password TEXT) AS ` + util.SQLQuoteLiteral(`
  SELECT rolname::TEXT, rolpassword::TEXT
  FROM pg_catalog.pg_authid
  WHERE pg_authid.rolname = $1
    AND `+sqlAuthorizationConditions) + `
LANGUAGE SQL STABLE SECURITY DEFINER;`)
}

// DisableInPostgreSQL removes any objects created by EnableInPostgreSQL.
func DisableInPostgreSQL(ctx context.Context, exec postgres.Executor) error {
	log := logging.FromContext(ctx)

	// First, remove PgBouncer objects from all databases and database templates.
	// The PgBouncer user is removed later.
	stdout, stderr, err := exec.ExecInAllDatabases(ctx,
		strings.Join([]string{
			// Quiet NOTICE messages from IF EXISTS statements.
			// - https://www.postgresql.org/docs/current/runtime-config-client.html
			`SET client_min_messages = WARNING;`,

			// Drop the following objects in a transaction.
			`BEGIN;`,

			// Remove the "get_auth" function that returns user credentials to PgBouncer.
			`DROP FUNCTION IF EXISTS :"namespace".get_auth(username TEXT);`,

			// Drop the PgBouncer schema and anything within it.
			`DROP SCHEMA IF EXISTS :"namespace" CASCADE;`,

			// Ensure there's nothing else unexpectedly owned by the PgBouncer
			// user in this database. Any privileges on shared objects are also
			// removed.
			strings.TrimSpace(`
SELECT pg_catalog.format('DROP OWNED BY %I CASCADE', :'username')
 WHERE EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = :'username')
\gexec`),

			// Commit (finish) the transaction.
			`COMMIT;`,
		}, "\n"),
		map[string]string{
			"username":  postgresqlUser,
			"namespace": postgresqlSchema,

			"ON_ERROR_STOP": "on", // Abort when any one statement fails.
			"QUIET":         "on", // Do not print successful statements to stdout.
		})

	log.V(1).Info("removed PgBouncer objects", "stdout", stdout, "stderr", stderr)

	if err == nil {
		// Remove the PgBouncer user now that the objects and other privileges are gone.
		stdout, stderr, err = exec.ExecInDatabasesFromQuery(ctx,
			`SELECT pg_catalog.current_database()`,
			`SET client_min_messages = WARNING; DROP ROLE IF EXISTS :"username";`,
			map[string]string{
				"username": postgresqlUser,

				"ON_ERROR_STOP": "on", // Abort when any one statement fails.
				"QUIET":         "on", // Do not print successful statements to stdout.
			})

		log.V(1).Info("removed PgBouncer user", "stdout", stdout, "stderr", stderr)
	}

	return err
}

// EnableInPostgreSQL creates the PgBouncer user, schema, and SECURITY DEFINER
// function that allows it to authenticate clients using their password stored
// in PostgreSQL.
func EnableInPostgreSQL(
	ctx context.Context, exec postgres.Executor, clusterSecret *corev1.Secret,
) error {
	log := logging.FromContext(ctx)

	stdout, stderr, err := exec.ExecInAllDatabases(ctx,
		strings.Join([]string{
			// Quiet NOTICE messages from IF NOT EXISTS statements.
			// - https://www.postgresql.org/docs/current/runtime-config-client.html
			`SET client_min_messages = WARNING;`,

			// Create the following objects in a transaction so that permissions
			// are correct before any other session sees them.
			// - https://www.postgresql.org/docs/current/ddl-priv.html
			`BEGIN;`,

			// Create the PgBouncer user if it does not already exist.
			// Permissions are granted later.
			strings.TrimSpace(`
SELECT pg_catalog.format('CREATE ROLE %I NOLOGIN', :'username')
 WHERE NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = :'username')
\gexec`),

			// Ensure the user can only access the one schema. Revoke anything
			// that might have been granted on other schemas, like "public".
			strings.TrimSpace(`
SELECT pg_catalog.format('REVOKE ALL PRIVILEGES ON SCHEMA %I FROM %I', nspname, :'username')
  FROM pg_catalog.pg_namespace
 WHERE pg_catalog.has_schema_privilege(:'username', oid, 'CREATE, USAGE')
   AND nspname NOT IN ('pg_catalog', :'namespace')
\gexec`),

			// Create the one schema and lock it down. Only the one user is
			// allowed to use it.
			strings.TrimSpace(`
CREATE SCHEMA IF NOT EXISTS :"namespace";
REVOKE ALL PRIVILEGES
    ON SCHEMA :"namespace" FROM PUBLIC, :"username";
 GRANT USAGE
    ON SCHEMA :"namespace" TO :"username";`),

			// The "get_auth" function returns the appropriate credentials for
			// a user's password-based authentication and works with PgBouncer's
			// "auth_query" setting. Only the one user is allowed to execute it.
			// - https://www.pgbouncer.org/config.html#auth_query
			sqlAuthenticationQuery(`:"namespace".get_auth`),
			strings.TrimSpace(`
REVOKE ALL PRIVILEGES
    ON FUNCTION :"namespace".get_auth(username TEXT) FROM PUBLIC, :"username";
 GRANT EXECUTE
    ON FUNCTION :"namespace".get_auth(username TEXT) TO :"username";`),

			// Remove "public" from the PgBouncer user's search_path.
			// - https://www.postgresql.org/docs/current/perm-functions.html
			`ALTER ROLE :"username" SET search_path TO :'namespace';`,

			// Allow the PgBouncer user to to login.
			`ALTER ROLE :"username" LOGIN PASSWORD :'verifier';`,

			// Commit (finish) the transaction.
			`COMMIT;`,
		}, "\n"),
		map[string]string{
			"username":  postgresqlUser,
			"namespace": postgresqlSchema,
			"verifier":  string(clusterSecret.Data[verifierSecretKey]),

			"ON_ERROR_STOP": "on", // Abort when any one statement fails.
			"QUIET":         "on", // Do not print successful statements to stdout.
		})

	log.V(1).Info("applied PgBouncer objects", "stdout", stdout, "stderr", stderr)

	return err
}

func generatePassword() (plaintext, verifier string, err error) {
	// PgBouncer can login to PostgreSQL using either MD5 or SCRAM-SHA-256.
	// When using MD5, the (hashed) verifier can be stored in PgBouncer's
	// authentication file. When using SCRAM, the plaintext password must be
	// stored.
	// - https://www.pgbouncer.org/config.html#authentication-file-format
	// - https://github.com/pgbouncer/pgbouncer/issues/508#issuecomment-713339834

	plaintext, err = util.GenerateASCIIPassword(32)
	if err == nil {
		verifier, err = password.NewSCRAMPassword(plaintext).Build()
	}
	return
}

func postgresqlHBAs() []postgres.HostBasedAuthentication {
	// PgBouncer must connect over TLS using a SCRAM password. Other network
	// connections are forbidden.
	// - https://www.postgresql.org/docs/current/auth-pg-hba-conf.html
	// - https://www.postgresql.org/docs/current/auth-password.html

	return []postgres.HostBasedAuthentication{
		*postgres.NewHBA().User(postgresqlUser).TLS().Method("scram-sha-256"),
		*postgres.NewHBA().User(postgresqlUser).TCP().Method("reject"),
	}
}
