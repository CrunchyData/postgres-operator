/*
 Copyright 2021 - 2023 Crunchy Data Solutions, Inc.
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

package pgmonitor

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

const (
	ExporterPort = int32(9187)

	// TODO: With the current implementation of the crunchy-postgres-exporter
	// it makes sense to hard-code the database. When moving away from the
	// crunchy-postgres-exporter start.sh script we should re-evaluate always
	// setting the exporter database to `postgres`.
	ExporterDB = "postgres"

	// The exporter connects to all databases over loopback using a password.
	// Kubernetes guarantees localhost resolves to loopback:
	// https://kubernetes.io/docs/concepts/cluster-administration/networking/
	// https://releases.k8s.io/v1.21.0/pkg/kubelet/kubelet_pods.go#L343
	ExporterHost = "localhost"
)

// postgres_exporter command flags
var (
	ExporterExtendQueryPathFlag  = "--extend.query-path=/tmp/queries.yml"
	ExporterWebListenAddressFlag = fmt.Sprintf("--web.listen-address=:%d", ExporterPort)
	ExporterWebConfigFileFlag    = "--web.config.file=/web-config/web-config.yml"
)

// Defaults for certain values used in queries.yml
// TODO(dsessler7): make these values configurable via spec
var DefaultValuesForQueries = map[string]string{
	"PGBACKREST_INFO_THROTTLE_MINUTES":    "10",
	"PG_STAT_STATEMENTS_LIMIT":            "20",
	"PG_STAT_STATEMENTS_THROTTLE_MINUTES": "-1",
}

// GenerateDefaultExporterQueries generates the default queries used by exporter
func GenerateDefaultExporterQueries(ctx context.Context, cluster *v1beta1.PostgresCluster) string {
	log := logging.FromContext(ctx)
	var queries string
	baseQueries := []string{"backrest", "global", "per_db", "nodemx"}
	queriesConfigDir := GetQueriesConfigDir(ctx)

	// TODO: When we add pgbouncer support we will do something like the following:
	// if pgbouncerEnabled() {
	// 	baseQueries = append(baseQueries, "pgbouncer")
	// }

	for _, queryType := range baseQueries {
		queriesContents, err := os.ReadFile(fmt.Sprintf("%s/queries_%s.yml", queriesConfigDir, queryType))
		if err != nil {
			// log an error, but continue to next iteration
			log.Error(err, fmt.Sprintf("Query file queries_%s.yml does not exist (it should)...", queryType))
			continue
		}
		queries += string(queriesContents) + "\n"
	}

	// Add general queries for specific postgres version
	queriesGeneral, err := os.ReadFile(fmt.Sprintf("%s/pg%d/queries_general.yml", queriesConfigDir, cluster.Spec.PostgresVersion))
	if err != nil {
		// log an error, but continue
		log.Error(err, fmt.Sprintf("Query file %s/pg%d/queries_general.yml does not exist (it should)...", queriesConfigDir, cluster.Spec.PostgresVersion))
	} else {
		queries += string(queriesGeneral) + "\n"
	}

	// Add pg_stat_statement queries for specific postgres version
	queriesPgStatStatements, err := os.ReadFile(fmt.Sprintf("%s/pg%d/queries_pg_stat_statements.yml", queriesConfigDir, cluster.Spec.PostgresVersion))
	if err != nil {
		// log an error, but continue
		log.Error(err, fmt.Sprintf("Query file %s/pg%d/queries_pg_stat_statements.yml not loaded.", queriesConfigDir, cluster.Spec.PostgresVersion))
	} else {
		queries += string(queriesPgStatStatements) + "\n"
	}

	// If postgres version >= 12, add pg_stat_statements_reset queries
	if cluster.Spec.PostgresVersion >= 12 {
		queriesPgStatStatementsReset, err := os.ReadFile(fmt.Sprintf("%s/pg%d/queries_pg_stat_statements_reset_info.yml", queriesConfigDir, cluster.Spec.PostgresVersion))
		if err != nil {
			// log an error, but continue
			log.Error(err, fmt.Sprintf("Query file %s/pg%d/queries_pg_stat_statements_reset_info.yml not loaded.", queriesConfigDir, cluster.Spec.PostgresVersion))
		} else {
			queries += string(queriesPgStatStatementsReset) + "\n"
		}
	}

	// Find and replace default values in queries
	for k, v := range DefaultValuesForQueries {
		queries = strings.ReplaceAll(queries, fmt.Sprintf("#%s#", k), v)
	}

	// TODO: Add ability to exclude certain user-specified queries

	return queries
}

// ExporterStartCommand generates an entrypoint that will create a master queries file and
// start the postgres_exporter. It will repeat those steps if it notices a change in
// the source queries files.
func ExporterStartCommand(commandFlags []string) []string {
	script := strings.Join([]string{
		// Set up temporary file to hold postgres_exporter process id
		`POSTGRES_EXPORTER_PIDFILE=/tmp/postgres_exporter.pid`,

		// declare function that will combine custom queries file and default
		// queries and start the postgres_exporter
		`start_postgres_exporter() {`,
		`	cat /conf/* > /tmp/queries.yml`,
		`	echo "Starting postgres_exporter with the following flags..."`,
		`	echo "$@"`,
		`	postgres_exporter "$@" &`,
		`	echo $! > $POSTGRES_EXPORTER_PIDFILE`,
		`}`,

		// run function to combine queries files and start postgres_exporter
		`start_postgres_exporter "$@"`,

		// set directory to watch
		`declare -r directory=/conf`,

		// Create a file descriptor with a no-op process that will not get
		// cleaned up
		`exec {fd}<> <(:)`,

		// Set up loop. Use read's timeout setting instead of sleep,
		// which uses up a lot of memory
		`while read -r -t 3 -u "${fd}" || true; do`,

		// If the directory's modify time is newer than our file descriptor's,
		// something must have changed, so kill the postgres_exporter and rerun
		// the function to combine queries files and start postgres_exporter
		`  if [ "${directory}" -nt "/proc/self/fd/${fd}" ] && echo "Something changed..." &&`,
		`    kill $(head -1 ${POSTGRES_EXPORTER_PIDFILE?}) && start_postgres_exporter "$@"`,

		// We then want to get rid of the old file descriptor, get a fresh one
		// and restart the loop
		`  then`,
		`    exec {fd}>&- && exec {fd}<> <(:)`,
		`    stat --format='Latest queries file dated %%y' "${directory}"`,
		`  fi`,
		`done`,
	}, "\n")

	return append([]string{"bash", "-ceu", "--", script, "postgres_exporter_watcher"}, commandFlags...)
}
