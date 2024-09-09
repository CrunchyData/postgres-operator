// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

var gvks = []schema.GroupVersionKind{{
	Group:   corev1.SchemeGroupVersion.Group,
	Version: corev1.SchemeGroupVersion.Version,
	Kind:    "ConfigMapList",
}, {
	Group:   corev1.SchemeGroupVersion.Group,
	Version: corev1.SchemeGroupVersion.Version,
	Kind:    "SecretList",
}, {
	Group:   appsv1.SchemeGroupVersion.Group,
	Version: appsv1.SchemeGroupVersion.Version,
	Kind:    "StatefulSetList",
}, {
	Group:   appsv1.SchemeGroupVersion.Group,
	Version: appsv1.SchemeGroupVersion.Version,
	Kind:    "DeploymentList",
}, {
	Group:   batchv1.SchemeGroupVersion.Group,
	Version: batchv1.SchemeGroupVersion.Version,
	Kind:    "CronJobList",
}, {
	Group:   corev1.SchemeGroupVersion.Group,
	Version: corev1.SchemeGroupVersion.Version,
	Kind:    "PersistentVolumeClaimList",
}, {
	Group:   corev1.SchemeGroupVersion.Group,
	Version: corev1.SchemeGroupVersion.Version,
	Kind:    "ServiceList",
}, {
	Group:   corev1.SchemeGroupVersion.Group,
	Version: corev1.SchemeGroupVersion.Version,
	Kind:    "EndpointsList",
}, {
	Group:   corev1.SchemeGroupVersion.Group,
	Version: corev1.SchemeGroupVersion.Version,
	Kind:    "ServiceAccountList",
}, {
	Group:   rbacv1.SchemeGroupVersion.Group,
	Version: rbacv1.SchemeGroupVersion.Version,
	Kind:    "RoleBindingList",
}, {
	Group:   rbacv1.SchemeGroupVersion.Group,
	Version: rbacv1.SchemeGroupVersion.Version,
	Kind:    "RoleList",
}}

