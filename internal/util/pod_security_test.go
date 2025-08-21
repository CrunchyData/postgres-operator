// Copyright 2017 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/internal/kubernetes"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
)

func TestPodSecurityContext(t *testing.T) {
	ctx := context.Background()
	assert.Assert(t, cmp.MarshalMatches(PodSecurityContext(ctx, 2, []int64{}), `
fsGroup: 2
fsGroupChangePolicy: OnRootMismatch
	`))

	supplementalGroups := []int64{3, 4}
	assert.Assert(t, cmp.MarshalMatches(PodSecurityContext(ctx, 26, supplementalGroups), `
fsGroup: 26
fsGroupChangePolicy: OnRootMismatch
supplementalGroups:
- 3
- 4
	`))

	ctx = kubernetes.NewAPIContext(ctx, kubernetes.NewAPISet(kubernetes.API{
		Group: "security.openshift.io", Version: "v1",
		Kind: "SecurityContextConstraints",
	}))
	assert.Assert(t, cmp.MarshalMatches(PodSecurityContext(ctx, 2, []int64{}),
		`fsGroupChangePolicy: OnRootMismatch`))

	assert.Assert(t, cmp.MarshalMatches(PodSecurityContext(ctx, 2, supplementalGroups), `
fsGroupChangePolicy: OnRootMismatch
supplementalGroups:
- 3
- 4
	`))

	t.Run("NoRootGID", func(t *testing.T) {
		supplementalGroups = []int64{999, 0, 100, 0}
		assert.DeepEqual(t, []int64{999, 100}, PodSecurityContext(ctx, 2, supplementalGroups).SupplementalGroups)

		supplementalGroups = []int64{0}
		assert.Assert(t, PodSecurityContext(ctx, 2, supplementalGroups).SupplementalGroups == nil)
	})
}
