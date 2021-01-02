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
				require.Contains(t, output, "Total Volume Size")
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
