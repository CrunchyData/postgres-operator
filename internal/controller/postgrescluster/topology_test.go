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

package postgrescluster

import (
	"testing"

	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDefaultTopologySpreadConstraints(t *testing.T) {
	constraints := defaultTopologySpreadConstraints(metav1.LabelSelector{
		MatchLabels: map[string]string{"basic": "stuff"},
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "k1", Operator: "op", Values: []string{"v1", "v2"}},
		},
	})

	// Entire selector, hostname, zone, and ScheduleAnyway.
	assert.Assert(t, marshalMatches(constraints, `
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
