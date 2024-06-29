/*
 Copyright 2017 - 2024 Crunchy Data Solutions, Inc.
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

package feature

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
)

func TestDefaults(t *testing.T) {
	t.Parallel()
	gate := NewGate()

	assert.Assert(t, false == gate.Enabled(AppendCustomQueries))
	assert.Assert(t, false == gate.Enabled(AutoCreateUserSchema))
	assert.Assert(t, false == gate.Enabled(AutoGrowVolumes))
	assert.Assert(t, false == gate.Enabled(BridgeIdentifiers))
	assert.Assert(t, false == gate.Enabled(InstanceSidecars))
	assert.Assert(t, false == gate.Enabled(PGBouncerSidecars))
	assert.Assert(t, false == gate.Enabled(TablespaceVolumes))

	assert.Equal(t, gate.String(), "")
}

func TestStringFormat(t *testing.T) {
	t.Parallel()
	gate := NewGate()

	assert.NilError(t, gate.Set(""))
	assert.NilError(t, gate.Set("TablespaceVolumes=true"))
	assert.Equal(t, gate.String(), "TablespaceVolumes=true")
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

	assert.NilError(t, gate.Set("TablespaceVolumes=true"))
	assert.Assert(t, true == Enabled(ctx, TablespaceVolumes))

	assert.NilError(t, gate.SetFromMap(map[string]bool{TablespaceVolumes: false}))
	assert.Assert(t, false == Enabled(ctx, TablespaceVolumes))
}
