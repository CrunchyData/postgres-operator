// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"io"
	"strconv"
	"testing"

	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestGeneratePGAdminConfigMap(t *testing.T) {
	_, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	reconciler := &Reconciler{Client: cc}

	cluster := &v1beta1.PostgresCluster{}
	cluster.Namespace = "some-ns"
	cluster.Name = "pg1"

	t.Run("Unspecified", func(t *testing.T) {
		for _, spec := range []*v1beta1.UserInterfaceSpec{
			nil, new(v1beta1.UserInterfaceSpec),
		} {
			cluster := cluster.DeepCopy()
			cluster.Spec.UserInterface = spec

			configmap, specified, err := reconciler.generatePGAdminConfigMap(cluster)
			assert.NilError(t, err)
			assert.Assert(t, !specified)

			assert.Equal(t, configmap.Namespace, cluster.Namespace)
			assert.Equal(t, configmap.Name, "pg1-pgadmin")
		}
	})

	cluster.Spec.UserInterface = &v1beta1.UserInterfaceSpec{
		PGAdmin: &v1beta1.PGAdminPodSpec{},
	}

	t.Run("Data,ObjectMeta,TypeMeta", func(t *testing.T) {
		cluster := cluster.DeepCopy()

		configmap, specified, err := reconciler.generatePGAdminConfigMap(cluster)
		assert.NilError(t, err)
		assert.Assert(t, specified)

		assert.Assert(t, cmp.MarshalMatches(configmap.TypeMeta, `
apiVersion: v1
kind: ConfigMap
		`))
		assert.Assert(t, cmp.MarshalMatches(configmap.ObjectMeta, `
creationTimestamp: null
labels:
  postgres-operator.crunchydata.com/cluster: pg1
  postgres-operator.crunchydata.com/role: pgadmin
name: pg1-pgadmin
namespace: some-ns
ownerReferences:
- apiVersion: postgres-operator.crunchydata.com/v1beta1
  blockOwnerDeletion: true
  controller: true
  kind: PostgresCluster
  name: pg1
  uid: ""
		`))

		assert.Assert(t, len(configmap.Data) > 0, "expected some configuration")
	})

	t.Run("Annotations,Labels", func(t *testing.T) {
		cluster := cluster.DeepCopy()
		cluster.Spec.Metadata = &v1beta1.Metadata{
			Annotations: map[string]string{"a": "v1", "b": "v2"},
			Labels:      map[string]string{"c": "v3", "d": "v4"},
		}
		cluster.Spec.UserInterface.PGAdmin.Metadata = &v1beta1.Metadata{
			Annotations: map[string]string{"a": "v5", "e": "v6"},
			Labels:      map[string]string{"c": "v7", "f": "v8"},
		}

		configmap, specified, err := reconciler.generatePGAdminConfigMap(cluster)
		assert.NilError(t, err)
		assert.Assert(t, specified)

		// Annotations present in the metadata.
		assert.DeepEqual(t, configmap.ObjectMeta.Annotations, map[string]string{
			"a": "v5", "b": "v2", "e": "v6",
		})

		// Labels present in the metadata.
		assert.DeepEqual(t, configmap.ObjectMeta.Labels, map[string]string{
			"c": "v7", "d": "v4", "f": "v8",
			"postgres-operator.crunchydata.com/cluster": "pg1",
			"postgres-operator.crunchydata.com/role":    "pgadmin",
		})
	})
}

