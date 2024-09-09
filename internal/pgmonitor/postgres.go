// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgmonitor

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

const (
	// MonitoringUser is a Postgres user created by pgMonitor configuration
	MonitoringUser = "ccp_monitoring"
)

// PostgreSQLHBAs provides the Postgres HBA rules for allowing the monitoring
// exporter to be accessible
func PostgreSQLHBAs(inCluster *v1beta1.PostgresCluster, outHBAs *postgres.HBAs) {
	if ExporterEnabled(inCluster) {
		// Limit the monitoring user to local connections using SCRAM.
		outHBAs.Mandatory = append(outHBAs.Mandatory,
			*postgres.NewHBA().TCP().User(MonitoringUser).Method("scram-sha-256").Network("127.0.0.0/8"),
			*postgres.NewHBA().TCP().User(MonitoringUser).Method("scram-sha-256").Network("::1/128"),
			*postgres.NewHBA().TCP().User(MonitoringUser).Method("reject"))
	}
}

// PostgreSQLParameters provides additional required configuration parameters
// that Postgres needs to support monitoring
func PostgreSQLParameters(inCluster *v1beta1.PostgresCluster, outParameters *postgres.Parameters) {
	if ExporterEnabled(inCluster) {
		// Exporter expects that shared_preload_libraries are installed
		// pg_stat_statements: https://access.crunchydata.com/documentation/pgmonitor/latest/exporter/
		// pgnodemx: https://github.com/CrunchyData/pgnodemx
		outParameters.Mandatory.AppendToList("shared_preload_libraries", "pg_stat_statements", "pgnodemx")
		outParameters.Mandatory.Add("pgnodemx.kdapi_path",
			postgres.DownwardAPIVolumeMount().MountPath)
	}
}

// DisableExporterInPostgreSQL disables the exporter configuration in PostgreSQL.
// Currently the exporter is disabled by removing login permissions for the
// monitoring user.
// TODO: evaluate other uninstall/removal options
func DisableExporterInPostgreSQL(ctx context.Context, exec postgres.Executor) error {
	log := logging.FromContext(ctx)

	stdout, stderr, err := exec.Exec(ctx, strings.NewReader(`
		SELECT pg_catalog.format('ALTER ROLE %I NOLOGIN', :'username')
		 WHERE EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = :'username')
		\gexec`),
		map[string]string{
			"username": MonitoringUser,
		})

	log.V(1).Info("monitoring user disabled", "stdout", stdout, "stderr", stderr)

	return err
}

// EnableExporterInPostgreSQL runs SQL setup commands in `database` to enable
// the exporter to retrieve metrics. pgMonitor objects are created and expected
// extensions are installed. We also ensure that the monitoring user has the
// current password and can login.
func EnableExporterInPostgreSQL(ctx context.Context, exec postgres.Executor,
	monitoringSecret *corev1.Secret, database, setup string) error {
	log := logging.FromContext(ctx)

	stdout, stderr, err := exec.ExecInAllDatabases(ctx,
		strings.Join([]string{
			// Quiet NOTICE messages from IF EXISTS statements.
			// - https://www.postgresql.org/docs/current/runtime-config-client.html
			`SET client_min_messages = WARNING;`,

			// Exporter expects that extension(s) to be installed in all databases
			// pg_stat_statements: https://access.crunchydata.com/documentation/pgmonitor/latest/exporter/
			"CREATE EXTENSION IF NOT EXISTS pg_stat_statements;",

			// Run idempotent update
			"ALTER EXTENSION pg_stat_statements UPDATE;",
		}, "\n"),
		map[string]string{
			"ON_ERROR_STOP": "on", // Abort when any one statement fails.
			"QUIET":         "on", // Do not print successful commands to stdout.
		},
	)

	log.V(1).Info("applied pgMonitor objects", "database", "current and future databases", "stdout", stdout, "stderr", stderr)

	// NOTE: Setup is run last to ensure that the setup sql is used in the hash
	if err == nil {
		stdout, stderr, err = exec.ExecInDatabasesFromQuery(ctx,
			`SELECT :'database'`,
			strings.Join([]string{
				// Quiet NOTICE messages from IF EXISTS statements.
				// - https://www.postgresql.org/docs/current/runtime-config-client.html
				`SET client_min_messages = WARNING;`,

				// Setup.sql file from the exporter image. sql is specific
				// to the PostgreSQL version
				setup,

				// pgnodemx: https://github.com/CrunchyData/pgnodemx
				// The `monitor` schema is hard-coded in the setup SQL files
				// from pgMonitor configuration
				// https://github.com/CrunchyData/pgmonitor/blob/master/postgres_exporter/common/queries_nodemx.yml
				"CREATE EXTENSION IF NOT EXISTS pgnodemx WITH SCHEMA monitor;",

				// Run idempotent update
				"ALTER EXTENSION pgnodemx UPDATE;",

				// ccp_monitoring user is created in Setup.sql without a
				// password; update the password and ensure that the ROLE
				// can login to the database
				`ALTER ROLE :"username" LOGIN PASSWORD :'verifier';`,
			}, "\n"),
			map[string]string{
				"database": database,
				"username": MonitoringUser,
				"verifier": string(monitoringSecret.Data["verifier"]),

				"ON_ERROR_STOP": "on", // Abort when any one statement fails.
				"QUIET":         "on", // Do not print successful commands to stdout.
			},
		)

		log.V(1).Info("applied pgMonitor objects", "database", database, "stdout", stdout, "stderr", stderr)
	}

	return err
}
