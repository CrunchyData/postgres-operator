// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
)

func TestReconcileSnapshots(t *testing.T) {
	ctx := context.Background()
	cfg, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 1)
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	assert.NilError(t, err)

	r := &Reconciler{
		Client:          cc,
		Owner:           client.FieldOwner(t.Name()),
		DiscoveryClient: discoveryClient,
	}
	ns := setupNamespace(t, cc)

	t.Run("SnapshotsDisabledDeleteSnapshots", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns.Name
		cluster.ObjectMeta.UID = "the-uid-123"

		instances := newObservedInstances(cluster, nil, nil)
		volumes := []corev1.PersistentVolumeClaim{}

		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: "instance1-abc-def",
			},
		}
		volumeSnapshotClassName := "my-snapshotclass"
		snapshot, err := r.generateVolumeSnapshot(cluster, *pvc, volumeSnapshotClassName)
		assert.NilError(t, err)
		err = errors.WithStack(r.apply(ctx, snapshot))
		assert.NilError(t, err)

		err = r.reconcileVolumeSnapshots(ctx, cluster, instances, volumes)
		assert.NilError(t, err)

		// Get all snapshots for this cluster
		selectSnapshots, err := naming.AsSelector(naming.Cluster(cluster.Name))
		assert.NilError(t, err)
		snapshots := &volumesnapshotv1.VolumeSnapshotList{}
		err = errors.WithStack(
			r.Client.List(ctx, snapshots,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selectSnapshots},
			))
		assert.NilError(t, err)
		assert.Equal(t, len(snapshots.Items), 0)
	})

	t.Run("SnapshotsEnabledNoJobsNoSnapshots", func(t *testing.T) {
		gate := feature.NewGate()
		assert.NilError(t, gate.SetFromMap(map[string]bool{
			feature.VolumeSnapshots: true,
		}))
		ctx := feature.NewContext(ctx, gate)

		cluster := testCluster()
		cluster.Namespace = ns.Name
		cluster.ObjectMeta.UID = "the-uid-123"
		volumeSnapshotClassName := "my-snapshotclass"
		cluster.Spec.Backups.Snapshots = &v1beta1.VolumeSnapshots{
			VolumeSnapshotClassName: volumeSnapshotClassName,
		}

		instances := newObservedInstances(cluster, nil, nil)
		volumes := []corev1.PersistentVolumeClaim{}

		err := r.reconcileVolumeSnapshots(ctx, cluster, instances, volumes)
		assert.NilError(t, err)

		// Get all snapshots for this cluster
		selectSnapshots, err := naming.AsSelector(naming.Cluster(cluster.Name))
		assert.NilError(t, err)
		snapshots := &volumesnapshotv1.VolumeSnapshotList{}
		err = errors.WithStack(
			r.Client.List(ctx, snapshots,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selectSnapshots},
			))
		assert.NilError(t, err)
		assert.Equal(t, len(snapshots.Items), 0)
	})
}

