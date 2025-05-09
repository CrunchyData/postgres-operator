// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/kubernetes"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/events"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestReconcileVolumeSnapshots(t *testing.T) {
	ctx := context.Background()
	_, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 1)

	recorder := events.NewRecorder(t, runtime.Scheme)
	r := &Reconciler{
		Client:   cc,
		Owner:    client.FieldOwner(t.Name()),
		Recorder: recorder,
	}
	ns := setupNamespace(t, cc)

	// Enable snapshots feature gate and API
	gate := feature.NewGate()
	assert.NilError(t, gate.SetFromMap(map[string]bool{
		feature.VolumeSnapshots: true,
	}))
	ctx = feature.NewContext(ctx, gate)
	ctx = kubernetes.NewAPIContext(ctx, kubernetes.NewAPISet(
		volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshot"),
	))

	t.Run("SnapshotsDisabledDeleteSnapshots", func(t *testing.T) {
		// Create cluster (without snapshots spec)
		cluster := testCluster()
		cluster.Namespace = ns.Name
		cluster.UID = "the-uid-123"
		assert.NilError(t, r.Client.Create(ctx, cluster))
		t.Cleanup(func() { assert.Check(t, r.Client.Delete(ctx, cluster)) })

		// Create a snapshot
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dedicated-snapshot-volume",
			},
		}
		volumeSnapshotClassName := "my-snapshotclass"
		snapshot, err := r.generateVolumeSnapshot(cluster, *pvc, volumeSnapshotClassName)
		assert.NilError(t, err)
		assert.NilError(t, r.Client.Create(ctx, snapshot))

		// Get all snapshots for this cluster and assert 1 exists
		selectSnapshots, err := naming.AsSelector(naming.Cluster(cluster.Name))
		assert.NilError(t, err)
		snapshots := &volumesnapshotv1.VolumeSnapshotList{}
		assert.NilError(t,
			r.Client.List(ctx, snapshots,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selectSnapshots},
			))
		assert.Equal(t, len(snapshots.Items), 1)

		// Reconcile snapshots
		assert.NilError(t, r.reconcileVolumeSnapshots(ctx, cluster, pvc))

		// Get all snapshots for this cluster and assert 0 exist
		snapshots = &volumesnapshotv1.VolumeSnapshotList{}
		assert.NilError(t,
			r.Client.List(ctx, snapshots,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selectSnapshots},
			))
		assert.Equal(t, len(snapshots.Items), 0)
	})

	t.Run("SnapshotsEnabledTablespacesEnabled", func(t *testing.T) {
		// Enable both tablespaces and snapshots feature gates
		gate := feature.NewGate()
		assert.NilError(t, gate.SetFromMap(map[string]bool{
			feature.TablespaceVolumes: true,
			feature.VolumeSnapshots:   true,
		}))
		ctx := feature.NewContext(ctx, gate)

		// Create a cluster with snapshots and tablespaces enabled
		volumeSnapshotClassName := "my-snapshotclass"
		cluster := testCluster()
		cluster.Namespace = ns.Name
		cluster.Spec.Backups.Snapshots = &v1beta1.VolumeSnapshots{
			VolumeSnapshotClassName: volumeSnapshotClassName,
		}
		cluster.Spec.InstanceSets[0].TablespaceVolumes = []v1beta1.TablespaceVolume{{
			Name: "volume-1",
		}}

		// Create pvc for reconcile
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dedicated-snapshot-volume",
			},
		}

		// Reconcile
		assert.NilError(t, r.reconcileVolumeSnapshots(ctx, cluster, pvc))

		// Assert warning event was created and has expected attributes
		if assert.Check(t, len(recorder.Events) > 0) {
			assert.Equal(t, recorder.Events[0].Type, "Warning")
			assert.Equal(t, recorder.Events[0].Regarding.Kind, "PostgresCluster")
			assert.Equal(t, recorder.Events[0].Regarding.Name, "hippo")
			assert.Equal(t, recorder.Events[0].Reason, "IncompatibleFeatures")
			assert.Assert(t, cmp.Contains(recorder.Events[0].Note, "VolumeSnapshots not currently compatible with TablespaceVolumes"))
		}
	})

	t.Run("SnapshotsEnabledNoPvcAnnotation", func(t *testing.T) {
		// Create a volume snapshot class
		volumeSnapshotClassName := "my-snapshotclass"
		volumeSnapshotClass := &volumesnapshotv1.VolumeSnapshotClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeSnapshotClassName,
			},
			DeletionPolicy: "Delete",
		}
		assert.NilError(t, r.Client.Create(ctx, volumeSnapshotClass))
		t.Cleanup(func() { assert.Check(t, r.Client.Delete(ctx, volumeSnapshotClass)) })

		// Create a cluster with snapshots enabled
		cluster := testCluster()
		cluster.Namespace = ns.Name
		cluster.Spec.Backups.Snapshots = &v1beta1.VolumeSnapshots{
			VolumeSnapshotClassName: volumeSnapshotClassName,
		}
		assert.NilError(t, r.Client.Create(ctx, cluster))
		t.Cleanup(func() { assert.Check(t, r.Client.Delete(ctx, cluster)) })

		// Create pvc for reconcile
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dedicated-snapshot-volume",
			},
		}

		// Reconcile
		assert.NilError(t, r.reconcileVolumeSnapshots(ctx, cluster, pvc))

		// Assert no snapshots exist
		selectSnapshots, err := naming.AsSelector(naming.Cluster(cluster.Name))
		assert.NilError(t, err)
		snapshots := &volumesnapshotv1.VolumeSnapshotList{}
		assert.NilError(t,
			r.Client.List(ctx, snapshots,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selectSnapshots},
			))
		assert.Equal(t, len(snapshots.Items), 0)
	})

	t.Run("SnapshotsEnabledReadySnapshotsExist", func(t *testing.T) {
		if strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
			t.Skip("requires mocking of Job conditions")
		}
		// Create a volume snapshot class
		volumeSnapshotClassName := "my-snapshotclass"
		volumeSnapshotClass := &volumesnapshotv1.VolumeSnapshotClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeSnapshotClassName,
			},
			DeletionPolicy: "Delete",
		}
		assert.NilError(t, r.Client.Create(ctx, volumeSnapshotClass))
		t.Cleanup(func() { assert.Check(t, r.Client.Delete(ctx, volumeSnapshotClass)) })

		// Create a cluster with snapshots enabled
		cluster := testCluster()
		cluster.Namespace = ns.Name
		cluster.UID = "the-uid-123"
		cluster.Spec.Backups.Snapshots = &v1beta1.VolumeSnapshots{
			VolumeSnapshotClassName: volumeSnapshotClassName,
		}
		assert.NilError(t, r.Client.Create(ctx, cluster))
		t.Cleanup(func() { assert.Check(t, r.Client.Delete(ctx, cluster)) })

		// Create pvc with annotation
		pvcName := initialize.String("dedicated-snapshot-volume")
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: *pvcName,
				Annotations: map[string]string{
					naming.PGBackRestBackupJobCompletion: "backup-timestamp",
				},
			},
		}

		// Create snapshot with annotation matching the pvc annotation
		snapshot1 := &volumesnapshotv1.VolumeSnapshot{
			TypeMeta: metav1.TypeMeta{
				APIVersion: volumesnapshotv1.SchemeGroupVersion.String(),
				Kind:       "VolumeSnapshot",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "first-snapshot",
				Namespace: ns.Name,
				Annotations: map[string]string{
					naming.PGBackRestBackupJobCompletion: "backup-timestamp",
				},
				Labels: map[string]string{
					naming.LabelCluster: "hippo",
				},
			},
			Spec: volumesnapshotv1.VolumeSnapshotSpec{
				Source: volumesnapshotv1.VolumeSnapshotSource{
					PersistentVolumeClaimName: pvcName,
				},
			},
		}
		assert.NilError(t, r.setControllerReference(cluster, snapshot1))
		assert.NilError(t, r.Client.Create(ctx, snapshot1))

		// Update snapshot status
		truePtr := initialize.Bool(true)
		snapshot1.Status = &volumesnapshotv1.VolumeSnapshotStatus{
			ReadyToUse: truePtr,
		}
		assert.NilError(t, r.Client.Status().Update(ctx, snapshot1))

		// Create second snapshot with different annotation value
		snapshot2 := &volumesnapshotv1.VolumeSnapshot{
			TypeMeta: metav1.TypeMeta{
				APIVersion: volumesnapshotv1.SchemeGroupVersion.String(),
				Kind:       "VolumeSnapshot",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "second-snapshot",
				Namespace: ns.Name,
				Annotations: map[string]string{
					naming.PGBackRestBackupJobCompletion: "older-backup-timestamp",
				},
				Labels: map[string]string{
					naming.LabelCluster: "hippo",
				},
			},
			Spec: volumesnapshotv1.VolumeSnapshotSpec{
				Source: volumesnapshotv1.VolumeSnapshotSource{
					PersistentVolumeClaimName: pvcName,
				},
			},
		}
		assert.NilError(t, r.setControllerReference(cluster, snapshot2))
		assert.NilError(t, r.Client.Create(ctx, snapshot2))

		// Update second snapshot's status
		snapshot2.Status = &volumesnapshotv1.VolumeSnapshotStatus{
			ReadyToUse: truePtr,
		}
		assert.NilError(t, r.Client.Status().Update(ctx, snapshot2))

		// Reconcile
		assert.NilError(t, r.reconcileVolumeSnapshots(ctx, cluster, pvc))

		// Assert first snapshot exists and second snapshot was deleted
		selectSnapshots, err := naming.AsSelector(naming.Cluster(cluster.Name))
		assert.NilError(t, err)
		snapshots := &volumesnapshotv1.VolumeSnapshotList{}
		assert.NilError(t,
			r.Client.List(ctx, snapshots,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selectSnapshots},
			))
		assert.Equal(t, len(snapshots.Items), 1)
		assert.Equal(t, snapshots.Items[0].Name, "first-snapshot")

		// Cleanup
		assert.NilError(t, r.deleteControlled(ctx, cluster, snapshot1))
	})

	t.Run("SnapshotsEnabledCreateSnapshot", func(t *testing.T) {
		// Create a volume snapshot class
		volumeSnapshotClassName := "my-snapshotclass"
		volumeSnapshotClass := &volumesnapshotv1.VolumeSnapshotClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeSnapshotClassName,
			},
			DeletionPolicy: "Delete",
		}
		assert.NilError(t, r.Client.Create(ctx, volumeSnapshotClass))
		t.Cleanup(func() { assert.Check(t, r.Client.Delete(ctx, volumeSnapshotClass)) })

		// Create a cluster with snapshots enabled
		cluster := testCluster()
		cluster.Namespace = ns.Name
		cluster.UID = "the-uid-123"
		cluster.Spec.Backups.Snapshots = &v1beta1.VolumeSnapshots{
			VolumeSnapshotClassName: volumeSnapshotClassName,
		}
		assert.NilError(t, r.Client.Create(ctx, cluster))
		t.Cleanup(func() { assert.Check(t, r.Client.Delete(ctx, cluster)) })

		// Create pvc with annotation
		pvcName := initialize.String("dedicated-snapshot-volume")
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: *pvcName,
				Annotations: map[string]string{
					naming.PGBackRestBackupJobCompletion: "another-backup-timestamp",
				},
			},
		}

		// Reconcile
		assert.NilError(t, r.reconcileVolumeSnapshots(ctx, cluster, pvc))

		// Assert that a snapshot was created
		selectSnapshots, err := naming.AsSelector(naming.Cluster(cluster.Name))
		assert.NilError(t, err)
		snapshots := &volumesnapshotv1.VolumeSnapshotList{}
		assert.NilError(t,
			r.Client.List(ctx, snapshots,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selectSnapshots},
			))
		assert.Equal(t, len(snapshots.Items), 1)
		assert.Equal(t, snapshots.Items[0].Annotations[naming.PGBackRestBackupJobCompletion],
			"another-backup-timestamp")
	})
}

