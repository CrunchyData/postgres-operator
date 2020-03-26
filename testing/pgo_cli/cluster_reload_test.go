package pgo_cli_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClusterReload(t *testing.T) {
	t.Parallel()

	withNamespace(t, func(namespace func() string) {
		withCluster(t, namespace, func(cluster func() string) {
			t.Run("reload", func(t *testing.T) {
				t.Run("applies PostgreSQL configuration", func(t *testing.T) {
					requireClusterReady(t, namespace(), cluster(), time.Minute)

					_, stderr := clusterPSQL(t, namespace(), cluster(),
						`ALTER SYSTEM SET checkpoint_completion_target = 1`)
					require.Empty(t, stderr)

					stdout, stderr := clusterPSQL(t, namespace(), cluster(), `
						SELECT name, s.setting, fs.setting
						FROM pg_settings s JOIN pg_file_settings fs USING (name)
						WHERE name = 'checkpoint_completion_target'
						AND (fs.sourcefile, fs.sourceline, fs.setting)
						IS NOT DISTINCT FROM (s.sourcefile, s.sourceline, s.setting)
					`)
					require.Empty(t, stderr)
					require.Contains(t, stdout, "(0 rows)",
						"bug in test: expected ALTER SYSTEM to change settings")

					output, err := pgo("reload", cluster(), "--no-prompt", "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "reload")

					applied := func() bool {
						stdout, stderr := clusterPSQL(t, namespace(), cluster(), `
							SELECT name, s.setting, fs.setting
							FROM pg_settings s JOIN pg_file_settings fs USING (name)
							WHERE name = 'checkpoint_completion_target'
							AND (fs.sourcefile, fs.sourceline, fs.setting)
							IS DISTINCT FROM (s.sourcefile, s.sourceline, s.setting)
						`)
						require.Empty(t, stderr)
						return strings.Contains(stdout, "(0 rows)")
					}
					requireWaitFor(t, applied, 20*time.Second, time.Second,
						"expected settings to take effect")
				})
			})
		})
	})
}
