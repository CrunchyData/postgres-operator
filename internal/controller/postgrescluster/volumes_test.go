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
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestPersistentVolumeClaimLimitations(t *testing.T) {
	if !strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
		t.Skip("requires a running persistent volume controller")
	}

	ctx := context.Background()
	_, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 1)

	ns := setupNamespace(t, cc)

	// Stub to see that handlePersistentVolumeClaimError returns nil.
	cluster := new(v1beta1.PostgresCluster)
	reconciler := &Reconciler{
		Recorder: new(record.FakeRecorder),
	}

	apiErrorStatus := func(t testing.TB, err error) metav1.Status {
		t.Helper()
		var status apierrors.APIStatus
		assert.Assert(t, errors.As(err, &status))
		return status.Status()
	}

	// NOTE(cbandy): use multiples of 1Gi below to stay compatible with AWS, GCP, etc.

	// Statically provisioned volumes cannot be resized. The API response depends
	// on the phase of the volume claim.
	t.Run("StaticNoResize", func(t *testing.T) {
		// A static PVC is one with a present-and-blank storage class.
		// - https://docs.k8s.io/concepts/storage/persistent-volumes/#static
		// - https://docs.k8s.io/concepts/storage/persistent-volumes/#class-1
		base := &corev1.PersistentVolumeClaim{}
		assert.NilError(t, yaml.Unmarshal([]byte(`{
			spec: {
				storageClassName: "",
				accessModes: [ReadWriteOnce],
				selector: { matchLabels: { postgres-operator-test: static-no-resize } },
				resources: { requests: { storage: 2Gi } },
			},
		}`), base))
		base.Namespace = ns.Name

		t.Run("Pending", func(t *testing.T) {
			// No persistent volume for this claim.
			pvc := base.DeepCopy()
			pvc.Name = "static-pvc-pending"
			assert.NilError(t, cc.Create(ctx, pvc))
			t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, pvc)) })

			// Not able to shrink the storage request.
			pvc.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("1Gi")

			err := cc.Update(ctx, pvc)
			assert.Assert(t, apierrors.IsInvalid(err), "expected Invalid, got\n%#v", err)
			assert.ErrorContains(t, err, "less than previous")
			assert.ErrorContains(t, err, pvc.Name, "expected mention of the object")

			status := apiErrorStatus(t, err)
			assert.Assert(t, status.Details != nil)
			assert.Assert(t, len(status.Details.Causes) != 0)
			assert.Equal(t, status.Details.Causes[0].Field, "spec")
			assert.Equal(t, status.Details.Causes[0].Type, metav1.CauseType(field.ErrorTypeForbidden))

			assert.NilError(t, reconciler.handlePersistentVolumeClaimError(cluster, err))

			// Not able to grow the storage request.
			pvc.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("4Gi")

			err = cc.Update(ctx, pvc)
			assert.Assert(t, apierrors.IsInvalid(err), "expected Invalid, got\n%#v", err)
			assert.ErrorContains(t, err, "bound claim")
			assert.ErrorContains(t, err, pvc.Name, "expected mention of the object")

			status = apiErrorStatus(t, err)
			assert.Assert(t, status.Details != nil)
			assert.Assert(t, len(status.Details.Causes) != 0)
			assert.Equal(t, status.Details.Causes[0].Field, "spec")
			assert.Equal(t, status.Details.Causes[0].Type, metav1.CauseType(field.ErrorTypeForbidden))

			assert.NilError(t, reconciler.handlePersistentVolumeClaimError(cluster, err))
		})

		t.Run("Bound", func(t *testing.T) {
			// A persistent volume that will match the claim.
			pv := &corev1.PersistentVolume{}
			assert.NilError(t, yaml.Unmarshal([]byte(`{
				metadata: {
					generateName: postgres-operator-test-,
					labels: { postgres-operator-test: static-no-resize },
				},
				spec: {
					accessModes: [ReadWriteOnce],
					capacity: { storage: 4Gi },
					hostPath: { path: /tmp },
					persistentVolumeReclaimPolicy: Delete,
				},
			}`), pv))

			assert.NilError(t, cc.Create(ctx, pv))
			t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, pv)) })

			assert.NilError(t, wait.PollImmediate(time.Second, Scale(10*time.Second), func() (bool, error) {
				err := cc.Get(ctx, client.ObjectKeyFromObject(pv), pv)
				return pv.Status.Phase != corev1.VolumePending, err
			}), "expected Available, got %#v", pv.Status)

			pvc := base.DeepCopy()
			pvc.Name = "static-pvc-bound"
			assert.NilError(t, cc.Create(ctx, pvc))
			t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, pvc)) })

			assert.NilError(t, wait.PollImmediate(time.Second, Scale(10*time.Second), func() (bool, error) {
				err := cc.Get(ctx, client.ObjectKeyFromObject(pvc), pvc)
				return pvc.Status.Phase != corev1.ClaimPending, err
			}), "expected Bound, got %#v", pvc.Status)

			// Not able to shrink the storage request.
			pvc.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("1Gi")

			err := cc.Update(ctx, pvc)
			assert.Assert(t, apierrors.IsInvalid(err), "expected Invalid, got\n%#v", err)
			assert.ErrorContains(t, err, "less than previous")
			assert.ErrorContains(t, err, pvc.Name, "expected mention of the object")

			status := apiErrorStatus(t, err)
			assert.Assert(t, status.Details != nil)
			assert.Assert(t, len(status.Details.Causes) != 0)
			assert.Equal(t, status.Details.Causes[0].Field, "spec.resources.requests.storage")
			assert.Equal(t, status.Details.Causes[0].Type, metav1.CauseType(field.ErrorTypeForbidden))

			assert.NilError(t, reconciler.handlePersistentVolumeClaimError(cluster, err))

			// Not able to grow the storage request.
			pvc.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("4Gi")

			err = cc.Update(ctx, pvc)
			assert.Assert(t, apierrors.IsForbidden(err), "expected Forbidden, got\n%#v", err)
			assert.ErrorContains(t, err, "only dynamic")
			assert.ErrorContains(t, err, pvc.Name, "expected mention of the object")

			assert.NilError(t, reconciler.handlePersistentVolumeClaimError(cluster, err))
		})
	})

	// Dynamically provisioned volumes can be resized under certain conditions.
	// The API response depends on the phase of the volume claim.
	// - https://releases.k8s.io/v1.21.0/plugin/pkg/admission/storage/persistentvolume/resize/admission.go
	t.Run("Dynamic", func(t *testing.T) {
		// Create a claim without a storage class to detect the default.
		find := &corev1.PersistentVolumeClaim{}
		assert.NilError(t, yaml.Unmarshal([]byte(`{
			spec: {
				accessModes: [ReadWriteOnce],
				selector: { matchLabels: { postgres-operator-test: find-dynamic } },
				resources: { requests: { storage: 1Gi } },
			},
		}`), find))
		find.Namespace, find.Name = ns.Name, "find-dynamic"

		assert.NilError(t, cc.Create(ctx, find))
		t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, find)) })

		if find.Spec.StorageClassName == nil {
			t.Skip("requires a default storage class and expansion controller")
		}

		base := &storagev1.StorageClass{}
		base.Name = *find.Spec.StorageClassName

		if err := cc.Get(ctx, client.ObjectKeyFromObject(base), base); err != nil {
			t.Skipf("requires a default storage class, got\n%#v", err)
		}

		t.Run("Pending", func(t *testing.T) {
			// A storage class that will not bind until there is a pod.
			sc := base.DeepCopy()
			sc.ObjectMeta = metav1.ObjectMeta{
				GenerateName: "postgres-operator-test-",
				Labels: map[string]string{
					"postgres-operator-test": "pvc-limitations-pending",
				},
			}
			sc.ReclaimPolicy = new(corev1.PersistentVolumeReclaimPolicy)
			*sc.ReclaimPolicy = corev1.PersistentVolumeReclaimDelete
			sc.VolumeBindingMode = new(storagev1.VolumeBindingMode)
			*sc.VolumeBindingMode = storagev1.VolumeBindingWaitForFirstConsumer

			assert.NilError(t, cc.Create(ctx, sc))
			t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, sc)) })

			pvc := &corev1.PersistentVolumeClaim{}
			assert.NilError(t, yaml.Unmarshal([]byte(`{
				spec: {
					accessModes: [ReadWriteOnce],
					resources: { requests: { storage: 2Gi } },
				},
			}`), pvc))
			pvc.Namespace, pvc.Name = ns.Name, "dynamic-pvc-pending"
			pvc.Spec.StorageClassName = &sc.Name

			assert.NilError(t, cc.Create(ctx, pvc))
			t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, pvc)) })

			// Not able to shrink the storage request.
			pvc.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("1Gi")

			err := cc.Update(ctx, pvc)
			assert.Assert(t, apierrors.IsInvalid(err), "expected Invalid, got\n%#v", err)
			assert.ErrorContains(t, err, "less than previous")
			assert.ErrorContains(t, err, pvc.Name, "expected mention of the object")

			status := apiErrorStatus(t, err)
			assert.Assert(t, status.Details != nil)
			assert.Assert(t, len(status.Details.Causes) != 0)
			assert.Equal(t, status.Details.Causes[0].Field, "spec")
			assert.Equal(t, status.Details.Causes[0].Type, metav1.CauseType(field.ErrorTypeForbidden))

			assert.NilError(t, reconciler.handlePersistentVolumeClaimError(cluster, err))

			// Not able to grow the storage request.
			pvc.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("4Gi")

			err = cc.Update(ctx, pvc)
			assert.Assert(t, apierrors.IsInvalid(err), "expected Invalid, got\n%#v", err)
			assert.ErrorContains(t, err, "bound claim")
			assert.ErrorContains(t, err, pvc.Name, "expected mention of the object")

			status = apiErrorStatus(t, err)
			assert.Assert(t, status.Details != nil)
			assert.Assert(t, len(status.Details.Causes) != 0)
			assert.Equal(t, status.Details.Causes[0].Field, "spec")
			assert.Equal(t, status.Details.Causes[0].Type, metav1.CauseType(field.ErrorTypeForbidden))

			assert.NilError(t, reconciler.handlePersistentVolumeClaimError(cluster, err))
		})

		t.Run("Bound", func(t *testing.T) {
			setup := func(t testing.TB, expansion bool) *corev1.PersistentVolumeClaim {
				// A storage class that binds when there is a pod and deletes volumes.
				sc := base.DeepCopy()
				sc.ObjectMeta = metav1.ObjectMeta{
					GenerateName: "postgres-operator-test-",
					Labels: map[string]string{
						"postgres-operator-test": "pvc-limitations-bound",
					},
				}
				sc.AllowVolumeExpansion = &expansion
				sc.ReclaimPolicy = new(corev1.PersistentVolumeReclaimPolicy)
				*sc.ReclaimPolicy = corev1.PersistentVolumeReclaimDelete

				assert.NilError(t, cc.Create(ctx, sc))
				t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, sc)) })

				pvc := &corev1.PersistentVolumeClaim{}
				pvc.ObjectMeta = metav1.ObjectMeta{
					Namespace:    ns.Name,
					GenerateName: "postgres-operator-test-",
					Labels: map[string]string{
						"postgres-operator-test": "pvc-limitations-bound",
					},
				}
				assert.NilError(t, yaml.Unmarshal([]byte(`{
					spec: {
						accessModes: [ReadWriteOnce],
						resources: { requests: { storage: 2Gi } },
					},
				}`), pvc))
				pvc.Spec.StorageClassName = &sc.Name

				assert.NilError(t, cc.Create(ctx, pvc))
				t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, pvc)) })

				pod := &corev1.Pod{}
				pod.Namespace, pod.Name = ns.Name, pvc.Name
				pod.Spec.Containers = []corev1.Container{{
					Name:    "any",
					Image:   CrunchyPostgresHAImage,
					Command: []string{"true"},
					VolumeMounts: []corev1.VolumeMount{{
						MountPath: "/tmp", Name: "volume",
					}},
				}}
				pod.Spec.Volumes = []corev1.Volume{{
					Name: "volume",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvc.Name,
						},
					},
				}}

				assert.NilError(t, cc.Create(ctx, pod))
				t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, pod)) })

				assert.NilError(t, wait.PollImmediate(time.Second, Scale(30*time.Second), func() (bool, error) {
					err := cc.Get(ctx, client.ObjectKeyFromObject(pvc), pvc)
					return pvc.Status.Phase != corev1.ClaimPending, err
				}), "expected Bound, got %#v", pvc.Status)

				return pvc
			}

			t.Run("NoExpansionNoResize", func(t *testing.T) {
				pvc := setup(t, false)

				// Not able to shrink the storage request.
				pvc.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("1Gi")

				err := cc.Update(ctx, pvc)
				assert.Assert(t, apierrors.IsInvalid(err), "expected Invalid, got\n%#v", err)
				assert.ErrorContains(t, err, "less than previous")
				assert.ErrorContains(t, err, pvc.Name, "expected mention of the object")

				status := apiErrorStatus(t, err)
				assert.Assert(t, status.Details != nil)
				assert.Assert(t, len(status.Details.Causes) != 0)
				assert.Equal(t, status.Details.Causes[0].Field, "spec.resources.requests.storage")
				assert.Equal(t, status.Details.Causes[0].Type, metav1.CauseType(field.ErrorTypeForbidden))

				assert.NilError(t, reconciler.handlePersistentVolumeClaimError(cluster, err))

				// Not able to grow the storage request.
				pvc.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("4Gi")

				err = cc.Update(ctx, pvc)
				assert.Assert(t, apierrors.IsForbidden(err), "expected Forbidden, got\n%#v", err)
				assert.ErrorContains(t, err, "only dynamic")
				assert.ErrorContains(t, err, pvc.Name, "expected mention of the object")

				assert.NilError(t, reconciler.handlePersistentVolumeClaimError(cluster, err))
			})

			t.Run("ExpansionNoShrink", func(t *testing.T) {
				if base.AllowVolumeExpansion == nil || !*base.AllowVolumeExpansion {
					t.Skip("requires a default storage class that allows expansion")
				}

				// Not able to shrink the storage request.
				pvc := setup(t, true)
				pvc.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("1Gi")

				err := cc.Update(ctx, pvc)
				assert.Assert(t, apierrors.IsInvalid(err), "expected Invalid, got\n%#v", err)
				assert.ErrorContains(t, err, "less than previous")
				assert.ErrorContains(t, err, pvc.Name, "expected mention of the object")

				status := apiErrorStatus(t, err)
				assert.Assert(t, status.Details != nil)
				assert.Assert(t, len(status.Details.Causes) != 0)
				assert.Equal(t, status.Details.Causes[0].Field, "spec.resources.requests.storage")
				assert.Equal(t, status.Details.Causes[0].Type, metav1.CauseType(field.ErrorTypeForbidden))

				assert.NilError(t, reconciler.handlePersistentVolumeClaimError(cluster, err))
			})

			t.Run("ExpansionResizeConditions", func(t *testing.T) {
				if base.AllowVolumeExpansion == nil || !*base.AllowVolumeExpansion {
					t.Skip("requires a default storage class that allows expansion")
				}

				pvc := setup(t, true)
				pvc.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("4Gi")
				assert.NilError(t, cc.Update(ctx, pvc))

				var condition *corev1.PersistentVolumeClaimCondition

				// When the resize controller sees that `spec.resources != status.capacity`,
				// it sets a "Resizing" condition and invokes the storage provider.
				// The provider could work very quickly and we miss the condition.
				// NOTE(cbandy): The oldest KEP talks about "ResizeStarted", but
				// that changed to "Resizing" during the merge to Kubernetes v1.8.
				// - https://git.k8s.io/enhancements/keps/sig-storage/284-enable-volume-expansion
				// - https://pr.k8s.io/49727#discussion_r136678508
				assert.NilError(t, wait.PollImmediate(time.Second, Scale(10*time.Second), func() (bool, error) {
					err := cc.Get(ctx, client.ObjectKeyFromObject(pvc), pvc)
					for i := range pvc.Status.Conditions {
						if pvc.Status.Conditions[i].Type == corev1.PersistentVolumeClaimResizing {
							condition = &pvc.Status.Conditions[i]
						}
					}
					return condition != nil ||
						equality.Semantic.DeepEqual(pvc.Spec.Resources, pvc.Status.Capacity), err
				}), "expected Resizing, got %+v", pvc.Status)

				if condition != nil {
					assert.Equal(t, condition.Status, corev1.ConditionTrue,
						"expected Resizing, got %+v", condition)
				}

				// Kubernetes v1.10 added the "FileSystemResizePending" condition
				// to indicate when the storage provider has finished its work.
				// When a CSI implementation indicates that it performed the
				// *entire* resize, this condition does not appear.
				// - https://pr.k8s.io/58415
				// - https://git.k8s.io/enhancements/keps/sig-storage/556-csi-volume-resizing
				assert.NilError(t, wait.PollImmediate(time.Second, Scale(30*time.Second), func() (bool, error) {
					err := cc.Get(ctx, client.ObjectKeyFromObject(pvc), pvc)
					for i := range pvc.Status.Conditions {
						if pvc.Status.Conditions[i].Type == corev1.PersistentVolumeClaimFileSystemResizePending {
							condition = &pvc.Status.Conditions[i]
						}
					}
					return condition != nil ||
						equality.Semantic.DeepEqual(pvc.Spec.Resources, pvc.Status.Capacity), err
				}), "expected FileSystemResizePending, got %+v", pvc.Status)

				if condition != nil {
					assert.Equal(t, condition.Status, corev1.ConditionTrue,
						"expected FileSystemResizePending, got %+v", condition)
				}

				// Kubernetes v1.15 ("ExpandInUsePersistentVolumes" feature gate)
				// will finish the resize of mounted and writable PVCs that have
				// the "FileSystemResizePending" condition. When the work is done,
				// the condition is removed and `spec.resources == status.capacity`.
				// - https://git.k8s.io/enhancements/keps/sig-storage/531-online-pv-resizing

				// A future version of Kubernetes will allow `spec.resources` to
				// shrink so long as it is greater than `status.capacity`.
				// - https://git.k8s.io/enhancements/keps/sig-storage/1790-recover-resize-failure
			})
		})
	})
}

