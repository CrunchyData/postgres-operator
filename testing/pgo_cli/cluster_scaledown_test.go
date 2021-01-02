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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClusterScaleDown(t *testing.T) {
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
			t.Run("scaledown", func(t *testing.T) {
				t.Run("shows replicas", func(t *testing.T) {
					requireReplica(t, namespace(), cluster())

					pods := replicaPods(t, namespace(), cluster())
					require.NotEmpty(t, pods, "expected replicas to exist")

					output, err := pgo("scaledown", cluster(), "-n", namespace(),
						"--query",
					).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, pods[0].Labels["deployment-name"])
				})

				t.Run("removes one replica", func(t *testing.T) {
					requireReplica(t, namespace(), cluster())

					before := replicaPods(t, namespace(), cluster())
					require.NotEmpty(t, before, "expected replicas to exist")

					output, err := pgo("scaledown", cluster(), "-n", namespace(),
						"--target="+before[0].Labels["deployment-name"], "--no-prompt",
					).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "deleted")
					require.Contains(t, output, before[0].Labels["deployment-name"])

					gone := func() bool {
						after := replicaPods(t, namespace(), cluster())
						return len(before) != len(after)
					}
					requireWaitFor(t, gone, time.Minute, time.Second,
						"timeout waiting for replica of %q in %q", cluster(), namespace())
				})
			})
		})
	})
}
