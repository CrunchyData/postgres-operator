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

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/util"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
	"github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
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
	})

	t.Run("validate postgres user password is not changed after another reconcile", func(t *testing.T) {

		pgUserSecret2, err := r.reconcilePGUserSecret(ctx, postgresCluster)
		assert.NilError(t, err)

		returnedConnectionString2, ok := pgUserSecret2.Data["uri"]

		assert.Assert(t, ok)
		assert.Equal(t, string(returnedConnectionString2), returnedConnectionString)

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

func TestCustomGlobalLabels(t *testing.T) {
	if !strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
		t.Skip("requires a running garbage collection controller")
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
	ns.Labels = labels.Set{"postgres-operator-test": t.Name()}
	assert.NilError(t, cc.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, ns)) })

	mustReconcile := func(t *testing.T, cluster *v1beta1.PostgresCluster) reconcile.Result {
		t.Helper()
		key := client.ObjectKeyFromObject(cluster)
		request := reconcile.Request{NamespacedName: key}
		result, err := reconciler.Reconcile(ctx, request)
		assert.NilError(t, err, "%+v", err)
		return result
	}

	cluster := testCluster()
	cluster.ObjectMeta.Name = "global-cluster"
	cluster.ObjectMeta.Namespace = ns.Name
	cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{
		Name:            "daisy-instance1",
		Replicas:        Int32(1),
		VolumeClaimSpec: testVolumeClaimSpec(),
	}, {
		Name:            "daisy-instance2",
		Replicas:        Int32(1),
		VolumeClaimSpec: testVolumeClaimSpec(),
	}}
	cluster.Spec.Metadata.Labels = map[string]string{"my.cluster.label": "daisy"}

	assert.NilError(t, errors.WithStack(reconciler.Client.Create(ctx, cluster)))
	t.Cleanup(func() {
		// Remove finalizers, if any, so the namespace can terminate.
		assert.Check(t, client.IgnoreNotFound(
			reconciler.Client.Patch(ctx, cluster, client.RawPatch(
				client.Merge.Type(), []byte(`{"metadata":{"finalizers":[]}}`)))))
	})

	// Continue until instances are healthy.
	g := gomega.NewWithT(t)
	g.Eventually(func() (instances []appsv1.StatefulSet) {
		mustReconcile(t, cluster)

		list := appsv1.StatefulSetList{}
		selector, err := labels.Parse(strings.Join([]string{
			"postgres-operator.crunchydata.com/cluster=" + cluster.Name,
			"postgres-operator.crunchydata.com/instance",
		}, ","))
		assert.NilError(t, err)
		assert.NilError(t, cc.List(ctx, &list,
			client.InNamespace(cluster.Namespace),
			client.MatchingLabelsSelector{Selector: selector}))
		return list.Items
	},
		"120s", // timeout
		"5s",   // interval
	).Should(gomega.WithTransform(func(instances []appsv1.StatefulSet) int {
		ready := 0
		for _, sts := range instances {
			ready += int(sts.Status.ReadyReplicas)
		}
		return ready
	}, gomega.BeNumerically("==", len(cluster.Spec.InstanceSets))))

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
			var resourceLabels map[string]string
			var templateLabels map[string]string

			if !metav1.IsControlledBy(&u, cluster) {
				continue
			}

			t.Run(u.GetKind()+"/"+u.GetName(), func(t *testing.T) {
				switch u.GetKind() {
				case "StatefulSetList":
					var resource appsv1.StatefulSet
					assert.NilError(t, runtime.DefaultUnstructuredConverter.
						FromUnstructured(u.UnstructuredContent(), &resource))
					resourceLabels = resource.GetLabels()
					templateLabels = resource.Spec.Template.GetLabels()
				case "DeploymentList":
					var resource appsv1.Deployment
					assert.NilError(t, runtime.DefaultUnstructuredConverter.
						FromUnstructured(u.UnstructuredContent(), &resource))
					resourceLabels = resource.GetLabels()
					templateLabels = resource.Spec.Template.GetLabels()
				default:
					resourceLabels = u.GetLabels()
				}

				assert.Equal(t, resourceLabels["my.cluster.label"], "daisy")
				if templateLabels != nil {
					t.Run("template", func(t *testing.T) {
						assert.Equal(t, templateLabels["my.cluster.label"], "daisy")
					})
				}
			})
		}
	}
}

