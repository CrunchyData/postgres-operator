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
	"os"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
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

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestPersistentVolumeClaimLimitations(t *testing.T) {
	if !strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
		t.Skip("requires a running persistent volume controller")
	}

	ctx := context.Background()
	tEnv, cc, _ := setupTestEnv(t, t.Name())
	t.Cleanup(func() { teardownTestEnv(t, tEnv) })

	ns := &corev1.Namespace{}
	ns.GenerateName = "postgres-operator-test-"
	ns.Labels = map[string]string{"postgres-operator-test": t.Name()}
	assert.NilError(t, cc.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, ns)) })

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

			assert.NilError(t, wait.PollImmediate(time.Second, 10*time.Second, func() (bool, error) {
				err := cc.Get(ctx, client.ObjectKeyFromObject(pv), pv)
				return pv.Status.Phase != corev1.VolumePending, err
			}), "expected Available, got %#v", pv.Status)

			pvc := base.DeepCopy()
			pvc.Name = "static-pvc-bound"
			assert.NilError(t, cc.Create(ctx, pvc))
			t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, pvc)) })

			assert.NilError(t, wait.PollImmediate(time.Second, 10*time.Second, func() (bool, error) {
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

				assert.NilError(t, wait.PollImmediate(time.Second, 30*time.Second, func() (bool, error) {
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
				assert.NilError(t, wait.PollImmediate(time.Second, 10*time.Second, func() (bool, error) {
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
				assert.NilError(t, wait.PollImmediate(time.Second, 30*time.Second, func() (bool, error) {
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

	ctx := context.Background()
	tEnv, cc, _ := setupTestEnv(t, t.Name())
	t.Cleanup(func() { teardownTestEnv(t, tEnv) })

	ns := &corev1.Namespace{}
	ns.GenerateName = "postgres-operator-test-"
	ns.Labels = map[string]string{"postgres-operator-test": t.Name()}
	assert.NilError(t, cc.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, ns)) })

	// Stub to see that handlePersistentVolumeClaimError returns nil.
	cluster := &v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testcluster",
			Namespace: ns.Name,
		},
	}
	cluster.Spec.Backups.PGBackRest.Repos = []v1beta1.PGBackRestRepo{{
		Name:   "testrepo1",
		Volume: &v1beta1.RepoPVC{},
	}}

	reconciler := &Reconciler{
		Recorder: new(record.FakeRecorder),
		Client:   cc,
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testvolume",
			Namespace: ns.Name,
			Labels: map[string]string{
				naming.LabelCluster: cluster.Name,
			},
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{
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
	assert.NilError(t, cc.Create(ctx, pgDataPVC))

	walPVC := pvc.DeepCopy()
	walPVC.Name = "testwalvol"
	walPVC.Labels = map[string]string{
		naming.LabelCluster:     cluster.Name,
		naming.LabelInstanceSet: "testinstance1",
		naming.LabelInstance:    "testinstance1-abcd",
		naming.LabelRole:        naming.RolePostgresWAL,
	}
	assert.NilError(t, cc.Create(ctx, walPVC))

	repoPVC1 := pvc.DeepCopy()
	repoPVC1.Name = "testrepovol1"
	repoPVC1.Labels = map[string]string{
		naming.LabelCluster:              cluster.Name,
		naming.LabelPGBackRest:           "",
		naming.LabelPGBackRestRepo:       "testrepo1",
		naming.LabelPGBackRestRepoVolume: "",
	}
	assert.NilError(t, cc.Create(ctx, repoPVC1))

	repoPVC2 := pvc.DeepCopy()
	repoPVC2.Name = "testrepovol2"
	repoPVC2.Labels = map[string]string{
		naming.LabelCluster:              cluster.Name,
		naming.LabelPGBackRest:           "",
		naming.LabelPGBackRestRepo:       "testrepo2",
		naming.LabelPGBackRestRepoVolume: "",
	}
	// don't create this one yet

	t.Run("get first volume created", func(t *testing.T) {
		// getPVCName should normally find 1 PVC, but in cases where multiples
		// are found, the first sorted PVC name will be returned.
		testMap := map[string]string{
			naming.LabelCluster: cluster.Name,
		}

		selector, err := naming.AsSelector(metav1.LabelSelector{
			MatchLabels: testMap,
		})

		assert.NilError(t, err)
		assert.Assert(t, reconciler.getPVCName(ctx, cluster, selector) == "testpgdatavol")

	})

	t.Run("get pgdata PVC", func(t *testing.T) {

		assert.Assert(t, reconciler.getPGPVCNames(ctx, cluster, map[string]string{
			naming.LabelCluster:     cluster.Name,
			naming.LabelInstanceSet: "testinstance1",
			naming.LabelInstance:    "testinstance1-abcd",
			naming.LabelRole:        naming.RolePostgresData,
		}) == "testpgdatavol")
	})

	t.Run("get wal PVC", func(t *testing.T) {

		assert.Assert(t, reconciler.getPGPVCNames(ctx, cluster, map[string]string{
			naming.LabelCluster:     cluster.Name,
			naming.LabelInstanceSet: "testinstance1",
			naming.LabelInstance:    "testinstance1-abcd",
			naming.LabelRole:        naming.RolePostgresWAL,
		}) == "testwalvol")
	})

	t.Run("get one repo PVC", func(t *testing.T) {
		expectedMap := map[string]string{
			"testrepo1": "testrepovol1",
		}
		assert.DeepEqual(t, reconciler.getRepoPVCNames(ctx, cluster), expectedMap)
	})

	t.Run("get two repo PVCs", func(t *testing.T) {
		assert.NilError(t, cc.Create(ctx, repoPVC2))

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
		assert.DeepEqual(t, reconciler.getRepoPVCNames(ctx, cluster), expectedMap)
	})
}
