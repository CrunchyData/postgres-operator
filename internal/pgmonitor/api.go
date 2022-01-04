/*
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/crunchydata/postgres-operator/internal/logging"
)

type Executor func(
	ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
) error

// GetExporterSQL takes the PostgreSQL version and returns the corresponding
// setup.sql file that is defined in the exporter container
func (exec Executor) GetExporterSetupSQL(ctx context.Context, version int) (string, string, error) {
	log := logging.FromContext(ctx)

	var stdout, stderr bytes.Buffer
	var sql string
	err := exec(ctx, nil, &stdout, &stderr,
		[]string{"cat", fmt.Sprintf("/opt/cpm/conf/pg%d/setup.sql", version)}...)

	log.V(1).Info("sql received from exporter", "stdout", stdout.String(), "stderr", stderr.String())

	if err == nil {
		// TODO: Revisit how pgbackrest_info.sh is used with pgMonitor.
		// pgMonitor queries expect a path to a script that runs pgBackRest
		// info and provides json output. In the queries yaml for pgBackRest
		// the default path is `/usr/bin/pgbackrest-info.sh`. We update
		// the path to point to the script in our database image.
		sql = strings.ReplaceAll(stdout.String(),
			"/usr/bin/pgbackrest-info.sh",
			"/opt/crunchy/bin/postgres/pgbackrest_info.sh")
	}

	log.V(1).Info("updated pgMonitor default configration", "sql", sql)

	return sql, stderr.String(), err
}
