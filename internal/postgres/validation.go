// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"path"
	"regexp"
	"strings"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// SanitizeParameters transforms parameters so they are safe for Postgres in cluster.
func SanitizeParameters(cluster *v1beta1.PostgresCluster, parameters *ParameterSet) {
	if v, ok := parameters.Get("log_directory"); ok {
		parameters.Add("log_directory", sanitizeLogDirectory(cluster, v))
	}
}

// sensitiveAbsolutePath matches absolute paths that Postgres expects to control.
// User input should not direct tools to write to these directories.
//
// See [sanitizeLogDirectory].
var sensitiveAbsolutePath = regexp.MustCompile(
	// Rooted in one of these volumes
	`^(` + dataMountPath + `|` + tmpMountPath + `|` + walMountPath + `)` +

		// First subdirectory is a Postgres directory
		`/(` + `pg\d+` + // [DataDirectory]
		`|` + `pgsql_tmp` + // https://www.postgresql.org/docs/current/storage-file-layout.html
		`|` + `pg\d+_wal` + // [WALDirectory]
		`)(/|$)`,
)

// sensitiveRelativePath matches paths relative to the Postgres "data_directory" that Postgres expects to control.
// Arguably, everything inside the data directory is sensitve, but this is here because
// Postges interprets some of its parameters relative to its data directory.
//
// User input should not direct tools to write to these directories.
//
// NOTE: This is not an exhaustive list! New code should use an allowlist rather than this denylist.
//
// See [sanitizeLogDirectory].
var sensitiveRelativePath = regexp.MustCompile(
	`^(archive|base|current|global|patroni|pg_|PG_|postgresql|postmaster|[[:xdigit:]]{24,})` +
		`|` + `[.](history|partial)$`,
)

// sanitizeLogDirectory returns the absolute path to input when it is a safe "log_directory" for cluster.
// Otherwise, it returns the absolute path to a good "log_directory" value.
//
// https://www.postgresql.org/docs/current/runtime-config-logging.html#GUC-LOG-DIRECTORY
func sanitizeLogDirectory(cluster *v1beta1.PostgresCluster, input string) string {
	directory := path.Clean(input)

	// [path.Clean] leaves leading parent directories. Eliminate these as a security measure.
	for strings.HasPrefix(directory, "../") {
		directory = directory[3:]
	}

	switch {
	case directory == "log":
		// This the Postgres default and the only relative path allowed in v1 of PostgresCluster.
		// Expand it relative to the data directory like Postgres does.
		return path.Join(DataDirectory(cluster), "log")

	case directory == "", directory == ".", directory == "/",
		sensitiveAbsolutePath.MatchString(directory),
		sensitiveRelativePath.MatchString(directory):
		// When the value is empty after cleaning or disallowed, choose one instead.
		// Keep it on the same volume, if possible.
		if strings.HasPrefix(directory, tmpMountPath) {
			return path.Join(tmpMountPath, "logs/postgres")
		}
		if strings.HasPrefix(directory, walMountPath) {
			return path.Join(walMountPath, "logs/postgres")
		}

		// There is always a data volume, so use that.
		return path.Join(dataMountPath, "logs/postgres")

	case !path.IsAbs(directory):
		// Directory is relative. This is disallowed since v1 of PostgresCluster.
		// Expand it relative to the data directory like Postgres does.
		return path.Join(DataDirectory(cluster), directory)

	default:
		// Directory is absolute and considered safe; use it.
		return directory
	}
}
