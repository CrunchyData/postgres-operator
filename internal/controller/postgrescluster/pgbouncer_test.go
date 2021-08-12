// +build envtest

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

package postgrescluster

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestGeneratePGBouncerService(t *testing.T) {
	env, cc, _ := setupTestEnv(t, ControllerName)
	t.Cleanup(func() { teardownTestEnv(t, env) })

	reconciler := &Reconciler{Client: cc}

	cluster := &v1beta1.PostgresCluster{}
	cluster.Namespace = "ns5"
	cluster.Name = "pg7"

	t.Run("Unspecified", func(t *testing.T) {
		for _, spec := range []*v1beta1.PostgresProxySpec{
			nil, new(v1beta1.PostgresProxySpec),
		} {
			cluster := cluster.DeepCopy()
			cluster.Spec.Proxy = spec

			service, specified, err := reconciler.generatePGBouncerService(cluster)
			assert.NilError(t, err)
			assert.Assert(t, !specified)

			assert.Assert(t, marshalMatches(service.ObjectMeta, `
creationTimestamp: null
name: pg7-pgbouncer
namespace: ns5
			`))
		}
	})

	cluster.Spec.Proxy = &v1beta1.PostgresProxySpec{
		PGBouncer: &v1beta1.PGBouncerPodSpec{
			Port: initialize.Int32(9651),
		},
	}

	alwaysExpect := func(t testing.TB, service *corev1.Service) {
		assert.Assert(t, marshalMatches(service.TypeMeta, `
apiVersion: v1
kind: Service
		`))
		assert.Assert(t, marshalMatches(service.ObjectMeta, `
creationTimestamp: null
labels:
  postgres-operator.crunchydata.com/cluster: pg7
  postgres-operator.crunchydata.com/role: pgbouncer
name: pg7-pgbouncer
namespace: ns5
ownerReferences:
- apiVersion: postgres-operator.crunchydata.com/v1beta1
  blockOwnerDeletion: true
  controller: true
  kind: PostgresCluster
  name: pg7
  uid: ""
		`))
		assert.Assert(t, marshalMatches(service.Spec.Ports, `
- name: pgbouncer
  port: 9651
  protocol: TCP
  targetPort: pgbouncer
		`))

		// Always gets a ClusterIP (never None).
		assert.Equal(t, service.Spec.ClusterIP, "")
		assert.DeepEqual(t, service.Spec.Selector, map[string]string{
			"postgres-operator.crunchydata.com/cluster": "pg7",
			"postgres-operator.crunchydata.com/role":    "pgbouncer",
		})
	}

	t.Run("AnnotationsLabels", func(t *testing.T) {
		cluster := cluster.DeepCopy()
		cluster.Spec.Metadata = &v1beta1.Metadata{
			Annotations: map[string]string{"a": "v1"},
			Labels:      map[string]string{"b": "v2"},
		}

		service, specified, err := reconciler.generatePGBouncerService(cluster)
		assert.NilError(t, err)
		assert.Assert(t, specified)

		// Annotations present in the metadata.
		assert.DeepEqual(t, service.ObjectMeta.Annotations, map[string]string{
			"a": "v1",
		})

		// Labels present in the metadata.
		assert.DeepEqual(t, service.ObjectMeta.Labels, map[string]string{
			"b": "v2",
			"postgres-operator.crunchydata.com/cluster": "pg7",
			"postgres-operator.crunchydata.com/role":    "pgbouncer",
		})

		// Labels not in the selector.
		assert.DeepEqual(t, service.Spec.Selector, map[string]string{
			"postgres-operator.crunchydata.com/cluster": "pg7",
			"postgres-operator.crunchydata.com/role":    "pgbouncer",
		})
	})

	t.Run("NoServiceSpec", func(t *testing.T) {
		service, specified, err := reconciler.generatePGBouncerService(cluster)
		assert.NilError(t, err)
		assert.Assert(t, specified)
		alwaysExpect(t, service)

		// Defaults to ClusterIP.
		assert.Equal(t, service.Spec.Type, corev1.ServiceTypeClusterIP)
	})

	types := []struct {
		Type   string
		Expect func(testing.TB, *corev1.Service)
	}{
		{Type: "ClusterIP", Expect: func(t testing.TB, service *corev1.Service) {
			assert.Equal(t, service.Spec.Type, corev1.ServiceTypeClusterIP)
		}},
		{Type: "NodePort", Expect: func(t testing.TB, service *corev1.Service) {
			assert.Equal(t, service.Spec.Type, corev1.ServiceTypeNodePort)
		}},
		{Type: "LoadBalancer", Expect: func(t testing.TB, service *corev1.Service) {
			assert.Equal(t, service.Spec.Type, corev1.ServiceTypeLoadBalancer)
		}},
	}

	for _, test := range types {
		t.Run(test.Type, func(t *testing.T) {
			cluster := cluster.DeepCopy()
			cluster.Spec.Proxy.PGBouncer.Service = &v1beta1.ServiceSpec{Type: test.Type}

			service, specified, err := reconciler.generatePGBouncerService(cluster)
			assert.NilError(t, err)
			assert.Assert(t, specified)
			alwaysExpect(t, service)
			test.Expect(t, service)
		})
	}
}

