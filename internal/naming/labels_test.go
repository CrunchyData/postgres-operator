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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation"
)

func TestLabelsValid(t *testing.T) {
	assert.Assert(t, nil == validation.IsQualifiedName(LabelCluster))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelInstance))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelInstanceSet))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelPatroni))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelRole))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelPGBackRest))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelPGBackRestBackup))
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
	assert.Assert(t, nil == validation.IsDNS1123Label(string(BackupReplicaCreate)))
}

func TestMerge(t *testing.T) {
	for _, test := range []struct {
		name   string
		sets   []map[string]string
		expect labels.Set
	}{{
		name:   "no sets",
		sets:   []map[string]string{},
		expect: labels.Set{},
	}, {
		name: "nil map",
		sets: []map[string]string{
			map[string]string(nil),
		},
		expect: labels.Set{},
	}, {
		name: "has empty sets",
		sets: []map[string]string{
			{"label.one": "one"},
			{},
		},
		expect: labels.Set{
			"label.one": "one",
		},
	}, {
		name: "two sets with no overlap",
		sets: []map[string]string{
			{"label.one": "one"},
			{"label.two": "two"},
		},
		expect: labels.Set{
			"label.one": "one",
			"label.two": "two",
		},
	}, {
		name: "two sets with overlap",
		sets: []map[string]string{
			{LabelCluster: "bad", "label.one": "one"},
			{LabelCluster: "good", "label.two": "two"},
		},
		expect: labels.Set{
			"label.one":  "one",
			"label.two":  "two",
			LabelCluster: "good",
		},
	}, {
		name: "three sets with no overlap",
		sets: []map[string]string{
			{"label.one": "one"},
			{"label.two": "two"},
			{"label.three": "three"},
		},
		expect: labels.Set{
			"label.one":   "one",
			"label.two":   "two",
			"label.three": "three",
		},
	}, {
		name: "three sets with overlap",
		sets: []map[string]string{
			{LabelCluster: "bad-one", "label.one": "one"},
			{LabelCluster: "bad-two", "label.two": "two"},
			{LabelCluster: "good", "label.three": "three"},
		},
		expect: labels.Set{
			"label.one":   "one",
			"label.two":   "two",
			"label.three": "three",
			LabelCluster:  "good",
		},
	}} {
		t.Run(test.name, func(t *testing.T) {
			merged := Merge(test.sets...)
			assert.DeepEqual(t, merged, test.expect)
		})
	}
}

// validate various functions that return pgBackRest labels
func TestPGBackRestLabelFuncs(t *testing.T) {

	clusterName := "hippo"
	repoName := "hippo-repo"

	// verify the labels that identify pgBackRest resources
	pgBackRestLabels := PGBackRestLabels(clusterName)
	assert.Equal(t, pgBackRestLabels.Get(LabelCluster), clusterName)
	assert.Check(t, pgBackRestLabels.Has(LabelPGBackRest))

	// verify that the labels selector is created as expected
	pgBackRestSelector := PGBackRestSelector(clusterName)
	assert.Check(t, pgBackRestSelector.Matches(pgBackRestLabels))

	// verify the labels that identify pgBackRest backup resources
	pgBackRestReplicaBackupLabels := PGBackRestBackupJobLabels(clusterName, repoName,
		BackupReplicaCreate)
	assert.Equal(t, pgBackRestReplicaBackupLabels.Get(LabelCluster), clusterName)
	assert.Check(t, pgBackRestReplicaBackupLabels.Has(LabelPGBackRest))
	assert.Equal(t, pgBackRestReplicaBackupLabels.Get(LabelPGBackRestRepo), repoName)
	assert.Equal(t, pgBackRestReplicaBackupLabels.Get(LabelPGBackRestBackup),
		string(BackupReplicaCreate))

	// verify the pgBackRest label selector function
	// PGBackRestBackupJobSelector
	pgBackRestBackupJobSelector := PGBackRestBackupJobSelector(clusterName, repoName,
		BackupReplicaCreate)
	assert.Check(t, pgBackRestBackupJobSelector.Matches(pgBackRestReplicaBackupLabels))

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

	// verify the labels that identify pgBackRest repo resources
	pgBackRestCronJobLabels := PGBackRestCronJobLabels(clusterName, repoName,
		"testBackupType")
	assert.Equal(t, pgBackRestCronJobLabels.Get(LabelCluster), clusterName)
	assert.Check(t, pgBackRestCronJobLabels.Has(LabelPGBackRest))
	assert.Equal(t, pgBackRestCronJobLabels.Get(LabelPGBackRestRepo), repoName)

	// verify the labels that identify pgBackRest dedicated repository host resources
	pgBackRestDedicatedLabels := PGBackRestDedicatedLabels(clusterName)
	assert.Equal(t, pgBackRestDedicatedLabels.Get(LabelCluster), clusterName)
	assert.Check(t, pgBackRestDedicatedLabels.Has(LabelPGBackRest))
	assert.Check(t, pgBackRestDedicatedLabels.Has(LabelPGBackRestRepoHost))
	assert.Check(t, pgBackRestDedicatedLabels.Has(LabelPGBackRestDedicated))

	// verify that the dedicated labels selector is created as expected
	pgBackRestDedicatedSelector := PGBackRestDedicatedSelector(clusterName)
	assert.Check(t, pgBackRestDedicatedSelector.Matches(pgBackRestDedicatedLabels))

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
