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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TC110 ✓
func TestOperatorRBAC(t *testing.T) {
	t.Parallel()

	withNamespace(t, func(namespace1 func() string) {
		withNamespace(t, func(namespace2 func() string) {
			t.Run("create pgouser", func(t *testing.T) {
				t.Run("uses namespaces", func(t *testing.T) {
					var err error
					_, err = pgo("create", "pgouser", "heihei",
						"--pgouser-namespaces="+namespace1()+","+namespace2(),
						"--pgouser-password=moana",
						"--pgouser-roles=pgoadmin",
					).Exec(t)
					require.NoError(t, err)
					defer pgo("delete", "pgouser", "heihei", "--no-prompt").Exec(t)

					var output string
					output, err = pgo("create", "pgouser", "maui",
						"--pgouser-namespaces=pgo-test-does-not-exist",
						"--pgouser-password=demigod",
						"--pgouser-roles=pgoadmin",
					).Exec(t)
					require.Error(t, err)
					require.Contains(t, output, "not watched by")

					_, err = pgo("create", "pgouser", "pua",
						"--all-namespaces",
						"--pgouser-password=tafiti",
						"--pgouser-roles=pgoadmin",
					).Exec(t)
					require.NoError(t, err)
					defer pgo("delete", "pgouser", "pua", "--no-prompt").Exec(t)

					output, err = pgo("show", "pgouser", "--all").Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "heihei")
					require.NotContains(t, output, "maui")
					require.Contains(t, output, "pua")
				})
			})

			t.Run("create pgorole", func(t *testing.T) {
				var err error
				_, err = pgo("create", "pgorole", "junker", "--permissions=CreateCluster").Exec(t)
				require.NoError(t, err)
				defer pgo("delete", "pgorole", "junker", "--no-prompt").Exec(t)

				var output string
				output, err = pgo("show", "pgorole", "--all").Exec(t)
				require.NoError(t, err)
				require.Contains(t, output, "junker")
			})

			t.Run("update pgouser", func(t *testing.T) {
				t.Run("constrains actions", func(t *testing.T) {
					var err error

					// initially pgoadmin
					_, err = pgo("create", "pgouser", "heihei",
						"--pgouser-namespaces="+namespace1()+","+namespace2(),
						"--pgouser-password=moana",
						"--pgouser-roles=pgoadmin",
					).Exec(t)
					require.NoError(t, err)
					defer pgo("delete", "pgouser", "heihei", "--no-prompt").Exec(t)

					_, err = pgo("create", "pgorole", "junker", "--permissions=CreateCluster").Exec(t)
					require.NoError(t, err)
					defer pgo("delete", "pgorole", "junker", "--no-prompt").Exec(t)

					// change to junker
					_, err = pgo("update", "pgouser", "heihei", "--pgouser-roles=junker").Exec(t)
					require.NoError(t, err)

					// allowed
					_, err = pgo("create", "cluster", "test-permissions", "-n", namespace1()).
						WithEnvironment("PGOUSERNAME", "heihei").
						WithEnvironment("PGOUSERPASS", "moana").
						Exec(t)
					require.NoError(t, err)
					defer teardownCluster(t, namespace1(), "test-permissions", time.Now())

					// forbidden
					_, err = pgo("update", "namespace", namespace2()).
						WithEnvironment("PGOUSERNAME", "heihei").
						WithEnvironment("PGOUSERPASS", "moana").
						Exec(t)
					require.Error(t, err)
				})
			})

			t.Run("delete pgouser", func(t *testing.T) {
				var err error
				_, err = pgo("create", "pgouser", "heihei",
					"--pgouser-namespaces="+namespace1()+","+namespace2(),
					"--pgouser-password=moana",
					"--pgouser-roles=pgoadmin",
				).Exec(t)
				require.NoError(t, err)
				defer pgo("delete", "pgouser", "heihei", "--no-prompt").Exec(t)

				var output string
				output, err = pgo("show", "pgouser", "--all").Exec(t)
				require.NoError(t, err)
				require.Contains(t, output, "heihei")

				_, err = pgo("delete", "pgouser", "heihei", "--no-prompt").Exec(t)
				require.NoError(t, err)

				output, err = pgo("show", "pgouser", "--all").Exec(t)
				require.NoError(t, err)
				require.NotContains(t, output, "heihei")
			})
		})
	})
}
