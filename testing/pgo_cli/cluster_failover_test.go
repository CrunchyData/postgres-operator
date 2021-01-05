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

// TC48 ✓
// TC99 ✓
// TC100 ✓
// TC101 ✓
// TC102 ✓
// TC103 ✓
func TestClusterFailover(t *testing.T) {
	t.Parallel()

	var replicaOnce sync.Once
	requireReplica := func(t *testing.T, namespace, cluster string) {
		replicaOnce.Do(func() {
			_, err := pgo("scale", cluster, "--no-prompt", "-n", namespace).Exec(t)
			require.NoError(t, err)
			requireReplicasReady(t, namespace, cluster, 3*time.Minute)
		})
	}

	withNamespace(t, func(namespace func() string) {
		withCluster(t, namespace, func(cluster func() string) {
			t.Run("failover", func(t *testing.T) {
				t.Run("shows replicas", func(t *testing.T) {
					requireReplica(t, namespace(), cluster())

					pods := replicaPods(t, namespace(), cluster())
					require.NotEmpty(t, pods, "expected replicas to exist")

					output, err := pgo("failover", cluster(), "-n", namespace(),
						"--query",
					).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, pods[0].Labels["deployment-name"])
				})

				t.Run("swaps primary with replica", func(t *testing.T) {
					requireReplica(t, namespace(), cluster())

					before := replicaPods(t, namespace(), cluster())
					require.NotEmpty(t, before, "expected replicas to exist")

					output, err := pgo("failover", cluster(), "-n", namespace(),
						"--target="+before[0].Labels["deployment-name"], "--no-prompt",
					).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "success")

					replaced := func() bool {
						after := replicaPods(t, namespace(), cluster())
						return len(after) > 0 &&
							after[0].Labels["deployment-name"] != before[0].Labels["deployment-name"]
					}
					requireWaitFor(t, replaced, time.Minute, time.Second,
						"timeout waiting for failover of %q in %q", cluster(), namespace())

					requireReplicasReady(t, namespace(), cluster(), 5*time.Second)

					{
						var stdout, stderr string
						streaming := func() bool {
							primaries := primaryPods(t, namespace(), cluster())
							require.Len(t, primaries, 1)

							stdout, stderr, err = TestContext.Kubernetes.PodExec(
								primaries[0].Namespace, primaries[0].Name,
								strings.NewReader(`SELECT to_json(pg_stat_replication) FROM pg_stat_replication`),
								"psql", "-U", "postgres", "-f-")
							require.NoError(t, err)
							require.Empty(t, stderr)

							return strings.Contains(stdout, `"state":"streaming"`)
						}
						if !waitFor(t, streaming, 10*time.Second, time.Second) {
							require.Contains(t, stdout, `"state":"streaming"`)
						}
					}
				})
			})
		})
	})
}
