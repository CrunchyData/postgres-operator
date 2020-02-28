package pgo_cli_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TC42 ✓
// TC115 ✓
var _ = describe("Cluster Commands", func(t *testing.T) {
	t.Parallel()

	withNamespace(t, func(namespace func() string) {
		withCluster(t, namespace, func(cluster func() string) {
			t.Run("label", func(t *testing.T) {
				t.Run("modifies the cluster", func(t *testing.T) {
					output, err := pgo("label", cluster(), "--label=villain=hordak", "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "applied")

					output, err = pgo("show", "cluster", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "villain=hordak")

					output, err = pgo("show", "cluster", "--selector=villain=hordak", "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, cluster())
				})
			})

			t.Run("delete label", func(t *testing.T) {
				t.Run("modifies the cluster", func(t *testing.T) {
					_, err := pgo("label", cluster(), "--label=etheria=yes", "-n", namespace()).Exec(t)
					require.NoError(t, err)

					output, err := pgo("delete", "label", cluster(), "--label=etheria=yes", "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "deleting")

					output, err = pgo("show", "cluster", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.NotContains(t, output, "etheria=yes")
				})
			})
		})
	})
})
