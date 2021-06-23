/*
 Copyright 2021 Crunchy Data Solutions, Inc.
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
) {
	if outParameters.Mandatory == nil {
		outParameters.Mandatory = postgres.NewParameterSet()
	}

	// Send WAL files to all configured repositories when not in recovery.
	// - https://pgbackrest.org/user-guide.html#quickstart/configure-archiving
	// - https://pgbackrest.org/command.html#command-archive-push
	// - https://www.postgresql.org/docs/current/runtime-config-wal.html
	archive := `pgbackrest --stanza=` + DefaultStanzaName + ` archive-push "%p"`
	outParameters.Mandatory.Add("archive_mode", "on")
	outParameters.Mandatory.Add("archive_command", archive)

	// Fetch WAL files from any configured repository during recovery.
	// - https://pgbackrest.org/command.html#command-archive-get
	// - https://www.postgresql.org/docs/current/runtime-config-wal.html
	restore := `pgbackrest --stanza=` + DefaultStanzaName + ` archive-get %f "%p"`
	outParameters.Mandatory.Add("restore_command", restore)

	if inCluster.Spec.Standby != nil && inCluster.Spec.Standby.Enabled {

		// Fetch WAL files from the designated repository. The repository name
		// is validated by the Kubernetes API, so it does not need to be quoted
		// nor escaped.
		repoName := inCluster.Spec.Standby.RepoName
		restore += " --repo=" + strings.TrimPrefix(repoName, "repo")
		outParameters.Mandatory.Add("restore_command", restore)
	}
}
