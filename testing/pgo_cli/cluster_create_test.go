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
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TC41 ✓
func TestClusterCreate(t *testing.T) {
	t.Parallel()

	withNamespace(t, func(namespace func() string) {
		t.Run("create cluster", func(t *testing.T) {
			t.Run("creates a workflow", func(t *testing.T) {
				workflow := regexp.MustCompile(`(?m:^workflow id.*?(\S+)$)`)

				output, err := pgo("create", "cluster", "mycluster", "-n", namespace()).Exec(t)
				defer teardownCluster(t, namespace(), "mycluster", time.Now())
				require.NoError(t, err)
				require.Regexp(t, workflow, output, "expected pgo to show the workflow")

				_, err = pgo("show", "workflow", workflow.FindStringSubmatch(output)[1], "-n", namespace()).Exec(t)
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
}