func TestCustomLabels(t *testing.T) {
	ctx := context.Background()
	_, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 2)

	reconciler := &Reconciler{
		Client:   cc,
		Owner:    client.FieldOwner(t.Name()),
		Recorder: new(record.FakeRecorder),
		Tracer:   otel.Tracer(t.Name()),
	}

	ns := setupNamespace(t, cc)

	reconcileTestCluster := func(cluster *v1beta1.PostgresCluster) {
		assert.NilError(t, errors.WithStack(reconciler.Client.Create(ctx, cluster)))
		t.Cleanup(func() {
			// Remove finalizers, if any, so the namespace can terminate.
			assert.Check(t, client.IgnoreNotFound(
				reconciler.Client.Patch(ctx, cluster, client.RawPatch(
					client.Merge.Type(), []byte(`{"metadata":{"finalizers":[]}}`)))))
		})

		// Reconcile the cluster
		result, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(cluster),
		})
		assert.NilError(t, err)
		assert.Assert(t, result.Requeue == false)
	}

	getUnstructuredLabels := func(cluster v1beta1.PostgresCluster, u unstructured.Unstructured) (map[string]map[string]string, error) {
		var err error
		labels := map[string]map[string]string{}

		if metav1.IsControlledBy(&u, &cluster) {
			switch u.GetKind() {
			case "StatefulSet":
				var resource appsv1.StatefulSet
				err = runtime.DefaultUnstructuredConverter.
					FromUnstructured(u.UnstructuredContent(), &resource)
				labels["resource"] = resource.GetLabels()
				labels["podTemplate"] = resource.Spec.Template.GetLabels()
			case "Deployment":
				var resource appsv1.Deployment
				err = runtime.DefaultUnstructuredConverter.
					FromUnstructured(u.UnstructuredContent(), &resource)
				labels["resource"] = resource.GetLabels()
				labels["podTemplate"] = resource.Spec.Template.GetLabels()
			case "CronJob":
				var resource batchv1.CronJob
				err = runtime.DefaultUnstructuredConverter.
					FromUnstructured(u.UnstructuredContent(), &resource)
				labels["resource"] = resource.GetLabels()
				labels["jobTemplate"] = resource.Spec.JobTemplate.GetLabels()
				labels["jobPodTemplate"] = resource.Spec.JobTemplate.Spec.Template.GetLabels()
			default:
				labels["resource"] = u.GetLabels()
			}
		}
		return labels, err
	}

	t.Run("Cluster", func(t *testing.T) {
		cluster := testCluster()
		cluster.ObjectMeta.Name = "global-cluster"
		cluster.ObjectMeta.Namespace = ns.Name
		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{
			Name:                "daisy-instance1",
			Replicas:            initialize.Int32(1),
			DataVolumeClaimSpec: testVolumeClaimSpec(),
		}, {
			Name:                "daisy-instance2",
			Replicas:            initialize.Int32(1),
			DataVolumeClaimSpec: testVolumeClaimSpec(),
		}}
		cluster.Spec.Metadata = &v1beta1.Metadata{
			Labels: map[string]string{"my.cluster.label": "daisy"},
		}
		testCronSchedule := "@yearly"
		cluster.Spec.Backups.PGBackRest.Repos[0].BackupSchedules = &v1beta1.PGBackRestBackupSchedules{
			Full:         &testCronSchedule,
			Differential: &testCronSchedule,
			Incremental:  &testCronSchedule,
		}
		selector, err := naming.AsSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{
				naming.LabelCluster: cluster.Name,
			},
		})
		assert.NilError(t, err)
		reconcileTestCluster(cluster)

		for _, gvk := range gvks {
			uList := &unstructured.UnstructuredList{}
			uList.SetGroupVersionKind(gvk)
			assert.NilError(t, reconciler.Client.List(ctx, uList,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selector}))

			for i := range uList.Items {
				u := uList.Items[i]
				labels, err := getUnstructuredLabels(*cluster, u)
				assert.NilError(t, err)
				for resourceType, resourceLabels := range labels {
					t.Run(u.GetKind()+"/"+u.GetName()+"/"+resourceType, func(t *testing.T) {
						assert.Equal(t, resourceLabels["my.cluster.label"], "daisy")
					})
				}
			}
		}
	})

	t.Run("Instance", func(t *testing.T) {
		cluster := testCluster()
		cluster.ObjectMeta.Name = "instance-cluster"
		cluster.ObjectMeta.Namespace = ns.Name
		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{
			Name:                "max-instance",
			Replicas:            initialize.Int32(1),
			DataVolumeClaimSpec: testVolumeClaimSpec(),
			Metadata: &v1beta1.Metadata{
				Labels: map[string]string{"my.instance.label": "max"},
			},
		}, {
			Name:                "lucy-instance",
			Replicas:            initialize.Int32(1),
			DataVolumeClaimSpec: testVolumeClaimSpec(),
			Metadata: &v1beta1.Metadata{
				Labels: map[string]string{"my.instance.label": "lucy"},
			},
		}}
		reconcileTestCluster(cluster)
		for _, set := range cluster.Spec.InstanceSets {
			t.Run(set.Name, func(t *testing.T) {
				selector, err := naming.AsSelector(metav1.LabelSelector{
					MatchLabels: map[string]string{
						naming.LabelCluster:     cluster.Name,
						naming.LabelInstanceSet: set.Name,
					},
				})
				assert.NilError(t, err)

				for _, gvk := range gvks {
					uList := &unstructured.UnstructuredList{}
					uList.SetGroupVersionKind(gvk)
					assert.NilError(t, reconciler.Client.List(ctx, uList,
						client.InNamespace(cluster.Namespace),
						client.MatchingLabelsSelector{Selector: selector}))

					for i := range uList.Items {
						u := uList.Items[i]

						labels, err := getUnstructuredLabels(*cluster, u)
						assert.NilError(t, err)
						for resourceType, resourceLabels := range labels {
							t.Run(u.GetKind()+"/"+u.GetName()+"/"+resourceType, func(t *testing.T) {
								assert.Equal(t, resourceLabels["my.instance.label"], set.Metadata.Labels["my.instance.label"])
							})
						}
					}
				}
			})
		}

	})

	t.Run("PGBackRest", func(t *testing.T) {
		cluster := testCluster()
		cluster.ObjectMeta.Name = "pgbackrest-cluster"
		cluster.ObjectMeta.Namespace = ns.Name
		cluster.Spec.Backups.PGBackRest.Metadata = &v1beta1.Metadata{
			Labels: map[string]string{"my.pgbackrest.label": "lucy"},
		}
		testCronSchedule := "@yearly"
		cluster.Spec.Backups.PGBackRest.Repos[0].BackupSchedules = &v1beta1.PGBackRestBackupSchedules{
			Full:         &testCronSchedule,
			Differential: &testCronSchedule,
			Incremental:  &testCronSchedule,
		}
		reconcileTestCluster(cluster)

		selector, err := naming.AsSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{
				naming.LabelCluster: cluster.Name,
			},
			MatchExpressions: []metav1.LabelSelectorRequirement{{
				Key:      naming.LabelPGBackRest,
				Operator: metav1.LabelSelectorOpExists},
			},
		})
		assert.NilError(t, err)

		for _, gvk := range gvks {
			uList := &unstructured.UnstructuredList{}
			uList.SetGroupVersionKind(gvk)
			assert.NilError(t, reconciler.Client.List(ctx, uList,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selector}))

			for i := range uList.Items {
				u := uList.Items[i]

				labels, err := getUnstructuredLabels(*cluster, u)
				assert.NilError(t, err)
				for resourceType, resourceLabels := range labels {
					t.Run(u.GetKind()+"/"+u.GetName()+"/"+resourceType, func(t *testing.T) {
						assert.Equal(t, resourceLabels["my.pgbackrest.label"], "lucy")
					})
				}
			}
		}
	})

	t.Run("PGBouncer", func(t *testing.T) {
		cluster := testCluster()
		cluster.ObjectMeta.Name = "pgbouncer-cluster"
		cluster.ObjectMeta.Namespace = ns.Name
		cluster.Spec.Proxy.PGBouncer.Metadata = &v1beta1.Metadata{
			Labels: map[string]string{"my.pgbouncer.label": "lucy"},
		}
		reconcileTestCluster(cluster)

		selector, err := naming.AsSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{
				naming.LabelCluster: cluster.Name,
				naming.LabelRole:    naming.RolePGBouncer,
			},
		})
		assert.NilError(t, err)

		for _, gvk := range gvks {
			uList := &unstructured.UnstructuredList{}
			uList.SetGroupVersionKind(gvk)
			assert.NilError(t, reconciler.Client.List(ctx, uList,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selector}))

			for i := range uList.Items {
				u := uList.Items[i]

				labels, err := getUnstructuredLabels(*cluster, u)
				assert.NilError(t, err)
				for resourceType, resourceLabels := range labels {
					t.Run(u.GetKind()+"/"+u.GetName()+"/"+resourceType, func(t *testing.T) {
						assert.Equal(t, resourceLabels["my.pgbouncer.label"], "lucy")
					})
				}
			}
		}
	})
}

