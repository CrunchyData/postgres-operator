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
	"errors"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestReconcilePostgresVolumes(t *testing.T) {
	ctx := context.Background()
	tEnv, tClient, _ := setupTestEnv(t, ControllerName)
	t.Cleanup(func() { teardownTestEnv(t, tEnv) })

	reconciler := &Reconciler{
		Client: tClient,
		Owner:  client.FieldOwner(t.Name()),
	}

	ns := &corev1.Namespace{}
	ns.GenerateName = "postgres-operator-test-"
	assert.NilError(t, tClient.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, tClient.Delete(ctx, ns)) })

	cluster := testCluster()
	cluster.Namespace = ns.Name

	assert.NilError(t, tClient.Create(ctx, cluster))
	t.Cleanup(func() { assert.Check(t, tClient.Delete(ctx, cluster)) })

	spec := &v1beta1.PostgresInstanceSetSpec{}
	assert.NilError(t, yaml.Unmarshal([]byte(`{
		name: "some-instance",
		volumeClaimSpec: {
			accessModes: [ReadWriteOnce],
			resources: { requests: { storage: 1Gi } },
			storageClassName: "storage-class-for-data",
		},
	}`), spec))

	instance := &appsv1.StatefulSet{ObjectMeta: naming.GenerateInstance(cluster, spec)}

	t.Run("DataVolume", func(t *testing.T) {
		pvc, err := reconciler.reconcilePostgresDataVolume(ctx, cluster, spec, instance)
		assert.NilError(t, err)

		assert.Assert(t, metav1.IsControlledBy(pvc, cluster))

		assert.Equal(t, pvc.Labels[naming.LabelCluster], cluster.Name)
		assert.Equal(t, pvc.Labels[naming.LabelInstance], instance.Name)
		assert.Equal(t, pvc.Labels[naming.LabelInstanceSet], spec.Name)
		assert.Equal(t, pvc.Labels[naming.LabelRole], "pgdata")

		assert.Assert(t, marshalMatches(pvc.Spec, `
accessModes:
- ReadWriteOnce
resources:
  requests:
    storage: 1Gi
storageClassName: storage-class-for-data
volumeMode: Filesystem
		`))
	})

	t.Run("WALVolume", func(t *testing.T) {
		observed := &Instance{}

		t.Run("None", func(t *testing.T) {
			pvc, err := reconciler.reconcilePostgresWALVolume(ctx, cluster, spec, instance, observed)
			assert.NilError(t, err)
			assert.Assert(t, pvc == nil)
		})

		t.Run("Specified", func(t *testing.T) {
			spec := spec.DeepCopy()
			assert.NilError(t, yaml.Unmarshal([]byte(`{
				walVolumeClaimSpec: {
					accessModes: [ReadWriteMany],
					resources: { requests: { storage: 2Gi } },
					storageClassName: "storage-class-for-wal",
				},
			}`), spec))

			pvc, err := reconciler.reconcilePostgresWALVolume(ctx, cluster, spec, instance, observed)
			assert.NilError(t, err)

			assert.Assert(t, metav1.IsControlledBy(pvc, cluster))

			assert.Equal(t, pvc.Labels[naming.LabelCluster], cluster.Name)
			assert.Equal(t, pvc.Labels[naming.LabelInstance], instance.Name)
			assert.Equal(t, pvc.Labels[naming.LabelInstanceSet], spec.Name)
			assert.Equal(t, pvc.Labels[naming.LabelRole], "pgwal")

			assert.Assert(t, marshalMatches(pvc.Spec, `
accessModes:
- ReadWriteMany
resources:
  requests:
    storage: 2Gi
storageClassName: storage-class-for-wal
volumeMode: Filesystem
			`))

			t.Run("Removed", func(t *testing.T) {
				spec := spec.DeepCopy()
				spec.WALVolumeClaimSpec = nil

				ignoreTypeMeta := cmpopts.IgnoreFields(corev1.PersistentVolumeClaim{}, "TypeMeta")

				t.Run("FilesAreNotSafe", func(t *testing.T) {
					// No pods; expect no changes to the PVC.
					returned, err := reconciler.reconcilePostgresWALVolume(ctx, cluster, spec, instance, observed)
					assert.NilError(t, err)
					assert.DeepEqual(t, returned, pvc, ignoreTypeMeta)

					// Not running; expect no changes to the PVC.
					observed.Pods = []*corev1.Pod{{}}

					returned, err = reconciler.reconcilePostgresWALVolume(ctx, cluster, spec, instance, observed)
					assert.NilError(t, err)
					assert.DeepEqual(t, returned, pvc, ignoreTypeMeta)

					// Cannot find WAL files; expect no changes to the PVC.
					observed.Pods[0].Namespace, observed.Pods[0].Name = "pod-ns", "pod-name"
					observed.Pods[0].Status.ContainerStatuses = []corev1.ContainerStatus{{
						Name: naming.ContainerDatabase,
					}}
					observed.Pods[0].Status.ContainerStatuses[0].State.Running =
						new(corev1.ContainerStateRunning)

					expected := errors.New("flop")
					reconciler.PodExec = func(
						namespace, pod, container string,
						_ io.Reader, _, _ io.Writer, command ...string,
					) error {
						assert.Equal(t, namespace, "pod-ns")
						assert.Equal(t, pod, "pod-name")
						assert.Equal(t, container, "database")
						assert.DeepEqual(t, command,
							[]string{"bash", "-ceu", "--", `exec realpath "${PGDATA}/pg_wal"`})
						return expected
					}

					returned, err = reconciler.reconcilePostgresWALVolume(ctx, cluster, spec, instance, observed)
					assert.Equal(t, expected, errors.Unwrap(err), "expected pod exec")
					assert.DeepEqual(t, returned, pvc, ignoreTypeMeta)

					// Files are in the wrong place; expect no changes to the PVC.
					reconciler.PodExec = func(
						_, _, _ string, _ io.Reader, stdout, _ io.Writer, _ ...string,
					) error {
						assert.Assert(t, stdout != nil)
						_, err := stdout.Write([]byte("some-place\n"))
						assert.NilError(t, err)
						return nil
					}

					returned, err = reconciler.reconcilePostgresWALVolume(ctx, cluster, spec, instance, observed)
					assert.NilError(t, err)
					assert.DeepEqual(t, returned, pvc, ignoreTypeMeta)
				})

				t.Run("FilesAreSafe", func(t *testing.T) {
					// Files are seen in the directory intended by the specification.
					observed.Pods = []*corev1.Pod{{}}
					observed.Pods[0].Status.ContainerStatuses = []corev1.ContainerStatus{{
						Name: naming.ContainerDatabase,
					}}
					observed.Pods[0].Status.ContainerStatuses[0].State.Running =
						new(corev1.ContainerStateRunning)

					reconciler.PodExec = func(
						_, _, _ string, _ io.Reader, stdout, _ io.Writer, _ ...string,
					) error {
						assert.Assert(t, stdout != nil)
						_, err := stdout.Write([]byte(postgres.WALDirectory(cluster, spec) + "\n"))
						assert.NilError(t, err)
						return nil
					}

					returned, err := reconciler.reconcilePostgresWALVolume(ctx, cluster, spec, instance, observed)
					assert.NilError(t, err)
					assert.Assert(t, returned == nil)

					key, fetched := client.ObjectKeyFromObject(pvc), &corev1.PersistentVolumeClaim{}
					assert.NilError(t, tClient.Get(ctx, key, fetched))
					assert.Assert(t, fetched.DeletionTimestamp != nil, "expected deleted")

					// Pods will redeploy while the PVC is scheduled for deletion.
					observed.Pods = nil
					returned, err = reconciler.reconcilePostgresWALVolume(ctx, cluster, spec, instance, observed)
					assert.NilError(t, err)
					assert.Assert(t, returned == nil)
				})
			})
		})
	})
}