func TestGeneratePGAdminService(t *testing.T) {
	_, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	reconciler := &Reconciler{
		Client:   cc,
		Recorder: new(record.FakeRecorder),
	}

	cluster := &v1beta1.PostgresCluster{}
	cluster.Namespace = "my-ns"
	cluster.Name = "my-cluster"

	t.Run("Unspecified", func(t *testing.T) {
		for _, spec := range []*v1beta1.UserInterfaceSpec{
			nil, new(v1beta1.UserInterfaceSpec),
		} {
			cluster := cluster.DeepCopy()
			cluster.Spec.UserInterface = spec

			service, specified, err := reconciler.generatePGAdminService(cluster)
			assert.NilError(t, err)
			assert.Assert(t, !specified)

			assert.Assert(t, cmp.MarshalMatches(service.ObjectMeta, `
creationTimestamp: null
name: my-cluster-pgadmin
namespace: my-ns
			`))
		}
	})

	cluster.Spec.UserInterface = &v1beta1.UserInterfaceSpec{
		PGAdmin: &v1beta1.PGAdminPodSpec{},
	}

	alwaysExpect := func(t testing.TB, service *corev1.Service) {
		assert.Assert(t, cmp.MarshalMatches(service.TypeMeta, `
apiVersion: v1
kind: Service
		`))
		assert.Assert(t, cmp.MarshalMatches(service.ObjectMeta, `
creationTimestamp: null
labels:
  postgres-operator.crunchydata.com/cluster: my-cluster
  postgres-operator.crunchydata.com/role: pgadmin
name: my-cluster-pgadmin
namespace: my-ns
ownerReferences:
- apiVersion: postgres-operator.crunchydata.com/v1beta1
  blockOwnerDeletion: true
  controller: true
  kind: PostgresCluster
  name: my-cluster
  uid: ""
		`))

		// Always gets a ClusterIP (never None).
		assert.Equal(t, service.Spec.ClusterIP, "")
		assert.DeepEqual(t, service.Spec.Selector, map[string]string{
			"postgres-operator.crunchydata.com/cluster": "my-cluster",
			"postgres-operator.crunchydata.com/role":    "pgadmin",
		})
	}

	t.Run("AnnotationsLabels", func(t *testing.T) {
		cluster := cluster.DeepCopy()
		cluster.Spec.Metadata = &v1beta1.Metadata{
			Annotations: map[string]string{"a": "v1"},
			Labels:      map[string]string{"b": "v2"},
		}

		service, specified, err := reconciler.generatePGAdminService(cluster)
		assert.NilError(t, err)
		assert.Assert(t, specified)

		// Annotations present in the metadata.
		assert.DeepEqual(t, service.ObjectMeta.Annotations, map[string]string{
			"a": "v1",
		})

		// Labels present in the metadata.
		assert.DeepEqual(t, service.ObjectMeta.Labels, map[string]string{
			"b": "v2",
			"postgres-operator.crunchydata.com/cluster": "my-cluster",
			"postgres-operator.crunchydata.com/role":    "pgadmin",
		})

		// Labels not in the selector.
		assert.DeepEqual(t, service.Spec.Selector, map[string]string{
			"postgres-operator.crunchydata.com/cluster": "my-cluster",
			"postgres-operator.crunchydata.com/role":    "pgadmin",
		})

		// Add metadata to individual service
		cluster.Spec.UserInterface.PGAdmin.Service = &v1beta1.ServiceSpec{
			Metadata: &v1beta1.Metadata{
				Annotations: map[string]string{"c": "v3"},
				Labels: map[string]string{"d": "v4",
					"postgres-operator.crunchydata.com/cluster": "wrongName"},
			},
		}

		service, specified, err = reconciler.generatePGAdminService(cluster)
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
			"postgres-operator.crunchydata.com/cluster": "my-cluster",
			"postgres-operator.crunchydata.com/role":    "pgadmin",
		})

		// Labels not in the selector.
		assert.DeepEqual(t, service.Spec.Selector, map[string]string{
			"postgres-operator.crunchydata.com/cluster": "my-cluster",
			"postgres-operator.crunchydata.com/role":    "pgadmin",
		})
	})

	t.Run("NoServiceSpec", func(t *testing.T) {
		service, specified, err := reconciler.generatePGAdminService(cluster)
		assert.NilError(t, err)
		assert.Assert(t, specified)
		alwaysExpect(t, service)
		// Defaults to ClusterIP.
		assert.Equal(t, service.Spec.Type, corev1.ServiceTypeClusterIP)
		assert.Assert(t, cmp.MarshalMatches(service.Spec.Ports, `
- name: pgadmin
  port: 5050
  protocol: TCP
  targetPort: pgadmin
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
			cluster.Spec.UserInterface.PGAdmin.Service = &v1beta1.ServiceSpec{Type: test.Type}

			service, specified, err := reconciler.generatePGAdminService(cluster)
			assert.NilError(t, err)
			assert.Assert(t, specified)
			alwaysExpect(t, service)
			test.Expect(t, service)
			assert.Assert(t, cmp.MarshalMatches(service.Spec.Ports, `
- name: pgadmin
  port: 5050
  protocol: TCP
  targetPort: pgadmin
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
				assert.ErrorContains(t, err, "NodePort cannot be set with type ClusterIP on Service \"my-cluster-pgadmin\"")
				assert.Assert(t, service == nil)
			}},
		{Description: "NodePort with Port 32001", Type: "NodePort",
			NodePort: initialize.Int32(32001), Expect: func(t testing.TB, service *corev1.Service, err error) {
				assert.NilError(t, err)
				assert.Equal(t, service.Spec.Type, corev1.ServiceTypeNodePort)
				alwaysExpect(t, service)
				assert.Assert(t, cmp.MarshalMatches(service.Spec.Ports, `
- name: pgadmin
  nodePort: 32001
  port: 5050
  protocol: TCP
  targetPort: pgadmin
`))
			}},
		{Description: "LoadBalancer with Port 32002", Type: "LoadBalancer",
			NodePort: initialize.Int32(32002), Expect: func(t testing.TB, service *corev1.Service, err error) {
				assert.NilError(t, err)
				assert.Equal(t, service.Spec.Type, corev1.ServiceTypeLoadBalancer)
				alwaysExpect(t, service)
				assert.Assert(t, cmp.MarshalMatches(service.Spec.Ports, `
- name: pgadmin
  nodePort: 32002
  port: 5050
  protocol: TCP
  targetPort: pgadmin
`))
			}},
	}

	for _, test := range typesAndPort {
		t.Run(test.Description, func(t *testing.T) {
			cluster := cluster.DeepCopy()
			cluster.Spec.UserInterface.PGAdmin.Service =
				&v1beta1.ServiceSpec{Type: test.Type, NodePort: test.NodePort}

			service, specified, err := reconciler.generatePGAdminService(cluster)
			test.Expect(t, service, err)
			// whether or not an error is encountered, 'specified' is true because
			// the service *should* exist
			assert.Assert(t, specified)

		})
	}
}