func TestCustomAnnotations(t *testing.T) {
	ctx := context.Background()
	_, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 2)

	reconciler := &Reconciler{
		Client:   cc,
		Owner:    client.FieldOwner(t.Name()),
		Recorder: new(record.FakeRecorder),
		Tracer:   otel.Tracer(t.Name()),
	}

	ns := setupNamespace(t, cc)

	reconcileTestCluster := func(cluster *v1beta1.PostgresCluster) {
		assert.NilError(t, errors.WithStack(reconciler.Client.Create(ctx, cluster)))
		t.Cleanup(func() {
			// Remove finalizers, if any, so the namespace can terminate.
			assert.Check(t, client.IgnoreNotFound(
				reconciler.Client.Patch(ctx, cluster, client.RawPatch(
					client.Merge.Type(), []byte(`{"metadata":{"finalizers":[]}}`)))))
		})

		// Reconcile the cluster
		result, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(cluster),
		})
		assert.NilError(t, err)
		assert.Assert(t, result.Requeue == false)
	}

	getUnstructuredAnnotations := func(cluster v1beta1.PostgresCluster, u unstructured.Unstructured) (map[string]map[string]string, error) {
		var err error
		annotations := map[string]map[string]string{}

		if metav1.IsControlledBy(&u, &cluster) {
			switch u.GetKind() {
			case "StatefulSet":
				var resource appsv1.StatefulSet
				err = runtime.DefaultUnstructuredConverter.
					FromUnstructured(u.UnstructuredContent(), &resource)
				annotations["resource"] = resource.GetAnnotations()
				annotations["podTemplate"] = resource.Spec.Template.GetAnnotations()
			case "Deployment":
				var resource appsv1.Deployment
				err = runtime.DefaultUnstructuredConverter.
					FromUnstructured(u.UnstructuredContent(), &resource)
				annotations["resource"] = resource.GetAnnotations()
				annotations["podTemplate"] = resource.Spec.Template.GetAnnotations()
			case "CronJob":
				var resource batchv1.CronJob
				err = runtime.DefaultUnstructuredConverter.
					FromUnstructured(u.UnstructuredContent(), &resource)
				annotations["resource"] = resource.GetAnnotations()
				annotations["jobTemplate"] = resource.Spec.JobTemplate.GetAnnotations()
				annotations["jobPodTemplate"] = resource.Spec.JobTemplate.Spec.Template.GetAnnotations()
			default:
				annotations["resource"] = u.GetAnnotations()
			}
		}
		return annotations, err
	}

	t.Run("Cluster", func(t *testing.T) {
		cluster := testCluster()
		cluster.ObjectMeta.Name = "global-cluster"
		cluster.ObjectMeta.Namespace = ns.Name
		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{
			Name:                "daisy-instance1",
			Replicas:            initialize.Int32(1),
			DataVolumeClaimSpec: testVolumeClaimSpec(),
		}, {
			Name:                "daisy-instance2",
			Replicas:            initialize.Int32(1),
			DataVolumeClaimSpec: testVolumeClaimSpec(),
		}}
		cluster.Spec.Metadata = &v1beta1.Metadata{
			Annotations: map[string]string{"my.cluster.annotation": "daisy"},
		}
		testCronSchedule := "@yearly"
		cluster.Spec.Backups.PGBackRest.Repos[0].BackupSchedules = &v1beta1.PGBackRestBackupSchedules{
			Full:         &testCronSchedule,
			Differential: &testCronSchedule,
			Incremental:  &testCronSchedule,
		}
		reconcileTestCluster(cluster)

		selector, err := naming.AsSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{
				naming.LabelCluster: cluster.Name,
			},
		})
		assert.NilError(t, err)

		for _, gvk := range gvks {
			uList := &unstructured.UnstructuredList{}
			uList.SetGroupVersionKind(gvk)
			assert.NilError(t, reconciler.Client.List(ctx, uList,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selector}))

			for i := range uList.Items {
				u := uList.Items[i]
				annotations, err := getUnstructuredAnnotations(*cluster, u)
				assert.NilError(t, err)
				for resourceType, resourceAnnotations := range annotations {
					t.Run(u.GetKind()+"/"+u.GetName()+"/"+resourceType, func(t *testing.T) {
						assert.Equal(t, resourceAnnotations["my.cluster.annotation"], "daisy")
					})
				}
			}
		}
	})

	t.Run("Instance", func(t *testing.T) {
		cluster := testCluster()
		cluster.ObjectMeta.Name = "instance-cluster"
		cluster.ObjectMeta.Namespace = ns.Name
		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{
			Name:                "max-instance",
			Replicas:            initialize.Int32(1),
			DataVolumeClaimSpec: testVolumeClaimSpec(),
			Metadata: &v1beta1.Metadata{
				Annotations: map[string]string{"my.instance.annotation": "max"},
			},
		}, {
			Name:                "lucy-instance",
			Replicas:            initialize.Int32(1),
			DataVolumeClaimSpec: testVolumeClaimSpec(),
			Metadata: &v1beta1.Metadata{
				Annotations: map[string]string{"my.instance.annotation": "lucy"},
			},
		}}
		reconcileTestCluster(cluster)
		for _, set := range cluster.Spec.InstanceSets {
			t.Run(set.Name, func(t *testing.T) {
				selector, err := naming.AsSelector(metav1.LabelSelector{
					MatchLabels: map[string]string{
						naming.LabelCluster:     cluster.Name,
						naming.LabelInstanceSet: set.Name,
					},
				})
				assert.NilError(t, err)

				for _, gvk := range gvks {
					uList := &unstructured.UnstructuredList{}
					uList.SetGroupVersionKind(gvk)
					assert.NilError(t, reconciler.Client.List(ctx, uList,
						client.InNamespace(cluster.Namespace),
						client.MatchingLabelsSelector{Selector: selector}))

					for i := range uList.Items {
						u := uList.Items[i]

						annotations, err := getUnstructuredAnnotations(*cluster, u)
						assert.NilError(t, err)
						for resourceType, resourceAnnotations := range annotations {
							t.Run(u.GetKind()+"/"+u.GetName()+"/"+resourceType, func(t *testing.T) {
								assert.Equal(t, resourceAnnotations["my.instance.annotation"], set.Metadata.Annotations["my.instance.annotation"])
							})
						}
					}
				}
			})
		}

	})

	t.Run("PGBackRest", func(t *testing.T) {
		cluster := testCluster()
		cluster.ObjectMeta.Name = "pgbackrest-cluster"
		cluster.ObjectMeta.Namespace = ns.Name
		cluster.Spec.Backups.PGBackRest.Metadata = &v1beta1.Metadata{
			Annotations: map[string]string{"my.pgbackrest.annotation": "lucy"},
		}
		testCronSchedule := "@yearly"
		cluster.Spec.Backups.PGBackRest.Repos[0].BackupSchedules = &v1beta1.PGBackRestBackupSchedules{
			Full:         &testCronSchedule,
			Differential: &testCronSchedule,
			Incremental:  &testCronSchedule,
		}
		reconcileTestCluster(cluster)

		selector, err := naming.AsSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{
				naming.LabelCluster: cluster.Name,
			},
			MatchExpressions: []metav1.LabelSelectorRequirement{{
				Key:      naming.LabelPGBackRest,
				Operator: metav1.LabelSelectorOpExists},
			},
		})
		assert.NilError(t, err)

		for _, gvk := range gvks {
			uList := &unstructured.UnstructuredList{}
			uList.SetGroupVersionKind(gvk)
			assert.NilError(t, reconciler.Client.List(ctx, uList,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selector}))

			for i := range uList.Items {
				u := uList.Items[i]

				annotations, err := getUnstructuredAnnotations(*cluster, u)
				assert.NilError(t, err)
				for resourceType, resourceAnnotations := range annotations {
					t.Run(u.GetKind()+"/"+u.GetName()+"/"+resourceType, func(t *testing.T) {
						assert.Equal(t, resourceAnnotations["my.pgbackrest.annotation"], "lucy")
					})
				}
			}
		}
	})

	t.Run("PGBouncer", func(t *testing.T) {
		cluster := testCluster()
		cluster.ObjectMeta.Name = "pgbouncer-cluster"
		cluster.ObjectMeta.Namespace = ns.Name
		cluster.Spec.Proxy.PGBouncer.Metadata = &v1beta1.Metadata{
			Annotations: map[string]string{"my.pgbouncer.annotation": "lucy"},
		}
		reconcileTestCluster(cluster)

		selector, err := naming.AsSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{
				naming.LabelCluster: cluster.Name,
				naming.LabelRole:    naming.RolePGBouncer,
			},
		})
		assert.NilError(t, err)

		for _, gvk := range gvks {
			uList := &unstructured.UnstructuredList{}
			uList.SetGroupVersionKind(gvk)
			assert.NilError(t, reconciler.Client.List(ctx, uList,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selector}))

			for i := range uList.Items {
				u := uList.Items[i]

				annotations, err := getUnstructuredAnnotations(*cluster, u)
				assert.NilError(t, err)
				for resourceType, resourceAnnotations := range annotations {
					t.Run(u.GetKind()+"/"+u.GetName()+"/"+resourceType, func(t *testing.T) {
						assert.Equal(t, resourceAnnotations["my.pgbouncer.annotation"], "lucy")
					})
				}
			}
		}
	})
}

