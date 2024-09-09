// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package naming

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestAnyCluster(t *testing.T) {
	s, err := AsSelector(AnyCluster())
	assert.NilError(t, err)
	assert.DeepEqual(t, s.String(), strings.Join([]string{
		"postgres-operator.crunchydata.com/cluster",
	}, ","))
}

func TestCluster(t *testing.T) {
	s, err := AsSelector(Cluster("something"))
	assert.NilError(t, err)
	assert.DeepEqual(t, s.String(), strings.Join([]string{
		"postgres-operator.crunchydata.com/cluster=something",
	}, ","))

	_, err = AsSelector(Cluster("--whoa/yikes"))
	assert.ErrorContains(t, err, "Invalid")
}

func TestClusterBackupJobs(t *testing.T) {
	s, err := AsSelector(ClusterBackupJobs("something"))
	assert.NilError(t, err)
	assert.DeepEqual(t, s.String(), strings.Join([]string{
		"postgres-operator.crunchydata.com/cluster=something",
		"postgres-operator.crunchydata.com/pgbackrest-backup",
	}, ","))

	_, err = AsSelector(Cluster("--whoa/yikes"))
	assert.ErrorContains(t, err, "Invalid")
}

func TestClusterDataForPostgresAndPGBackRest(t *testing.T) {
	s, err := AsSelector(ClusterDataForPostgresAndPGBackRest("something"))
	assert.NilError(t, err)
	assert.DeepEqual(t, s.String(), strings.Join([]string{
		"postgres-operator.crunchydata.com/cluster=something",
		"postgres-operator.crunchydata.com/data in (pgbackrest,postgres)",
	}, ","))

	_, err = AsSelector(ClusterDataForPostgresAndPGBackRest("--whoa/yikes"))
	assert.ErrorContains(t, err, "Invalid")
}

func TestClusterInstance(t *testing.T) {
	s, err := AsSelector(ClusterInstance("daisy", "dog"))
	assert.NilError(t, err)
	assert.DeepEqual(t, s.String(), strings.Join([]string{
		"postgres-operator.crunchydata.com/cluster=daisy",
		"postgres-operator.crunchydata.com/instance=dog",
	}, ","))

	_, err = AsSelector(ClusterInstance("--whoa/son", "--whoa/yikes"))
	assert.ErrorContains(t, err, "Invalid")
}

func TestClusterInstances(t *testing.T) {
	s, err := AsSelector(ClusterInstances("something"))
	assert.NilError(t, err)
	assert.DeepEqual(t, s.String(), strings.Join([]string{
		"postgres-operator.crunchydata.com/cluster=something",
		"postgres-operator.crunchydata.com/instance",
	}, ","))

	_, err = AsSelector(ClusterInstances("--whoa/yikes"))
	assert.ErrorContains(t, err, "Invalid")
}

func TestClusterInstanceSet(t *testing.T) {
	s, err := AsSelector(ClusterInstanceSet("something", "also"))
	assert.NilError(t, err)
	assert.DeepEqual(t, s.String(), strings.Join([]string{
		"postgres-operator.crunchydata.com/cluster=something",
		"postgres-operator.crunchydata.com/instance-set=also",
	}, ","))

	_, err = AsSelector(ClusterInstanceSet("--whoa/yikes", "ok"))
	assert.ErrorContains(t, err, "Invalid")
}

func TestClusterInstanceSets(t *testing.T) {
	s, err := AsSelector(ClusterInstanceSets("something"))
	assert.NilError(t, err)
	assert.DeepEqual(t, s.String(), strings.Join([]string{
		"postgres-operator.crunchydata.com/cluster=something",
		"postgres-operator.crunchydata.com/instance-set",
	}, ","))

	_, err = AsSelector(ClusterInstanceSets("--whoa/yikes"))
	assert.ErrorContains(t, err, "Invalid")
}

func TestClusterPatronis(t *testing.T) {
	cluster := &v1beta1.PostgresCluster{}
	cluster.Name = "something"

	s, err := AsSelector(ClusterPatronis(cluster))
	assert.NilError(t, err)
	assert.DeepEqual(t, s.String(), strings.Join([]string{
		"postgres-operator.crunchydata.com/cluster=something",
		"postgres-operator.crunchydata.com/patroni=something-ha",
	}, ","))

	cluster.Name = "--nope--"
	_, err = AsSelector(ClusterPatronis(cluster))
	assert.ErrorContains(t, err, "Invalid")
}

func TestClusterPGBouncerSelector(t *testing.T) {
	cluster := &v1beta1.PostgresCluster{}
	cluster.Name = "something"

	s, err := AsSelector(ClusterPGBouncerSelector(cluster))
	assert.NilError(t, err)
	assert.DeepEqual(t, s.String(), strings.Join([]string{
		"postgres-operator.crunchydata.com/cluster=something",
		"postgres-operator.crunchydata.com/role=pgbouncer",
	}, ","))

	cluster.Name = "--bad--dog"
	_, err = AsSelector(ClusterPGBouncerSelector(cluster))
	assert.ErrorContains(t, err, "Invalid")
}

func TestClusterPostgresUsers(t *testing.T) {
	s, err := AsSelector(ClusterPostgresUsers("something"))
	assert.NilError(t, err)
	assert.DeepEqual(t, s.String(), strings.Join([]string{
		"postgres-operator.crunchydata.com/cluster=something",
		"postgres-operator.crunchydata.com/pguser",
	}, ","))

	_, err = AsSelector(ClusterPostgresUsers("--nope--"))
	assert.ErrorContains(t, err, "Invalid")
}

func TestCrunchyBridgeClusterPostgresRoles(t *testing.T) {
	s, err := AsSelector(CrunchyBridgeClusterPostgresRoles("something"))
	assert.NilError(t, err)
	assert.DeepEqual(t, s.String(), strings.Join([]string{
		"postgres-operator.crunchydata.com/cluster=something",
		"postgres-operator.crunchydata.com/role=cbc-pgrole",
	}, ","))

	_, err = AsSelector(CrunchyBridgeClusterPostgresRoles("--nope--"))
	assert.ErrorContains(t, err, "Invalid")
}
