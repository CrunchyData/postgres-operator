/*
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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

package pgbackrest

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func isUniqueAndSorted(slice []string) bool {
	if len(slice) > 1 {
		previous := slice[0]
		for _, next := range slice[1:] {
			if next <= previous {
				return false
			}
			previous = next
		}
	}
	return true
}

func TestPermissions(t *testing.T) {
	cluster := new(v1beta1.PostgresCluster)
	cluster.Default()

	permissions := Permissions(cluster)
	for _, rule := range permissions {
		assert.Assert(t, isUniqueAndSorted(rule.APIGroups), "got %q", rule.APIGroups)
		assert.Assert(t, isUniqueAndSorted(rule.Resources), "got %q", rule.Resources)
		assert.Assert(t, isUniqueAndSorted(rule.Verbs), "got %q", rule.Verbs)
	}

	assert.Assert(t, cmp.MarshalMatches(permissions, `
- apiGroups:
  - ""
  resources:
  - pods
  verbs:
  - list
- apiGroups:
  - ""
  resources:
  - pods/exec
  verbs:
  - create
	`))
}