func TestGenerateClusterPrimaryService(t *testing.T) {
	_, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	reconciler := &Reconciler{Client: cc}

	cluster := &v1beta1.PostgresCluster{}
	cluster.Namespace = "ns2"
	cluster.Name = "pg5"
	cluster.Spec.Port = initialize.Int32(2600)

	leader := &corev1.Service{}
	leader.Spec.ClusterIP = "1.9.8.3"

	_, _, err := reconciler.generateClusterPrimaryService(cluster, nil)
	assert.ErrorContains(t, err, "not implemented")

	alwaysExpect := func(t testing.TB, service *corev1.Service, endpoints *corev1.Endpoints) {
		assert.Assert(t, cmp.MarshalMatches(service.TypeMeta, `
apiVersion: v1
kind: Service
		`))
		assert.Assert(t, cmp.MarshalMatches(service.ObjectMeta, `
creationTimestamp: null
labels:
  postgres-operator.crunchydata.com/cluster: pg5
  postgres-operator.crunchydata.com/role: primary
name: pg5-primary
namespace: ns2
ownerReferences:
- apiVersion: postgres-operator.crunchydata.com/v1beta1
  blockOwnerDeletion: true
  controller: true
  kind: PostgresCluster
  name: pg5
  uid: ""
		`))
		assert.Assert(t, cmp.MarshalMatches(service.Spec.Ports, `
- name: postgres
  port: 2600
  protocol: TCP
  targetPort: postgres
		`))

		assert.Equal(t, service.Spec.ClusterIP, "None")
		assert.Assert(t, service.Spec.Selector == nil,
			"got %v", service.Spec.Selector)

		assert.Assert(t, cmp.MarshalMatches(endpoints, `
apiVersion: v1
kind: Endpoints
metadata:
  creationTimestamp: null
  labels:
    postgres-operator.crunchydata.com/cluster: pg5
    postgres-operator.crunchydata.com/role: primary
  name: pg5-primary
  namespace: ns2
  ownerReferences:
  - apiVersion: postgres-operator.crunchydata.com/v1beta1
    blockOwnerDeletion: true
    controller: true
    kind: PostgresCluster
    name: pg5
    uid: ""
subsets:
- addresses:
  - ip: 1.9.8.3
  ports:
  - name: postgres
    port: 2600
    protocol: TCP
		`))
	}

	service, endpoints, err := reconciler.generateClusterPrimaryService(cluster, leader)
	assert.NilError(t, err)
	alwaysExpect(t, service, endpoints)

	t.Run("LeaderLoadBalancer", func(t *testing.T) {
		leader := leader.DeepCopy()
		leader.Spec.Type = corev1.ServiceTypeLoadBalancer
		leader.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{
			{IP: "55.44.33.22"},
			{IP: "99.88.77.66", Hostname: "some.host"},
			{IP: "1.2.3.4", Hostname: "only.the.first"},
		}

		service, endpoints, err := reconciler.generateClusterPrimaryService(cluster, leader)
		assert.NilError(t, err)
		alwaysExpect(t, service, endpoints)

		// generateClusterPrimaryService no longer sets ExternalIPs or ExternalName from
		// LoadBalancer-type leader service
		// - https://cloud.google.com/anthos/clusters/docs/security-bulletins#gcp-2020-015
		assert.Equal(t, len(service.Spec.ExternalIPs), 0)
		assert.Equal(t, service.Spec.ExternalName, "")
	})
}

