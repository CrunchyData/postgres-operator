// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"errors"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/events"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestHandlePersistentVolumeClaimError(t *testing.T) {
	recorder := events.NewRecorder(t, runtime.Scheme)
	reconciler := &Reconciler{
		Recorder: recorder,
	}

	cluster := new(v1beta1.PostgresCluster)
	cluster.Namespace = "ns1"
	cluster.Name = "pg2"

	reset := func() {
		cluster.Status.Conditions = cluster.Status.Conditions[:0]
		recorder.Events = recorder.Events[:0]
	}

	// It returns any error it does not recognize completely.
	t.Run("Unexpected", func(t *testing.T) {
		t.Cleanup(reset)

		err := errors.New("whomp")

		assert.Equal(t, err, reconciler.handlePersistentVolumeClaimError(cluster, err))
		assert.Assert(t, len(cluster.Status.Conditions) == 0)
		assert.Assert(t, len(recorder.Events) == 0)

		err = apierrors.NewInvalid(
			corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim").GroupKind(),
			"some-pvc",
			field.ErrorList{
				field.Forbidden(field.NewPath("metadata"), "dunno"),
			})

		assert.Equal(t, err, reconciler.handlePersistentVolumeClaimError(cluster, err))
		assert.Assert(t, len(cluster.Status.Conditions) == 0)
		assert.Assert(t, len(recorder.Events) == 0)
	})

	// Neither statically nor dynamically provisioned claims can be resized
	// before they are bound to a persistent volume. Kubernetes rejects such
	// changes during PVC validation.
	//
	// A static PVC is one with a present-and-blank storage class. It is
	// pending until a PV exists that matches its selector, requests, etc.
	// - https://docs.k8s.io/concepts/storage/persistent-volumes/#static
	// - https://docs.k8s.io/concepts/storage/persistent-volumes/#class-1
	//
	// A dynamic PVC is associated with a storage class. Storage classes that
	// "WaitForFirstConsumer" do not bind a PV until there is a pod.
	// - https://docs.k8s.io/concepts/storage/persistent-volumes/#dynamic
	t.Run("Pending", func(t *testing.T) {
		t.Run("Grow", func(t *testing.T) {
			t.Cleanup(reset)

			err := apierrors.NewInvalid(
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim").GroupKind(),
				"my-pending-pvc",
				field.ErrorList{
					// - https://releases.k8s.io/v1.24.0/pkg/apis/core/validation/validation.go#L2184
					field.Forbidden(field.NewPath("spec"), "… immutable … bound claim …"),
				})

			// PVCs will bind eventually. This error should become an event without a condition.
			assert.NilError(t, reconciler.handlePersistentVolumeClaimError(cluster, err))

			assert.Check(t, len(cluster.Status.Conditions) == 0)
			assert.Check(t, len(recorder.Events) > 0)

			for _, event := range recorder.Events {
				assert.Equal(t, event.Type, "Warning")
				assert.Equal(t, event.Reason, "PersistentVolumeError")
				assert.Assert(t, cmp.Contains(event.Note, "PersistentVolumeClaim"))
				assert.Assert(t, cmp.Contains(event.Note, "my-pending-pvc"))
				assert.Assert(t, cmp.Contains(event.Note, "bound claim"))
				assert.DeepEqual(t, event.Regarding, corev1.ObjectReference{
					APIVersion: v1beta1.GroupVersion.Identifier(),
					Kind:       "PostgresCluster",
					Namespace:  "ns1", Name: "pg2",
				})
			}
		})

		t.Run("Shrink", func(t *testing.T) {
			t.Cleanup(reset)

			// Requests to make a pending PVC smaller fail for multiple reasons.
			err := apierrors.NewInvalid(
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim").GroupKind(),
				"my-pending-pvc",
				field.ErrorList{
					// - https://releases.k8s.io/v1.24.0/pkg/apis/core/validation/validation.go#L2184
					field.Forbidden(field.NewPath("spec"), "… immutable … bound claim …"),

					// - https://releases.k8s.io/v1.24.0/pkg/apis/core/validation/validation.go#L2188
					field.Forbidden(field.NewPath("spec", "resources", "requests", "storage"), "… not be less …"),
				})

			// PVCs will bind eventually, but the size is rejected.
			assert.NilError(t, reconciler.handlePersistentVolumeClaimError(cluster, err))

			assert.Check(t, len(cluster.Status.Conditions) > 0)
			assert.Check(t, len(recorder.Events) > 0)

			for _, condition := range cluster.Status.Conditions {
				assert.Equal(t, condition.Type, "PersistentVolumeResizing")
				assert.Equal(t, condition.Status, metav1.ConditionFalse)
				assert.Equal(t, condition.Reason, "Invalid")
				assert.Assert(t, cmp.Contains(condition.Message, "cannot be resized"))
			}

			for _, event := range recorder.Events {
				assert.Equal(t, event.Type, "Warning")
				assert.Equal(t, event.Reason, "PersistentVolumeError")
				assert.Assert(t, cmp.Contains(event.Note, "PersistentVolumeClaim"))
				assert.Assert(t, cmp.Contains(event.Note, "my-pending-pvc"))
				assert.Assert(t, cmp.Contains(event.Note, "bound claim"))
				assert.Assert(t, cmp.Contains(event.Note, "not be less"))
				assert.DeepEqual(t, event.Regarding, corev1.ObjectReference{
					APIVersion: v1beta1.GroupVersion.Identifier(),
					Kind:       "PostgresCluster",
					Namespace:  "ns1", Name: "pg2",
				})
			}
		})
	})

	// Statically provisioned claims cannot be resized. Kubernetes responds
	// differently based on the size growing or shrinking.
	//
	// Dynamically provisioned claims of storage classes that do *not*
	// "allowVolumeExpansion" behave the same way.
	t.Run("NoExpansion", func(t *testing.T) {
		t.Run("Grow", func(t *testing.T) {
			t.Cleanup(reset)

			// - https://releases.k8s.io/v1.24.0/plugin/pkg/admission/storage/persistentvolume/resize/admission.go#L108
			err := apierrors.NewForbidden(
				corev1.Resource("persistentvolumeclaims"), "my-static-pvc",
				errors.New("… only dynamically provisioned …"))

			// This PVC cannot resize. The error should become an event and condition.
			assert.NilError(t, reconciler.handlePersistentVolumeClaimError(cluster, err))

			assert.Check(t, len(cluster.Status.Conditions) > 0)
			assert.Check(t, len(recorder.Events) > 0)

			for _, condition := range cluster.Status.Conditions {
				assert.Equal(t, condition.Type, "PersistentVolumeResizing")
				assert.Equal(t, condition.Status, metav1.ConditionFalse)
				assert.Equal(t, condition.Reason, "Forbidden")
				assert.Assert(t, cmp.Contains(condition.Message, "cannot be resized"))
			}

			for _, event := range recorder.Events {
				assert.Equal(t, event.Type, "Warning")
				assert.Equal(t, event.Reason, "PersistentVolumeError")
				assert.Assert(t, cmp.Contains(event.Note, "persistentvolumeclaim"))
				assert.Assert(t, cmp.Contains(event.Note, "my-static-pvc"))
				assert.Assert(t, cmp.Contains(event.Note, "only dynamic"))
				assert.DeepEqual(t, event.Regarding, corev1.ObjectReference{
					APIVersion: v1beta1.GroupVersion.Identifier(),
					Kind:       "PostgresCluster",
					Namespace:  "ns1", Name: "pg2",
				})
			}
		})

		// Dynamically provisioned claims of storage classes that *do*
		// "allowVolumeExpansion" can grow but cannot shrink. Kubernetes
		// rejects such changes during PVC validation, just like static claims.
		//
		// A future version of Kubernetes will allow `spec.resources` to shrink
		// so long as it is greater than `status.capacity`.
		// - https://git.k8s.io/enhancements/keps/sig-storage/1790-recover-resize-failure
		t.Run("Shrink", func(t *testing.T) {
			t.Cleanup(reset)

			err := apierrors.NewInvalid(
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim").GroupKind(),
				"my-static-pvc",
				field.ErrorList{
					// - https://releases.k8s.io/v1.24.0/pkg/apis/core/validation/validation.go#L2188
					field.Forbidden(field.NewPath("spec", "resources", "requests", "storage"), "… not be less …"),
				})

			// The PVC size is rejected. This error should become an event and condition.
			assert.NilError(t, reconciler.handlePersistentVolumeClaimError(cluster, err))

			assert.Check(t, len(cluster.Status.Conditions) > 0)
			assert.Check(t, len(recorder.Events) > 0)

			for _, condition := range cluster.Status.Conditions {
				assert.Equal(t, condition.Type, "PersistentVolumeResizing")
				assert.Equal(t, condition.Status, metav1.ConditionFalse)
				assert.Equal(t, condition.Reason, "Invalid")
				assert.Assert(t, cmp.Contains(condition.Message, "cannot be resized"))
			}

			for _, event := range recorder.Events {
				assert.Equal(t, event.Type, "Warning")
				assert.Equal(t, event.Reason, "PersistentVolumeError")
				assert.Assert(t, cmp.Contains(event.Note, "PersistentVolumeClaim"))
				assert.Assert(t, cmp.Contains(event.Note, "my-static-pvc"))
				assert.Assert(t, cmp.Contains(event.Note, "not be less"))
				assert.DeepEqual(t, event.Regarding, corev1.ObjectReference{
					APIVersion: v1beta1.GroupVersion.Identifier(),
					Kind:       "PostgresCluster",
					Namespace:  "ns1", Name: "pg2",
				})
			}
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
			Resources: corev1.VolumeResourceRequirements{
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
					Resources: corev1.VolumeResourceRequirements{
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
								Resources: corev1.VolumeResourceRequirements{
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
		err = wait.PollUntilContextTimeout(ctx, time.Second/2, Scale(time.Second*15), false, func(ctx context.Context) (bool, error) {
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
		err = wait.PollUntilContextTimeout(ctx, time.Second/2, Scale(time.Second*15), false, func(ctx context.Context) (bool, error) {
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
		err = wait.PollUntilContextTimeout(ctx, time.Second/2, Scale(time.Second*15), false, func(ctx context.Context) (bool, error) {
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
					Resources: corev1.VolumeResourceRequirements{
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
								Resources: corev1.VolumeResourceRequirements{
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
		Namespace:     cluster.Namespace,
		LabelSelector: naming.DirectoryMoveJobLabels(cluster.Name).AsSelector(),
	})
	assert.NilError(t, err)

	t.Run("check pgdata move job pod spec", func(t *testing.T) {

		for i := range moveJobs.Items {
			if moveJobs.Items[i].Name == "testcluster-move-pgdata-dir" {
				compare := `
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
    capabilities:
      drop:
      - ALL
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
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
  fsGroupChangePolicy: OnRootMismatch
terminationGracePeriodSeconds: 30
volumes:
- name: postgres-data
  persistentVolumeClaim:
    claimName: testpgdata
	`

				assert.Assert(t, cmp.MarshalMatches(moveJobs.Items[i].Spec.Template.Spec, compare+"\n"))
			}
		}

	})

	t.Run("check pgwal move job pod spec", func(t *testing.T) {

		for i := range moveJobs.Items {
			if moveJobs.Items[i].Name == "testcluster-move-pgwal-dir" {
				compare := `
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
    capabilities:
      drop:
      - ALL
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
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
  fsGroupChangePolicy: OnRootMismatch
terminationGracePeriodSeconds: 30
volumes:
- name: postgres-wal
  persistentVolumeClaim:
    claimName: testwal
	`

				assert.Assert(t, cmp.MarshalMatches(moveJobs.Items[i].Spec.Template.Spec, compare+"\n"))
			}
		}

	})

	t.Run("check repo move job pod spec", func(t *testing.T) {

		for i := range moveJobs.Items {
			if moveJobs.Items[i].Name == "testcluster-move-pgbackrest-repo-dir" {
				compare := `
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
    capabilities:
      drop:
      - ALL
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
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
  fsGroupChangePolicy: OnRootMismatch
terminationGracePeriodSeconds: 30
volumes:
- name: pgbackrest-repo
  persistentVolumeClaim:
    claimName: testrepo
	`
				assert.Assert(t, cmp.MarshalMatches(moveJobs.Items[i].Spec.Template.Spec, compare+"\n"))
			}
		}

	})
}
