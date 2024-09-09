// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"time"

	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"

	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// +kubebuilder:rbac:groups="snapshot.storage.k8s.io",resources="volumesnapshots",verbs={get,list,create,patch,delete}

// reconcileVolumeSnapshots creates and manages VolumeSnapshots if the proper VolumeSnapshot CRDs
// are installed and VolumeSnapshots are enabled for the PostgresCluster. A VolumeSnapshot of the
// primary instance's pgdata volume will be created whenever a backup is completed.
func (r *Reconciler) reconcileVolumeSnapshots(ctx context.Context,
	postgrescluster *v1beta1.PostgresCluster, instances *observedInstances,
	clusterVolumes []corev1.PersistentVolumeClaim) error {

	// Get feature gate state
	volumeSnapshotsFeatureEnabled := feature.Enabled(ctx, feature.VolumeSnapshots)

	// Check if the Kube cluster has VolumeSnapshots installed. If VolumeSnapshots
	// are not installed we need to return early. If user is attempting to use
	// VolumeSnapshots, return an error, otherwise return nil.
	volumeSnapshotsExist, err := r.GroupVersionKindExists("snapshot.storage.k8s.io/v1", "VolumeSnapshot")
	if err != nil {
		return err
	}
	if !*volumeSnapshotsExist {
		if postgrescluster.Spec.Backups.Snapshots != nil && volumeSnapshotsFeatureEnabled {
			return errors.New("VolumeSnapshots are not installed/enabled in this Kubernetes cluster; cannot create snapshot.")
		} else {
			return nil
		}
	}

	// Get all snapshots for this cluster
	snapshots, err := r.getSnapshotsForCluster(ctx, postgrescluster)
	if err != nil {
		return err
	}

	// If snapshots are disabled, delete any existing snapshots and return early.
	if postgrescluster.Spec.Backups.Snapshots == nil || !volumeSnapshotsFeatureEnabled {
		for i := range snapshots.Items {
			if err == nil {
				err = errors.WithStack(client.IgnoreNotFound(
					r.deleteControlled(ctx, postgrescluster, &snapshots.Items[i])))
			}
		}

		return err
	}

	// Check snapshots for errors; if present, create an event. If there
	// are multiple snapshots with errors, create event for the latest error.
	latestSnapshotWithError := getLatestSnapshotWithError(snapshots)
	if latestSnapshotWithError != nil {
		r.Recorder.Event(postgrescluster, corev1.EventTypeWarning, "VolumeSnapshotError",
			*latestSnapshotWithError.Status.Error.Message)
	}

	// Get all backup jobs for this cluster
	jobs := &batchv1.JobList{}
	selectJobs, err := naming.AsSelector(naming.ClusterBackupJobs(postgrescluster.Name))
	if err == nil {
		err = errors.WithStack(
			r.Client.List(ctx, jobs,
				client.InNamespace(postgrescluster.Namespace),
				client.MatchingLabelsSelector{Selector: selectJobs},
			))
	}
	if err != nil {
		return err
	}

	// Find most recently completed backup job
	backupJob := getLatestCompleteBackupJob(jobs)

	// Return early if no completed backup job found
	if backupJob == nil {
		return nil
	}

	// Find snapshot associated with latest backup
	snapshotFound := false
	snapshotIdx := 0
	for idx, snapshot := range snapshots.Items {
		if snapshot.GetAnnotations()[naming.PGBackRestBackupJobId] == string(backupJob.UID) {
			snapshotFound = true
			snapshotIdx = idx
		}
	}

	// If snapshot exists for latest backup and it is Ready, delete all other snapshots.
	// If it exists, but is not ready, do nothing. If it does not exist, create a snapshot.
	if snapshotFound {
		if *snapshots.Items[snapshotIdx].Status.ReadyToUse {
			// Snapshot found and ready. We only keep one snapshot, so delete any other snapshots.
			for idx := range snapshots.Items {
				if idx != snapshotIdx {
					err = r.deleteControlled(ctx, postgrescluster, &snapshots.Items[idx])
					if err != nil {
						return err
					}
				}
			}
		}
	} else {
		// Snapshot not found. Create snapshot.
		var snapshot *volumesnapshotv1.VolumeSnapshot
		snapshot, err = r.generateVolumeSnapshotOfPrimaryPgdata(postgrescluster,
			instances, clusterVolumes, backupJob)
		if err == nil {
			err = errors.WithStack(r.apply(ctx, snapshot))
		}
	}

	return err
}

// generateVolumeSnapshotOfPrimaryPgdata will generate a VolumeSnapshot of a
// PostgresCluster's primary instance's pgdata PersistentVolumeClaim and
// annotate it with the provided backup job's UID.
func (r *Reconciler) generateVolumeSnapshotOfPrimaryPgdata(
	postgrescluster *v1beta1.PostgresCluster, instances *observedInstances,
	clusterVolumes []corev1.PersistentVolumeClaim, backupJob *batchv1.Job,
) (*volumesnapshotv1.VolumeSnapshot, error) {

	// Find primary instance
	primaryInstance := &Instance{}
	for _, instance := range instances.forCluster {
		if isPrimary, known := instance.IsPrimary(); isPrimary && known {
			primaryInstance = instance
		}
	}
	// Return error if primary instance not found
	if primaryInstance.Name == "" {
		return nil, errors.New("Could not find primary instance. Cannot create volume snapshot.")
	}

	// Find pvc associated with primary instance
	primaryPvc := corev1.PersistentVolumeClaim{}
	for _, pvc := range clusterVolumes {
		pvcInstance := pvc.GetLabels()[naming.LabelInstance]
		pvcRole := pvc.GetLabels()[naming.LabelRole]
		if pvcRole == naming.RolePostgresData && pvcInstance == primaryInstance.Name {
			primaryPvc = pvc
		}
	}
	// Return error if primary pvc not found
	if primaryPvc.Name == "" {
		return nil, errors.New("Could not find primary's pgdata pvc. Cannot create volume snapshot.")
	}

	// generate VolumeSnapshot
	snapshot, err := r.generateVolumeSnapshot(postgrescluster, primaryPvc,
		postgrescluster.Spec.Backups.Snapshots.VolumeSnapshotClassName)
	if err == nil {
		// Add annotation for associated backup job's UID
		if snapshot.Annotations == nil {
			snapshot.Annotations = map[string]string{}
		}
		snapshot.Annotations[naming.PGBackRestBackupJobId] = string(backupJob.UID)
	}

	return snapshot, err
}

