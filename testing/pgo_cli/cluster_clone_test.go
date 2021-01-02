package pgo_cli_test

/*
 Copyright 2020 - 2021 Crunchy Data Solutions, Inc.
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

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClusterClone(t *testing.T) {
	t.Parallel()

	withNamespace(t, func(namespace func() string) {
		withCluster(t, namespace, func(cluster func() string) {
			t.Run("clone", func(t *testing.T) {
				t.Run("creates a copy of a cluster", func(t *testing.T) {
					requireClusterReady(t, namespace(), cluster(), time.Minute)
					requireStanzaExists(t, namespace(), cluster(), 2*time.Minute)

					// data in the origin cluster followed by a WAL flush
					_, stderr := clusterPSQL(t, namespace(), cluster(), `
						CREATE TABLE original (data) AS VALUES ('one'), ('two');
						DO $$ BEGIN IF current_setting('server_version_num')::int > 100000
							THEN PERFORM pg_switch_wal();
							ELSE PERFORM pg_switch_xlog();
						END IF; END $$`)
					require.Empty(t, stderr)

					output, err := pgo("clone", cluster(), "rex", "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.NotEmpty(t, output)

					defer teardownCluster(t, namespace(), "rex", time.Now())
					requireClusterReady(t, namespace(), "rex", 4*time.Minute)

					stdout, stderr := clusterPSQL(t, namespace(), "rex", `TABLE original`)
					require.Empty(t, stderr)
					require.Contains(t, stdout, "(2 rows)",
						"expected original data to be present in the clone")
				})
			})
		})
	})
}