func TestReconcileDedicatedSnapshotVolume(t *testing.T) {
	ctx := context.Background()
	_, cc := setupKubernetes(t)

	recorder := events.NewRecorder(t, runtime.Scheme)
	r := &Reconciler{
		Client:   cc,
		Owner:    client.FieldOwner(t.Name()),
		Recorder: recorder,
	}

	// Enable snapshots feature gate
	gate := feature.NewGate()
	assert.NilError(t, gate.SetFromMap(map[string]bool{
		feature.VolumeSnapshots: true,
	}))
	ctx = feature.NewContext(ctx, gate)

	t.Run("SnapshotsDisabledDeletePvc", func(t *testing.T) {
		// Create cluster without snapshots spec
		ns := setupNamespace(t, cc)
		cluster := testCluster()
		cluster.Namespace = ns.Name
		cluster.UID = "the-uid-123"
		assert.NilError(t, r.Client.Create(ctx, cluster))
		t.Cleanup(func() { assert.Check(t, r.Client.Delete(ctx, cluster)) })

		// Create a dedicated snapshot volume
		pvc := &corev1.PersistentVolumeClaim{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PersistentVolumeClaim",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dedicated-snapshot-volume",
				Namespace: ns.Name,
				Labels: map[string]string{
					naming.LabelCluster: cluster.Name,
					naming.LabelRole:    naming.RoleSnapshot,
					naming.LabelData:    naming.DataPostgres,
				},
			},
		}
		spec := testVolumeClaimSpec()
		pvc.Spec = spec.AsPersistentVolumeClaimSpec()
		assert.NilError(t, r.setControllerReference(cluster, pvc))
		assert.NilError(t, r.Client.Create(ctx, pvc))

		// Assert that the pvc was created
		selectPvcs, err := naming.AsSelector(naming.Cluster(cluster.Name))
		assert.NilError(t, err)
		pvcs := &corev1.PersistentVolumeClaimList{}
		assert.NilError(t,
			r.Client.List(ctx, pvcs,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selectPvcs},
			))
		assert.Equal(t, len(pvcs.Items), 1)

		// Create volumes for reconcile
		clusterVolumes := []*corev1.PersistentVolumeClaim{pvc}

		// Reconcile
		returned, err := r.reconcileDedicatedSnapshotVolume(ctx, cluster, clusterVolumes)
		assert.NilError(t, err)
		assert.Check(t, returned == nil)

		// Assert that the pvc has been deleted or marked for deletion
		key, fetched := client.ObjectKeyFromObject(pvc), &corev1.PersistentVolumeClaim{}
		if err := r.Client.Get(ctx, key, fetched); err == nil {
			assert.Assert(t, fetched.DeletionTimestamp != nil, "expected deleted")
		} else {
			assert.Assert(t, apierrors.IsNotFound(err), "expected NotFound, got %v", err)
		}
	})

	t.Run("SnapshotsEnabledCreatePvcNoBackupNoRestore", func(t *testing.T) {
		// Create cluster with snapshots enabled
		ns := setupNamespace(t, cc)
		cluster := testCluster()
		cluster.Namespace = ns.Name
		cluster.UID = "the-uid-123"
		cluster.Spec.Backups.Snapshots = &v1beta1.VolumeSnapshots{
			VolumeSnapshotClassName: "my-snapshotclass",
		}
		assert.NilError(t, r.Client.Create(ctx, cluster))
		t.Cleanup(func() { assert.Check(t, r.Client.Delete(ctx, cluster)) })

		// Create volumes for reconcile
		clusterVolumes := []*corev1.PersistentVolumeClaim{}

		// Reconcile
		pvc, err := r.reconcileDedicatedSnapshotVolume(ctx, cluster, clusterVolumes)
		assert.NilError(t, err)
		assert.Assert(t, pvc != nil)

		// Assert pvc was created
		selectPvcs, err := naming.AsSelector(naming.Cluster(cluster.Name))
		assert.NilError(t, err)
		pvcs := &corev1.PersistentVolumeClaimList{}
		assert.NilError(t,
			r.Client.List(ctx, pvcs,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selectPvcs},
			))
		assert.Equal(t, len(pvcs.Items), 1)
	})

	t.Run("SnapshotsEnabledBackupExistsCreateRestore", func(t *testing.T) {
		if strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
			t.Skip("requires mocking of Job conditions")
		}
		// Create cluster with snapshots enabled
		ns := setupNamespace(t, cc)
		cluster := testCluster()
		cluster.Namespace = ns.Name
		cluster.UID = "the-uid-123"
		cluster.Spec.Backups.Snapshots = &v1beta1.VolumeSnapshots{
			VolumeSnapshotClassName: "my-snapshotclass",
		}
		assert.NilError(t, r.Client.Create(ctx, cluster))
		t.Cleanup(func() { assert.Check(t, r.Client.Delete(ctx, cluster)) })

		// Create successful backup job
		backupJob := testBackupJob(cluster)
		assert.NilError(t, r.setControllerReference(cluster, backupJob))
		assert.NilError(t, r.Client.Create(ctx, backupJob))

		currentTime := metav1.Now()
		startTime := metav1.NewTime(currentTime.AddDate(0, 0, -1))
		backupJob.Status = succeededJobStatus(startTime, currentTime)
		assert.NilError(t, r.Client.Status().Update(ctx, backupJob))

		// Create instance set and volumes for reconcile
		sts := &appsv1.StatefulSet{}
		generateInstanceStatefulSetIntent(ctx, cluster, &cluster.Spec.InstanceSets[0], "pod-service", "service-account", sts, 1)
		clusterVolumes := []*corev1.PersistentVolumeClaim{}

		// Reconcile
		pvc, err := r.reconcileDedicatedSnapshotVolume(ctx, cluster, clusterVolumes)
		assert.NilError(t, err)
		assert.Assert(t, pvc != nil)

		// Assert restore job with annotation was created
		restoreJobs := &batchv1.JobList{}
		selectJobs, err := naming.AsSelector(naming.ClusterRestoreJobs(cluster.Name))
		assert.NilError(t, err)
		assert.NilError(t,
			r.Client.List(ctx, restoreJobs,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selectJobs},
			))
		assert.Equal(t, len(restoreJobs.Items), 1)
		assert.Assert(t, restoreJobs.Items[0].Annotations[naming.PGBackRestBackupJobCompletion] != "")
	})

	t.Run("SnapshotsEnabledSuccessfulRestoreExists", func(t *testing.T) {
		if strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
			t.Skip("requires mocking of Job conditions")
		}
		// Create cluster with snapshots enabled
		ns := setupNamespace(t, cc)
		cluster := testCluster()
		cluster.Namespace = ns.Name
		cluster.UID = "the-uid-123"
		cluster.Spec.Backups.Snapshots = &v1beta1.VolumeSnapshots{
			VolumeSnapshotClassName: "my-snapshotclass",
		}
		assert.NilError(t, r.Client.Create(ctx, cluster))
		t.Cleanup(func() { assert.Check(t, r.Client.Delete(ctx, cluster)) })

		// Create times for jobs
		currentTime := metav1.Now()
		currentStartTime := metav1.NewTime(currentTime.AddDate(0, 0, -1))
		earlierTime := metav1.NewTime(currentTime.AddDate(-1, 0, 0))
		earlierStartTime := metav1.NewTime(earlierTime.AddDate(0, 0, -1))

		// Create successful backup job
		backupJob := testBackupJob(cluster)
		assert.NilError(t, r.setControllerReference(cluster, backupJob))
		assert.NilError(t, r.Client.Create(ctx, backupJob))

		backupJob.Status = succeededJobStatus(earlierStartTime, earlierTime)
		assert.NilError(t, r.Client.Status().Update(ctx, backupJob))

		// Create successful restore job
		restoreJob := testRestoreJob(cluster)
		restoreJob.Annotations = map[string]string{
			naming.PGBackRestBackupJobCompletion: backupJob.Status.CompletionTime.Format(time.RFC3339),
		}
		assert.NilError(t, r.setControllerReference(cluster, restoreJob))
		assert.NilError(t, r.Client.Create(ctx, restoreJob))

		restoreJob.Status = succeededJobStatus(currentStartTime, currentTime)
		assert.NilError(t, r.Client.Status().Update(ctx, restoreJob))

		// Create instance set and volumes for reconcile
		sts := &appsv1.StatefulSet{}
		generateInstanceStatefulSetIntent(ctx, cluster, &cluster.Spec.InstanceSets[0], "pod-service", "service-account", sts, 1)
		clusterVolumes := []*corev1.PersistentVolumeClaim{}

		// Reconcile
		pvc, err := r.reconcileDedicatedSnapshotVolume(ctx, cluster, clusterVolumes)
		assert.NilError(t, err)
		assert.Assert(t, pvc != nil)

		// Assert restore job was deleted
		restoreJobs := &batchv1.JobList{}
		selectJobs, err := naming.AsSelector(naming.ClusterRestoreJobs(cluster.Name))
		assert.NilError(t, err)
		assert.NilError(t,
			r.Client.List(ctx, restoreJobs,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selectJobs},
			))
		assert.Equal(t, len(restoreJobs.Items), 0)

		// Assert pvc was annotated
		assert.Equal(t, pvc.GetAnnotations()[naming.PGBackRestBackupJobCompletion], backupJob.Status.CompletionTime.Format(time.RFC3339))
	})

	t.Run("SnapshotsEnabledFailedRestoreExists", func(t *testing.T) {
		if strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
			t.Skip("requires mocking of Job conditions")
		}
		// Create cluster with snapshots enabled
		ns := setupNamespace(t, cc)
		cluster := testCluster()
		cluster.Namespace = ns.Name
		cluster.UID = "the-uid-123"
		cluster.Spec.Backups.Snapshots = &v1beta1.VolumeSnapshots{
			VolumeSnapshotClassName: "my-snapshotclass",
		}
		assert.NilError(t, r.Client.Create(ctx, cluster))
		t.Cleanup(func() { assert.Check(t, r.Client.Delete(ctx, cluster)) })

		// Create times for jobs
		currentTime := metav1.Now()
		earlierTime := metav1.NewTime(currentTime.AddDate(-1, 0, 0))
		startTime := metav1.NewTime(earlierTime.AddDate(0, 0, -1))

		// Create successful backup job
		backupJob := testBackupJob(cluster)
		assert.NilError(t, r.setControllerReference(cluster, backupJob))
		assert.NilError(t, r.Client.Create(ctx, backupJob))

		backupJob.Status = succeededJobStatus(startTime, earlierTime)
		assert.NilError(t, r.Client.Status().Update(ctx, backupJob))

		// Create failed restore job
		restoreJob := testRestoreJob(cluster)
		restoreJob.Annotations = map[string]string{
			naming.PGBackRestBackupJobCompletion: backupJob.Status.CompletionTime.Format(time.RFC3339),
		}
		assert.NilError(t, r.setControllerReference(cluster, restoreJob))
		assert.NilError(t, r.Client.Create(ctx, restoreJob))

		restoreJob.Status = batchv1.JobStatus{
			Succeeded: 0,
			Failed:    1,
		}
		assert.NilError(t, r.Client.Status().Update(ctx, restoreJob))

		// Setup instances and volumes for reconcile
		sts := &appsv1.StatefulSet{}
		generateInstanceStatefulSetIntent(ctx, cluster, &cluster.Spec.InstanceSets[0], "pod-service", "service-account", sts, 1)
		clusterVolumes := []*corev1.PersistentVolumeClaim{}

		// Reconcile
		pvc, err := r.reconcileDedicatedSnapshotVolume(ctx, cluster, clusterVolumes)
		assert.NilError(t, err)
		assert.Assert(t, pvc != nil)

		// Assert warning event was created and has expected attributes
		if assert.Check(t, len(recorder.Events) > 0) {
			assert.Equal(t, recorder.Events[0].Type, "Warning")
			assert.Equal(t, recorder.Events[0].Regarding.Kind, "PostgresCluster")
			assert.Equal(t, recorder.Events[0].Regarding.Name, "hippo")
			assert.Equal(t, recorder.Events[0].Reason, "DedicatedSnapshotVolumeRestoreJobError")
			assert.Assert(t, cmp.Contains(recorder.Events[0].Note, "restore job failed, check the logs"))
		}
	})
}

