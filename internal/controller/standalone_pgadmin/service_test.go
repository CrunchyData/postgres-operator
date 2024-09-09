// Copyright 2023 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package standalone_pgadmin

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestService(t *testing.T) {
	pgadmin := new(v1beta1.PGAdmin)
	pgadmin.Name = "daisy"
	pgadmin.Namespace = "daisy-service-ns"
	pgadmin.Spec.ServiceName = "daisy-service"
	pgadmin.Spec.Metadata = &v1beta1.Metadata{
		Labels: map[string]string{
			"test-label": "test-label-val",
			"postgres-operator.crunchydata.com/pgadmin": "bad-val",
			"postgres-operator.crunchydata.com/role":    "bad-val",
		},
		Annotations: map[string]string{
			"test-annotation": "test-annotation-val",
		},
	}

	service := service(pgadmin)
	assert.Assert(t, service != nil)
	assert.Assert(t, cmp.MarshalMatches(service.TypeMeta, `
apiVersion: v1
kind: Service
	`))

	assert.Assert(t, cmp.MarshalMatches(service.ObjectMeta, `
annotations:
  test-annotation: test-annotation-val
creationTimestamp: null
labels:
  postgres-operator.crunchydata.com/pgadmin: daisy
  postgres-operator.crunchydata.com/role: pgadmin
  test-label: test-label-val
name: daisy-service
namespace: daisy-service-ns
	`))

	assert.Assert(t, cmp.MarshalMatches(service.Spec, `
ports:
- name: pgadmin-port
  port: 5050
  protocol: TCP
  targetPort: 5050
selector:
  postgres-operator.crunchydata.com/pgadmin: daisy
type: ClusterIP
	`))
}
