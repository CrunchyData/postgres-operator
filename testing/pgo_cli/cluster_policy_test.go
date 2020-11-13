package pgo_cli_test

/*
 Copyright 2020 Crunchy Data Solutions, Inc.
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
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TC45 ✓
// TC52 ✓
// TC115 ✓
func TestClusterPolicy(t *testing.T) {
	t.Parallel()

	withNamespace(t, func(namespace func() string) {
		t.Run("create policy", func(t *testing.T) {
			t.Run("requires argument", func(t *testing.T) {
				t.Skip("BUG: exits zero")
				output, err := pgo("create", "policy", "hello", "-n", namespace()).Exec(t)
				require.Error(t, err)
				require.Contains(t, output, "flags are required")
			})

			t.Run("keeps content", func(t *testing.T) {
				const policyPath = "../testdata/policy1.sql"
				policyContent, err := ioutil.ReadFile(policyPath)
				if err != nil {
					t.Fatalf("bug in test: %v", err)
				}

				output, err := pgo("create", "policy", "hello", "--in-file="+policyPath, "-n", namespace()).Exec(t)
				defer pgo("delete", "policy", "hello", "--no-prompt", "-n", namespace()).Exec(t)
				require.NoError(t, err)
				require.NotEmpty(t, output)

				output, err = pgo("show", "policy", "hello", "-n", namespace()).Exec(t)
				require.NoError(t, err)
				require.Contains(t, output, "hello")
				require.Contains(t, output, string(policyContent))
			})
		})

		withCluster(t, namespace, func(cluster func() string) {
			t.Run("apply", func(t *testing.T) {
				t.Run("requires selector", func(t *testing.T) {
					t.Skip("BUG: exits zero")
					output, err := pgo("apply", "nope", "-n", namespace()).Exec(t)
					require.Error(t, err)
					require.Contains(t, output, "required")
				})

				t.Run("executes a policy", func(t *testing.T) {
					t.Skip("BUG: how to choose a database")
					const policyPath = "../testdata/policy1.sql"

					_, err := pgo("create", "policy", "p1-apply", "--in-file="+policyPath, "-n", namespace()).Exec(t)
					defer pgo("delete", "policy", "p1-apply", "--no-prompt", "-n", namespace()).Exec(t)
					require.NoError(t, err)

					requireClusterReady(t, namespace(), cluster(), time.Minute)

					output, err := pgo("apply", "p1-apply", "--selector=name="+cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.NotEmpty(t, output)

					stdout, stderr := clusterPSQL(t, namespace(), cluster(), `
						\c userdb
						\dt policy1
					`)
					require.Empty(t, stderr)
					require.Contains(t, stdout, "(1 row)")
				})
			})
		})
	})
}