func TestReconcilePGAdminService(t *testing.T) {
	ctx := context.Background()
	_, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 1)

	reconciler := &Reconciler{Client: cc, Owner: client.FieldOwner(t.Name())}

	cluster := testCluster()
	cluster.Namespace = setupNamespace(t, cc).Name
	assert.NilError(t, cc.Create(ctx, cluster))

	t.Run("Unspecified", func(t *testing.T) {
		cluster := cluster.DeepCopy()
		cluster.Spec.UserInterface = nil

		service, err := reconciler.reconcilePGAdminService(ctx, cluster)
		assert.NilError(t, err)
		assert.Assert(t, service == nil)
	})

	cluster.Spec.UserInterface = &v1beta1.UserInterfaceSpec{
		PGAdmin: &v1beta1.PGAdminPodSpec{},
	}

	t.Run("NoServiceSpec", func(t *testing.T) {
		service, err := reconciler.reconcilePGAdminService(ctx, cluster)
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
			cluster.Spec.UserInterface.PGAdmin.Service = &v1beta1.ServiceSpec{Type: serviceType}

			service, err := reconciler.reconcilePGAdminService(ctx, cluster)
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

				cluster.Spec.UserInterface = &v1beta1.UserInterfaceSpec{
					PGAdmin: &v1beta1.PGAdminPodSpec{},
				}
				cluster.Spec.UserInterface.PGAdmin.Service = &v1beta1.ServiceSpec{Type: beforeType}

				before, err := reconciler.reconcilePGAdminService(ctx, cluster)
				assert.NilError(t, err)
				t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, before)) })

				cluster.Spec.UserInterface.PGAdmin.Service.Type = changeType

				after, err := reconciler.reconcilePGAdminService(ctx, cluster)

				// LoadBalancers are provisioned by a separate controller that
				// updates the Service soon after creation. The API may return
				// a conflict error when we race to update it, even though we
				// don't send a resourceVersion in our payload. Retry.
				if apierrors.IsConflict(err) {
					t.Log("conflict:", err)
					after, err = reconciler.reconcilePGAdminService(ctx, cluster)
				}

				assert.NilError(t, err, "\n%#v", errors.Unwrap(err))
				assert.Equal(t, after.Spec.ClusterIP, before.Spec.ClusterIP,
					"expected to keep the same ClusterIP")
				serviceTypeChangeClusterCounter++
			})
		}
	}
}