func TestCustomInstanceLabels(t *testing.T) {
	if !strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
		t.Skip("requires a running garbage collection controller")
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
	ns.Labels = labels.Set{"postgres-operator-test": t.Name()}
	assert.NilError(t, cc.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, ns)) })

	mustReconcile := func(t *testing.T, cluster *v1beta1.PostgresCluster) reconcile.Result {
		t.Helper()
		key := client.ObjectKeyFromObject(cluster)
		request := reconcile.Request{NamespacedName: key}
		result, err := reconciler.Reconcile(ctx, request)
		assert.NilError(t, err, "%+v", err)
		return result
	}

	cluster := testCluster()
	cluster.ObjectMeta.Name = "instance-cluster"
	cluster.ObjectMeta.Namespace = ns.Name
	cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{
		Name:            "max-instance",
		Replicas:        Int32(1),
		VolumeClaimSpec: testVolumeClaimSpec(),
		Metadata: v1beta1.Metadata{
			Labels: map[string]string{"my.instance.label": "max"},
		},
	}, {
		Name:            "lucy-instance",
		Replicas:        Int32(1),
		VolumeClaimSpec: testVolumeClaimSpec(),
		Metadata: v1beta1.Metadata{
			Labels: map[string]string{"my.instance.label": "lucy"},
		},
	}}

	assert.NilError(t, errors.WithStack(reconciler.Client.Create(ctx, cluster)))
	t.Cleanup(func() {
		// Remove finalizers, if any, so the namespace can terminate.
		assert.Check(t, client.IgnoreNotFound(
			reconciler.Client.Patch(ctx, cluster, client.RawPatch(
				client.Merge.Type(), []byte(`{"metadata":{"finalizers":[]}}`)))))
	})

	// Continue until instances are healthy.
	g := gomega.NewWithT(t)
	g.Eventually(func() (instances []appsv1.StatefulSet) {
		mustReconcile(t, cluster)

		list := appsv1.StatefulSetList{}
		selector, err := labels.Parse(strings.Join([]string{
			"postgres-operator.crunchydata.com/cluster=" + cluster.Name,
			"postgres-operator.crunchydata.com/instance",
		}, ","))
		assert.NilError(t, err)
		assert.NilError(t, cc.List(ctx, &list,
			client.InNamespace(cluster.Namespace),
			client.MatchingLabelsSelector{Selector: selector}))
		return list.Items
	},
		"120s", // timeout
		"5s",   // interval
	).Should(gomega.WithTransform(func(instances []appsv1.StatefulSet) int {
		ready := 0
		for _, sts := range instances {
			ready += int(sts.Status.ReadyReplicas)
		}
		return ready
	}, gomega.BeNumerically("==", len(cluster.Spec.InstanceSets))))

	for _, instanceSet := range cluster.Spec.InstanceSets {
		t.Run(instanceSet.Name, func(t *testing.T) {
			selector, err := naming.AsSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{
					naming.LabelCluster:     cluster.Name,
					naming.LabelInstanceSet: instanceSet.Name,
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
					var resourceLabels map[string]string
					var templateLabels map[string]string

					if !metav1.IsControlledBy(&u, cluster) {
						continue
					}

					t.Run(u.GetKind()+"/"+u.GetName(), func(t *testing.T) {
						switch u.GetKind() {
						case "StatefulSetList":
							var resource appsv1.StatefulSet
							assert.NilError(t, runtime.DefaultUnstructuredConverter.
								FromUnstructured(u.UnstructuredContent(), &resource))
							resourceLabels = resource.GetLabels()
							templateLabels = resource.Spec.Template.Labels
						case "DeploymentList":
							var resource appsv1.Deployment
							assert.NilError(t, runtime.DefaultUnstructuredConverter.
								FromUnstructured(u.UnstructuredContent(), &resource))
							resourceLabels = resource.GetLabels()
							templateLabels = resource.Spec.Template.GetLabels()
						default:
							resourceLabels = u.GetLabels()
						}

						assert.Equal(t, resourceLabels["my.instance.label"], instanceSet.Metadata.Labels["my.instance.label"])
						if templateLabels != nil {
							t.Run("template", func(t *testing.T) {
								assert.Equal(t, templateLabels["my.instance.label"], instanceSet.Metadata.Labels["my.instance.label"])
							})
						}
					})
				}
			}
		})
	}
}

