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

// TC115 ✓
// TC116 ✓
// TC119 ✓
func TestClusterDelete(t *testing.T) {
	t.Parallel()

	withNamespace(t, func(namespace func() string) {
		t.Run("delete cluster", func(t *testing.T) {
			t.Run("removes data and backups", func(t *testing.T) {
				t.Parallel()
				withCluster(t, namespace, func(cluster func() string) {
					requireClusterReady(t, namespace(), cluster(), time.Minute)
					require.NotEmpty(t, clusterPVCs(t, namespace(), cluster()), "expected data to exist")

					output, err := pgo("delete", "cluster", cluster(), "--no-prompt", "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "deleted")

					gone := func() bool {
						return len(clusterPVCs(t, namespace(), cluster())) == 0
					}
					requireWaitFor(t, gone, time.Minute, time.Second,
						"timeout waiting for data of %q in %q", cluster(), namespace())

					output, err = pgo("show", "cluster", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.NotContains(t, output, cluster())
				})
			})

			t.Run("can keep backups", func(t *testing.T) {
				t.Parallel()
				withCluster(t, namespace, func(cluster func() string) {
					requireClusterReady(t, namespace(), cluster(), time.Minute)
					require.NotEmpty(t, clusterPVCs(t, namespace(), cluster()), "expected data to exist")

					output, err := pgo("delete", "cluster", cluster(), "--keep-backups", "--no-prompt", "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "deleted")

					gone := func() bool {
						return len(clusterPVCs(t, namespace(), cluster())) == 1
					}
					requireWaitFor(t, gone, time.Minute, time.Second,
						"timeout waiting for data of %q in %q", cluster(), namespace())

					pvcs := clusterPVCs(t, namespace(), cluster())
					require.NotEmpty(t, pvcs)
					require.Contains(t, pvcs[0].Name, "pgbr-repo")

					output, err = pgo("show", "cluster", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.NotContains(t, output, cluster())
				})
			})

			t.Run("can keep data", func(t *testing.T) {
				t.Parallel()
				withCluster(t, namespace, func(cluster func() string) {
					requireClusterReady(t, namespace(), cluster(), time.Minute)
					require.NotEmpty(t, clusterPVCs(t, namespace(), cluster()), "expected data to exist")

					output, err := pgo("delete", "cluster", cluster(), "--keep-data", "--no-prompt", "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "deleted")

					gone := func() bool {
						return len(clusterPVCs(t, namespace(), cluster())) == 1
					}
					requireWaitFor(t, gone, time.Minute, time.Second,
						"timeout waiting for data of %q in %q", cluster(), namespace())

					pvcs := clusterPVCs(t, namespace(), cluster())
					require.NotEmpty(t, pvcs)
					require.Equal(t, cluster(), pvcs[0].Name)
				})
			})
		})
	})
}