func TestCreateDedicatedSnapshotVolume(t *testing.T) {
	ctx := context.Background()
	_, cc := setupKubernetes(t)

	r := &Reconciler{
		Client: cc,
		Owner:  client.FieldOwner(t.Name()),
	}

	ns := setupNamespace(t, cc)
	cluster := testCluster()
	cluster.Namespace = ns.Name
	cluster.UID = "the-uid-123"

	labelMap := map[string]string{
		naming.LabelCluster: cluster.Name,
		naming.LabelRole:    naming.RoleSnapshot,
		naming.LabelData:    naming.DataPostgres,
	}
	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: naming.ClusterDedicatedSnapshotVolume(cluster)}
	pvc.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"))

	pvc, err := r.createDedicatedSnapshotVolume(ctx, cluster, labelMap, pvc)
	assert.NilError(t, err)
	assert.Assert(t, metav1.IsControlledBy(pvc, cluster))
	assert.Equal(t, pvc.Spec.Resources.Requests[corev1.ResourceStorage], resource.MustParse("1Gi"))
}

func TestDedicatedSnapshotVolumeRestore(t *testing.T) {
	ctx := context.Background()
	_, cc := setupKubernetes(t)

	r := &Reconciler{
		Client: cc,
		Owner:  client.FieldOwner(t.Name()),
	}

	ns := setupNamespace(t, cc)
	cluster := testCluster()
	cluster.Namespace = ns.Name
	cluster.UID = "the-uid-123"

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dedicated-snapshot-volume",
		},
	}

	sts := &appsv1.StatefulSet{}
	generateInstanceStatefulSetIntent(ctx, cluster, &cluster.Spec.InstanceSets[0], "pod-service", "service-account", sts, 1)
	currentTime := metav1.Now()
	backupJob := testBackupJob(cluster)
	backupJob.Status.CompletionTime = &currentTime

	assert.NilError(t, r.dedicatedSnapshotVolumeRestore(ctx, cluster, pvc, backupJob))

	// Assert a restore job was created that has the correct annotation
	jobs := &batchv1.JobList{}
	selectJobs, err := naming.AsSelector(naming.ClusterRestoreJobs(cluster.Name))
	assert.NilError(t, err)
	assert.NilError(t,
		r.Client.List(ctx, jobs,
			client.InNamespace(cluster.Namespace),
			client.MatchingLabelsSelector{Selector: selectJobs},
		))
	assert.Equal(t, len(jobs.Items), 1)
	assert.Equal(t, jobs.Items[0].Annotations[naming.PGBackRestBackupJobCompletion],
		backupJob.Status.CompletionTime.Format(time.RFC3339))
}

