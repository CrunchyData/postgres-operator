/*
 Copyright 2021 Crunchy Data Solutions, Inc.
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

package naming

import (
	"testing"

	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/util/validation"
)

func TestLabelsValid(t *testing.T) {
	assert.Assert(t, nil == validation.IsQualifiedName(LabelCluster))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelInstance))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelInstanceSet))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelPatroni))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelRole))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelPGBackRest))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelPGBackRestConfig))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelPGBackRestDedicated))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelPGBackRestRepo))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelPGBackRestRepoHost))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelPGBackRestRepoVolume))
}

func TestLabelValuesValid(t *testing.T) {
	assert.Assert(t, nil == validation.IsDNS1123Label(RolePatroniLeader))
	assert.Assert(t, nil == validation.IsDNS1123Label(RolePatroniReplica))
	assert.Assert(t, nil == validation.IsDNS1123Label(RolePGBouncer))
	assert.Assert(t, nil == validation.IsDNS1123Label(RolePrimary))
	assert.Assert(t, nil == validation.IsDNS1123Label(RoleReplica))
}

// validate various functions that return pgBackRest labels
func TestPGBackRestLabelFuncs(t *testing.T) {

	clusterName := "hippo"
	repoName := "hippo-repo"

	// verify the labels that identify pgBackRest resources
	pgBackRestLabels := PGBackRestLabels(clusterName)
	assert.Equal(t, pgBackRestLabels.Get(LabelCluster), clusterName)
	assert.Check(t, pgBackRestLabels.Has(LabelPGBackRest))

	// verify the labels that identify pgBackRest repo resources
	pgBackRestRepoLabels := PGBackRestRepoLabels(clusterName, repoName)
	assert.Equal(t, pgBackRestRepoLabels.Get(LabelCluster), clusterName)
	assert.Check(t, pgBackRestRepoLabels.Has(LabelPGBackRest))
	assert.Equal(t, pgBackRestRepoLabels.Get(LabelPGBackRestRepo), repoName)

	// verify the labels that identify pgBackRest configuration resources
	pgBackRestConfigLabels := PGBackRestConfigLabels(clusterName)
	assert.Equal(t, pgBackRestConfigLabels.Get(LabelCluster), clusterName)
	assert.Check(t, pgBackRestConfigLabels.Has(LabelPGBackRest))
	assert.Check(t, pgBackRestConfigLabels.Has(LabelPGBackRestConfig))

	// verify the labels that identify pgBackRest dedicated repository host resources
	pgBackRestDedicatedLabels := PGBackRestDedicatedLabels(clusterName)
	assert.Equal(t, pgBackRestDedicatedLabels.Get(LabelCluster), clusterName)
	assert.Check(t, pgBackRestDedicatedLabels.Has(LabelPGBackRest))
	assert.Check(t, pgBackRestDedicatedLabels.Has(LabelPGBackRestRepoHost))
	assert.Check(t, pgBackRestDedicatedLabels.Has(LabelPGBackRestDedicated))

	// verify the labels that identify pgBackRest repository host resources
	pgBackRestRepoHostLabels := PGBackRestRepoHostLabels(clusterName)
	assert.Equal(t, pgBackRestRepoHostLabels.Get(LabelCluster), clusterName)
	assert.Check(t, pgBackRestRepoHostLabels.Has(LabelPGBackRest))
	assert.Check(t, pgBackRestRepoHostLabels.Has(LabelPGBackRestRepoHost))

	// verify the labels that identify pgBackRest repository volume resources
	pgBackRestRepoVolumeLabels := PGBackRestRepoVolumeLabels(clusterName, repoName)
	assert.Equal(t, pgBackRestRepoVolumeLabels.Get(LabelCluster), clusterName)
	assert.Check(t, pgBackRestRepoVolumeLabels.Has(LabelPGBackRest))
	assert.Equal(t, pgBackRestRepoVolumeLabels.Get(LabelPGBackRestRepo), repoName)
	assert.Check(t, pgBackRestRepoVolumeLabels.Has(LabelPGBackRestRepoVolume))
}
