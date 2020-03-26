package pgo_cli_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TC40 ✓
func TestOperatorCommands(t *testing.T) {
	t.Parallel()

	t.Run("version", func(t *testing.T) {
		t.Run("reports the API version", func(t *testing.T) {
			output, err := pgo("version").Exec(t)
			require.NoError(t, err)
			require.Contains(t, output, "pgo-apiserver version")
		})
	})

	withNamespace(t, func(namespace func() string) {
		t.Run("status", func(t *testing.T) {
			t.Run("shows something", func(t *testing.T) {
				output, err := pgo("status", "-n", namespace()).Exec(t)
				require.NoError(t, err)
				require.Contains(t, output, "Operator Start")
			})
		})

		t.Run("show config", func(t *testing.T) {
			t.Run("shows something", func(t *testing.T) {
				output, err := pgo("show", "config", "-n", namespace()).Exec(t)
				require.NoError(t, err)
				require.Contains(t, output, "PrimaryStorage")
			})
		})
	})
}
