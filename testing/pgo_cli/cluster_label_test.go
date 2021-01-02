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

	"github.com/stretchr/testify/require"
)

// TC42 ✓
// TC115 ✓
func TestClusterLabel(t *testing.T) {
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
}