func TestGenerateSnapshotOfDedicatedSnapshotVolume(t *testing.T) {
	_, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 1)

	r := &Reconciler{
		Client: cc,
		Owner:  client.FieldOwner(t.Name()),
	}
	ns := setupNamespace(t, cc)

	cluster := testCluster()
	cluster.Namespace = ns.Name
	cluster.Spec.Backups.Snapshots = &v1beta1.VolumeSnapshots{
		VolumeSnapshotClassName: "my-snapshot",
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				naming.PGBackRestBackupJobCompletion: "backup-completion-timestamp",
			},
			Name: "dedicated-snapshot-volume",
		},
	}

	snapshot, err := r.generateSnapshotOfDedicatedSnapshotVolume(cluster, pvc)
	assert.NilError(t, err)
	assert.Equal(t, snapshot.GetAnnotations()[naming.PGBackRestBackupJobCompletion],
		"backup-completion-timestamp")
}

func TestGenerateVolumeSnapshot(t *testing.T) {
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
			Name: "dedicated-snapshot-volume",
		},
	}
	volumeSnapshotClassName := "my-snapshot"

	snapshot, err := r.generateVolumeSnapshot(cluster, *pvc, volumeSnapshotClassName)
	assert.NilError(t, err)
	assert.Equal(t, *snapshot.Spec.VolumeSnapshotClassName, "my-snapshot")
	assert.Equal(t, *snapshot.Spec.Source.PersistentVolumeClaimName, "dedicated-snapshot-volume")
	assert.Equal(t, snapshot.Labels[naming.LabelCluster], "hippo")
	assert.Equal(t, snapshot.ObjectMeta.OwnerReferences[0].Name, "hippo")
}

