// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"strconv"
	"testing"

	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestGeneratePGBouncerService(t *testing.T) {
	_, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	reconciler := &Reconciler{
		Client:   cc,
		Recorder: new(record.FakeRecorder),
	}

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

			assert.Assert(t, cmp.MarshalMatches(service.ObjectMeta, `
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
		assert.Assert(t, cmp.MarshalMatches(service.TypeMeta, `
apiVersion: v1
kind: Service
		`))
		assert.Assert(t, cmp.MarshalMatches(service.ObjectMeta, `
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

		// Add metadata to individual service
		cluster.Spec.Proxy.PGBouncer.Service = &v1beta1.ServiceSpec{
			Metadata: &v1beta1.Metadata{
				Annotations: map[string]string{"c": "v3"},
				Labels: map[string]string{"d": "v4",
					"postgres-operator.crunchydata.com/cluster": "wrongName"},
			},
		}

		service, specified, err = reconciler.generatePGBouncerService(cluster)
		assert.NilError(t, err)
		assert.Assert(t, specified)

		// Annotations present in the metadata.
		assert.DeepEqual(t, service.ObjectMeta.Annotations, map[string]string{
			"a": "v1",
			"c": "v3",
		})

		// Labels present in the metadata.
		assert.DeepEqual(t, service.ObjectMeta.Labels, map[string]string{
			"b": "v2",
			"d": "v4",
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
		assert.Assert(t, cmp.MarshalMatches(service.Spec.Ports, `
- name: pgbouncer
  port: 9651
  protocol: TCP
  targetPort: pgbouncer
		`))
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
			assert.Assert(t, cmp.MarshalMatches(service.Spec.Ports, `
- name: pgbouncer
  port: 9651
  protocol: TCP
  targetPort: pgbouncer
		`))
		})
	}

	typesAndPort := []struct {
		Description string
		Type        string
		NodePort    *int32
		Expect      func(testing.TB, *corev1.Service, error)
	}{
		{Description: "ClusterIP with Port 32000", Type: "ClusterIP",
			NodePort: initialize.Int32(32000), Expect: func(t testing.TB, service *corev1.Service, err error) {
				assert.ErrorContains(t, err, "NodePort cannot be set with type ClusterIP on Service \"pg7-pgbouncer\"")
				assert.Assert(t, service == nil)
			}},
		{Description: "NodePort with Port 32001", Type: "NodePort",
			NodePort: initialize.Int32(32001), Expect: func(t testing.TB, service *corev1.Service, err error) {
				assert.NilError(t, err)
				assert.Equal(t, service.Spec.Type, corev1.ServiceTypeNodePort)
				alwaysExpect(t, service)
				assert.Assert(t, cmp.MarshalMatches(service.Spec.Ports, `
- name: pgbouncer
  nodePort: 32001
  port: 9651
  protocol: TCP
  targetPort: pgbouncer
`))
			}},
		{Description: "LoadBalancer with Port 32002", Type: "LoadBalancer",
			NodePort: initialize.Int32(32002), Expect: func(t testing.TB, service *corev1.Service, err error) {
				assert.NilError(t, err)
				assert.Equal(t, service.Spec.Type, corev1.ServiceTypeLoadBalancer)
				alwaysExpect(t, service)
				assert.Assert(t, cmp.MarshalMatches(service.Spec.Ports, `
- name: pgbouncer
  nodePort: 32002
  port: 9651
  protocol: TCP
  targetPort: pgbouncer
`))
			}},
	}

	for _, test := range typesAndPort {
		t.Run(test.Type, func(t *testing.T) {
			cluster := cluster.DeepCopy()
			cluster.Spec.Proxy.PGBouncer.Service = &v1beta1.ServiceSpec{Type: test.Type, NodePort: test.NodePort}

			service, specified, err := reconciler.generatePGBouncerService(cluster)
			test.Expect(t, service, err)
			// whether or not an error is encountered, 'specified' is true because
			// the service *should* exist
			assert.Assert(t, specified)
		})
	}
}

func TestReconcilePGBouncerService(t *testing.T) {
	ctx := context.Background()
	_, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 1)

	reconciler := &Reconciler{Client: cc, Owner: client.FieldOwner(t.Name())}

	cluster := testCluster()
	cluster.Namespace = setupNamespace(t, cc).Name
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
	serviceTypeChangeClusterCounter := 0
	for _, beforeType := range serviceTypes {
		for _, changeType := range serviceTypes {
			t.Run(beforeType+"To"+changeType, func(t *testing.T) {
				// Creating fresh clusters for these tests
				clusterNamespace := cluster.Namespace
				cluster := testCluster()
				cluster.Namespace = clusterNamespace

				// Note (dsessler): Adding a number to each cluster name to make cluster/service
				// names unique to work around an intermittent race condition where a service
				// from a prior test has not been deleted yet when the next test runs, causing
				// the test to fail due to non-matching IP addresses.
				cluster.Name += "-" + strconv.Itoa(serviceTypeChangeClusterCounter)
				assert.NilError(t, cc.Create(ctx, cluster))

				cluster.Spec.Proxy = &v1beta1.PostgresProxySpec{
					PGBouncer: &v1beta1.PGBouncerPodSpec{
						Port: initialize.Int32(19041),
					},
				}
				cluster.Spec.Proxy.PGBouncer.Service = &v1beta1.ServiceSpec{Type: beforeType}

				before, err := reconciler.reconcilePGBouncerService(ctx, cluster)
				assert.NilError(t, err)
				t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, before)) })

				cluster.Spec.Proxy.PGBouncer.Service.Type = changeType

				after, err := reconciler.reconcilePGBouncerService(ctx, cluster)

				// LoadBalancers are provisioned by a separate controller that
				// updates the Service soon after creation. The API may return
				// a conflict error when we race to update it, even though we
				// don't send a resourceVersion in our payload. Retry.
				if apierrors.IsConflict(err) {
					t.Log("conflict:", err)
					after, err = reconciler.reconcilePGBouncerService(ctx, cluster)
				}

				assert.NilError(t, err, "\n%#v", errors.Unwrap(err))
				assert.Equal(t, after.Spec.ClusterIP, before.Spec.ClusterIP,
					"expected to keep the same ClusterIP")
				serviceTypeChangeClusterCounter++
			})
		}
	}
}