func TestReconcilePGAdminStatefulSet(t *testing.T) {
	ctx := context.Background()
	_, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 1)

	reconciler := &Reconciler{Client: cc, Owner: client.FieldOwner(t.Name())}

	ns := setupNamespace(t, cc)
	cluster := pgAdminTestCluster(*ns)

	assert.NilError(t, cc.Create(ctx, cluster))
	t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, cluster)) })

	configmap := &corev1.ConfigMap{}
	configmap.Name = "test-cm"

	pvc := &corev1.PersistentVolumeClaim{}
	pvc.Name = "test-pvc"

	t.Run("verify StatefulSet", func(t *testing.T) {
		err := reconciler.reconcilePGAdminStatefulSet(ctx, cluster, configmap, pvc)
		assert.NilError(t, err)

		selector, err := naming.AsSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{
				naming.LabelCluster: cluster.Name,
			},
		})
		assert.NilError(t, err)

		list := appsv1.StatefulSetList{}
		assert.NilError(t, cc.List(ctx, &list, client.InNamespace(cluster.Namespace),
			client.MatchingLabelsSelector{Selector: selector}))
		assert.Equal(t, len(list.Items), 1)
		assert.Equal(t, list.Items[0].Spec.ServiceName, "test-cluster-pods")

		template := list.Items[0].Spec.Template.DeepCopy()

		// Containers and Volumes should be populated.
		assert.Assert(t, len(template.Spec.Containers) != 0)
		assert.Assert(t, len(template.Spec.InitContainers) != 0)
		assert.Assert(t, len(template.Spec.Volumes) != 0)

		// Ignore Containers and Volumes in the comparison below.
		template.Spec.Containers = nil
		template.Spec.InitContainers = nil
		template.Spec.Volumes = nil

		assert.Assert(t, cmp.MarshalMatches(template.ObjectMeta, `
creationTimestamp: null
labels:
  postgres-operator.crunchydata.com/cluster: test-cluster
  postgres-operator.crunchydata.com/data: pgadmin
  postgres-operator.crunchydata.com/role: pgadmin
		`))

		compare := `
automountServiceAccountToken: false
containers: null
dnsPolicy: ClusterFirst
enableServiceLinks: false
restartPolicy: Always
schedulerName: default-scheduler
securityContext:
  fsGroup: 26
  fsGroupChangePolicy: OnRootMismatch
terminationGracePeriodSeconds: 30
		`

		assert.Assert(t, cmp.MarshalMatches(template.Spec, compare))
	})

	t.Run("verify customized deployment", func(t *testing.T) {

		customcluster := pgAdminTestCluster(*ns)

		// add pod level customizations
		customcluster.Name = "custom-cluster"

		// annotation and label
		customcluster.Spec.UserInterface.PGAdmin.Metadata = &v1beta1.Metadata{
			Annotations: map[string]string{
				"annotation1": "annotationvalue",
			},
			Labels: map[string]string{
				"label1": "labelvalue",
			},
		}

		// scheduling constraints
		customcluster.Spec.UserInterface.PGAdmin.Affinity = &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{{
						MatchExpressions: []corev1.NodeSelectorRequirement{{
							Key:      "key",
							Operator: "Exists",
						}},
					}},
				},
			},
		}
		customcluster.Spec.UserInterface.PGAdmin.Tolerations = []corev1.Toleration{
			{Key: "sometoleration"},
		}

		if cluster.Spec.UserInterface.PGAdmin.PriorityClassName != nil {
			customcluster.Spec.UserInterface.PGAdmin.PriorityClassName = initialize.String("testpriorityclass")
		}

		customcluster.Spec.UserInterface.PGAdmin.TopologySpreadConstraints = []corev1.TopologySpreadConstraint{
			{
				MaxSkew:           int32(1),
				TopologyKey:       "fakekey",
				WhenUnsatisfiable: corev1.ScheduleAnyway,
				LabelSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{Key: naming.LabelCluster, Operator: "In", Values: []string{"somename"}},
						{Key: naming.LabelData, Operator: "Exists"},
					},
				},
			},
		}

		// set an image pull secret
		customcluster.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{
			Name: "myImagePullSecret"}}

		assert.NilError(t, cc.Create(ctx, customcluster))
		t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, customcluster)) })

		err := reconciler.reconcilePGAdminStatefulSet(ctx, customcluster, configmap, pvc)
		assert.NilError(t, err)

		selector, err := naming.AsSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{
				naming.LabelCluster: customcluster.Name,
			},
		})
		assert.NilError(t, err)

		list := appsv1.StatefulSetList{}
		assert.NilError(t, cc.List(ctx, &list, client.InNamespace(customcluster.Namespace),
			client.MatchingLabelsSelector{Selector: selector}))
		assert.Equal(t, len(list.Items), 1)
		assert.Equal(t, list.Items[0].Spec.ServiceName, "custom-cluster-pods")

		template := list.Items[0].Spec.Template.DeepCopy()

		// Containers and Volumes should be populated.
		assert.Assert(t, len(template.Spec.Containers) != 0)
		assert.Assert(t, len(template.Spec.InitContainers) != 0)
		assert.Assert(t, len(template.Spec.Volumes) != 0)

		// Ignore Containers and Volumes in the comparison below.
		template.Spec.Containers = nil
		template.Spec.InitContainers = nil
		template.Spec.Volumes = nil

		assert.Assert(t, cmp.MarshalMatches(template.ObjectMeta, `
annotations:
  annotation1: annotationvalue
creationTimestamp: null
labels:
  label1: labelvalue
  postgres-operator.crunchydata.com/cluster: custom-cluster
  postgres-operator.crunchydata.com/data: pgadmin
  postgres-operator.crunchydata.com/role: pgadmin
		`))

		compare := `
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
      - matchExpressions:
        - key: key
          operator: Exists
automountServiceAccountToken: false
containers: null
dnsPolicy: ClusterFirst
enableServiceLinks: false
imagePullSecrets:
- name: myImagePullSecret
restartPolicy: Always
schedulerName: default-scheduler
securityContext:
  fsGroup: 26
  fsGroupChangePolicy: OnRootMismatch
terminationGracePeriodSeconds: 30
tolerations:
- key: sometoleration
topologySpreadConstraints:
- labelSelector:
    matchExpressions:
    - key: postgres-operator.crunchydata.com/cluster
      operator: In
      values:
      - somename
    - key: postgres-operator.crunchydata.com/data
      operator: Exists
  maxSkew: 1
  topologyKey: fakekey
  whenUnsatisfiable: ScheduleAnyway
`

		assert.Assert(t, cmp.MarshalMatches(template.Spec, compare))
	})
}