func TestGetDedicatedSnapshotVolumeRestoreJob(t *testing.T) {
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

	t.Run("NoRestoreJobs", func(t *testing.T) {
		dsvRestoreJob, err := r.getDedicatedSnapshotVolumeRestoreJob(ctx, cluster)
		assert.NilError(t, err)
		assert.Check(t, dsvRestoreJob == nil)
	})

	t.Run("NoDsvRestoreJobs", func(t *testing.T) {
		job1 := testRestoreJob(cluster)
		job1.Namespace = ns.Name

		err := r.Client.Create(ctx, job1)
		assert.NilError(t, err)

		dsvRestoreJob, err := r.getDedicatedSnapshotVolumeRestoreJob(ctx, cluster)
		assert.NilError(t, err)
		assert.Check(t, dsvRestoreJob == nil)
	})

	t.Run("DsvRestoreJobExists", func(t *testing.T) {
		job2 := testRestoreJob(cluster)
		job2.Name = "restore-job-2"
		job2.Namespace = ns.Name
		job2.Annotations = map[string]string{
			naming.PGBackRestBackupJobCompletion: "backup-timestamp",
		}

		err := r.Client.Create(ctx, job2)
		assert.NilError(t, err)

		job3 := testRestoreJob(cluster)
		job3.Name = "restore-job-3"
		job3.Namespace = ns.Name

		assert.NilError(t, r.Client.Create(ctx, job3))

		dsvRestoreJob, err := r.getDedicatedSnapshotVolumeRestoreJob(ctx, cluster)
		assert.NilError(t, err)
		assert.Assert(t, dsvRestoreJob != nil)
		assert.Equal(t, dsvRestoreJob.Name, "restore-job-2")
	})
}