func TestGetPVCNameMethods(t *testing.T) {

	namespace := "postgres-operator-test-get-pvc-name"

	// Stub to see that handlePersistentVolumeClaimError returns nil.
	cluster := &v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testcluster",
			Namespace: namespace,
		},
	}
	cluster.Spec.Backups.PGBackRest.Repos = []v1beta1.PGBackRestRepo{{
		Name:   "testrepo1",
		Volume: &v1beta1.RepoPVC{},
	}}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testvolume",
			Namespace: namespace,
			Labels: map[string]string{
				naming.LabelCluster: cluster.Name,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				"ReadWriteMany",
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}

	pgDataPVC := pvc.DeepCopy()
	pgDataPVC.Name = "testpgdatavol"
	pgDataPVC.Labels = map[string]string{
		naming.LabelCluster:     cluster.Name,
		naming.LabelInstanceSet: "testinstance1",
		naming.LabelInstance:    "testinstance1-abcd",
		naming.LabelRole:        naming.RolePostgresData,
	}

	walPVC := pvc.DeepCopy()
	walPVC.Name = "testwalvol"
	walPVC.Labels = map[string]string{
		naming.LabelCluster:     cluster.Name,
		naming.LabelInstanceSet: "testinstance1",
		naming.LabelInstance:    "testinstance1-abcd",
		naming.LabelRole:        naming.RolePostgresWAL,
	}
	clusterVolumes := []corev1.PersistentVolumeClaim{*pgDataPVC, *walPVC}

	repoPVC1 := pvc.DeepCopy()
	repoPVC1.Name = "testrepovol1"
	repoPVC1.Labels = map[string]string{
		naming.LabelCluster:              cluster.Name,
		naming.LabelPGBackRest:           "",
		naming.LabelPGBackRestRepo:       "testrepo1",
		naming.LabelPGBackRestRepoVolume: "",
	}
	repoPVCs := []*corev1.PersistentVolumeClaim{repoPVC1}

	repoPVC2 := pvc.DeepCopy()
	repoPVC2.Name = "testrepovol2"
	repoPVC2.Labels = map[string]string{
		naming.LabelCluster:              cluster.Name,
		naming.LabelPGBackRest:           "",
		naming.LabelPGBackRestRepo:       "testrepo2",
		naming.LabelPGBackRestRepoVolume: "",
	}
	// don't create this one yet

	t.Run("get pgdata PVC", func(t *testing.T) {

		pvcNames, err := getPGPVCName(map[string]string{
			naming.LabelCluster:     cluster.Name,
			naming.LabelInstanceSet: "testinstance1",
			naming.LabelInstance:    "testinstance1-abcd",
			naming.LabelRole:        naming.RolePostgresData,
		}, clusterVolumes)
		assert.NilError(t, err)

		assert.Assert(t, pvcNames == "testpgdatavol")
	})

	t.Run("get wal PVC", func(t *testing.T) {

		pvcNames, err := getPGPVCName(map[string]string{
			naming.LabelCluster:     cluster.Name,
			naming.LabelInstanceSet: "testinstance1",
			naming.LabelInstance:    "testinstance1-abcd",
			naming.LabelRole:        naming.RolePostgresWAL,
		}, clusterVolumes)
		assert.NilError(t, err)

		assert.Assert(t, pvcNames == "testwalvol")
	})

	t.Run("get one repo PVC", func(t *testing.T) {
		expectedMap := map[string]string{
			"testrepo1": "testrepovol1",
		}

		assert.DeepEqual(t, getRepoPVCNames(cluster, repoPVCs), expectedMap)
	})

	t.Run("get two repo PVCs", func(t *testing.T) {
		repoPVCs2 := append(repoPVCs, repoPVC2)

		cluster.Spec.Backups.PGBackRest.Repos = []v1beta1.PGBackRestRepo{{
			Name:   "testrepo1",
			Volume: &v1beta1.RepoPVC{},
		}, {
			Name:   "testrepo2",
			Volume: &v1beta1.RepoPVC{},
		}}

		expectedMap := map[string]string{
			"testrepo1": "testrepovol1",
			"testrepo2": "testrepovol2",
		}

		assert.DeepEqual(t, getRepoPVCNames(cluster, repoPVCs2), expectedMap)
	})
}