func TestGenerateVolumeSnapshotOfPrimaryPgdata(t *testing.T) {
	// ctx := context.Background()
	_, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 1)

	r := &Reconciler{
		Client: cc,
		Owner:  client.FieldOwner(t.Name()),
	}
	ns := setupNamespace(t, cc)

	t.Run("NoPrimary", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns.Name
		instances := newObservedInstances(cluster, nil, nil)
		volumes := []corev1.PersistentVolumeClaim{}
		backupJob := &batchv1.Job{}

		snapshot, err := r.generateVolumeSnapshotOfPrimaryPgdata(cluster, instances, volumes, backupJob)
		assert.Error(t, err, "Could not find primary instance. Cannot create volume snapshot.")
		assert.Check(t, snapshot == nil)
	})

	t.Run("NoVolume", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns.Name
		instances := newObservedInstances(cluster,
			[]appsv1.StatefulSet{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "instance1-abc",
						Labels: map[string]string{
							"postgres-operator.crunchydata.com/instance-set": "00",
						},
					},
				},
			},
			[]corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "some-pod-name",
						Labels: map[string]string{
							"postgres-operator.crunchydata.com/instance-set": "00",
							"postgres-operator.crunchydata.com/instance":     "instance1-abc",
							"postgres-operator.crunchydata.com/role":         "master",
						},
					},
				},
			})
		volumes := []corev1.PersistentVolumeClaim{}
		backupJob := &batchv1.Job{}

		snapshot, err := r.generateVolumeSnapshotOfPrimaryPgdata(cluster, instances, volumes, backupJob)
		assert.Error(t, err, "Could not find primary's pgdata pvc. Cannot create volume snapshot.")
		assert.Check(t, snapshot == nil)
	})

	t.Run("Success", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns.Name
		cluster.Spec.Backups.Snapshots = &v1beta1.VolumeSnapshots{
			VolumeSnapshotClassName: "my-volume-snapshot-class",
		}
		cluster.ObjectMeta.UID = "the-uid-123"
		instances := newObservedInstances(cluster,
			[]appsv1.StatefulSet{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "instance1-abc",
						Labels: map[string]string{
							"postgres-operator.crunchydata.com/instance-set": "00",
						},
					},
				},
			},
			[]corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "some-pod-name",
						Labels: map[string]string{
							"postgres-operator.crunchydata.com/instance-set": "00",
							"postgres-operator.crunchydata.com/instance":     "instance1-abc",
							"postgres-operator.crunchydata.com/role":         "master",
						},
					},
				},
			},
		)
		volumes := []corev1.PersistentVolumeClaim{{
			ObjectMeta: metav1.ObjectMeta{
				Name: "instance1-abc-def",
				Labels: map[string]string{
					naming.LabelRole:        naming.RolePostgresData,
					naming.LabelInstanceSet: "instance1",
					naming.LabelInstance:    "instance1-abc"},
			},
		}}
		backupJob := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name: "backup1",
				UID:  "the-uid-456",
			},
		}

		snapshot, err := r.generateVolumeSnapshotOfPrimaryPgdata(cluster, instances, volumes, backupJob)
		assert.NilError(t, err)
		assert.Equal(t, snapshot.Annotations[naming.PGBackRestBackupJobId], "the-uid-456")
	})
}

func TestGenerateVolumeSnapshot(t *testing.T) {
	// ctx := context.Background()
	_, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 1)

	r := &Reconciler{
		Client: cc,
		Owner:  client.FieldOwner(t.Name()),
	}
	ns := setupNamespace(t, cc)

	cluster := testCluster()
	cluster.Namespace = ns.Name

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "instance1-abc-def",
		},
	}
	volumeSnapshotClassName := "my-snapshot"

	snapshot, err := r.generateVolumeSnapshot(cluster, *pvc, volumeSnapshotClassName)
	assert.NilError(t, err)
	assert.Equal(t, *snapshot.Spec.VolumeSnapshotClassName, "my-snapshot")
	assert.Equal(t, *snapshot.Spec.Source.PersistentVolumeClaimName, "instance1-abc-def")
	assert.Equal(t, snapshot.Labels[naming.LabelCluster], "hippo")
	assert.Equal(t, snapshot.ObjectMeta.OwnerReferences[0].Name, "hippo")
}