func TestGetLatestCompleteBackupJob(t *testing.T) {
	ctx := context.Background()
	_, cc := setupKubernetes(t)

	r := &Reconciler{
		Client: cc,
		Owner:  client.FieldOwner(t.Name()),
	}
	ns := setupNamespace(t, cc)

	cluster := testCluster()
	cluster.Namespace = ns.Name

	t.Run("NoJobs", func(t *testing.T) {
		latestCompleteBackupJob, err := r.getLatestCompleteBackupJob(ctx, cluster)
		assert.NilError(t, err)
		assert.Check(t, latestCompleteBackupJob == nil)
	})

	t.Run("NoCompleteJobs", func(t *testing.T) {
		job1 := testBackupJob(cluster)
		job1.Namespace = ns.Name

		err := r.Client.Create(ctx, job1)
		assert.NilError(t, err)

		latestCompleteBackupJob, err := r.getLatestCompleteBackupJob(ctx, cluster)
		assert.NilError(t, err)
		assert.Check(t, latestCompleteBackupJob == nil)
	})

	t.Run("OneCompleteBackupJob", func(t *testing.T) {
		if strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
			t.Skip("requires mocking of Job conditions")
		}
		currentTime := metav1.Now()
		currentStartTime := metav1.NewTime(currentTime.AddDate(0, 0, -1))

		job1 := testBackupJob(cluster)
		job1.Namespace = ns.Name

		err := r.apply(ctx, job1)
		assert.NilError(t, err)

		job2 := testBackupJob(cluster)
		job2.Namespace = ns.Name
		job2.Name = "backup-job-2"

		assert.NilError(t, r.Client.Create(ctx, job2))

		// Get job1 and update Status.
		assert.NilError(t, r.Client.Get(ctx, client.ObjectKeyFromObject(job1), job1))

		job1.Status = succeededJobStatus(currentStartTime, currentTime)
		assert.NilError(t, r.Client.Status().Update(ctx, job1))

		latestCompleteBackupJob, err := r.getLatestCompleteBackupJob(ctx, cluster)
		assert.NilError(t, err)
		assert.Check(t, latestCompleteBackupJob.Name == "backup-job-1")
	})

	t.Run("TwoCompleteBackupJobs", func(t *testing.T) {
		if strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
			t.Skip("requires mocking of Job conditions")
		}
		currentTime := metav1.Now()
		currentStartTime := metav1.NewTime(currentTime.AddDate(0, 0, -1))
		earlierTime := metav1.NewTime(currentTime.AddDate(-1, 0, 0))
		earlierStartTime := metav1.NewTime(earlierTime.AddDate(0, 0, -1))
		assert.Check(t, earlierTime.Before(&currentTime))

		job1 := testBackupJob(cluster)
		job1.Namespace = ns.Name

		err := r.apply(ctx, job1)
		assert.NilError(t, err)

		job2 := testBackupJob(cluster)
		job2.Namespace = ns.Name
		job2.Name = "backup-job-2"

		assert.NilError(t, r.apply(ctx, job2))

		// Get job1 and update Status.
		assert.NilError(t, r.Client.Get(ctx, client.ObjectKeyFromObject(job1), job1))

		job1.Status = succeededJobStatus(currentStartTime, currentTime)
		assert.NilError(t, r.Client.Status().Update(ctx, job1))

		// Get job2 and update Status.
		assert.NilError(t, r.Client.Get(ctx, client.ObjectKeyFromObject(job2), job2))

		job2.Status = succeededJobStatus(earlierStartTime, earlierTime)
		assert.NilError(t, r.Client.Status().Update(ctx, job2))

		latestCompleteBackupJob, err := r.getLatestCompleteBackupJob(ctx, cluster)
		assert.NilError(t, err)
		assert.Check(t, latestCompleteBackupJob.Name == "backup-job-1")
	})
}