func TestReconcilePGBouncerService(t *testing.T) {
	ctx := context.Background()
	env, cc, _ := setupTestEnv(t, ControllerName)
	t.Cleanup(func() { teardownTestEnv(t, env) })

	ns := &corev1.Namespace{}
	ns.GenerateName = "postgres-operator-test-"
	assert.NilError(t, cc.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, ns)) })

	reconciler := &Reconciler{Client: cc, Owner: client.FieldOwner(t.Name())}

	cluster := &v1beta1.PostgresCluster{}
	cluster.Namespace = ns.Name
	cluster.Name = "pg2"
	cluster.Spec.PostgresVersion = 12
	cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{}}

	assert.NilError(t, cc.Create(ctx, cluster))

	t.Run("Unspecified", func(t *testing.T) {
		cluster := cluster.DeepCopy()
		cluster.Spec.Proxy = nil

		service, err := reconciler.reconcilePGBouncerService(ctx, cluster)
		assert.NilError(t, err)
		assert.Assert(t, service == nil)
	})

	cluster.Spec.Proxy = &v1beta1.PostgresProxySpec{
		PGBouncer: &v1beta1.PGBouncerPodSpec{
			Port: initialize.Int32(19041),
		},
	}

	t.Run("NoServiceSpec", func(t *testing.T) {
		service, err := reconciler.reconcilePGBouncerService(ctx, cluster)
		assert.NilError(t, err)
		assert.Assert(t, service != nil)
		t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, service)) })

		assert.Assert(t, service.Spec.ClusterIP != "",
			"expected to be assigned a ClusterIP")
	})

	serviceTypes := []string{"ClusterIP", "NodePort", "LoadBalancer"}

	// Confirm that each ServiceType can be reconciled.
	for _, serviceType := range serviceTypes {
		t.Run(serviceType, func(t *testing.T) {
			cluster := cluster.DeepCopy()
			cluster.Spec.Proxy.PGBouncer.Service = &v1beta1.ServiceSpec{Type: serviceType}

			service, err := reconciler.reconcilePGBouncerService(ctx, cluster)
			assert.NilError(t, err)
			assert.Assert(t, service != nil)
			t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, service)) })

			assert.Assert(t, service.Spec.ClusterIP != "",
				"expected to be assigned a ClusterIP")
		})
	}

	// CRD validation looks only at the new/incoming value of fields. Confirm
	// that each ServiceType can change to any other ServiceType. Forbidding
	// certain transitions requires a validating webhook.
	for _, beforeType := range serviceTypes {
		for _, changeType := range serviceTypes {
			t.Run(beforeType+"To"+changeType, func(t *testing.T) {
				cluster := cluster.DeepCopy()
				cluster.Spec.Proxy.PGBouncer.Service = &v1beta1.ServiceSpec{Type: beforeType}

				before, err := reconciler.reconcilePGBouncerService(ctx, cluster)
				assert.NilError(t, err)
				t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, before)) })

				cluster.Spec.Proxy.PGBouncer.Service.Type = changeType

				after, err := reconciler.reconcilePGBouncerService(ctx, cluster)
				assert.NilError(t, err)
				assert.Equal(t, after.Spec.ClusterIP, before.Spec.ClusterIP,
					"expected to keep the same ClusterIP")
			})
		}
	}
}
