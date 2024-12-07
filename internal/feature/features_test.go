// Copyright 2017 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package feature

import (
	"context"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestDefaults(t *testing.T) {
	t.Parallel()
	gate := NewGate()

	assert.Assert(t, false == gate.Enabled(AppendCustomQueries))
	assert.Assert(t, true == gate.Enabled(AutoCreateUserSchema))
	assert.Assert(t, false == gate.Enabled(AutoGrowVolumes))
	assert.Assert(t, false == gate.Enabled(BridgeIdentifiers))
	assert.Assert(t, false == gate.Enabled(InstanceSidecars))
	assert.Assert(t, false == gate.Enabled(PGBouncerSidecars))
	assert.Assert(t, false == gate.Enabled(PGUpgradeCPUConcurrency))
	assert.Assert(t, false == gate.Enabled(TablespaceVolumes))
	assert.Assert(t, false == gate.Enabled(VolumeSnapshots))
}

func TestStringFormat(t *testing.T) {
	t.Parallel()
	gate := NewGate()

	assert.NilError(t, gate.Set(""))
	assert.NilError(t, gate.Set("TablespaceVolumes=true"))
	assert.Assert(t, true == gate.Enabled(TablespaceVolumes))

	err := gate.Set("NotAGate=true")
	assert.ErrorContains(t, err, "unrecognized feature gate")
	assert.ErrorContains(t, err, "NotAGate")

	err = gate.Set("GateNotSet")
	assert.ErrorContains(t, err, "missing bool")
	assert.ErrorContains(t, err, "GateNotSet")

	err = gate.Set("GateNotSet=foo")
	assert.ErrorContains(t, err, "invalid value")
	assert.ErrorContains(t, err, "GateNotSet")
}

func TestContext(t *testing.T) {
	t.Parallel()
	gate := NewGate()
	ctx := NewContext(context.Background(), gate)

	assert.Equal(t, ShowAssigned(ctx), "")
	assert.Assert(t, ShowEnabled(ctx) != "") // This assumes some feature is enabled by default.

	assert.NilError(t, gate.Set("TablespaceVolumes=true"))
	assert.Assert(t, Enabled(ctx, TablespaceVolumes))
	assert.Equal(t, ShowAssigned(ctx), "TablespaceVolumes=true")
	assert.Assert(t,
		strings.Contains(ShowEnabled(ctx), "TablespaceVolumes=true"),
		"got: %v", ShowEnabled(ctx))

	assert.NilError(t, gate.SetFromMap(map[string]bool{TablespaceVolumes: false}))
	assert.Assert(t, !Enabled(ctx, TablespaceVolumes))
	assert.Equal(t, ShowAssigned(ctx), "TablespaceVolumes=false")
	assert.Assert(t,
		!strings.Contains(ShowEnabled(ctx), "TablespaceVolumes"),
		"got: %v", ShowEnabled(ctx))
}