func TestGetSnapshotWithLatestError(t *testing.T) {
	t.Run("NoSnapshots", func(t *testing.T) {
		snapshots := []*volumesnapshotv1.VolumeSnapshot{}
		snapshotWithLatestError := getSnapshotWithLatestError(snapshots)
		assert.Check(t, snapshotWithLatestError == nil)
	})

	t.Run("NoSnapshotsWithStatus", func(t *testing.T) {
		snapshots := []*volumesnapshotv1.VolumeSnapshot{
			{},
			{},
		}
		snapshotWithLatestError := getSnapshotWithLatestError(snapshots)
		assert.Check(t, snapshotWithLatestError == nil)
	})

	t.Run("NoSnapshotsWithErrors", func(t *testing.T) {
		snapshots := []*volumesnapshotv1.VolumeSnapshot{
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
		}
		snapshotWithLatestError := getSnapshotWithLatestError(snapshots)
		assert.Check(t, snapshotWithLatestError == nil)
	})

	t.Run("OneSnapshotWithError", func(t *testing.T) {
		currentTime := metav1.Now()
		earlierTime := metav1.NewTime(currentTime.AddDate(-1, 0, 0))
		snapshots := []*volumesnapshotv1.VolumeSnapshot{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "good-snapshot",
					UID:  "the-uid-123",
				},
				Status: &volumesnapshotv1.VolumeSnapshotStatus{
					CreationTime: &currentTime,
					ReadyToUse:   initialize.Bool(true),
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "bad-snapshot",
					UID:  "the-uid-456",
				},
				Status: &volumesnapshotv1.VolumeSnapshotStatus{
					ReadyToUse: initialize.Bool(false),
					Error: &volumesnapshotv1.VolumeSnapshotError{
						Time: &earlierTime,
					},
				},
			},
		}
		snapshotWithLatestError := getSnapshotWithLatestError(snapshots)
		assert.Equal(t, snapshotWithLatestError.Name, "bad-snapshot")
	})

	t.Run("TwoSnapshotsWithErrors", func(t *testing.T) {
		currentTime := metav1.Now()
		earlierTime := metav1.NewTime(currentTime.AddDate(-1, 0, 0))
		snapshots := []*volumesnapshotv1.VolumeSnapshot{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "first-bad-snapshot",
					UID:  "the-uid-123",
				},
				Status: &volumesnapshotv1.VolumeSnapshotStatus{
					ReadyToUse: initialize.Bool(false),
					Error: &volumesnapshotv1.VolumeSnapshotError{
						Time: &earlierTime,
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "second-bad-snapshot",
					UID:  "the-uid-456",
				},
				Status: &volumesnapshotv1.VolumeSnapshotStatus{
					ReadyToUse: initialize.Bool(false),
					Error: &volumesnapshotv1.VolumeSnapshotError{
						Time: &currentTime,
					},
				},
			},
		}
		snapshotWithLatestError := getSnapshotWithLatestError(snapshots)
		assert.Equal(t, snapshotWithLatestError.Name, "second-bad-snapshot")
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
		assert.Equal(t, len(snapshots), 0)
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
		assert.NilError(t, r.Client.Create(ctx, snapshot))

		snapshots, err := r.getSnapshotsForCluster(ctx, cluster)
		assert.NilError(t, err)
		assert.Equal(t, len(snapshots), 0)
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
		assert.NilError(t, r.Client.Create(ctx, snapshot2))

		snapshots, err := r.getSnapshotsForCluster(ctx, cluster)
		assert.NilError(t, err)
		assert.Equal(t, len(snapshots), 1)
		assert.Equal(t, snapshots[0].Name, "another-snapshot")
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
		assert.NilError(t, r.apply(ctx, snapshot2))

		snapshots, err := r.getSnapshotsForCluster(ctx, cluster)
		assert.NilError(t, err)
		assert.Equal(t, len(snapshots), 2)
	})
}

func TestGetLatestReadySnapshot(t *testing.T) {
	t.Run("NoSnapshots", func(t *testing.T) {
		snapshots := []*volumesnapshotv1.VolumeSnapshot{}
		latestReadySnapshot := getLatestReadySnapshot(snapshots)
		assert.Assert(t, latestReadySnapshot == nil)
	})

	t.Run("NoSnapshotsWithStatus", func(t *testing.T) {
		snapshots := []*volumesnapshotv1.VolumeSnapshot{
			{},
			{},
		}
		latestReadySnapshot := getLatestReadySnapshot(snapshots)
		assert.Assert(t, latestReadySnapshot == nil)
	})

	t.Run("NoReadySnapshots", func(t *testing.T) {
		snapshots := []*volumesnapshotv1.VolumeSnapshot{
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
		}
		latestReadySnapshot := getLatestReadySnapshot(snapshots)
		assert.Assert(t, latestReadySnapshot == nil)
	})

	t.Run("OneReadySnapshot", func(t *testing.T) {
		currentTime := metav1.Now()
		earlierTime := metav1.NewTime(currentTime.AddDate(-1, 0, 0))
		snapshots := []*volumesnapshotv1.VolumeSnapshot{
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
		}
		latestReadySnapshot := getLatestReadySnapshot(snapshots)
		assert.Equal(t, latestReadySnapshot.Name, "good-snapshot")
	})

	t.Run("TwoReadySnapshots", func(t *testing.T) {
		currentTime := metav1.Now()
		earlierTime := metav1.NewTime(currentTime.AddDate(-1, 0, 0))
		snapshots := []*volumesnapshotv1.VolumeSnapshot{
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
		}
		latestReadySnapshot := getLatestReadySnapshot(snapshots)
		assert.Equal(t, latestReadySnapshot.Name, "second-good-snapshot")
	})
}

