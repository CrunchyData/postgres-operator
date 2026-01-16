// Copyright 2017 - 2026 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
)

func TestPodSecurityContext(t *testing.T) {
	t.Run("Non-Openshift", func(t *testing.T) {
		assert.Assert(t, cmp.MarshalMatches(PodSecurityContext(2, []int64{}), `
fsGroup: 2
fsGroupChangePolicy: OnRootMismatch
	`))

		supplementalGroups := []int64{3, 4}
		assert.Assert(t, cmp.MarshalMatches(PodSecurityContext(26, supplementalGroups), `
fsGroup: 26
fsGroupChangePolicy: OnRootMismatch
supplementalGroups:
- 3
- 4
	`))
	})

	t.Run("OpenShift", func(t *testing.T) {
		assert.Assert(t, cmp.MarshalMatches(PodSecurityContext(0, []int64{}),
			`fsGroupChangePolicy: OnRootMismatch`))

		supplementalGroups := []int64{3, 4}
		assert.Assert(t, cmp.MarshalMatches(PodSecurityContext(0, supplementalGroups), `
fsGroupChangePolicy: OnRootMismatch
supplementalGroups:
- 3
- 4
	`))
	})

	t.Run("NoRootGID", func(t *testing.T) {
		supplementalGroups := []int64{999, 0, 100, 0}
		assert.DeepEqual(t, []int64{999, 100}, PodSecurityContext(2, supplementalGroups).SupplementalGroups)

		supplementalGroups = []int64{0}
		assert.Assert(t, PodSecurityContext(2, supplementalGroups).SupplementalGroups == nil)
	})
}