func TestCustomPGBackRestLabels(t *testing.T) {
	if !strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
		t.Skip("requires a running garbage collection controller")
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
	ns.Labels = labels.Set{"postgres-operator-test": t.Name()}
	assert.NilError(t, cc.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, ns)) })

	mustReconcile := func(t *testing.T, cluster *v1beta1.PostgresCluster) reconcile.Result {
		t.Helper()
		key := client.ObjectKeyFromObject(cluster)
		request := reconcile.Request{NamespacedName: key}
		result, err := reconciler.Reconcile(ctx, request)
		assert.NilError(t, err, "%+v", err)
		return result
	}

	cluster := testCluster()
	cluster.ObjectMeta.Name = "pgbackrest-cluster"
	cluster.ObjectMeta.Namespace = ns.Name
	cluster.Spec.Archive.PGBackRest.Metadata.Labels = map[string]string{
		"my.pgbackrest.label": "lucy",
	}

	assert.NilError(t, errors.WithStack(reconciler.Client.Create(ctx, cluster)))
	t.Cleanup(func() {
		// Remove finalizers, if any, so the namespace can terminate.
		assert.Check(t, client.IgnoreNotFound(
			reconciler.Client.Patch(ctx, cluster, client.RawPatch(
				client.Merge.Type(), []byte(`{"metadata":{"finalizers":[]}}`)))))
	})

	// Continue until instances are healthy.
	g := gomega.NewWithT(t)
	g.Eventually(func() (instances []appsv1.StatefulSet) {
		mustReconcile(t, cluster)

		list := appsv1.StatefulSetList{}
		selector, err := labels.Parse(strings.Join([]string{
			"postgres-operator.crunchydata.com/cluster=" + cluster.Name,
			"postgres-operator.crunchydata.com/instance",
		}, ","))
		assert.NilError(t, err)
		assert.NilError(t, cc.List(ctx, &list,
			client.InNamespace(cluster.Namespace),
			client.MatchingLabelsSelector{Selector: selector}))
		return list.Items
	},
		"120s", // timeout
		"5s",   // interval
	).Should(gomega.WithTransform(func(instances []appsv1.StatefulSet) int {
		ready := 0
		for _, sts := range instances {
			ready += int(sts.Status.ReadyReplicas)
		}
		return ready
	}, gomega.BeNumerically("==", len(cluster.Spec.InstanceSets))))

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
			var resourceLabels map[string]string
			var templateLabels map[string]string

			if !metav1.IsControlledBy(&u, cluster) {
				continue
			}

			t.Run(u.GetKind()+"/"+u.GetName(), func(t *testing.T) {
				switch u.GetKind() {
				case "StatefulSetList":
					var resource appsv1.StatefulSet
					assert.NilError(t, runtime.DefaultUnstructuredConverter.
						FromUnstructured(u.UnstructuredContent(), &resource))
					resourceLabels = resource.GetLabels()
					templateLabels = resource.Spec.Template.Labels
				case "DeploymentList":
					var resource appsv1.Deployment
					assert.NilError(t, runtime.DefaultUnstructuredConverter.
						FromUnstructured(u.UnstructuredContent(), &resource))
					resourceLabels = resource.GetLabels()
					templateLabels = resource.Spec.Template.GetLabels()
				default:
					resourceLabels = u.GetLabels()
				}

				assert.Equal(t, resourceLabels["my.pgbackrest.label"], "lucy")
				if templateLabels != nil {
					t.Run("template", func(t *testing.T) {
						assert.Equal(t, templateLabels["my.pgbackrest.label"], "lucy")
					})
				}
			})
		}
	}
}

