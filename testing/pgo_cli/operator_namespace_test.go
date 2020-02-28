package pgo_cli_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var _ = describe("Operator Commands", func(t *testing.T) {
	t.Parallel()

	withNamespace(t, func(namespace func() string) {
		t.Run("show namespace", func(t *testing.T) {
			t.Run("shows something", func(t *testing.T) {
				output, err := pgo("show", "namespace", namespace()).Exec(t)
				require.NoError(t, err)
				require.Contains(t, output, namespace())
			})
		})
	})
})
