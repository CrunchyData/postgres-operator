// +build envtest

package postgrescluster

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

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/util"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcilePGUserSecret(t *testing.T) {

	// setup the test environment and ensure a clean teardown
	tEnv, tClient, cfg := setupTestEnv(t, ControllerName)
	t.Cleanup(func() { teardownTestEnv(t, tEnv) })

	r := &Reconciler{}
	ctx, cancel := setupManager(t, cfg, func(mgr manager.Manager) {
		r = &Reconciler{
			Client:   tClient,
			Recorder: mgr.GetEventRecorderFor(ControllerName),
			Tracer:   otel.Tracer(ControllerName),
			Owner:    ControllerName,
		}
	})
	t.Cleanup(func() { teardownManager(cancel, t) })

	// test postgrescluster values
	var (
		postgresPort             = int32(5432)
		clusterName              = "hippocluster"
		namespace                = "postgres-operator-test-" + rand.String(6)
		clusterUID               = types.UID("hippouid")
		returnedConnectionString string
	)

	ns := &corev1.Namespace{}
	ns.Name = namespace
	assert.NilError(t, tClient.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, tClient.Delete(ctx, ns)) })

	// create a PostgresCluster to test with
	postgresCluster := &v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: namespace,
			UID:       clusterUID,
		},
		Spec: v1beta1.PostgresClusterSpec{
			Port: &postgresPort,
		},
	}

	t.Run("create postgres user secret", func(t *testing.T) {

		_, err := r.reconcilePGUserSecret(ctx, postgresCluster)
		assert.NilError(t, err)
	})

	t.Run("validate postgres user secret", func(t *testing.T) {

		pgUserSecret := &v1.Secret{ObjectMeta: naming.PostgresUserSecret(postgresCluster)}
		pgUserSecret.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Secret"))
		err := r.Client.Get(ctx, client.ObjectKeyFromObject(pgUserSecret), pgUserSecret)
		assert.NilError(t, err)

		databasename, ok := pgUserSecret.Data["dbname"]
		assert.Assert(t, ok)
		assert.Equal(t, string(databasename), postgresCluster.Name)

		host, ok := pgUserSecret.Data["host"]
		assert.Assert(t, ok)
		assert.Equal(t, string(host), fmt.Sprintf("%s.%s.svc", "hippocluster-primary", namespace))

		port, ok := pgUserSecret.Data["port"]
		assert.Assert(t, ok)
		assert.Equal(t, string(port), fmt.Sprintf("%d", *postgresCluster.Spec.Port))

		username, ok := pgUserSecret.Data["user"]
		assert.Assert(t, ok)
		assert.Equal(t, string(username), postgresCluster.Name)

		password, ok := pgUserSecret.Data["password"]
		assert.Assert(t, ok)
		assert.Equal(t, len(password), util.DefaultGeneratedPasswordLength)

		secretConnectionString1, ok := pgUserSecret.Data["uri"]

		assert.Assert(t, ok)

		returnedConnectionString = string(secretConnectionString1)

		testConnectionString := (&url.URL{
			Scheme: "postgresql",
			Host:   fmt.Sprintf("%s.%s.svc:%d", "hippocluster-primary", namespace, *postgresCluster.Spec.Port),
			User:   url.UserPassword(string(pgUserSecret.Data["user"]), string(pgUserSecret.Data["password"])),
			Path:   string(pgUserSecret.Data["dbname"]),
		}).String()

		assert.Equal(t, returnedConnectionString, testConnectionString)

		// ensure none of the pgBouncer information is set
		_, ok = pgUserSecret.Data["pgbouncer-host"]
		assert.Assert(t, !ok)

		_, ok = pgUserSecret.Data["pgbouncer-port"]
		assert.Assert(t, !ok)

		_, ok = pgUserSecret.Data["pgbouncer-uri"]
		assert.Assert(t, !ok)
	})

	t.Run("validate postgres user password is not changed after another reconcile", func(t *testing.T) {

		pgUserSecret2, err := r.reconcilePGUserSecret(ctx, postgresCluster)
		assert.NilError(t, err)

		returnedConnectionString2, ok := pgUserSecret2.Data["uri"]

		assert.Assert(t, ok)
		assert.Equal(t, string(returnedConnectionString2), returnedConnectionString)

	})

	t.Run("validate addition of pgbouncer", func(t *testing.T) {
		postgresCluster.Spec.Proxy = new(v1beta1.PostgresProxySpec)
		postgresCluster.Spec.Proxy.PGBouncer = new(v1beta1.PGBouncerPodSpec)
		postgresCluster.Spec.Proxy.PGBouncer.Port = new(int32)
		*postgresCluster.Spec.Proxy.PGBouncer.Port = 6432

		pgUserSecret2, err := r.reconcilePGUserSecret(ctx, postgresCluster)
		assert.NilError(t, err)

		host, ok := pgUserSecret2.Data["pgbouncer-host"]
		assert.Assert(t, ok)
		assert.Equal(t, string(host), fmt.Sprintf("%s-pgbouncer.%s.svc",
			postgresCluster.Name, postgresCluster.Namespace))

		port, ok := pgUserSecret2.Data["pgbouncer-port"]
		assert.Assert(t, ok)
		assert.Equal(t, string(port), fmt.Sprintf("%d", *postgresCluster.Spec.Proxy.PGBouncer.Port))

		pgBouncerConnectionString := (&url.URL{
			Scheme: "postgresql",
			Host: fmt.Sprintf("%s-pgbouncer.%s.svc:%d",
				postgresCluster.Name, postgresCluster.Namespace, *postgresCluster.Spec.Proxy.PGBouncer.Port),
			User: url.UserPassword(string(pgUserSecret2.Data["user"]), string(pgUserSecret2.Data["password"])),
			Path: string(pgUserSecret2.Data["dbname"]),
		}).String()
		uri, ok := pgUserSecret2.Data["pgbouncer-uri"]
		assert.Assert(t, ok)
		assert.Equal(t, string(uri), pgBouncerConnectionString)
	})
}

