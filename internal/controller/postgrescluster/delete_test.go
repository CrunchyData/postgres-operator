//go:build envtest
// +build envtest

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

package postgrescluster

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/internal/patroni"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestReconcilerHandleDelete(t *testing.T) {
	if !strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
		t.Skip("requires a running garbage collection controller")
	}

	ctx := context.Background()
	env, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 2)

	ns := setupNamespace(t, cc)
	reconciler := Reconciler{
		Client:   cc,
		Owner:    client.FieldOwner(t.Name()),
		Recorder: new(record.FakeRecorder),
		Tracer:   otel.Tracer(t.Name()),
	}

	var err error
	reconciler.PodExec, err = newPodExecutor(env.Config)
	assert.NilError(t, err)

	mustReconcile := func(t *testing.T, cluster *v1beta1.PostgresCluster) reconcile.Result {
		t.Helper()
		key := client.ObjectKeyFromObject(cluster)
		request := reconcile.Request{NamespacedName: key}
		result, err := reconciler.Reconcile(ctx, request)
		assert.NilError(t, err, "%+v", err)
		return result
	}

	for _, test := range []struct {
		name         string
		beforeCreate func(*testing.T, *v1beta1.PostgresCluster)
		beforeDelete func(*testing.T, *v1beta1.PostgresCluster)
		propagation  metav1.DeletionPropagation

		waitForRunningInstances int32
	}{
		// Normal delete of a healthly cluster.
		{
			name: "Background", propagation: metav1.DeletePropagationBackground,
			waitForRunningInstances: 2,
		},
		// TODO(cbandy): metav1.DeletePropagationForeground

		// Normal delete of a healthy cluster after a failover.
		{
			name: "AfterFailover", propagation: metav1.DeletePropagationBackground,
			waitForRunningInstances: 2,

			beforeDelete: func(t *testing.T, cluster *v1beta1.PostgresCluster) {
				list := corev1.PodList{}
				selector, err := labels.Parse(strings.Join([]string{
					"postgres-operator.crunchydata.com/cluster=" + cluster.Name,
					"postgres-operator.crunchydata.com/instance",
				}, ","))
				assert.NilError(t, err)
				assert.NilError(t, cc.List(ctx, &list,
					client.InNamespace(cluster.Namespace),
					client.MatchingLabelsSelector{Selector: selector}))

				var primary *corev1.Pod
				var replica *corev1.Pod
				for i := range list.Items {
					if list.Items[i].Labels["postgres-operator.crunchydata.com/role"] == "replica" {
						replica = &list.Items[i]
					} else {
						primary = &list.Items[i]
					}
				}

				if true &&
					assert.Check(t, primary != nil, "expected to find a primary in %+v", list.Items) &&
					assert.Check(t, replica != nil, "expected to find a replica in %+v", list.Items) {
					success, err := patroni.Executor(
						func(_ context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string) error {
							return reconciler.PodExec(replica.Namespace, replica.Name, "database", stdin, stdout, stderr, command...)
						},
					).ChangePrimaryAndWait(ctx, primary.Name, replica.Name)

					assert.NilError(t, err)
					assert.Assert(t, success)
				}
			},
		},

		// Normal delete of cluster that could never run PostgreSQL.
		{
			name: "NeverRunning", propagation: metav1.DeletePropagationBackground,
			waitForRunningInstances: 0,

			beforeCreate: func(_ *testing.T, cluster *v1beta1.PostgresCluster) {
				cluster.Spec.Image = "example.com/does-not-exist"
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cluster := &v1beta1.PostgresCluster{}
			assert.NilError(t, yaml.Unmarshal([]byte(`{
				spec: {
					postgresVersion: 13,
					instances: [
						{
							replicas: 2,
							dataVolumeClaimSpec: {
								accessModes: [ReadWriteOnce],
								resources: { requests: { storage: 1Gi } },
							},
						},
					],
					backups: { 
						pgbackrest: {
							repos: [{
								name: repo1,
								volume: {
									volumeClaimSpec: {
										accessModes: [ReadWriteOnce],
										resources: { requests: { storage: 1Gi } },
									},
								},
							}],
						},
					},
				},
			}`), cluster))

			cluster.Namespace = ns.Name
			cluster.Name = strings.ToLower(test.name)
			cluster.Spec.Image = CrunchyPostgresHAImage
			cluster.Spec.Backups.PGBackRest.Image = CrunchyPGBackRestImage

			if test.beforeCreate != nil {
				test.beforeCreate(t, cluster)
			}

			assert.NilError(t, cc.Create(ctx, cluster))

			t.Cleanup(func() {
				// Remove finalizers, if any, so the namespace can terminate.
				assert.Check(t, client.IgnoreNotFound(
					cc.Patch(ctx, cluster, client.RawPatch(
						client.Merge.Type(), []byte(`{"metadata":{"finalizers":[]}}`)))))
			})

			// Start cluster.
			mustReconcile(t, cluster)

			assert.NilError(t,
				cc.Get(ctx, client.ObjectKeyFromObject(cluster), cluster))
			assert.Assert(t,
				sets.NewString(cluster.Finalizers...).
					Has("postgres-operator.crunchydata.com/finalizer"),
				"cluster should immediately have a finalizer")

			// Continue until instances are healthy.
			if ready := int32(0); !assert.Check(t,
				wait.Poll(time.Second, Scale(time.Minute), func() (bool, error) {
					mustReconcile(t, cluster)
					assert.NilError(t, cc.Get(ctx, client.ObjectKeyFromObject(cluster), cluster))

					ready = 0
					for _, set := range cluster.Status.InstanceSets {
						ready += set.ReadyReplicas
					}
					return ready >= test.waitForRunningInstances, nil
				}), "expected %v instances to be ready, got: %v", test.waitForRunningInstances, ready,
			) {
				t.FailNow()
			}

			if test.beforeDelete != nil {
				test.beforeDelete(t, cluster)
			}

			switch test.propagation {
			case metav1.DeletePropagationBackground:
				// Background deletion is the default for custom resources.
				// - https://issue.k8s.io/81628
				assert.NilError(t, cc.Delete(ctx, cluster))
			default:
				assert.NilError(t, cc.Delete(ctx, cluster,
					client.PropagationPolicy(test.propagation)))
			}

			// Stop cluster.
			result := mustReconcile(t, cluster)

			// If things started running, then they should stop in a certain order.
			if test.waitForRunningInstances > 0 {

				// Replicas should stop first, leaving just the one primary.
				var instances []corev1.Pod
				assert.NilError(t, wait.Poll(time.Second, Scale(time.Minute), func() (bool, error) {
					if result.Requeue {
						result = mustReconcile(t, cluster)
					}

					list := corev1.PodList{}
					selector, err := labels.Parse(strings.Join([]string{
						"postgres-operator.crunchydata.com/cluster=" + cluster.Name,
						"postgres-operator.crunchydata.com/instance",
					}, ","))
					assert.NilError(t, err)
					assert.NilError(t, cc.List(ctx, &list,
						client.InNamespace(cluster.Namespace),
						client.MatchingLabelsSelector{Selector: selector}))

					instances = list.Items

					// Patroni doesn't use "primary" to identify the primary.
					return len(instances) == 1 &&
						instances[0].Labels["postgres-operator.crunchydata.com/role"] == "master", nil
				}), "expected one instance, got:\n%+v", instances)

				// Patroni DCS objects should not be deleted yet.
				{
					list := corev1.EndpointsList{}
					selector, err := labels.Parse(strings.Join([]string{
						"postgres-operator.crunchydata.com/cluster=" + cluster.Name,
						"postgres-operator.crunchydata.com/patroni",
					}, ","))
					assert.NilError(t, err)
					assert.NilError(t, cc.List(ctx, &list,
						client.InNamespace(cluster.Namespace),
						client.MatchingLabelsSelector{Selector: selector}))

					assert.Assert(t, len(list.Items) >= 2, // config + leader
						"expected Patroni DCS objects to remain, there are %v",
						len(list.Items))

					// Endpoints are deleted differently than other resources, and
					// Patroni might have recreated them to stay alive. Check that
					// they are all from before the cluster delete operation.
					// - https://issue.k8s.io/99407
					assert.NilError(t,
						cc.Get(ctx, client.ObjectKeyFromObject(cluster), cluster))

					for _, endpoints := range list.Items {
						assert.Assert(t,
							endpoints.CreationTimestamp.Time.Before(cluster.DeletionTimestamp.Time),
							`expected %q to be after %+v`, cluster.DeletionTimestamp, endpoints)
					}
				}
			}

			// Continue until cluster is gone.
			assert.NilError(t, wait.Poll(time.Second, Scale(time.Minute), func() (bool, error) {
				mustReconcile(t, cluster)

				err := cc.Get(ctx, client.ObjectKeyFromObject(cluster), cluster)
				return apierrors.IsNotFound(err), client.IgnoreNotFound(err)
			}), "expected cluster to be deleted, got:\n%+v", *cluster)

			var endpoints []corev1.Endpoints
			assert.NilError(t, wait.Poll(time.Second, Scale(time.Minute/3), func() (bool, error) {
				list := corev1.EndpointsList{}
				selector, err := labels.Parse(strings.Join([]string{
					"postgres-operator.crunchydata.com/cluster=" + cluster.Name,
					"postgres-operator.crunchydata.com/patroni",
				}, ","))
				assert.NilError(t, err)
				assert.NilError(t, cc.List(ctx, &list,
					client.InNamespace(cluster.Namespace),
					client.MatchingLabelsSelector{Selector: selector}))

				endpoints = list.Items

				return len(endpoints) == 0, nil
			}), "Patroni DCS objects should be gone, got:\n%+v", endpoints)
		})
	}
}

