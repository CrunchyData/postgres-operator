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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TC126 ✓
func TestClusterTest(t *testing.T) {
	t.Parallel()

	withNamespace(t, func(namespace func() string) {
		withCluster(t, namespace, func(cluster func() string) {
			t.Run("test", func(t *testing.T) {
				t.Run("shows something immediately", func(t *testing.T) {
					output, err := pgo("test", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, cluster())
				})

				t.Run("detects the cluster eventually", func(t *testing.T) {
					var output string
					var err error

					check := func() bool {
						output, err = pgo("test", cluster(), "-n", namespace()).Exec(t)
						require.NoError(t, err)
						return strings.Contains(output, "UP")
					}

					if !check() && !waitFor(t, check, time.Minute, time.Second) {
						require.Contains(t, output, "UP")
					}
				})
			})
		})
	})
}