var gvks = []schema.GroupVersionKind{{
	Group:   v1.SchemeGroupVersion.Group,
	Version: v1.SchemeGroupVersion.Version,
	Kind:    "ConfigMapList",
}, {
	Group:   v1.SchemeGroupVersion.Group,
	Version: v1.SchemeGroupVersion.Version,
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
	Group:   batchv1beta1.SchemeGroupVersion.Group,
	Version: batchv1beta1.SchemeGroupVersion.Version,
	Kind:    "CronJobList",
}, {
	Group:   v1.SchemeGroupVersion.Group,
	Version: v1.SchemeGroupVersion.Version,
	Kind:    "PersistentVolumeClaimList",
}, {
	Group:   v1.SchemeGroupVersion.Group,
	Version: v1.SchemeGroupVersion.Version,
	Kind:    "ServiceList",
}, {
	Group:   v1.SchemeGroupVersion.Group,
	Version: v1.SchemeGroupVersion.Version,
	Kind:    "EndpointsList",
}, {
	Group:   v1.SchemeGroupVersion.Group,
	Version: v1.SchemeGroupVersion.Version,
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
	t.Parallel()

	env, cc, config := setupTestEnv(t, ControllerName)
	t.Cleanup(func() { teardownTestEnv(t, env) })

	reconciler := &Reconciler{}
	ctx, cancel := setupManager(t, config, func(mgr manager.Manager) {
		reconciler = &Reconciler{
			Client:   cc,
			Owner:    client.FieldOwner(t.Name()),
			Recorder: mgr.GetEventRecorderFor(ControllerName),
			Tracer:   otel.Tracer(t.Name()),
		}
	})
	t.Cleanup(func() { teardownManager(cancel, t) })

	ns := &v1.Namespace{}
	ns.GenerateName = "postgres-operator-test-"
	ns.Labels = labels.Set{"postgres-operator-test": t.Name()}
	assert.NilError(t, cc.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, ns)) })

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
				var resource batchv1beta1.CronJob
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
			Replicas:            Int32(1),
			DataVolumeClaimSpec: testVolumeClaimSpec(),
		}, {
			Name:                "daisy-instance2",
			Replicas:            Int32(1),
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
			Replicas:            Int32(1),
			DataVolumeClaimSpec: testVolumeClaimSpec(),
			Metadata: &v1beta1.Metadata{
				Labels: map[string]string{"my.instance.label": "max"},
			},
		}, {
			Name:                "lucy-instance",
			Replicas:            Int32(1),
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
	t.Parallel()

	env, cc, config := setupTestEnv(t, ControllerName)
	t.Cleanup(func() { teardownTestEnv(t, env) })

	reconciler := &Reconciler{}
	ctx, cancel := setupManager(t, config, func(mgr manager.Manager) {
		reconciler = &Reconciler{
			Client:   cc,
			Owner:    client.FieldOwner(t.Name()),
			Recorder: mgr.GetEventRecorderFor(ControllerName),
			Tracer:   otel.Tracer(t.Name()),
		}
	})
	t.Cleanup(func() { teardownManager(cancel, t) })

	ns := &v1.Namespace{}
	ns.GenerateName = "postgres-operator-test-"
	ns.Labels = labels.Set{"postgres-operator-test": ""}
	assert.NilError(t, cc.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, ns)) })

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
				var resource batchv1beta1.CronJob
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
			Replicas:            Int32(1),
			DataVolumeClaimSpec: testVolumeClaimSpec(),
		}, {
			Name:                "daisy-instance2",
			Replicas:            Int32(1),
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
			Replicas:            Int32(1),
			DataVolumeClaimSpec: testVolumeClaimSpec(),
			Metadata: &v1beta1.Metadata{
				Annotations: map[string]string{"my.instance.annotation": "max"},
			},
		}, {
			Name:                "lucy-instance",
			Replicas:            Int32(1),
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

func TestContainerSecurityContext(t *testing.T) {
	if !strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
		t.Skip("Test requires pods to be created")
	}

	t.Parallel()

	env, cc, config := setupTestEnv(t, ControllerName)
	t.Cleanup(func() { teardownTestEnv(t, env) })

	reconciler := &Reconciler{}
	ctx, cancel := setupManager(t, config, func(mgr manager.Manager) {
		reconciler = &Reconciler{
			Client:   cc,
			Owner:    client.FieldOwner(t.Name()),
			Recorder: mgr.GetEventRecorderFor(ControllerName),
			Tracer:   otel.Tracer(t.Name()),
		}
		podExec, err := newPodExecutor(config)
		assert.NilError(t, err)
		reconciler.PodExec = podExec
	})
	t.Cleanup(func() { teardownManager(cancel, t) })

	ns := &v1.Namespace{}
	ns.GenerateName = "postgres-operator-test-"
	ns.Labels = labels.Set{"postgres-operator-test": ""}
	assert.NilError(t, cc.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, ns)) })

	cluster := testCluster()
	cluster.Namespace = ns.Name

	assert.NilError(t, errors.WithStack(reconciler.Client.Create(ctx, cluster)))
	t.Cleanup(func() {
		// Remove finalizers, if any, so the namespace can terminate.
		assert.Check(t, client.IgnoreNotFound(
			reconciler.Client.Patch(ctx, cluster, client.RawPatch(
				client.Merge.Type(), []byte(`{"metadata":{"finalizers":[]}}`)))))
	})

	pods := &corev1.PodList{}
	assert.NilError(t, wait.Poll(time.Second, time.Second*120, func() (done bool, err error) {
		// Reconcile the cluster
		result, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(cluster),
		})
		if err != nil {
			return false, err
		}
		if result.Requeue {
			return false, nil
		}

		err = reconciler.Client.List(ctx, pods,
			client.InNamespace(ns.Name),
			client.MatchingLabels{
				naming.LabelCluster: cluster.Name,
			})
		if err != nil {
			return false, err
		}

		// Can expect 4 pods from a cluster
		// instance, repo-host, pgbouncer, backup(s)
		if len(pods.Items) < 4 {
			return false, nil
		}
		return true, nil
	}))

	// Once we have a pod list with pods of each type, check that the
	// pods containers have the expected Security Context options
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			assert.Equal(t, *container.SecurityContext.Privileged, false)
			assert.Equal(t, *container.SecurityContext.ReadOnlyRootFilesystem, true)
			assert.Equal(t, *container.SecurityContext.AllowPrivilegeEscalation, false)
		}
		for _, initContainer := range pod.Spec.InitContainers {
			assert.Equal(t, *initContainer.SecurityContext.Privileged, false)
			assert.Equal(t, *initContainer.SecurityContext.ReadOnlyRootFilesystem, true)
			assert.Equal(t, *initContainer.SecurityContext.AllowPrivilegeEscalation, false)
		}
	}
}