func TestGetLatestCompleteBackupJob(t *testing.T) {
	t.Run("NoJobs", func(t *testing.T) {
		jobList := &batchv1.JobList{}
		latestCompleteBackupJob := getLatestCompleteBackupJob(jobList)
		assert.Check(t, latestCompleteBackupJob == nil)
	})

	t.Run("NoCompleteJobs", func(t *testing.T) {
		jobList := &batchv1.JobList{
			Items: []batchv1.Job{
				{
					Status: batchv1.JobStatus{
						Succeeded: 0,
					},
				},
				{
					Status: batchv1.JobStatus{
						Succeeded: 0,
					},
				},
			},
		}
		latestCompleteBackupJob := getLatestCompleteBackupJob(jobList)
		assert.Check(t, latestCompleteBackupJob == nil)
	})

	t.Run("OneCompleteBackupJob", func(t *testing.T) {
		currentTime := metav1.Now()
		jobList := &batchv1.JobList{
			Items: []batchv1.Job{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "backup1",
						UID:  "something-here",
					},
					Status: batchv1.JobStatus{
						Succeeded:      1,
						CompletionTime: &currentTime,
					},
				},
				{
					Status: batchv1.JobStatus{
						Succeeded: 0,
					},
				},
			},
		}
		latestCompleteBackupJob := getLatestCompleteBackupJob(jobList)
		assert.Check(t, latestCompleteBackupJob.UID == "something-here")
	})

	t.Run("TwoCompleteBackupJobs", func(t *testing.T) {
		currentTime := metav1.Now()
		earlierTime := metav1.NewTime(currentTime.AddDate(-1, 0, 0))
		assert.Check(t, earlierTime.Before(&currentTime))

		jobList := &batchv1.JobList{
			Items: []batchv1.Job{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "backup2",
						UID:  "newer-one",
					},
					Status: batchv1.JobStatus{
						Succeeded:      1,
						CompletionTime: &currentTime,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "backup1",
						UID:  "older-one",
					},
					Status: batchv1.JobStatus{
						Succeeded:      1,
						CompletionTime: &earlierTime,
					},
				},
			},
		}
		latestCompleteBackupJob := getLatestCompleteBackupJob(jobList)
		assert.Check(t, latestCompleteBackupJob.UID == "newer-one")
	})
}

func TestGetLatestSnapshotWithError(t *testing.T) {
	t.Run("NoSnapshots", func(t *testing.T) {
		snapshotList := &volumesnapshotv1.VolumeSnapshotList{}
		latestSnapshotWithError := getLatestSnapshotWithError(snapshotList)
		assert.Check(t, latestSnapshotWithError == nil)
	})

	t.Run("NoSnapshotsWithErrors", func(t *testing.T) {
		snapshotList := &volumesnapshotv1.VolumeSnapshotList{
			Items: []volumesnapshotv1.VolumeSnapshot{
				{
					Status: &volumesnapshotv1.VolumeSnapshotStatus{
						ReadyToUse: initialize.Bool(true),
					},
				},
				{
					Status: &volumesnapshotv1.VolumeSnapshotStatus{
						ReadyToUse: initialize.Bool(false),
					},
				},
			},
		}
		latestSnapshotWithError := getLatestSnapshotWithError(snapshotList)
		assert.Check(t, latestSnapshotWithError == nil)
	})

	t.Run("OneSnapshotWithError", func(t *testing.T) {
		currentTime := metav1.Now()
		snapshotList := &volumesnapshotv1.VolumeSnapshotList{
			Items: []volumesnapshotv1.VolumeSnapshot{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "good-snapshot",
						UID:  "the-uid-123",
					},
					Status: &volumesnapshotv1.VolumeSnapshotStatus{
						ReadyToUse: initialize.Bool(true),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bad-snapshot",
						UID:  "the-uid-456",
					},
					Status: &volumesnapshotv1.VolumeSnapshotStatus{
						CreationTime: &currentTime,
						ReadyToUse:   initialize.Bool(false),
						Error:        &volumesnapshotv1.VolumeSnapshotError{},
					},
				},
			},
		}
		latestSnapshotWithError := getLatestSnapshotWithError(snapshotList)
		assert.Equal(t, latestSnapshotWithError.ObjectMeta.Name, "bad-snapshot")
	})

	t.Run("TwoSnapshotsWithErrors", func(t *testing.T) {
		currentTime := metav1.Now()
		earlierTime := metav1.NewTime(currentTime.AddDate(-1, 0, 0))
		snapshotList := &volumesnapshotv1.VolumeSnapshotList{
			Items: []volumesnapshotv1.VolumeSnapshot{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "first-bad-snapshot",
						UID:  "the-uid-123",
					},
					Status: &volumesnapshotv1.VolumeSnapshotStatus{
						CreationTime: &earlierTime,
						ReadyToUse:   initialize.Bool(false),
						Error:        &volumesnapshotv1.VolumeSnapshotError{},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "second-bad-snapshot",
						UID:  "the-uid-456",
					},
					Status: &volumesnapshotv1.VolumeSnapshotStatus{
						CreationTime: &currentTime,
						ReadyToUse:   initialize.Bool(false),
						Error:        &volumesnapshotv1.VolumeSnapshotError{},
					},
				},
			},
		}
		latestSnapshotWithError := getLatestSnapshotWithError(snapshotList)
		assert.Equal(t, latestSnapshotWithError.ObjectMeta.Name, "second-bad-snapshot")
	})
}