func TestGeneratePGBouncerDeployment(t *testing.T) {
	_, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	ctx := context.Background()
	reconciler := &Reconciler{Client: cc}

	cluster := &v1beta1.PostgresCluster{}
	cluster.Namespace = "ns3"
	cluster.Name = "test-cluster"

	t.Run("Unspecified", func(t *testing.T) {
		for _, spec := range []*v1beta1.PostgresProxySpec{
			nil, new(v1beta1.PostgresProxySpec),
		} {
			cluster := cluster.DeepCopy()
			cluster.Spec.Proxy = spec

			deploy, specified, err := reconciler.generatePGBouncerDeployment(ctx, cluster, nil, nil, nil)
			assert.NilError(t, err)
			assert.Assert(t, !specified)

			assert.Assert(t, cmp.MarshalMatches(deploy.ObjectMeta, `
creationTimestamp: null
name: test-cluster-pgbouncer
namespace: ns3
			`))
		}
	})

	cluster.Spec.Proxy = &v1beta1.PostgresProxySpec{
		PGBouncer: &v1beta1.PGBouncerPodSpec{},
	}
	cluster.Default()

	configmap := &corev1.ConfigMap{}
	configmap.Name = "some-cm2"

	secret := &corev1.Secret{}
	secret.Name = "some-secret3"

	primary := &corev1.SecretProjection{}

	t.Run("AnnotationsLabels", func(t *testing.T) {
		cluster := cluster.DeepCopy()
		cluster.Spec.Metadata = &v1beta1.Metadata{
			Annotations: map[string]string{"a": "v1"},
			Labels:      map[string]string{"b": "v2"},
		}

		deploy, specified, err := reconciler.generatePGBouncerDeployment(
			ctx, cluster, primary, configmap, secret)
		assert.NilError(t, err)
		assert.Assert(t, specified)

		// Annotations present in the metadata.
		assert.DeepEqual(t, deploy.ObjectMeta.Annotations, map[string]string{
			"a": "v1",
		})

		// Labels present in the metadata.
		assert.DeepEqual(t, deploy.ObjectMeta.Labels, map[string]string{
			"b": "v2",
			"postgres-operator.crunchydata.com/cluster": "test-cluster",
			"postgres-operator.crunchydata.com/role":    "pgbouncer",
		})

		// Labels not in the pod selector.
		assert.DeepEqual(t, deploy.Spec.Selector,
			&metav1.LabelSelector{
				MatchLabels: map[string]string{
					"postgres-operator.crunchydata.com/cluster": "test-cluster",
					"postgres-operator.crunchydata.com/role":    "pgbouncer",
				},
			})

		// Annotations present in the pod template.
		assert.DeepEqual(t, deploy.Spec.Template.Annotations, map[string]string{
			"a": "v1",
		})

		// Labels present in the pod template.
		assert.DeepEqual(t, deploy.Spec.Template.Labels, map[string]string{
			"b": "v2",
			"postgres-operator.crunchydata.com/cluster": "test-cluster",
			"postgres-operator.crunchydata.com/role":    "pgbouncer",
		})
	})

	t.Run("PodSpec", func(t *testing.T) {
		deploy, specified, err := reconciler.generatePGBouncerDeployment(
			ctx, cluster, primary, configmap, secret)
		assert.NilError(t, err)
		assert.Assert(t, specified)

		// Containers and Volumes should be populated.
		assert.Assert(t, len(deploy.Spec.Template.Spec.Containers) != 0)
		assert.Assert(t, len(deploy.Spec.Template.Spec.Volumes) != 0)

		// Ignore Containers and Volumes in the comparison below.
		deploy.Spec.Template.Spec.Containers = nil
		deploy.Spec.Template.Spec.Volumes = nil

		// TODO(tjmoore4): Add additional tests to test appending existing
		// topology spread constraints and spec.disableDefaultPodScheduling being
		// set to true (as done in instance StatefulSet tests).

		assert.Assert(t, cmp.MarshalMatches(deploy.Spec.Template.Spec, `
automountServiceAccountToken: false
containers: null
enableServiceLinks: false
restartPolicy: Always
securityContext:
  fsGroupChangePolicy: OnRootMismatch
shareProcessNamespace: true
topologySpreadConstraints:
- labelSelector:
    matchLabels:
      postgres-operator.crunchydata.com/cluster: test-cluster
      postgres-operator.crunchydata.com/role: pgbouncer
  maxSkew: 1
  topologyKey: kubernetes.io/hostname
  whenUnsatisfiable: ScheduleAnyway
- labelSelector:
    matchLabels:
      postgres-operator.crunchydata.com/cluster: test-cluster
      postgres-operator.crunchydata.com/role: pgbouncer
  maxSkew: 1
  topologyKey: topology.kubernetes.io/zone
  whenUnsatisfiable: ScheduleAnyway
		`))

		t.Run("DisableDefaultPodScheduling", func(t *testing.T) {
			cluster := cluster.DeepCopy()
			cluster.Spec.DisableDefaultPodScheduling = initialize.Bool(true)

			deploy, specified, err := reconciler.generatePGBouncerDeployment(
				ctx, cluster, primary, configmap, secret)
			assert.NilError(t, err)
			assert.Assert(t, specified)

			assert.Assert(t, deploy.Spec.Template.Spec.TopologySpreadConstraints == nil)
		})
	})
}

