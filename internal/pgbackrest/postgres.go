// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgbackrest

import (
	"strings"

	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// PostgreSQL populates outParameters with any settings needed to run pgBackRest.
func PostgreSQL(
	inCluster *v1beta1.PostgresCluster,
	outParameters *postgres.Parameters,
	backupsEnabled bool,
) {
	if outParameters.Mandatory == nil {
		outParameters.Mandatory = postgres.NewParameterSet()
	}
	if outParameters.Default == nil {
		outParameters.Default = postgres.NewParameterSet()
	}

	// Send WAL files to all configured repositories when not in recovery.
	// - https://pgbackrest.org/user-guide.html#quickstart/configure-archiving
	// - https://pgbackrest.org/command.html#command-archive-push
	// - https://www.postgresql.org/docs/current/runtime-config-wal.html
	outParameters.Mandatory.Add("archive_mode", "on")
	if backupsEnabled {
		archive := `pgbackrest --stanza=` + DefaultStanzaName + ` archive-push "%p"`
		outParameters.Mandatory.Add("archive_command", archive)
	} else {
		// If backups are disabled, keep archive_mode on (to avoid a Postgres restart)
		// and throw away WAL.
		outParameters.Mandatory.Add("archive_command", `true`)
	}

	// archive_timeout is used to determine at what point a WAL file is switched,
	// if the WAL archive has not reached its full size in # of transactions
	// (16MB). This has ramifications for log shipping, i.e. it ensures a WAL file
	// is shipped to an archive every X seconds to help reduce the risk of data
	// loss in a disaster recovery scenario. For standby servers that are not
	// connected using streaming replication, this also ensures that new data is
	// available at least once a minute.
	//
	// PostgreSQL documentation considers an archive_timeout of 60 seconds to be
	// reasonable. There are cases where you may want to set archive_timeout to 0,
	// for example, when the remote archive (pgBackRest repo) is unavailable; this
	// is to prevent WAL accumulation on your primary.
	// - https://www.postgresql.org/docs/current/runtime-config-wal.html#GUC-ARCHIVE-TIMEOUT
	outParameters.Default.Add("archive_timeout", "60s")

	// Fetch WAL files from any configured repository during recovery.
	// - https://pgbackrest.org/command.html#command-archive-get
	// - https://www.postgresql.org/docs/current/runtime-config-wal.html
	restore := `pgbackrest --stanza=` + DefaultStanzaName + ` archive-get %f "%p"`
	outParameters.Mandatory.Add("restore_command", restore)

	if inCluster.Spec.Standby != nil && inCluster.Spec.Standby.Enabled && inCluster.Spec.Standby.RepoName != "" {

		// Fetch WAL files from the designated repository. The repository name
		// is validated by the Kubernetes API, so it does not need to be quoted
		// nor escaped.
		repoName := inCluster.Spec.Standby.RepoName
		restore += " --repo=" + strings.TrimPrefix(repoName, "repo")
		outParameters.Mandatory.Add("restore_command", restore)
	}
}
