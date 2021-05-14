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

package postgres

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestConfigDirectory(t *testing.T) {
	cluster := new(v1beta1.PostgresCluster)
	cluster.Spec.PostgresVersion = 11

	assert.Equal(t, ConfigDirectory(cluster), "/pgdata/pg11")
}

func TestDataDirectory(t *testing.T) {
	cluster := new(v1beta1.PostgresCluster)
	cluster.Spec.PostgresVersion = 12

	assert.Equal(t, DataDirectory(cluster), "/pgdata/pg12")
}