func TestReconcilerHandleDeleteNamespace(t *testing.T) {
	if !strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
		t.Skip("requires a running garbage collection controller")
	}

	ctx := context.Background()
	env, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 2)

	ns := setupNamespace(t, cc)

	var mm struct {
		manager.Manager
		Context context.Context
		Error   chan error
		Stop    context.CancelFunc
	}

	var err error
	mm.Context, mm.Stop = context.WithCancel(context.Background())
	mm.Error = make(chan error, 1)
	mm.Manager, err = manager.New(env.Config, manager.Options{
		Namespace: ns.Name,
		Scheme:    cc.Scheme(),

		HealthProbeBindAddress: "0", // disable
		MetricsBindAddress:     "0", // disable
	})
	assert.NilError(t, err)

	reconciler := Reconciler{
		Client:   mm.GetClient(),
		Owner:    client.FieldOwner(t.Name()),
		Recorder: new(record.FakeRecorder),
		Tracer:   otel.Tracer(t.Name()),
	}
	assert.NilError(t, reconciler.SetupWithManager(mm.Manager))

	go func() { mm.Error <- mm.Start(mm.Context) }()
	t.Cleanup(func() { mm.Stop(); assert.Check(t, <-mm.Error) })

	cluster := &v1beta1.PostgresCluster{}
	assert.NilError(t, yaml.Unmarshal([]byte(`{
		spec: {
			postgresVersion: 13,
			instances: [
				{
					replicas: 2,
					dataVolumeClaimSpec: {
						accessModes: [ReadWriteOnce],
						resources: { requests: { storage: 1Gi } },
					},
				},
			],
			backups: { 
				pgbackrest: {
					repos: [{
						name: repo1,
						volume: {
							volumeClaimSpec: {
								accessModes: [ReadWriteOnce],
								resources: { requests: { storage: 1Gi } },
							},
						},
					}],
				},
			},
		},
	}`), cluster))

	cluster.Namespace = ns.Name
	cluster.Name = strings.ToLower("DeleteNamespace")
	cluster.Spec.Image = CrunchyPostgresHAImage
	cluster.Spec.Backups.PGBackRest.Image = CrunchyPGBackRestImage

	assert.NilError(t, cc.Create(ctx, cluster))

	t.Cleanup(func() {
		// Remove finalizers, if any, so the namespace can terminate.
		assert.Check(t, client.IgnoreNotFound(
			cc.Patch(ctx, cluster, client.RawPatch(
				client.Merge.Type(), []byte(`{"metadata":{"finalizers":[]}}`)))))
	})

	// Wait until instances are healthy.
	if ready := int32(0); !assert.Check(t,
		wait.Poll(time.Second, Scale(time.Minute), func() (bool, error) {
			assert.NilError(t, cc.Get(ctx, client.ObjectKeyFromObject(cluster), cluster))

			ready = 0
			for _, set := range cluster.Status.InstanceSets {
				ready += set.ReadyReplicas
			}
			return ready >= 2, nil
		}), "expected 2 instances to be ready, got: %v", ready,
	) {
		t.FailNow()
	}

	// Delete the namespace.
	assert.NilError(t, cc.Delete(ctx, ns))

	assert.NilError(t, wait.PollImmediate(time.Second, Scale(time.Minute), func() (bool, error) {
		err := cc.Get(ctx, client.ObjectKeyFromObject(cluster), cluster)
		return apierrors.IsNotFound(err), client.IgnoreNotFound(err)
	}), "expected cluster to be deleted, got:\n%+v", *cluster)

	// Kubernetes will continue to remove things after the PostgresCluster is gone.
	// In some cases, a Pod might get stuck in a deleted-and-creating state.
	// Conditions in the Namespace status indicate what is going on.
	var namespace corev1.Namespace
	assert.NilError(t, wait.PollImmediate(time.Second, Scale(3*time.Minute), func() (bool, error) {
		err := cc.Get(ctx, client.ObjectKeyFromObject(ns), &namespace)
		return apierrors.IsNotFound(err), client.IgnoreNotFound(err)
	}), "expected namespace to be deleted, got status:\n%+v", namespace.Status)
}