func TestReconcileConfigureExistingPVCs(t *testing.T) {
	ctx := context.Background()
	_, tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 1)

	r := &Reconciler{Client: tClient, Owner: client.FieldOwner(t.Name())}

	ns := setupNamespace(t, tClient)
	cluster := &v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testcluster",
			Namespace: ns.GetName(),
		},
		Spec: v1beta1.PostgresClusterSpec{
			PostgresVersion: 13,
			Image:           "example.com/crunchy-postgres-ha:test",
			DataSource: &v1beta1.DataSource{
				Volumes: &v1beta1.DataSourceVolumes{},
			},
			InstanceSets: []v1beta1.PostgresInstanceSetSpec{{
				Name: "instance1",
				DataVolumeClaimSpec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteMany},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
				},
			}},
			Backups: v1beta1.Backups{
				PGBackRest: v1beta1.PGBackRestArchive{
					Image: "example.com/crunchy-pgbackrest:test",
					Repos: []v1beta1.PGBackRestRepo{{
						Name: "repo1",
						Volume: &v1beta1.RepoPVC{
							VolumeClaimSpec: corev1.PersistentVolumeClaimSpec{
								AccessModes: []corev1.PersistentVolumeAccessMode{
									corev1.ReadWriteMany},
								Resources: corev1.ResourceRequirements{
									Requests: map[corev1.ResourceName]resource.
										Quantity{
										corev1.ResourceStorage: resource.
											MustParse("1Gi"),
									},
								},
							},
						},
					},
					},
				},
			},
		},
	}

	// create base PostgresCluster
	assert.NilError(t, tClient.Create(ctx, cluster))
	t.Cleanup(func() { assert.Check(t, tClient.Delete(ctx, cluster)) })

	t.Run("existing pgdata volume", func(t *testing.T) {
		volume := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pgdatavolume",
				Namespace: cluster.Namespace,
				Labels: map[string]string{
					"somelabel": "labelvalue-pgdata",
				},
			},
			Spec: cluster.Spec.InstanceSets[0].DataVolumeClaimSpec,
		}

		assert.NilError(t, tClient.Create(ctx, volume))

		// add the pgData PVC name to the CRD
		cluster.Spec.DataSource.Volumes.
			PGDataVolume = &v1beta1.DataSourceVolume{
			PVCName: "pgdatavolume",
		}

		clusterVolumes, err := r.observePersistentVolumeClaims(ctx, cluster)
		assert.NilError(t, err)
		// check that created volume does not show up in observed volumes since
		// it does not have appropriate labels
		assert.Assert(t, len(clusterVolumes) == 0)

		clusterVolumes, err = r.configureExistingPVCs(ctx, cluster,
			clusterVolumes)
		assert.NilError(t, err)

		// now, check that the label volume is returned
		assert.Assert(t, len(clusterVolumes) == 1)

		// observe again, but allow time for the change to be observed
		err = wait.Poll(time.Second/2, Scale(time.Second*15), func() (bool, error) {
			clusterVolumes, err = r.observePersistentVolumeClaims(ctx, cluster)
			return len(clusterVolumes) == 1, err
		})
		assert.NilError(t, err)
		// check that created volume is now in the list
		assert.Assert(t, len(clusterVolumes) == 1)

		// validate the expected labels are in place
		// expected volume labels, plus the original label
		expected := map[string]string{
			naming.LabelCluster:     cluster.Name,
			naming.LabelInstanceSet: cluster.Spec.InstanceSets[0].Name,
			naming.LabelInstance:    cluster.Status.StartupInstance,
			naming.LabelRole:        naming.RolePostgresData,
			naming.LabelData:        naming.DataPostgres,
			"somelabel":             "labelvalue-pgdata",
		}

		// ensure volume is found and labeled correctly
		var found bool
		for i := range clusterVolumes {
			if clusterVolumes[i].Name == cluster.Spec.DataSource.Volumes.
				PGDataVolume.PVCName {
				found = true
				assert.DeepEqual(t, expected, clusterVolumes[i].Labels)
			}
		}
		assert.Assert(t, found)
	})

	t.Run("existing pg_wal volume", func(t *testing.T) {
		pgWALVolume := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pgwalvolume",
				Namespace: cluster.Namespace,
				Labels: map[string]string{
					"somelabel": "labelvalue-pgwal",
				},
			},
			Spec: cluster.Spec.InstanceSets[0].DataVolumeClaimSpec,
		}

		assert.NilError(t, tClient.Create(ctx, pgWALVolume))

		// add the pg_wal PVC name to the CRD
		cluster.Spec.DataSource.Volumes.PGWALVolume =
			&v1beta1.DataSourceVolume{
				PVCName: "pgwalvolume",
			}

		clusterVolumes, err := r.observePersistentVolumeClaims(ctx, cluster)
		assert.NilError(t, err)
		// check that created pgwal volume does not show up in observed volumes
		// since it does not have appropriate labels, only the previously created
		// pgdata volume should be in the observed list
		assert.Assert(t, len(clusterVolumes) == 1)

		clusterVolumes, err = r.configureExistingPVCs(ctx, cluster,
			clusterVolumes)
		assert.NilError(t, err)

		// now, check that the label volume is returned
		assert.Assert(t, len(clusterVolumes) == 2)

		// observe again, but allow time for the change to be observed
		err = wait.Poll(time.Second/2, Scale(time.Second*15), func() (bool, error) {
			clusterVolumes, err = r.observePersistentVolumeClaims(ctx, cluster)
			return len(clusterVolumes) == 2, err
		})
		assert.NilError(t, err)
		// check that created volume is now in the list
		assert.Assert(t, len(clusterVolumes) == 2)

		// validate the expected labels are in place
		// expected volume labels, plus the original label
		expected := map[string]string{
			naming.LabelCluster:     cluster.Name,
			naming.LabelInstanceSet: cluster.Spec.InstanceSets[0].Name,
			naming.LabelInstance:    cluster.Status.StartupInstance,
			naming.LabelRole:        naming.RolePostgresWAL,
			naming.LabelData:        naming.DataPostgres,
			"somelabel":             "labelvalue-pgwal",
		}

		// ensure volume is found and labeled correctly
		var found bool
		for i := range clusterVolumes {
			if clusterVolumes[i].Name == cluster.Spec.DataSource.Volumes.
				PGWALVolume.PVCName {
				found = true
				assert.DeepEqual(t, expected, clusterVolumes[i].Labels)
			}
		}
		assert.Assert(t, found)
	})

	t.Run("existing repo volume", func(t *testing.T) {
		volume := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "repovolume",
				Namespace: cluster.Namespace,
				Labels: map[string]string{
					"somelabel": "labelvalue-repo",
				},
			},
			Spec: cluster.Spec.InstanceSets[0].DataVolumeClaimSpec,
		}

		assert.NilError(t, tClient.Create(ctx, volume))

		// add the pgBackRest repo PVC name to the CRD
		cluster.Spec.DataSource.Volumes.PGBackRestVolume =
			&v1beta1.DataSourceVolume{
				PVCName: "repovolume",
			}

		clusterVolumes, err := r.observePersistentVolumeClaims(ctx, cluster)
		assert.NilError(t, err)
		// check that created volume does not show up in observed volumes since
		// it does not have appropriate labels
		// check that created pgBackRest repo volume does not show up in observed
		// volumes since it does not have appropriate labels, only the previously
		// created pgdata and pg_wal volumes should be in the observed list
		assert.Assert(t, len(clusterVolumes) == 2)

		clusterVolumes, err = r.configureExistingPVCs(ctx, cluster,
			clusterVolumes)
		assert.NilError(t, err)

		// now, check that the label volume is returned
		assert.Assert(t, len(clusterVolumes) == 3)

		// observe again, but allow time for the change to be observed
		err = wait.Poll(time.Second/2, Scale(time.Second*15), func() (bool, error) {
			clusterVolumes, err = r.observePersistentVolumeClaims(ctx, cluster)
			return len(clusterVolumes) == 3, err
		})
		assert.NilError(t, err)
		// check that created volume is now in the list
		assert.Assert(t, len(clusterVolumes) == 3)

		// validate the expected labels are in place
		// expected volume labels, plus the original label
		expected := map[string]string{
			naming.LabelCluster:              cluster.Name,
			naming.LabelData:                 naming.DataPGBackRest,
			naming.LabelPGBackRest:           "",
			naming.LabelPGBackRestRepo:       "repo1",
			naming.LabelPGBackRestRepoVolume: "",
			"somelabel":                      "labelvalue-repo",
		}

		// ensure volume is found and labeled correctly
		var found bool
		for i := range clusterVolumes {
			if clusterVolumes[i].Name == cluster.Spec.DataSource.Volumes.
				PGBackRestVolume.PVCName {
				found = true
				assert.DeepEqual(t, expected, clusterVolumes[i].Labels)
			}
		}
		assert.Assert(t, found)
	})
}

