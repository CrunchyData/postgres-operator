// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

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
