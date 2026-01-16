// Copyright 2017 - 2026 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"path/filepath"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// GetInstanceLogPath is responsible for determining the appropriate log path for pgbackrest
// in instance pods. If the user has set a log path via the spec, use that. Otherwise, use
// the default log path set in the naming package. Ensure trailing slashes are trimmed.
//
// This function assumes that the backups/pgbackrest spec is present in postgresCluster.
func GetPGBackRestLogPathForInstance(postgresCluster *v1beta1.PostgresCluster) string {
	logPath := naming.PGBackRestPGDataLogPath
	if postgresCluster.Spec.Backups.PGBackRest.Log != nil &&
		postgresCluster.Spec.Backups.PGBackRest.Log.Path != "" {
		logPath = postgresCluster.Spec.Backups.PGBackRest.Log.Path
	}
	return filepath.Clean(logPath)
}