func TestReconcileMoveDirectories(t *testing.T) {
	ctx := context.Background()
	_, tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 1)

	r := &Reconciler{Client: tClient, Owner: client.FieldOwner(t.Name())}

	ns := setupNamespace(t, tClient)
	cluster := &v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testcluster",
			Namespace: ns.GetName(),
		},
		Spec: v1beta1.PostgresClusterSpec{
			PostgresVersion: 13,
			Image:           "example.com/crunchy-postgres-ha:test",
			ImagePullPolicy: corev1.PullAlways,
			ImagePullSecrets: []corev1.LocalObjectReference{{
				Name: "test-secret",
			}},
			DataSource: &v1beta1.DataSource{
				Volumes: &v1beta1.DataSourceVolumes{
					PGDataVolume: &v1beta1.DataSourceVolume{
						PVCName:   "testpgdata",
						Directory: "testpgdatadir",
					},
					PGWALVolume: &v1beta1.DataSourceVolume{
						PVCName:   "testwal",
						Directory: "testwaldir",
					},
					PGBackRestVolume: &v1beta1.DataSourceVolume{
						PVCName:   "testrepo",
						Directory: "testrepodir",
					},
				},
			},
			InstanceSets: []v1beta1.PostgresInstanceSetSpec{{
				Name: "instance1",
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("1m"),
					},
				},
				PriorityClassName: initialize.String("some-priority-class"),
				DataVolumeClaimSpec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteMany},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
				},
			}},
			Backups: v1beta1.Backups{
				PGBackRest: v1beta1.PGBackRestArchive{
					Image: "example.com/crunchy-pgbackrest:test",
					RepoHost: &v1beta1.PGBackRestRepoHost{
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU: resource.MustParse("1m"),
							},
						},
						PriorityClassName: initialize.String("some-priority-class"),
					},
					Repos: []v1beta1.PGBackRestRepo{{
						Name: "repo1",
						Volume: &v1beta1.RepoPVC{
							VolumeClaimSpec: corev1.PersistentVolumeClaimSpec{
								AccessModes: []corev1.PersistentVolumeAccessMode{
									corev1.ReadWriteMany},
								Resources: corev1.ResourceRequirements{
									Requests: map[corev1.ResourceName]resource.
										Quantity{
										corev1.ResourceStorage: resource.
											MustParse("1Gi"),
									},
								},
							},
						},
					},
					},
				},
			},
		},
	}

	// create PostgresCluster
	assert.NilError(t, tClient.Create(ctx, cluster))
	t.Cleanup(func() { assert.Check(t, tClient.Delete(ctx, cluster)) })

	returnEarly, err := r.reconcileDirMoveJobs(ctx, cluster)
	assert.NilError(t, err)
	// returnEarly will initially be true because the Jobs will not have
	// completed yet
	assert.Assert(t, returnEarly)

	moveJobs := &batchv1.JobList{}
	err = r.Client.List(ctx, moveJobs, &client.ListOptions{
		LabelSelector: naming.DirectoryMoveJobLabels(cluster.Name).AsSelector(),
	})
	assert.NilError(t, err)

	t.Run("check pgdata move job pod spec", func(t *testing.T) {

		for i := range moveJobs.Items {
			if moveJobs.Items[i].Name == "testcluster-move-pgdata-dir" {
				assert.Assert(t, marshalMatches(moveJobs.Items[i].Spec.Template.Spec, `
automountServiceAccountToken: false
containers:
- command:
  - bash
  - -ceu
  - "echo \"Preparing cluster testcluster volumes for PGO v5.x\"\n    echo \"pgdata_pvc=testpgdata\"\n
    \   echo \"Current PG data directory volume contents:\" \n    ls -lh \"/pgdata\"\n
    \   echo \"Now updating PG data directory...\"\n    [ -d \"/pgdata/testpgdatadir\"
    ] && mv \"/pgdata/testpgdatadir\" \"/pgdata/pg13_bootstrap\"\n    rm -f \"/pgdata/pg13/patroni.dynamic.json\"\n
    \   echo \"Updated PG data directory contents:\" \n    ls -lh \"/pgdata\"\n    echo
    \"PG Data directory preparation complete\"\n    "
  image: example.com/crunchy-postgres-ha:test
  imagePullPolicy: Always
  name: pgdata-move-job
  resources:
    requests:
      cpu: 1m
  securityContext:
    allowPrivilegeEscalation: false
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
  terminationMessagePath: /dev/termination-log
  terminationMessagePolicy: File
  volumeMounts:
  - mountPath: /pgdata
    name: postgres-data
dnsPolicy: ClusterFirst
enableServiceLinks: false
imagePullSecrets:
- name: test-secret
priorityClassName: some-priority-class
restartPolicy: Never
schedulerName: default-scheduler
securityContext:
  fsGroup: 26
  runAsNonRoot: true
terminationGracePeriodSeconds: 30
volumes:
- name: postgres-data
  persistentVolumeClaim:
    claimName: testpgdata
	`+"\n"))
			}
		}

	})

	t.Run("check pgwal move job pod spec", func(t *testing.T) {

		for i := range moveJobs.Items {
			if moveJobs.Items[i].Name == "testcluster-move-pgwal-dir" {
				assert.Assert(t, marshalMatches(moveJobs.Items[i].Spec.Template.Spec, `
automountServiceAccountToken: false
containers:
- command:
  - bash
  - -ceu
  - "echo \"Preparing cluster testcluster volumes for PGO v5.x\"\n    echo \"pg_wal_pvc=testwal\"\n
    \   echo \"Current PG WAL directory volume contents:\"\n    ls -lh \"/pgwal\"\n
    \   echo \"Now updating PG WAL directory...\"\n    [ -d \"/pgwal/testwaldir\"
    ] && mv \"/pgwal/testwaldir\" \"/pgwal/testcluster-wal\"\n    echo \"Updated PG
    WAL directory contents:\"\n    ls -lh \"/pgwal\"\n    echo \"PG WAL directory
    preparation complete\"\n    "
  image: example.com/crunchy-postgres-ha:test
  imagePullPolicy: Always
  name: pgwal-move-job
  resources:
    requests:
      cpu: 1m
  securityContext:
    allowPrivilegeEscalation: false
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
  terminationMessagePath: /dev/termination-log
  terminationMessagePolicy: File
  volumeMounts:
  - mountPath: /pgwal
    name: postgres-wal
dnsPolicy: ClusterFirst
enableServiceLinks: false
imagePullSecrets:
- name: test-secret
priorityClassName: some-priority-class
restartPolicy: Never
schedulerName: default-scheduler
securityContext:
  fsGroup: 26
  runAsNonRoot: true
terminationGracePeriodSeconds: 30
volumes:
- name: postgres-wal
  persistentVolumeClaim:
    claimName: testwal
	`+"\n"))
			}
		}

	})

	t.Run("check repo move job pod spec", func(t *testing.T) {

		for i := range moveJobs.Items {
			if moveJobs.Items[i].Name == "testcluster-move-pgbackrest-repo-dir" {
				assert.Assert(t, marshalMatches(moveJobs.Items[i].Spec.Template.Spec, `
automountServiceAccountToken: false
containers:
- command:
  - bash
  - -ceu
  - "echo \"Preparing cluster testcluster pgBackRest repo volume for PGO v5.x\"\n
    \   echo \"repo_pvc=testrepo\"\n    echo \"pgbackrest directory:\"\n    ls -lh
    /pgbackrest\n    echo \"Current pgBackRest repo directory volume contents:\" \n
    \   ls -lh \"/pgbackrest/testrepodir\"\n    echo \"Now updating repo directory...\"\n
    \   [ -d \"/pgbackrest/testrepodir\" ] && mv -t \"/pgbackrest/\" \"/pgbackrest/testrepodir/archive\"\n
    \   [ -d \"/pgbackrest/testrepodir\" ] && mv -t \"/pgbackrest/\" \"/pgbackrest/testrepodir/backup\"\n
    \   echo \"Updated /pgbackrest directory contents:\"\n    ls -lh \"/pgbackrest\"\n
    \   echo \"Repo directory preparation complete\"\n    "
  image: example.com/crunchy-pgbackrest:test
  imagePullPolicy: Always
  name: repo-move-job
  resources:
    requests:
      cpu: 1m
  securityContext:
    allowPrivilegeEscalation: false
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
  terminationMessagePath: /dev/termination-log
  terminationMessagePolicy: File
  volumeMounts:
  - mountPath: /pgbackrest
    name: pgbackrest-repo
dnsPolicy: ClusterFirst
enableServiceLinks: false
imagePullSecrets:
- name: test-secret
priorityClassName: some-priority-class
restartPolicy: Never
schedulerName: default-scheduler
securityContext:
  fsGroup: 26
  runAsNonRoot: true
terminationGracePeriodSeconds: 30
volumes:
- name: pgbackrest-repo
  persistentVolumeClaim:
    claimName: testrepo
	`+"\n"))
			}
		}

	})
}
