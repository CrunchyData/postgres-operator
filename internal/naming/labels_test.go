// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package naming

import (
	"testing"

	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation"
)

func TestLabelsValid(t *testing.T) {
	assert.Assert(t, nil == validation.IsQualifiedName(LabelCluster))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelData))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelInstance))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelInstanceSet))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelMoveJob))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelMovePGBackRestRepoDir))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelMovePGDataDir))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelMovePGWalDir))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelPatroni))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelRole))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelPGBackRest))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelPGBackRestBackup))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelPGBackRestConfig))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelPGBackRestDedicated))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelPGBackRestRepo))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelPGBackRestRepoVolume))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelPGBackRestRestore))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelPGBackRestRestoreConfig))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelPGMonitorDiscovery))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelPostgresUser))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelStandalonePGAdmin))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelStartupInstance))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelCrunchyBridgeClusterPostgresRole))
}

func TestLabelValuesValid(t *testing.T) {
	assert.Assert(t, nil == validation.IsValidLabelValue(DataPGAdmin))
	assert.Assert(t, nil == validation.IsValidLabelValue(DataPGBackRest))
	assert.Assert(t, nil == validation.IsValidLabelValue(DataPostgres))
	assert.Assert(t, nil == validation.IsValidLabelValue(RolePatroniLeader))
	assert.Assert(t, nil == validation.IsValidLabelValue(RolePatroniReplica))
	assert.Assert(t, nil == validation.IsValidLabelValue(RolePGAdmin))
	assert.Assert(t, nil == validation.IsValidLabelValue(RolePGBouncer))
	assert.Assert(t, nil == validation.IsValidLabelValue(RolePostgresData))
	assert.Assert(t, nil == validation.IsValidLabelValue(RolePostgresUser))
	assert.Assert(t, nil == validation.IsValidLabelValue(RolePostgresWAL))
	assert.Assert(t, nil == validation.IsValidLabelValue(RolePrimary))
	assert.Assert(t, nil == validation.IsValidLabelValue(RoleReplica))
	assert.Assert(t, nil == validation.IsValidLabelValue(string(BackupManual)))
	assert.Assert(t, nil == validation.IsValidLabelValue(string(BackupReplicaCreate)))
	assert.Assert(t, nil == validation.IsValidLabelValue(string(BackupScheduled)))
	assert.Assert(t, nil == validation.IsValidLabelValue(RoleMonitoring))
	assert.Assert(t, nil == validation.IsValidLabelValue(RoleCrunchyBridgeClusterPostgresRole))
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
	assert.Equal(t, pgBackRestCronJobLabels.Get(LabelPGBackRestBackup), string(BackupScheduled))

	// verify the labels that identify pgBackRest dedicated repository host resources
	pgBackRestDedicatedLabels := PGBackRestDedicatedLabels(clusterName)
	assert.Equal(t, pgBackRestDedicatedLabels.Get(LabelCluster), clusterName)
	assert.Check(t, pgBackRestDedicatedLabels.Has(LabelPGBackRest))
	assert.Check(t, pgBackRestDedicatedLabels.Has(LabelPGBackRestDedicated))

	// verify that the dedicated labels selector is created as expected
	pgBackRestDedicatedSelector := PGBackRestDedicatedSelector(clusterName)
	assert.Check(t, pgBackRestDedicatedSelector.Matches(pgBackRestDedicatedLabels))

	// verify the labels that identify pgBackRest repository volume resources
	pgBackRestRepoVolumeLabels := PGBackRestRepoVolumeLabels(clusterName, repoName)
	assert.Equal(t, pgBackRestRepoVolumeLabels.Get(LabelCluster), clusterName)
	assert.Check(t, pgBackRestRepoVolumeLabels.Has(LabelPGBackRest))
	assert.Equal(t, pgBackRestRepoVolumeLabels.Get(LabelPGBackRestRepo), repoName)
	assert.Check(t, pgBackRestRepoVolumeLabels.Has(LabelPGBackRestRepoVolume))

	// verify the labels that identify pgBackRest repository volume resources
	pgBackRestRestoreJobLabels := PGBackRestRestoreJobLabels(clusterName)
	assert.Equal(t, pgBackRestRestoreJobLabels.Get(LabelCluster), clusterName)
	assert.Check(t, pgBackRestRestoreJobLabels.Has(LabelPGBackRest))
	assert.Check(t, pgBackRestRestoreJobLabels.Has(LabelPGBackRestRestore))

	// verify the labels that identify pgBackRest restore configuration resources
	pgBackRestRestoreConfigLabels := PGBackRestRestoreConfigLabels(clusterName)
	assert.Equal(t, pgBackRestRestoreConfigLabels.Get(LabelCluster), clusterName)
	assert.Check(t, pgBackRestRestoreConfigLabels.Has(LabelPGBackRest))
	assert.Check(t, pgBackRestRestoreConfigLabels.Has(LabelPGBackRestRestoreConfig))

	pgBackRestRestoreConfigSelector := PGBackRestRestoreConfigSelector(clusterName)
	assert.Check(t, pgBackRestRestoreConfigSelector.Matches(pgBackRestRestoreConfigLabels))
}

// validate the DirectoryMoveJobLabels function
func TestMoveJobLabelFunc(t *testing.T) {

	clusterName := "hippo"

	// verify the labels that identify directory move jobs
	dirMoveJobLabels := DirectoryMoveJobLabels(clusterName)
	assert.Equal(t, dirMoveJobLabels.Get(LabelCluster), clusterName)
	assert.Check(t, dirMoveJobLabels.Has(LabelMoveJob))
}