func TestReconcilePGAdminDataVolume(t *testing.T) {
	ctx := context.Background()
	_, tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 1)

	reconciler := &Reconciler{
		Client: tClient,
		Owner:  client.FieldOwner(t.Name()),
	}

	ns := setupNamespace(t, tClient)
	cluster := pgAdminTestCluster(*ns)

	assert.NilError(t, tClient.Create(ctx, cluster))
	t.Cleanup(func() { assert.Check(t, tClient.Delete(ctx, cluster)) })

	t.Run("DataVolume", func(t *testing.T) {
		pvc, err := reconciler.reconcilePGAdminDataVolume(ctx, cluster)
		assert.NilError(t, err)

		assert.Assert(t, metav1.IsControlledBy(pvc, cluster))

		assert.Equal(t, pvc.Labels[naming.LabelCluster], cluster.Name)
		assert.Equal(t, pvc.Labels[naming.LabelRole], naming.RolePGAdmin)
		assert.Equal(t, pvc.Labels[naming.LabelData], naming.DataPGAdmin)

		assert.Assert(t, cmp.MarshalMatches(pvc.Spec, `
accessModes:
- ReadWriteOnce
resources:
  requests:
    storage: 1Gi
storageClassName: storage-class-for-data
volumeMode: Filesystem
		`))
	})
}

