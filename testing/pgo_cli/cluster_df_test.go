package pgo_cli_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TC44 ✓
var _ = describe("Cluster Commands", func(t *testing.T) {
	t.Parallel()

	withNamespace(t, func(namespace func() string) {
		withCluster(t, namespace, func(cluster func() string) {
			t.Run("df", func(t *testing.T) {
				t.Run("shows something", func(t *testing.T) {
					requireClusterReady(t, namespace(), cluster(), time.Minute)

					output, err := pgo("df", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.NotEmpty(t, output)
				})
			})
		})
	})
})
