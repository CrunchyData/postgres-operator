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
	assert.ErrorContains(t, err, "invalid")
}

func TestClusterDataForPostgresAndPGBackRest(t *testing.T) {
	s, err := AsSelector(ClusterDataForPostgresAndPGBackRest("something"))
	assert.NilError(t, err)
	assert.DeepEqual(t, s.String(), strings.Join([]string{
		"postgres-operator.crunchydata.com/cluster=something",
		"postgres-operator.crunchydata.com/data in (pgbackrest,postgres)",
	}, ","))

	_, err = AsSelector(ClusterDataForPostgresAndPGBackRest("--whoa/yikes"))
	assert.ErrorContains(t, err, "invalid")
}

func TestClusterInstance(t *testing.T) {
	s, err := AsSelector(ClusterInstance("daisy", "dog"))
	assert.NilError(t, err)
	assert.DeepEqual(t, s.String(), strings.Join([]string{
		"postgres-operator.crunchydata.com/cluster=daisy",
		"postgres-operator.crunchydata.com/instance=dog",
	}, ","))

	_, err = AsSelector(ClusterInstance("--whoa/son", "--whoa/yikes"))
	assert.ErrorContains(t, err, "invalid")
}

func TestClusterInstances(t *testing.T) {
	s, err := AsSelector(ClusterInstances("something"))
	assert.NilError(t, err)
	assert.DeepEqual(t, s.String(), strings.Join([]string{
		"postgres-operator.crunchydata.com/cluster=something",
		"postgres-operator.crunchydata.com/instance",
	}, ","))

	_, err = AsSelector(ClusterInstances("--whoa/yikes"))
	assert.ErrorContains(t, err, "invalid")
}

func TestClusterInstanceSet(t *testing.T) {
	s, err := AsSelector(ClusterInstanceSet("something", "also"))
	assert.NilError(t, err)
	assert.DeepEqual(t, s.String(), strings.Join([]string{
		"postgres-operator.crunchydata.com/cluster=something",
		"postgres-operator.crunchydata.com/instance-set=also",
	}, ","))

	_, err = AsSelector(ClusterInstanceSet("--whoa/yikes", "ok"))
	assert.ErrorContains(t, err, "invalid")
}

func TestClusterInstanceSets(t *testing.T) {
	s, err := AsSelector(ClusterInstanceSets("something"))
	assert.NilError(t, err)
	assert.DeepEqual(t, s.String(), strings.Join([]string{
		"postgres-operator.crunchydata.com/cluster=something",
		"postgres-operator.crunchydata.com/instance-set",
	}, ","))

	_, err = AsSelector(ClusterInstanceSets("--whoa/yikes"))
	assert.ErrorContains(t, err, "invalid")
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
	assert.ErrorContains(t, err, "invalid")
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
	assert.ErrorContains(t, err, "invalid")
}

func TestClusterPostgresUsers(t *testing.T) {
	s, err := AsSelector(ClusterPostgresUsers("something"))
	assert.NilError(t, err)
	assert.DeepEqual(t, s.String(), strings.Join([]string{
		"postgres-operator.crunchydata.com/cluster=something",
		"postgres-operator.crunchydata.com/pguser",
	}, ","))

	_, err = AsSelector(ClusterPostgresUsers("--nope--"))
	assert.ErrorContains(t, err, "invalid")
}

func TestClusterPrimary(t *testing.T) {
	s, err := AsSelector(ClusterPrimary("something"))
	assert.NilError(t, err)
	assert.DeepEqual(t, s.String(), strings.Join([]string{
		"postgres-operator.crunchydata.com/cluster=something",
		"postgres-operator.crunchydata.com/instance",
		"postgres-operator.crunchydata.com/role=master",
	}, ","))
}