func TestGetLatestReadySnapshot(t *testing.T) {
	t.Run("NoSnapshots", func(t *testing.T) {
		snapshotList := &volumesnapshotv1.VolumeSnapshotList{}
		latestReadySnapshot := getLatestReadySnapshot(snapshotList)
		assert.Check(t, latestReadySnapshot == nil)
	})

	t.Run("NoReadySnapshots", func(t *testing.T) {
		snapshotList := &volumesnapshotv1.VolumeSnapshotList{
			Items: []volumesnapshotv1.VolumeSnapshot{
				{
					Status: &volumesnapshotv1.VolumeSnapshotStatus{
						ReadyToUse: initialize.Bool(false),
					},
				},
				{
					Status: &volumesnapshotv1.VolumeSnapshotStatus{
						ReadyToUse: initialize.Bool(false),
					},
				},
			},
		}
		latestSnapshotWithError := getLatestReadySnapshot(snapshotList)
		assert.Check(t, latestSnapshotWithError == nil)
	})

	t.Run("OneReadySnapshot", func(t *testing.T) {
		currentTime := metav1.Now()
		earlierTime := metav1.NewTime(currentTime.AddDate(-1, 0, 0))
		snapshotList := &volumesnapshotv1.VolumeSnapshotList{
			Items: []volumesnapshotv1.VolumeSnapshot{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "good-snapshot",
						UID:  "the-uid-123",
					},
					Status: &volumesnapshotv1.VolumeSnapshotStatus{
						CreationTime: &earlierTime,
						ReadyToUse:   initialize.Bool(true),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bad-snapshot",
						UID:  "the-uid-456",
					},
					Status: &volumesnapshotv1.VolumeSnapshotStatus{
						CreationTime: &currentTime,
						ReadyToUse:   initialize.Bool(false),
					},
				},
			},
		}
		latestReadySnapshot := getLatestReadySnapshot(snapshotList)
		assert.Equal(t, latestReadySnapshot.ObjectMeta.Name, "good-snapshot")
	})

	t.Run("TwoReadySnapshots", func(t *testing.T) {
		currentTime := metav1.Now()
		earlierTime := metav1.NewTime(currentTime.AddDate(-1, 0, 0))
		snapshotList := &volumesnapshotv1.VolumeSnapshotList{
			Items: []volumesnapshotv1.VolumeSnapshot{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "first-good-snapshot",
						UID:  "the-uid-123",
					},
					Status: &volumesnapshotv1.VolumeSnapshotStatus{
						CreationTime: &earlierTime,
						ReadyToUse:   initialize.Bool(true),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "second-good-snapshot",
						UID:  "the-uid-456",
					},
					Status: &volumesnapshotv1.VolumeSnapshotStatus{
						CreationTime: &currentTime,
						ReadyToUse:   initialize.Bool(true),
					},
				},
			},
		}
		latestReadySnapshot := getLatestReadySnapshot(snapshotList)
		assert.Equal(t, latestReadySnapshot.ObjectMeta.Name, "second-good-snapshot")
	})
}

