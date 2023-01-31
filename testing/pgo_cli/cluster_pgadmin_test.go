package pgo_cli_test

/*
 Copyright 2020 - 2023 Crunchy Data Solutions, Inc.
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

func TestClusterPgAdmin(t *testing.T) {
	t.Parallel()

	var pgadminOnce sync.Once
	requirePgAdmin := func(t *testing.T, namespace, cluster string) {
		pgadminOnce.Do(func() {
			output, err := pgo("create", "pgadmin", cluster, "-n", namespace).Exec(t)
			require.NoError(t, err)
			require.Contains(t, output, "addition scheduled")
		})
	}

	withNamespace(t, func(namespace func() string) {
		withCluster(t, namespace, func(cluster func() string) {
			t.Run("create pgadmin", func(t *testing.T) {
				t.Run("starts PgAdmin", func(t *testing.T) {
					requireClusterReady(t, namespace(), cluster(), time.Minute)
					requirePgAdmin(t, namespace(), cluster())

					// PgAdmin does not appear immediately.
					requirePgAdminReady(t, namespace(), cluster(), time.Minute)

					// Here we wait 5 seconds to ensure pgAdmin has deployed in a stable way.
					// Without this sleep, the test can pass because the pgAdmin container is
					// (briefly) stable.
					time.Sleep(time.Duration(5) * time.Second)

					output, err := pgo("show", "cluster", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "pgadmin")

					output, err = pgo("test", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "pgadmin", "expected PgAdmin to be discoverable")

					for _, line := range strings.Split(output, "\n") {
						if strings.Contains(line, "pgadmin") {
							require.Contains(t, line, "UP", "expected PgAdmin to be accessible")
						}
					}
				})
			})

			t.Run("delete pgadmin", func(t *testing.T) {
				t.Run("stops PgAdmin", func(t *testing.T) {
					requireClusterReady(t, namespace(), cluster(), time.Minute)
					requirePgAdmin(t, namespace(), cluster())
					requirePgAdminReady(t, namespace(), cluster(), time.Minute)

					output, err := pgo("delete", "pgadmin", cluster(), "--no-prompt", "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "delete scheduled")

					gone := func() bool {
						deployments, err := TestContext.Kubernetes.ListDeployments(namespace(), map[string]string{
							"pg-cluster":      cluster(),
							"crunchy-pgadmin": "true",
						})
						require.NoError(t, err)
						return len(deployments) == 0
					}
					requireWaitFor(t, gone, time.Minute, time.Second,
						"timeout waiting for PgAdmin of %q in %q", cluster(), namespace())

					output, err = pgo("show", "cluster", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)

					//require.NotContains(t, output, "pgadmin")
					for _, line := range strings.Split(output, "\n") {
						// The service and deployment should be gone. The only remaining
						// reference could be in the labels.
						if strings.Contains(line, "pgadmin") {
							require.Contains(t, line, "pgadmin=false")
						}
					}
				})
			})
		})
	})
}