func TestDeleteSnapshots(t *testing.T) {
	ctx := context.Background()
	_, cc := setupKubernetes(t)

	r := &Reconciler{
		Client: cc,
		Owner:  client.FieldOwner(t.Name()),
	}
	ns := setupNamespace(t, cc)

	cluster := testCluster()
	cluster.Namespace = ns.Name
	cluster.UID = "the-uid-123"
	assert.NilError(t, r.Client.Create(ctx, cluster))

	rhinoCluster := testCluster()
	rhinoCluster.Name = "rhino"
	rhinoCluster.Namespace = ns.Name
	rhinoCluster.UID = "the-uid-456"
	assert.NilError(t, r.Client.Create(ctx, rhinoCluster))

	t.Cleanup(func() {
		assert.Check(t, r.Client.Delete(ctx, cluster))
		assert.Check(t, r.Client.Delete(ctx, rhinoCluster))
	})

	t.Run("NoSnapshots", func(t *testing.T) {
		snapshots := []*volumesnapshotv1.VolumeSnapshot{}
		err := r.deleteSnapshots(ctx, cluster, snapshots)
		assert.NilError(t, err)
	})

	t.Run("NoSnapshotsControlledByHippo", func(t *testing.T) {
		pvcName := initialize.String("dedicated-snapshot-volume")
		snapshot1 := &volumesnapshotv1.VolumeSnapshot{
			TypeMeta: metav1.TypeMeta{
				APIVersion: volumesnapshotv1.SchemeGroupVersion.String(),
				Kind:       "VolumeSnapshot",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "first-snapshot",
				Namespace: ns.Name,
			},
			Spec: volumesnapshotv1.VolumeSnapshotSpec{
				Source: volumesnapshotv1.VolumeSnapshotSource{
					PersistentVolumeClaimName: pvcName,
				},
			},
		}
		assert.NilError(t, r.setControllerReference(rhinoCluster, snapshot1))
		assert.NilError(t, r.Client.Create(ctx, snapshot1))

		snapshots := []*volumesnapshotv1.VolumeSnapshot{
			snapshot1,
		}
		assert.NilError(t, r.deleteSnapshots(ctx, cluster, snapshots))
		existingSnapshots := &volumesnapshotv1.VolumeSnapshotList{}
		assert.NilError(t,
			r.Client.List(ctx, existingSnapshots,
				client.InNamespace(ns.Namespace),
			))
		assert.Equal(t, len(existingSnapshots.Items), 1)
	})

	t.Run("OneSnapshotControlledByHippo", func(t *testing.T) {
		pvcName := initialize.String("dedicated-snapshot-volume")
		snapshot1 := &volumesnapshotv1.VolumeSnapshot{
			TypeMeta: metav1.TypeMeta{
				APIVersion: volumesnapshotv1.SchemeGroupVersion.String(),
				Kind:       "VolumeSnapshot",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "first-snapshot",
				Namespace: ns.Name,
			},
			Spec: volumesnapshotv1.VolumeSnapshotSpec{
				Source: volumesnapshotv1.VolumeSnapshotSource{
					PersistentVolumeClaimName: pvcName,
				},
			},
		}
		assert.NilError(t, r.setControllerReference(rhinoCluster, snapshot1))
		assert.NilError(t, r.apply(ctx, snapshot1))

		snapshot2 := &volumesnapshotv1.VolumeSnapshot{
			TypeMeta: metav1.TypeMeta{
				APIVersion: volumesnapshotv1.SchemeGroupVersion.String(),
				Kind:       "VolumeSnapshot",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "second-snapshot",
				Namespace: ns.Name,
			},
			Spec: volumesnapshotv1.VolumeSnapshotSpec{
				Source: volumesnapshotv1.VolumeSnapshotSource{
					PersistentVolumeClaimName: pvcName,
				},
			},
		}
		assert.NilError(t, r.setControllerReference(cluster, snapshot2))
		assert.NilError(t, r.Client.Create(ctx, snapshot2))

		snapshots := []*volumesnapshotv1.VolumeSnapshot{
			snapshot1, snapshot2,
		}
		assert.NilError(t, r.deleteSnapshots(ctx, cluster, snapshots))
		existingSnapshots := &volumesnapshotv1.VolumeSnapshotList{}
		assert.NilError(t,
			r.Client.List(ctx, existingSnapshots,
				client.InNamespace(ns.Namespace),
			))
		assert.Equal(t, len(existingSnapshots.Items), 1)
		assert.Equal(t, existingSnapshots.Items[0].Name, "first-snapshot")
	})
}

func TestClusterUsingTablespaces(t *testing.T) {
	ctx := context.Background()
	cluster := testCluster()

	t.Run("NoVolumesFeatureEnabled", func(t *testing.T) {
		// Enable Tablespaces feature gate
		gate := feature.NewGate()
		assert.NilError(t, gate.SetFromMap(map[string]bool{
			feature.TablespaceVolumes: true,
		}))
		ctx := feature.NewContext(ctx, gate)

		assert.Assert(t, !clusterUsingTablespaces(ctx, cluster))
	})

	t.Run("VolumesInPlaceFeatureDisabled", func(t *testing.T) {
		cluster.Spec.InstanceSets[0].TablespaceVolumes = []v1beta1.TablespaceVolume{{
			Name: "volume-1",
		}}

		assert.Assert(t, !clusterUsingTablespaces(ctx, cluster))
	})

	t.Run("VolumesInPlaceAndFeatureEnabled", func(t *testing.T) {
		// Enable Tablespaces feature gate
		gate := feature.NewGate()
		assert.NilError(t, gate.SetFromMap(map[string]bool{
			feature.TablespaceVolumes: true,
		}))
		ctx := feature.NewContext(ctx, gate)

		assert.Assert(t, clusterUsingTablespaces(ctx, cluster))
	})
}

func succeededJobStatus(startTime, completionTime metav1.Time) batchv1.JobStatus {
	return batchv1.JobStatus{
		Succeeded:      1,
		StartTime:      &startTime,
		CompletionTime: &completionTime,
		Conditions: []batchv1.JobCondition{
			{
				Type:   batchv1.JobSuccessCriteriaMet,
				Status: corev1.ConditionTrue,
			},
			{
				Type:   batchv1.JobComplete,
				Status: corev1.ConditionTrue,
			},
		},
	}
}
