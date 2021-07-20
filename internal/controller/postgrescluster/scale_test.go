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
	"os"
	"strings"
	"testing"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
	"github.com/onsi/gomega"
	"go.opentelemetry.io/otel"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func Int32(v int32) *int32 { return &v }

func TestScaleDown(t *testing.T) {
	if !strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
		t.Skip("requires a running garbage collection controller")
	}
	// TODO: Update tests that include envtest package to better handle
	// running in parallel
	// t.Parallel()
	env, cc, config := setupTestEnv(t, ControllerName)
	t.Cleanup(func() { teardownTestEnv(t, env) })

	reconciler := &Reconciler{}
	ctx, cancel := setupManager(t, config, func(mgr manager.Manager) {
		reconciler = &Reconciler{
			Client:   cc,
			Owner:    client.FieldOwner(t.Name()),
			Recorder: new(record.FakeRecorder),
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

	// Defines a volume claim spec that can be used to create instances
	volumeClaimSpec := v1.PersistentVolumeClaimSpec{
		AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
		Resources: v1.ResourceRequirements{
			Requests: map[v1.ResourceName]resource.Quantity{
				v1.ResourceStorage: resource.MustParse("1Gi"),
			},
		},
	}

	// Defines a base cluster spec that can be used by tests to generate a
	// cluster with an expected number of instances
	baseCluster := v1beta1.PostgresCluster{
		Spec: v1beta1.PostgresClusterSpec{
			PostgresVersion: 13,
			Image:           CrunchyPostgresHAImage,
			Backups: v1beta1.Backups{
				PGBackRest: v1beta1.PGBackRestArchive{
					Image: CrunchyPGBackRestImage,
					RepoHost: &v1beta1.PGBackRestRepoHost{
						Dedicated: &v1beta1.DedicatedRepo{},
					},
					Repos: []v1beta1.PGBackRestRepo{{
						Name:   "repo1",
						Volume: &v1beta1.RepoPVC{VolumeClaimSpec: volumeClaimSpec},
					}},
				},
			},
		},
	}

	for _, test := range []struct {
		name                   string
		createSet              []v1beta1.PostgresInstanceSetSpec
		createRunningInstances int
		updateSet              []v1beta1.PostgresInstanceSetSpec
		updateRunningInstances int
		primaryTest            func(*testing.T, string, string)
	}{
		{
			name: "OneSet",
			// Remove a single instance set from the spec
			createSet: []v1beta1.PostgresInstanceSetSpec{{
				Name:                "daisy",
				Replicas:            Int32(1),
				DataVolumeClaimSpec: volumeClaimSpec,
			}, {
				Name:                "max",
				Replicas:            Int32(1),
				DataVolumeClaimSpec: volumeClaimSpec,
			}},
			createRunningInstances: 2,
			updateSet: []v1beta1.PostgresInstanceSetSpec{{
				Name:                "daisy",
				Replicas:            Int32(1),
				DataVolumeClaimSpec: volumeClaimSpec,
			}},
			updateRunningInstances: 1,
		}, {
			name: "InstancesWithOneSet",
			// Decrease the number of replicas that are defined for one instance set
			createSet: []v1beta1.PostgresInstanceSetSpec{{
				Name:                "daisy",
				Replicas:            Int32(2),
				DataVolumeClaimSpec: volumeClaimSpec,
			}},
			createRunningInstances: 2,
			updateSet: []v1beta1.PostgresInstanceSetSpec{{
				Name:                "daisy",
				Replicas:            Int32(1),
				DataVolumeClaimSpec: volumeClaimSpec,
			}},
			updateRunningInstances: 1,
			primaryTest: func(t *testing.T, old, new string) {
				assert.Equal(t, old, new, "Primary instance should not have changed")
			},
		}, {
			name: "InstancesWithTwoSets",
			// Decrease the number of replicas that are defined for one instance set
			// and ensure that the other instance set is unchanged
			createSet: []v1beta1.PostgresInstanceSetSpec{{
				Name:                "daisy",
				Replicas:            Int32(2),
				DataVolumeClaimSpec: volumeClaimSpec,
			}, {
				Name:                "max",
				Replicas:            Int32(1),
				DataVolumeClaimSpec: volumeClaimSpec,
			}},
			createRunningInstances: 3,
			updateSet: []v1beta1.PostgresInstanceSetSpec{{
				Name:                "daisy",
				Replicas:            Int32(1),
				DataVolumeClaimSpec: volumeClaimSpec,
			}, {
				Name:                "max",
				Replicas:            Int32(1),
				DataVolumeClaimSpec: volumeClaimSpec,
			}},
			updateRunningInstances: 2,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var oldPrimaryInstanceName string
			var newPrimaryInstanceName string

			g := gomega.NewWithT(t)

			cluster := baseCluster.DeepCopy()
			cluster.ObjectMeta.Name = strings.ToLower(test.name)
			cluster.ObjectMeta.Namespace = ns.Name
			cluster.Spec.InstanceSets = test.createSet

			assert.NilError(t, reconciler.Client.Create(ctx, cluster))
			t.Cleanup(func() {
				// Remove finalizers, if any, so the namespace can terminate.
				assert.Check(t, client.IgnoreNotFound(
					reconciler.Client.Patch(ctx, cluster, client.RawPatch(
						client.Merge.Type(), []byte(`{"metadata":{"finalizers":[]}}`)))))

				// Set Cluster to delete after test
				assert.Check(t, reconciler.Client.Delete(ctx, cluster))
			})

			// Continue until instances are healthy.
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
				"60s", // timeout
				"1s",  // interval
			).Should(gomega.WithTransform(func(instances []appsv1.StatefulSet) int {
				ready := 0
				for _, sts := range instances {
					ready += int(sts.Status.ReadyReplicas)
				}
				return ready
			}, gomega.BeNumerically("==", test.createRunningInstances)))

			if test.primaryTest != nil {
				// Grab the old primary name to use later
				primaryPod := v1.PodList{}
				g.Eventually(func() (podLen int) {
					primarySelector, err := naming.AsSelector(metav1.LabelSelector{
						MatchLabels: map[string]string{
							naming.LabelCluster: cluster.Name,
							naming.LabelRole:    "master",
						},
					})
					assert.NilError(t, err)
					assert.NilError(t, cc.List(ctx, &primaryPod,
						client.InNamespace(cluster.Namespace),
						client.MatchingLabelsSelector{Selector: primarySelector}))
					return len(primaryPod.Items)
				},
					"10s", // timeout
					"1s",  // interval
				).Should(gomega.BeNumerically("==", 1))
				oldPrimaryInstanceName = primaryPod.Items[0].Labels[naming.LabelInstance]
			}

			// The cluster is running with the correct number of Ready Replicas
			// Now we can update the cluster by applying changes to the spec
			copy := cluster.DeepCopy()
			copy.Spec.InstanceSets = test.updateSet

			err := reconciler.Client.Patch(ctx, copy, client.MergeFrom(cluster))
			assert.NilError(t, err, "Error reconciling cluster")

			// Run the reconcile loop until we have the expected number of
			// Ready Replicas
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
				"60s", // timeout
				"1s",  // interval
			).Should(gomega.WithTransform(func(instances []appsv1.StatefulSet) int {
				ready := 0
				for _, sts := range instances {
					ready += int(sts.Status.ReadyReplicas)
				}
				return ready
			}, gomega.BeNumerically("==", test.updateRunningInstances)))

			// In the update case we need to ensure that the pods have deleted
			g.Eventually(func() (podLen int) {
				list := v1.PodList{}
				selector, err := labels.Parse(strings.Join([]string{
					"postgres-operator.crunchydata.com/cluster=" + cluster.Name,
					"postgres-operator.crunchydata.com/instance",
				}, ","))
				assert.NilError(t, err)
				assert.NilError(t, cc.List(ctx, &list,
					client.InNamespace(cluster.Namespace),
					client.MatchingLabelsSelector{Selector: selector}))
				return len(list.Items)
			},
				"30s", // timeout
				"1s",  // interval
			).Should(gomega.BeNumerically("==", test.updateRunningInstances))

			if test.primaryTest != nil {
				// If this is a primary test grab the updated primary
				primaryPod := v1.PodList{}
				g.Eventually(func() (podLen int) {
					primarySelector, err := naming.AsSelector(metav1.LabelSelector{
						MatchLabels: map[string]string{
							naming.LabelCluster: cluster.Name,
							naming.LabelRole:    "master",
						},
					})
					assert.NilError(t, err)
					assert.NilError(t, cc.List(ctx, &primaryPod,
						client.InNamespace(cluster.Namespace),
						client.MatchingLabelsSelector{Selector: primarySelector}))
					return len(primaryPod.Items)
				},
					"10s", // timeout
					"1s",  // interval
				).Should(gomega.BeNumerically("<=", 1)) // Can have 1 or 0 primaries

				if len(primaryPod.Items) == 1 {
					newPrimaryInstanceName = primaryPod.Items[0].Labels[naming.LabelInstance]
				}

				t.Run("Primary Test", func(t *testing.T) {
					test.primaryTest(t, oldPrimaryInstanceName, newPrimaryInstanceName)
				})
			}

			// The cluster has the correct number of total instances.
			// Does each instance set have the correct number of replicas?
			instances := &appsv1.StatefulSetList{}
			selector, err := naming.AsSelector(naming.ClusterInstances(cluster.Name))
			assert.NilError(t, err)
			assert.NilError(t, reconciler.Client.List(ctx, instances,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selector},
			))

			// Once again we make sure that the number of instances in the
			// environment reflect the number we expect
			assert.Equal(t, len(instances.Items), test.updateRunningInstances)

			// Group the instances by the instance set label and count the
			// replicas for each set
			replicas := map[string]int32{}
			for _, instance := range instances.Items {
				replicas[instance.Labels[naming.LabelInstanceSet]]++
			}

			// Ensure that each set has the number of replicas defined in
			// the test
			for _, set := range test.updateSet {
				assert.Equal(t, replicas[set.Name], *set.Replicas)
				delete(replicas, set.Name)
			}

			// Finally make sure that we don't have any extra sets
			assert.Equal(t, len(replicas), 0)
		})
	}
}
