package pgo_cli_test

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
