// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"testing"

	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
)

func TestDefaultTopologySpreadConstraints(t *testing.T) {
	constraints := defaultTopologySpreadConstraints(metav1.LabelSelector{
		MatchLabels: map[string]string{"basic": "stuff"},
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "k1", Operator: "op", Values: []string{"v1", "v2"}},
		},
	})

	// Entire selector, hostname, zone, and ScheduleAnyway.
	assert.Assert(t, cmp.MarshalMatches(constraints, `
- labelSelector:
    matchExpressions:
    - key: k1
      operator: op
      values:
      - v1
      - v2
    matchLabels:
      basic: stuff
  maxSkew: 1
  topologyKey: kubernetes.io/hostname
  whenUnsatisfiable: ScheduleAnyway
- labelSelector:
    matchExpressions:
    - key: k1
      operator: op
      values:
      - v1
      - v2
    matchLabels:
      basic: stuff
  maxSkew: 1
  topologyKey: topology.kubernetes.io/zone
  whenUnsatisfiable: ScheduleAnyway
	`))
}
