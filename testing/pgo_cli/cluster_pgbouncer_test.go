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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TC51 ✓
func TestClusterPgBouncer(t *testing.T) {
	t.Parallel()

	var pgbouncerOnce sync.Once
	requirePgBouncer := func(t *testing.T, namespace, cluster string) {
		pgbouncerOnce.Do(func() {
			output, err := pgo("create", "pgbouncer", cluster, "-n", namespace).Exec(t)
			require.NoError(t, err)
			require.Contains(t, output, "added")
		})
	}

	withNamespace(t, func(namespace func() string) {
		withCluster(t, namespace, func(cluster func() string) {
			t.Run("create pgbouncer", func(t *testing.T) {
				t.Run("starts PgBouncer", func(t *testing.T) {
					requireClusterReady(t, namespace(), cluster(), time.Minute)
					requirePgBouncer(t, namespace(), cluster())

					// PgBouncer does not appear immediately.
					requirePgBouncerReady(t, namespace(), cluster(), time.Minute)

					output, err := pgo("show", "cluster", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "pgbouncer")

					output, err = pgo("test", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "pgbouncer", "expected PgBouncer to be discoverable")

					for _, line := range strings.Split(output, "\n") {
						if strings.Contains(line, "pgbouncer") {
							require.Contains(t, line, "UP", "expected PgBouncer to be accessible")
						}
					}
				})
			})

			t.Run("delete pgbouncer", func(t *testing.T) {
				t.Run("stops PgBouncer", func(t *testing.T) {
					requireClusterReady(t, namespace(), cluster(), time.Minute)
					requirePgBouncer(t, namespace(), cluster())
					requirePgBouncerReady(t, namespace(), cluster(), time.Minute)

					output, err := pgo("delete", "pgbouncer", cluster(), "--no-prompt", "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "deleted")

					gone := func() bool {
						deployments, err := TestContext.Kubernetes.ListDeployments(namespace(), map[string]string{
							"pg-cluster":        cluster(),
							"crunchy-pgbouncer": "true",
						})
						require.NoError(t, err)
						return len(deployments) == 0
					}
					requireWaitFor(t, gone, time.Minute, time.Second,
						"timeout waiting for PgBouncer of %q in %q", cluster(), namespace())

					output, err = pgo("show", "cluster", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)

					//require.NotContains(t, output, "pgbouncer")
					for _, line := range strings.Split(output, "\n") {
						// The service and deployment should be gone. The only remaining
						// reference could be in the labels.
						if strings.Contains(line, "pgbouncer") {
							require.Contains(t, line, "pgbouncer=false")
						}
					}
				})
			})
		})
	})
}
