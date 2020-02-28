package pgo_cli_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TC47 ✓
// TC49 ✓
var _ = describe("Cluster Commands", func(t *testing.T) {
	t.Parallel()

	withNamespace(t, func(namespace func() string) {
		withCluster(t, namespace, func(cluster func() string) {
			t.Run("scale", func(t *testing.T) {
				t.Run("creates replica", func(t *testing.T) {
					requireClusterReady(t, namespace(), cluster(), time.Minute)

					output, err := pgo("scale", cluster(), "--no-prompt", "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.NotEmpty(t, output)

					requireReplicasReady(t, namespace(), cluster(), 2*time.Minute)
				})
			})
		})
	})
})