func TestReconcilePGAdminUsers(t *testing.T) {
	ctx := context.Background()

	t.Run("Disabled", func(t *testing.T) {
		r := new(Reconciler)
		cluster := new(v1beta1.PostgresCluster)
		assert.NilError(t, r.reconcilePGAdminUsers(ctx, cluster, nil, nil))
	})

	// pgAdmin enabled
	cluster := &v1beta1.PostgresCluster{}
	cluster.Namespace = "ns1"
	cluster.Name = "pgc1"
	cluster.Spec.Port = initialize.Int32(5432)
	cluster.Spec.UserInterface =
		&v1beta1.UserInterfaceSpec{PGAdmin: &v1beta1.PGAdminPodSpec{}}

	t.Run("NoPods", func(t *testing.T) {
		r := new(Reconciler)
		r.Client = fake.NewClientBuilder().Build()
		assert.NilError(t, r.reconcilePGAdminUsers(ctx, cluster, nil, nil))
	})

	// Pod in the namespace
	pod := corev1.Pod{}
	pod.Namespace = cluster.Namespace
	pod.Name = cluster.Name + "-pgadmin-0"

	t.Run("ContainerNotRunning", func(t *testing.T) {
		pod := pod.DeepCopy()

		pod.DeletionTimestamp = nil
		pod.Status.ContainerStatuses = nil

		r := new(Reconciler)
		r.Client = fake.NewClientBuilder().WithObjects(pod).Build()

		assert.NilError(t, r.reconcilePGAdminUsers(ctx, cluster, nil, nil))
	})

	t.Run("PodTerminating", func(t *testing.T) {
		pod := pod.DeepCopy()

		// Must add finalizer when adding deletion timestamp otherwise fake client will panic:
		// https://github.com/kubernetes-sigs/controller-runtime/pull/2316
		pod.Finalizers = append(pod.Finalizers, "some-finalizer")

		pod.DeletionTimestamp = new(metav1.Time)
		*pod.DeletionTimestamp = metav1.Now()
		pod.Status.ContainerStatuses =
			[]corev1.ContainerStatus{{Name: naming.ContainerPGAdmin}}
		pod.Status.ContainerStatuses[0].State.Running =
			new(corev1.ContainerStateRunning)

		r := new(Reconciler)
		r.Client = fake.NewClientBuilder().WithObjects(pod).Build()

		assert.NilError(t, r.reconcilePGAdminUsers(ctx, cluster, nil, nil))
	})

	t.Run("PodHealthy", func(t *testing.T) {
		cluster := cluster.DeepCopy()
		pod := pod.DeepCopy()

		pod.DeletionTimestamp = nil
		pod.Status.ContainerStatuses =
			[]corev1.ContainerStatus{{Name: naming.ContainerPGAdmin}}
		pod.Status.ContainerStatuses[0].State.Running =
			new(corev1.ContainerStateRunning)

		r := new(Reconciler)
		r.Client = fake.NewClientBuilder().WithObjects(pod).Build()

		calls := 0
		r.PodExec = func(
			ctx context.Context, namespace, pod, container string,
			stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			calls++

			assert.Equal(t, pod, "pgc1-pgadmin-0")
			assert.Equal(t, namespace, cluster.Namespace)
			assert.Equal(t, container, naming.ContainerPGAdmin)

			return nil
		}

		assert.NilError(t, r.reconcilePGAdminUsers(ctx, cluster, nil, nil))
		assert.Equal(t, calls, 1, "PodExec should be called once")

		assert.NilError(t, r.reconcilePGAdminUsers(ctx, cluster, nil, nil))
		assert.Equal(t, calls, 1, "PodExec should not be called again")

		// Do the thing when users change.
		users := []v1beta1.PostgresUserSpec{{Name: "u1"}}

		assert.NilError(t, r.reconcilePGAdminUsers(ctx, cluster, users, nil))
		assert.Equal(t, calls, 2, "PodExec should be called once")

		assert.NilError(t, r.reconcilePGAdminUsers(ctx, cluster, users, nil))
		assert.Equal(t, calls, 2, "PodExec should not be called again")

		// Do the thing when passwords change.
		passwords := map[string]*corev1.Secret{
			"u1": {Data: map[string][]byte{"password": []byte(`something`)}},
		}

		assert.NilError(t, r.reconcilePGAdminUsers(ctx, cluster, users, passwords))
		assert.Equal(t, calls, 3, "PodExec should be called once")

		assert.NilError(t, r.reconcilePGAdminUsers(ctx, cluster, users, passwords))
		assert.Equal(t, calls, 3, "PodExec should not be called again")

		passwords["u1"].Data["password"] = []byte(`rotated`)

		assert.NilError(t, r.reconcilePGAdminUsers(ctx, cluster, users, passwords))
		assert.Equal(t, calls, 4, "PodExec should be called once")

		assert.NilError(t, r.reconcilePGAdminUsers(ctx, cluster, users, passwords))
		assert.Equal(t, calls, 4, "PodExec should not be called again")

		t.Run("ThenDisabled", func(t *testing.T) {
			// TODO(cbandy): Revisit this when there is more than one UI.
			cluster := cluster.DeepCopy()
			cluster.Spec.UserInterface = nil

			assert.Assert(t, cluster.Status.UserInterface != nil, "expected some status")

			r := new(Reconciler)
			assert.NilError(t, r.reconcilePGAdminUsers(ctx, cluster, users, passwords))
			assert.Assert(t, cluster.Status.UserInterface == nil, "expected no status")
		})
	})
}

func pgAdminTestCluster(ns corev1.Namespace) *v1beta1.PostgresCluster {
	return &v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: ns.Name,
		},
		Spec: v1beta1.PostgresClusterSpec{
			PostgresVersion: 13,
			InstanceSets: []v1beta1.PostgresInstanceSetSpec{{
				DataVolumeClaimSpec: testVolumeClaimSpec(),
			}},
			Backups: v1beta1.Backups{
				PGBackRest: v1beta1.PGBackRestArchive{
					Repos: []v1beta1.PGBackRestRepo{{
						Name: "repo1",
						Volume: &v1beta1.RepoPVC{
							VolumeClaimSpec: corev1.PersistentVolumeClaimSpec{
								AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
								Resources: corev1.VolumeResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceStorage: resource.MustParse("1Gi"),
									},
								},
							},
						},
					}},
				},
			},
			UserInterface: &v1beta1.UserInterfaceSpec{
				PGAdmin: &v1beta1.PGAdminPodSpec{
					Image: "test-image",
					DataVolumeClaimSpec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("1Gi"),
							},
						},
						StorageClassName: initialize.String("storage-class-for-data"),
					},
				},
			},
		},
	}
}