func TestCustomPGBouncerLabels(t *testing.T) {
	if !strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
		t.Skip("requires a running garbage collection controller")
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
	ns.Labels = labels.Set{"postgres-operator-test": t.Name()}
	assert.NilError(t, cc.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, ns)) })

	mustReconcile := func(t *testing.T, cluster *v1beta1.PostgresCluster) reconcile.Result {
		t.Helper()
		key := client.ObjectKeyFromObject(cluster)
		request := reconcile.Request{NamespacedName: key}
		result, err := reconciler.Reconcile(ctx, request)
		assert.NilError(t, err, "%+v", err)
		return result
	}

	cluster := testCluster()
	cluster.ObjectMeta.Name = "pgbouncer-cluster"
	cluster.ObjectMeta.Namespace = ns.Name
	cluster.Spec.Proxy.PGBouncer.Metadata.Labels = map[string]string{
		"my.pgbouncer.label": "lucy",
	}

	assert.NilError(t, errors.WithStack(reconciler.Client.Create(ctx, cluster)))
	t.Cleanup(func() {
		// Remove finalizers, if any, so the namespace can terminate.
		assert.Check(t, client.IgnoreNotFound(
			reconciler.Client.Patch(ctx, cluster, client.RawPatch(
				client.Merge.Type(), []byte(`{"metadata":{"finalizers":[]}}`)))))
	})

	// Continue until instances are healthy.
	g := gomega.NewWithT(t)
	g.Eventually(func() (instances []appsv1.StatefulSet) {
		mustReconcile(t, cluster)

		list := appsv1.StatefulSetList{}
		selector, err := labels.Parse(strings.Join([]string{
			"postgres-operator.crunchydata.com/cluster=" + cluster.Name,
			"postgres-operator.crunchydata.com/instance",
		}, ","))
		assert.NilError(t, err)
		assert.NilError(t, cc.List(ctx, &list,
			client.InNamespace(cluster.Namespace),
			client.MatchingLabelsSelector{Selector: selector}))
		return list.Items
	},
		"120s", // timeout
		"5s",   // interval
	).Should(gomega.WithTransform(func(instances []appsv1.StatefulSet) int {
		ready := 0
		for _, sts := range instances {
			ready += int(sts.Status.ReadyReplicas)
		}
		return ready
	}, gomega.BeNumerically("==", len(cluster.Spec.InstanceSets))))

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
			var resourceLabels map[string]string
			var templateLabels map[string]string

			if !metav1.IsControlledBy(&u, cluster) {
				continue
			}

			t.Run(u.GetKind()+"/"+u.GetName(), func(t *testing.T) {
				switch u.GetKind() {
				case "StatefulSetList":
					var resource appsv1.StatefulSet
					assert.NilError(t, runtime.DefaultUnstructuredConverter.
						FromUnstructured(u.UnstructuredContent(), &resource))
					resourceLabels = resource.GetLabels()
					templateLabels = resource.Spec.Template.Labels
				case "DeploymentList":
					var resource appsv1.Deployment
					assert.NilError(t, runtime.DefaultUnstructuredConverter.
						FromUnstructured(u.UnstructuredContent(), &resource))
					resourceLabels = resource.GetLabels()
					templateLabels = resource.Spec.Template.GetLabels()
				default:
					resourceLabels = u.GetLabels()
				}

				assert.Equal(t, resourceLabels["my.pgbouncer.label"], "lucy")
				if templateLabels != nil {
					t.Run("template", func(t *testing.T) {
						assert.Equal(t, templateLabels["my.pgbouncer.label"], "lucy")
					})
				}
			})
		}
	}
}
