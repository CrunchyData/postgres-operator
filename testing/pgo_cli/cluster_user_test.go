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
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TC127 ✓
func TestClusterUser(t *testing.T) {
	t.Parallel()

	clusterDatabase := func(t *testing.T, namespace, cluster string) string {
		t.Helper()

		names := clusterDatabases(t, namespace, cluster)
		if len(names) > 0 && names[0] == "postgres" {
			names = names[:1]
		}
		require.NotEmpty(t, names, "expected database to exist")
		return names[0]
	}

	showPassword := func(t *testing.T, namespace, cluster, user string) string {
		t.Helper()

		output, err := pgo("show", "user", cluster, "-n", namespace, "--output=json").Exec(t)
		require.NoError(t, err)

		var response struct{ Results []map[string]interface{} }
		require.NoError(t, json.Unmarshal([]byte(output), &response))

		for _, result := range response.Results {
			if result["Username"].(string) == user {
				return result["Password"].(string)
			}
		}
		return ""
	}

	withNamespace(t, func(namespace func() string) {
		withCluster(t, namespace, func(cluster func() string) {
			t.Run("show user", func(t *testing.T) {
				t.Run("shows something", func(t *testing.T) {
					requireClusterReady(t, namespace(), cluster(), time.Minute)

					output, err := pgo("show", "user", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, cluster())
				})
			})

			t.Run("create user", func(t *testing.T) {
				t.Run("accepts password", func(t *testing.T) {
					t.Parallel()
					requireClusterReady(t, namespace(), cluster(), time.Minute)
					db := clusterDatabase(t, namespace(), cluster())

					output, err := pgo("create", "user", cluster(), "-n", namespace(),
						"--username=gandalf", "--password=wizard", "--managed",
					).Exec(t)
					require.NoError(t, err)
					require.NotEmpty(t, output)

					// Connect using the above credentials
					pool := clusterConnection(t, namespace(), cluster(),
						"user=gandalf password=wizard database="+db)
					pool.Close()
				})

				t.Run("accepts selector", func(t *testing.T) {
					t.Parallel()
					requireClusterReady(t, namespace(), cluster(), time.Minute)
					db := clusterDatabase(t, namespace(), cluster())

					output, err := pgo("create", "user",
						"--selector=name="+cluster(), "-n", namespace(),
						"--username=samwise", "--password=hobbit", "--managed",
					).Exec(t)
					require.NoError(t, err)
					require.NotEmpty(t, output)

					// Connect using the above credentials
					pool := clusterConnection(t, namespace(), cluster(),
						"user=samwise password=hobbit database="+db)
					pool.Close()
				})

				t.Run("generates password", func(t *testing.T) {
					t.Parallel()
					requireClusterReady(t, namespace(), cluster(), time.Minute)
					db := clusterDatabase(t, namespace(), cluster())
					password := regexp.MustCompile(`\s+gimli\s+(\S+)`)

					output, err := pgo("create", "user", cluster(), "-n", namespace(),
						"--username=gimli", "--password-length=16", "--managed",
					).Exec(t)
					require.NoError(t, err)
					require.Regexp(t, password, output, "expected pgo to show the generated password")

					// Connect using the above credentials
					pool := clusterConnection(t, namespace(), cluster(),
						"user=gimli password="+password.FindStringSubmatch(output)[1]+" database="+db)
					pool.Close()
				})

				t.Run("does not keep password", func(t *testing.T) {
					t.Parallel()
					requireClusterReady(t, namespace(), cluster(), time.Minute)

					output, err := pgo("create", "user", cluster(), "-n", namespace(),
						"--username=arwen", "--valid-days=60",
					).Exec(t)
					require.NoError(t, err)
					require.NotEmpty(t, output)

					_, err = pgo("show", "user", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Empty(t, showPassword(t, namespace(), cluster(), "arwen"))

					stdout, stderr := clusterPSQL(t, namespace(), cluster(), `\du arwen`)
					require.Empty(t, stderr)
					require.Contains(t, stdout, "arwen", "expected user to exist")
				})
			})

			t.Run("update user", func(t *testing.T) {
				t.Run("changes password", func(t *testing.T) {
					t.Parallel()
					requireClusterReady(t, namespace(), cluster(), time.Minute)
					db := clusterDatabase(t, namespace(), cluster())

					_, err := pgo("create", "user", cluster(), "-n", namespace(),
						"--username=howl", "--password=wizard",
					).Exec(t)
					require.NoError(t, err)

					output, err := pgo("update", "user", cluster(), "-n", namespace(),
						"--username=howl", "--password=jenkins",
					).Exec(t)
					require.NoError(t, err)
					require.NotEmpty(t, output)

					// Connect using the above credentials
					pool := clusterConnection(t, namespace(), cluster(),
						"user=howl password=jenkins database="+db)
					pool.Close()
				})

				t.Run("changes expiration", func(t *testing.T) {
					t.Parallel()
					requireClusterReady(t, namespace(), cluster(), time.Minute)
					db := clusterDatabase(t, namespace(), cluster())

					_, err := pgo("create", "user", cluster(), "-n", namespace(),
						"--username=sophie", "--password=hatter",
					).Exec(t)
					require.NoError(t, err)

					{
						output, err := pgo("update", "user", cluster(), "-n", namespace(),
							"--username=sophie", "--valid-days=10",
						).Exec(t)
						require.NoError(t, err)
						require.NotEmpty(t, output)

						// Connect using the above credentials
						pool := clusterConnection(t, namespace(), cluster(),
							"user=sophie password=hatter database="+db)
						pool.Close()

						stdout, stderr := clusterPSQL(t, namespace(), cluster(), `\du sophie`)
						require.Empty(t, stderr)
						require.Contains(t, stdout, time.Now().AddDate(0, 0, 10).Format("2006-01-02"),
							"expected expiry to be set")
					}

					{
						_, err := pgo("update", "user", cluster(), "-n", namespace(),
							"--username=sophie", "--expire-user",
						).Exec(t)
						require.NoError(t, err)

						stdout, stderr := clusterPSQL(t, namespace(), cluster(), `\du sophie`)
						require.Empty(t, stderr)
						require.Contains(t, stdout, "-infinity", "expected to find an expiry")

						expiry := regexp.MustCompile(`\d{4}-\d{2}-\d{2}`).FindString(stdout)
						require.Less(t, expiry, time.Now().Format("2006-01-02"),
							"expected expiry to have passed")
					}
				})

				t.Run("generates password", func(t *testing.T) {
					t.Skip("BUG: --expired silently requires --password-length")
					t.Parallel()
					requireClusterReady(t, namespace(), cluster(), time.Minute)
					db := clusterDatabase(t, namespace(), cluster())

					_, err := pgo("create", "user", cluster(), "-n", namespace(),
						"--username=calcifer", "--valid-days=2", "--managed",
					).Exec(t)
					require.NoError(t, err)

					original := showPassword(t, namespace(), cluster(), "calcifer")

					_, err = pgo("update", "user", cluster(), "-n", namespace(),
						"--expired=5",
					).Exec(t)
					require.NoError(t, err)

					generated := showPassword(t, namespace(), cluster(), "calcifer")
					require.True(t, original != generated,
						"expected password to be regenerated")

					// Connect using the above credentials
					pool := clusterConnection(t, namespace(), cluster(),
						"user=calcifer password="+generated+" database="+db)
					pool.Close()
				})
			})

			t.Run("delete user", func(t *testing.T) {
				t.Run("removes managed", func(t *testing.T) {
					t.Parallel()
					requireClusterReady(t, namespace(), cluster(), time.Minute)

					_, err := pgo("create", "user", cluster(), "-n", namespace(),
						"--username=ged", "--managed",
					).Exec(t)
					require.NoError(t, err)

					output, err := pgo("delete", "user", cluster(), "-n", namespace(),
						"--username=ged", "--no-prompt",
					).Exec(t)
					require.NoError(t, err)
					require.NotEmpty(t, output)

					_, err = pgo("show", "user", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Empty(t, showPassword(t, namespace(), cluster(), "ged"),
						"expected pgo to forget about this user")

					stdout, stderr := clusterPSQL(t, namespace(), cluster(), `\du ged`)
					require.Empty(t, stderr)
					require.NotRegexp(t, `\bged\b`, stdout,
						"expected user to be removed")
				})

				t.Run("removes unmanaged", func(t *testing.T) {
					t.Parallel()
					requireClusterReady(t, namespace(), cluster(), time.Minute)

					_, err := pgo("create", "user", cluster(), "-n", namespace(),
						"--username=ogion",
					).Exec(t)
					require.NoError(t, err)

					output, err := pgo("delete", "user", cluster(), "-n", namespace(),
						"--username=ogion", "--no-prompt",
					).Exec(t)
					require.NoError(t, err)
					require.NotEmpty(t, output)

					stdout, stderr := clusterPSQL(t, namespace(), cluster(), `\du ogion`)
					require.Empty(t, stderr)
					require.NotRegexp(t, `\bogion\b`, stdout,
						"expected user to be removed")
				})

				t.Run("accepts selector", func(t *testing.T) {
					t.Parallel()
					requireClusterReady(t, namespace(), cluster(), time.Minute)

					_, err := pgo("create", "user", cluster(), "-n", namespace(),
						"--username=vetch", "--managed",
					).Exec(t)
					require.NoError(t, err)

					output, err := pgo("delete", "user",
						"--selector=name="+cluster(), "-n", namespace(),
						"--username=vetch", "--no-prompt",
					).Exec(t)
					require.NoError(t, err)
					require.NotEmpty(t, output)

					_, err = pgo("show", "user", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Empty(t, showPassword(t, namespace(), cluster(), "vetch"),
						"expected pgo to forget about this user")

					stdout, stderr := clusterPSQL(t, namespace(), cluster(), `\du vetch`)
					require.Empty(t, stderr)
					require.NotRegexp(t, `\bvetch\b`, stdout,
						"expected user to be removed")
				})
			})
		})
	})
}
