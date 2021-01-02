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