// generateVolumeSnapshot generates a VolumeSnapshot that will use the supplied
// PersistentVolumeClaim and VolumeSnapshotClassName and will set the provided
// PostgresCluster as the owner.
func (r *Reconciler) generateVolumeSnapshot(postgrescluster *v1beta1.PostgresCluster,
	pvc corev1.PersistentVolumeClaim,
	volumeSnapshotClassName string) (*volumesnapshotv1.VolumeSnapshot, error) {

	snapshot := &volumesnapshotv1.VolumeSnapshot{
		TypeMeta: metav1.TypeMeta{
			APIVersion: volumesnapshotv1.SchemeGroupVersion.String(),
			Kind:       "VolumeSnapshot",
		},
		ObjectMeta: naming.ClusterVolumeSnapshot(postgrescluster),
	}
	snapshot.Spec.Source.PersistentVolumeClaimName = &pvc.Name
	snapshot.Spec.VolumeSnapshotClassName = &volumeSnapshotClassName

	snapshot.Annotations = postgrescluster.Spec.Metadata.GetAnnotationsOrNil()
	snapshot.Labels = naming.Merge(postgrescluster.Spec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster: postgrescluster.Name,
		})

	err := errors.WithStack(r.setControllerReference(postgrescluster, snapshot))

	return snapshot, err
}

// getLatestCompleteBackupJob takes a JobList and returns a pointer to the
// most recently completed backup job. If no completed backup job exists
// then it returns nil.
func getLatestCompleteBackupJob(jobs *batchv1.JobList) *batchv1.Job {
	zeroTime := metav1.NewTime(time.Time{})
	latestCompleteBackupJob := batchv1.Job{
		Status: batchv1.JobStatus{
			Succeeded:      1,
			CompletionTime: &zeroTime,
		},
	}
	for _, job := range jobs.Items {
		if job.Status.Succeeded > 0 &&
			latestCompleteBackupJob.Status.CompletionTime.Before(job.Status.CompletionTime) {
			latestCompleteBackupJob = job
		}
	}

	if latestCompleteBackupJob.Status.CompletionTime.Equal(&zeroTime) {
		return nil
	}

	return &latestCompleteBackupJob
}

// getLatestSnapshotWithError takes a VolumeSnapshotList and returns a pointer to the
// most recently created snapshot that has an error. If no snapshot errors exist
// then it returns nil.
func getLatestSnapshotWithError(snapshots *volumesnapshotv1.VolumeSnapshotList) *volumesnapshotv1.VolumeSnapshot {
	zeroTime := metav1.NewTime(time.Time{})
	latestSnapshotWithError := volumesnapshotv1.VolumeSnapshot{
		Status: &volumesnapshotv1.VolumeSnapshotStatus{
			CreationTime: &zeroTime,
		},
	}
	for _, snapshot := range snapshots.Items {
		if snapshot.Status.Error != nil &&
			latestSnapshotWithError.Status.CreationTime.Before(snapshot.Status.CreationTime) {
			latestSnapshotWithError = snapshot
		}
	}

	if latestSnapshotWithError.Status.CreationTime.Equal(&zeroTime) {
		return nil
	}

	return &latestSnapshotWithError
}

// getSnapshotsForCluster gets all the VolumeSnapshots for a given postgrescluster
func (r *Reconciler) getSnapshotsForCluster(ctx context.Context, cluster *v1beta1.PostgresCluster) (
	*volumesnapshotv1.VolumeSnapshotList, error) {

	selectSnapshots, err := naming.AsSelector(naming.Cluster(cluster.Name))
	if err != nil {
		return nil, err
	}
	snapshots := &volumesnapshotv1.VolumeSnapshotList{}
	err = errors.WithStack(
		r.Client.List(ctx, snapshots,
			client.InNamespace(cluster.Namespace),
			client.MatchingLabelsSelector{Selector: selectSnapshots},
		))

	return snapshots, err
}

// getLatestReadySnapshot takes a VolumeSnapshotList and returns the latest ready VolumeSnapshot
func getLatestReadySnapshot(snapshots *volumesnapshotv1.VolumeSnapshotList) *volumesnapshotv1.VolumeSnapshot {
	zeroTime := metav1.NewTime(time.Time{})
	latestReadySnapshot := volumesnapshotv1.VolumeSnapshot{
		Status: &volumesnapshotv1.VolumeSnapshotStatus{
			CreationTime: &zeroTime,
		},
	}
	for _, snapshot := range snapshots.Items {
		if *snapshot.Status.ReadyToUse &&
			latestReadySnapshot.Status.CreationTime.Before(snapshot.Status.CreationTime) {
			latestReadySnapshot = snapshot
		}
	}

	if latestReadySnapshot.Status.CreationTime.Equal(&zeroTime) {
		return nil
	}

	return &latestReadySnapshot
}
