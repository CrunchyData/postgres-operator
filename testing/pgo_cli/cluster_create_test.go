package pgo_cli_test

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TC41 ✓
var _ = describe("Cluster Commands", func(t *testing.T) {
	t.Parallel()

	withNamespace(t, func(namespace func() string) {
		t.Run("create cluster", func(t *testing.T) {
			t.Run("creates a workflow", func(t *testing.T) {
				output, err := pgo("create", "cluster", "mycluster", "-n", namespace()).Exec(t)
				defer teardownCluster(t, namespace(), "mycluster", time.Now())
				require.NoError(t, err)
				require.Contains(t, output, "workflow id")

				workflow := regexp.MustCompile(`\S+$`).FindString(strings.TrimSpace(output))
				require.NotEmpty(t, workflow)

				_, err = pgo("show", "workflow", workflow, "-n", namespace()).Exec(t)
				require.NoError(t, err)
			})
		})

		withCluster(t, namespace, func(cluster func() string) {
			t.Run("show cluster", func(t *testing.T) {
				t.Run("shows something", func(t *testing.T) {
					output, err := pgo("show", "cluster", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.NotEmpty(t, output)
				})
			})

			t.Run("show user", func(t *testing.T) {
				t.Run("shows something", func(t *testing.T) {
					output, err := pgo("show", "user", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.NotEmpty(t, output)
				})
			})
		})
	})
})
