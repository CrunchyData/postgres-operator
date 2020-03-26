package pgo_cli_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClusterCat(t *testing.T) {
	t.Parallel()

	withNamespace(t, func(namespace func() string) {
		withCluster(t, namespace, func(cluster func() string) {
			t.Run("cat", func(t *testing.T) {
				t.Run("shows something", func(t *testing.T) {
					requireClusterReady(t, namespace(), cluster(), time.Minute)

					output, err := pgo("cat", cluster(), "-n", namespace(),
						"/pgdata/"+cluster()+"/postgresql.conf",
					).Exec(t)
					require.NoError(t, err)
					require.NotEmpty(t, output)
				})
			})
		})
	})
}