func TestGetSnapshotsForCluster(t *testing.T) {
	ctx := context.Background()
	_, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 1)

	r := &Reconciler{
		Client: cc,
		Owner:  client.FieldOwner(t.Name()),
	}
	ns := setupNamespace(t, cc)

	cluster := testCluster()
	cluster.Namespace = ns.Name

	t.Run("NoSnapshots", func(t *testing.T) {
		snapshots, err := r.getSnapshotsForCluster(ctx, cluster)
		assert.NilError(t, err)
		assert.Equal(t, len(snapshots.Items), 0)
	})

	t.Run("NoSnapshotsForCluster", func(t *testing.T) {
		snapshot := &volumesnapshotv1.VolumeSnapshot{
			TypeMeta: metav1.TypeMeta{
				APIVersion: volumesnapshotv1.SchemeGroupVersion.String(),
				Kind:       "VolumeSnapshot",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "some-snapshot",
				Namespace: ns.Name,
				Labels: map[string]string{
					naming.LabelCluster: "rhino",
				},
			},
		}
		snapshot.Spec.Source.PersistentVolumeClaimName = initialize.String("some-pvc-name")
		snapshot.Spec.VolumeSnapshotClassName = initialize.String("some-class-name")
		err := r.apply(ctx, snapshot)
		assert.NilError(t, err)

		snapshots, err := r.getSnapshotsForCluster(ctx, cluster)
		assert.NilError(t, err)
		assert.Equal(t, len(snapshots.Items), 0)
	})

	t.Run("OneSnapshotForCluster", func(t *testing.T) {
		snapshot1 := &volumesnapshotv1.VolumeSnapshot{
			TypeMeta: metav1.TypeMeta{
				APIVersion: volumesnapshotv1.SchemeGroupVersion.String(),
				Kind:       "VolumeSnapshot",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "some-snapshot",
				Namespace: ns.Name,
				Labels: map[string]string{
					naming.LabelCluster: "rhino",
				},
			},
		}
		snapshot1.Spec.Source.PersistentVolumeClaimName = initialize.String("some-pvc-name")
		snapshot1.Spec.VolumeSnapshotClassName = initialize.String("some-class-name")
		err := r.apply(ctx, snapshot1)
		assert.NilError(t, err)

		snapshot2 := &volumesnapshotv1.VolumeSnapshot{
			TypeMeta: metav1.TypeMeta{
				APIVersion: volumesnapshotv1.SchemeGroupVersion.String(),
				Kind:       "VolumeSnapshot",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "another-snapshot",
				Namespace: ns.Name,
				Labels: map[string]string{
					naming.LabelCluster: "hippo",
				},
			},
		}
		snapshot2.Spec.Source.PersistentVolumeClaimName = initialize.String("another-pvc-name")
		snapshot2.Spec.VolumeSnapshotClassName = initialize.String("another-class-name")
		err = r.apply(ctx, snapshot2)
		assert.NilError(t, err)

		snapshots, err := r.getSnapshotsForCluster(ctx, cluster)
		assert.NilError(t, err)
		assert.Equal(t, len(snapshots.Items), 1)
		assert.Equal(t, snapshots.Items[0].Name, "another-snapshot")
	})

	t.Run("TwoSnapshotsForCluster", func(t *testing.T) {
		snapshot1 := &volumesnapshotv1.VolumeSnapshot{
			TypeMeta: metav1.TypeMeta{
				APIVersion: volumesnapshotv1.SchemeGroupVersion.String(),
				Kind:       "VolumeSnapshot",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "some-snapshot",
				Namespace: ns.Name,
				Labels: map[string]string{
					naming.LabelCluster: "hippo",
				},
			},
		}
		snapshot1.Spec.Source.PersistentVolumeClaimName = initialize.String("some-pvc-name")
		snapshot1.Spec.VolumeSnapshotClassName = initialize.String("some-class-name")
		err := r.apply(ctx, snapshot1)
		assert.NilError(t, err)

		snapshot2 := &volumesnapshotv1.VolumeSnapshot{
			TypeMeta: metav1.TypeMeta{
				APIVersion: volumesnapshotv1.SchemeGroupVersion.String(),
				Kind:       "VolumeSnapshot",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "another-snapshot",
				Namespace: ns.Name,
				Labels: map[string]string{
					naming.LabelCluster: "hippo",
				},
			},
		}
		snapshot2.Spec.Source.PersistentVolumeClaimName = initialize.String("another-pvc-name")
		snapshot2.Spec.VolumeSnapshotClassName = initialize.String("another-class-name")
		err = r.apply(ctx, snapshot2)
		assert.NilError(t, err)

		snapshots, err := r.getSnapshotsForCluster(ctx, cluster)
		assert.NilError(t, err)
		assert.Equal(t, len(snapshots.Items), 2)
	})
}
