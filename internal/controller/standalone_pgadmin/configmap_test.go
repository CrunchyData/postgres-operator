// Copyright 2023 Crunchy Data Solutions, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package standalone_pgadmin

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestGeneratePGAdminConfigMap(t *testing.T) {
	require.ParallelCapacity(t, 0)

	pgadmin := new(v1beta1.PGAdmin)
	pgadmin.Namespace = "some-ns"
	pgadmin.Name = "pg1"

	t.Run("Data,ObjectMeta,TypeMeta", func(t *testing.T) {
		pgadmin := pgadmin.DeepCopy()

		configmap := configmap(pgadmin)

		assert.Assert(t, cmp.MarshalMatches(configmap.TypeMeta, `
apiVersion: v1
kind: ConfigMap
		`))
		assert.Assert(t, cmp.MarshalMatches(configmap.ObjectMeta, `
creationTimestamp: null
labels:
  postgres-operator.crunchydata.com/role: pgadmin
  postgres-operator.crunchydata.com/standalone-pgadmin: pg1
name: pg1-standalone-pgadmin
namespace: some-ns
		`))

		assert.Assert(t, len(configmap.Data) > 0, "expected some configuration")
	})

	t.Run("Annotations,Labels", func(t *testing.T) {
		pgadmin := pgadmin.DeepCopy()
		pgadmin.Spec.Metadata = &v1beta1.Metadata{
			Annotations: map[string]string{"a": "v1", "b": "v2"},
			Labels:      map[string]string{"c": "v3", "d": "v4"},
		}

		configmap := configmap(pgadmin)

		// Annotations present in the metadata.
		assert.DeepEqual(t, configmap.ObjectMeta.Annotations, map[string]string{
			"a": "v1", "b": "v2",
		})

		// Labels present in the metadata.
		assert.DeepEqual(t, configmap.ObjectMeta.Labels, map[string]string{
			"c": "v3", "d": "v4",
			"postgres-operator.crunchydata.com/standalone-pgadmin": "pg1",
			"postgres-operator.crunchydata.com/role":               "pgadmin",
		})
	})
}