func TestReconcilePGBouncerDisruptionBudget(t *testing.T) {
	ctx := context.Background()
	_, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	r := &Reconciler{
		Client: cc,
		Owner:  client.FieldOwner(t.Name()),
	}

	foundPDB := func(
		cluster *v1beta1.PostgresCluster,
	) bool {
		got := &policyv1.PodDisruptionBudget{}
		err := r.Client.Get(ctx,
			naming.AsObjectKey(naming.ClusterPGBouncer(cluster)),
			got)
		return !apierrors.IsNotFound(err)
	}

	ns := setupNamespace(t, cc)

	t.Run("empty", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns.Name
		cluster.Spec.Proxy = nil

		assert.NilError(t, r.reconcilePGBouncerPodDisruptionBudget(ctx, cluster))
	})

	t.Run("no replicas in spec", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns.Name
		cluster.Spec.Proxy.PGBouncer.Replicas = nil
		assert.Error(t, r.reconcilePGBouncerPodDisruptionBudget(ctx, cluster),
			"Replicas should be defined")
	})

	t.Run("not created", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns.Name
		cluster.Spec.Proxy.PGBouncer.Replicas = initialize.Int32(1)
		cluster.Spec.Proxy.PGBouncer.MinAvailable = initialize.IntOrStringInt32(0)
		assert.NilError(t, r.reconcilePGBouncerPodDisruptionBudget(ctx, cluster))
		assert.Assert(t, !foundPDB(cluster))
	})

	t.Run("int created", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns.Name
		cluster.Spec.Proxy.PGBouncer.Replicas = initialize.Int32(1)
		cluster.Spec.Proxy.PGBouncer.MinAvailable = initialize.IntOrStringInt32(1)

		assert.NilError(t, r.Client.Create(ctx, cluster))
		t.Cleanup(func() { assert.Check(t, r.Client.Delete(ctx, cluster)) })

		assert.NilError(t, r.reconcilePGBouncerPodDisruptionBudget(ctx, cluster))
		assert.Assert(t, foundPDB(cluster))

		t.Run("deleted", func(t *testing.T) {
			cluster.Spec.Proxy.PGBouncer.MinAvailable = initialize.IntOrStringInt32(0)
			err := r.reconcilePGBouncerPodDisruptionBudget(ctx, cluster)
			if apierrors.IsConflict(err) {
				// When running in an existing environment another controller will sometimes update
				// the object. This leads to an error where the ResourceVersion of the object does
				// not match what we expect. When we run into this conflict, try to reconcile the
				// object again.
				err = r.reconcilePGBouncerPodDisruptionBudget(ctx, cluster)
			}
			assert.NilError(t, err, errors.Unwrap(err))
			assert.Assert(t, !foundPDB(cluster))
		})
	})

	t.Run("str created", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns.Name
		cluster.Spec.Proxy.PGBouncer.Replicas = initialize.Int32(1)
		cluster.Spec.Proxy.PGBouncer.MinAvailable = initialize.IntOrStringString("50%")

		assert.NilError(t, r.Client.Create(ctx, cluster))
		t.Cleanup(func() { assert.Check(t, r.Client.Delete(ctx, cluster)) })

		assert.NilError(t, r.reconcilePGBouncerPodDisruptionBudget(ctx, cluster))
		assert.Assert(t, foundPDB(cluster))

		t.Run("deleted", func(t *testing.T) {
			cluster.Spec.Proxy.PGBouncer.MinAvailable = initialize.IntOrStringString("0%")
			err := r.reconcilePGBouncerPodDisruptionBudget(ctx, cluster)
			if apierrors.IsConflict(err) {
				// When running in an existing environment another controller will sometimes update
				// the object. This leads to an error where the ResourceVersion of the object does
				// not match what we expect. When we run into this conflict, try to reconcile the
				// object again.
				err = r.reconcilePGBouncerPodDisruptionBudget(ctx, cluster)
			}
			assert.NilError(t, err, errors.Unwrap(err))
			assert.Assert(t, !foundPDB(cluster))
		})

		t.Run("delete with 00%", func(t *testing.T) {
			cluster.Spec.Proxy.PGBouncer.MinAvailable = initialize.IntOrStringString("50%")

			assert.NilError(t, r.reconcilePGBouncerPodDisruptionBudget(ctx, cluster))
			assert.Assert(t, foundPDB(cluster))

			t.Run("deleted", func(t *testing.T) {
				cluster.Spec.Proxy.PGBouncer.MinAvailable = initialize.IntOrStringString("00%")
				err := r.reconcilePGBouncerPodDisruptionBudget(ctx, cluster)
				if apierrors.IsConflict(err) {
					// When running in an existing environment another controller will sometimes update
					// the object. This leads to an error where the ResourceVersion of the object does
					// not match what we expect. When we run into this conflict, try to reconcile the
					// object again.
					err = r.reconcilePGBouncerPodDisruptionBudget(ctx, cluster)
				}
				assert.NilError(t, err, errors.Unwrap(err))
				assert.Assert(t, !foundPDB(cluster))
			})
		})
	})
}