func TestReconcileClusterPrimaryService(t *testing.T) {
	ctx := context.Background()
	_, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 1)

	reconciler := &Reconciler{Client: cc, Owner: client.FieldOwner(t.Name())}

	cluster := testCluster()
	cluster.Namespace = setupNamespace(t, cc).Name
	assert.NilError(t, cc.Create(ctx, cluster))

	_, err := reconciler.reconcileClusterPrimaryService(ctx, cluster, nil)
	assert.ErrorContains(t, err, "not implemented")

	leader := &corev1.Service{}
	leader.Spec.ClusterIP = "192.0.2.10"

	service, err := reconciler.reconcileClusterPrimaryService(ctx, cluster, leader)
	assert.NilError(t, err)
	assert.Assert(t, service != nil && service.UID != "", "expected created service")
}

func TestGenerateClusterReplicaServiceIntent(t *testing.T) {
	_, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	reconciler := &Reconciler{Client: cc}

	cluster := &v1beta1.PostgresCluster{}
	cluster.Namespace = "ns1"
	cluster.Name = "pg2"
	cluster.Spec.Port = initialize.Int32(9876)

	service, err := reconciler.generateClusterReplicaService(cluster)
	assert.NilError(t, err)

	alwaysExpect := func(t testing.TB, service *corev1.Service) {
		assert.Assert(t, cmp.MarshalMatches(service.TypeMeta, `
apiVersion: v1
kind: Service
		`))
		assert.Assert(t, cmp.MarshalMatches(service.ObjectMeta, `
creationTimestamp: null
labels:
  postgres-operator.crunchydata.com/cluster: pg2
  postgres-operator.crunchydata.com/role: replica
name: pg2-replicas
namespace: ns1
ownerReferences:
- apiVersion: postgres-operator.crunchydata.com/v1beta1
  blockOwnerDeletion: true
  controller: true
  kind: PostgresCluster
  name: pg2
  uid: ""
		`))
	}

	alwaysExpect(t, service)
	assert.Assert(t, cmp.MarshalMatches(service.Spec, `
ports:
- name: postgres
  port: 9876
  protocol: TCP
  targetPort: postgres
selector:
  postgres-operator.crunchydata.com/cluster: pg2
  postgres-operator.crunchydata.com/role: replica
type: ClusterIP
	`))

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
			cluster.Spec.ReplicaService = &v1beta1.ServiceSpec{Type: test.Type}

			service, err := reconciler.generateClusterReplicaService(cluster)
			assert.NilError(t, err)
			alwaysExpect(t, service)
			test.Expect(t, service)
			assert.Assert(t, cmp.MarshalMatches(service.Spec.Ports, `
- name: postgres
  port: 9876
  protocol: TCP
  targetPort: postgres
	`))
		})
	}

	t.Run("AnnotationsLabels", func(t *testing.T) {
		cluster := cluster.DeepCopy()
		cluster.Spec.Metadata = &v1beta1.Metadata{
			Annotations: map[string]string{"some": "note"},
			Labels:      map[string]string{"happy": "label"},
		}

		service, err := reconciler.generateClusterReplicaService(cluster)
		assert.NilError(t, err)

		// Annotations present in the metadata.
		assert.Assert(t, cmp.MarshalMatches(service.ObjectMeta.Annotations, `
some: note
		`))

		// Labels present in the metadata.
		assert.Assert(t, cmp.MarshalMatches(service.ObjectMeta.Labels, `
happy: label
postgres-operator.crunchydata.com/cluster: pg2
postgres-operator.crunchydata.com/role: replica
		`))

		// Labels not in the selector.
		assert.Assert(t, cmp.MarshalMatches(service.Spec.Selector, `
postgres-operator.crunchydata.com/cluster: pg2
postgres-operator.crunchydata.com/role: replica
		`))
	})
}
