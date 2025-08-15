// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"fmt"
	"strings"
	"time"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/kubernetes"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/pgbackrest"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

//+kubebuilder:rbac:groups="snapshot.storage.k8s.io",resources="volumesnapshots",verbs={get,list,create,patch,delete}

// The controller-runtime client sets up a cache that watches anything we "get" or "list".
//+kubebuilder:rbac:groups="snapshot.storage.k8s.io",resources="volumesnapshots",verbs={watch}

// reconcileVolumeSnapshots creates and manages VolumeSnapshots if the proper VolumeSnapshot CRDs
// are installed and VolumeSnapshots are enabled for the PostgresCluster. A VolumeSnapshot of the
// primary instance's pgdata volume will be created whenever a backup is completed. The steps to
// create snapshots include the following sequence:
//  1. We find the latest completed backup job and check the timestamp.
//  2. If the timestamp is later than what's on the dedicated snapshot PVC, a restore job runs in
//     the dedicated snapshot volume.
//  3. When the restore job completes, an annotation is updated on the PVC. If the restore job
//     fails, we don't run it again.
//  4. When the PVC annotation is updated, we see if there's a volume snapshot with an earlier
//     timestamp.
//  5. If there are no snapshots at all, we take a snapshot and put the backup job's completion
//     timestamp on the snapshot annotation.
//  6. If an earlier snapshot is found, we take a new snapshot, annotate it and delete the old
//     snapshot.
//  7. When the snapshot job completes, we delete the restore job.
func (r *Reconciler) reconcileVolumeSnapshots(ctx context.Context,
	postgrescluster *v1beta1.PostgresCluster, pvc *corev1.PersistentVolumeClaim) error {

	// If VolumeSnapshots feature gate is disabled. Do nothing and return early.
	if !feature.Enabled(ctx, feature.VolumeSnapshots) {
		return nil
	}

	// Return early when VolumeSnapshots are not installed in Kubernetes.
	// If user is attempting to use VolumeSnapshots, return an error.
	if !kubernetes.Has(
		ctx, volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshot"),
	) {
		if postgrescluster.Spec.Backups.Snapshots != nil {
			return errors.New("VolumeSnapshots are not installed/enabled in this Kubernetes cluster; cannot create snapshot.")
		} else {
			return nil
		}
	}

	// If user is attempting to use snapshots and has tablespaces enabled, we
	// need to create a warning event indicating that the two features are not
	// currently compatible and return early.
	if postgrescluster.Spec.Backups.Snapshots != nil &&
		clusterUsingTablespaces(ctx, postgrescluster) {
		r.Recorder.Event(postgrescluster, corev1.EventTypeWarning, "IncompatibleFeatures",
			"VolumeSnapshots not currently compatible with TablespaceVolumes; cannot create snapshot.")
		return nil
	}

	// Get all snapshots for the cluster.
	snapshots, err := r.getSnapshotsForCluster(ctx, postgrescluster)
	if err != nil {
		return err
	}

	// If snapshots are disabled, delete any existing snapshots and return early.
	if postgrescluster.Spec.Backups.Snapshots == nil {
		return r.deleteSnapshots(ctx, postgrescluster, snapshots)
	}

	// If we got here, then the snapshots are enabled (feature gate is enabled and the
	// cluster has a Spec.Backups.Snapshots section defined).

	// Check snapshots for errors; if present, create an event. If there are
	// multiple snapshots with errors, create event for the latest error and
	// delete any older snapshots with error.
	snapshotWithLatestError := getSnapshotWithLatestError(snapshots)
	if snapshotWithLatestError != nil {
		r.Recorder.Event(postgrescluster, corev1.EventTypeWarning, "VolumeSnapshotError",
			*snapshotWithLatestError.Status.Error.Message)
		for _, snapshot := range snapshots {
			if snapshot.Status != nil && snapshot.Status.Error != nil &&
				snapshot.Status.Error.Time.Before(snapshotWithLatestError.Status.Error.Time) {
				err = r.deleteControlled(ctx, postgrescluster, snapshot)
				if err != nil {
					return err
				}
			}
		}
	}

	// Get pvc backup job completion annotation. If it does not exist, there has not been
	// a successful restore yet, so return early.
	pvcUpdateTimeStamp, pvcAnnotationExists := pvc.GetAnnotations()[naming.PGBackRestBackupJobCompletion]
	if !pvcAnnotationExists {
		return err
	}

	// Check to see if snapshot exists for the latest backup that has been restored into
	// the dedicated pvc.
	var snapshotForPvcUpdateIdx int
	snapshotFoundForPvcUpdate := false
	for idx, snapshot := range snapshots {
		if snapshot.GetAnnotations()[naming.PGBackRestBackupJobCompletion] == pvcUpdateTimeStamp {
			snapshotForPvcUpdateIdx = idx
			snapshotFoundForPvcUpdate = true
		}
	}

	// If a snapshot exists for the latest backup that has been restored into the dedicated pvc
	// and the snapshot is Ready, delete all other snapshots.
	if snapshotFoundForPvcUpdate && snapshots[snapshotForPvcUpdateIdx].Status.ReadyToUse != nil &&
		*snapshots[snapshotForPvcUpdateIdx].Status.ReadyToUse {
		for idx, snapshot := range snapshots {
			if idx != snapshotForPvcUpdateIdx {
				err = r.deleteControlled(ctx, postgrescluster, snapshot)
				if err != nil {
					return err
				}
			}
		}
	}

	// If a snapshot for the latest backup/restore does not exist, create a snapshot.
	if !snapshotFoundForPvcUpdate {
		var snapshot *volumesnapshotv1.VolumeSnapshot
		snapshot, err = r.generateSnapshotOfDedicatedSnapshotVolume(postgrescluster, pvc)
		if err == nil {
			err = errors.WithStack(r.apply(ctx, snapshot))
		}
	}

	return err
}

// +kubebuilder:rbac:groups="",resources="persistentvolumeclaims",verbs={get}
// +kubebuilder:rbac:groups="",resources="persistentvolumeclaims",verbs={create,delete,patch}

// reconcileDedicatedSnapshotVolume reconciles the PersistentVolumeClaim that holds a
// copy of the pgdata and is dedicated for clean snapshots of the database. It creates
// and manages the volume as well as the restore jobs that bring the volume data forward
// after a successful backup.
func (r *Reconciler) reconcileDedicatedSnapshotVolume(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
	clusterVolumes []*corev1.PersistentVolumeClaim,
) (*corev1.PersistentVolumeClaim, error) {

	// If VolumeSnapshots feature gate is disabled, do nothing and return early.
	if !feature.Enabled(ctx, feature.VolumeSnapshots) {
		return nil, nil
	}

	// Set appropriate labels for dedicated snapshot volume
	labelMap := map[string]string{
		naming.LabelCluster: cluster.Name,
		naming.LabelRole:    naming.RoleSnapshot,
		naming.LabelData:    naming.DataPostgres,
	}

	// If volume already exists, use existing name. Otherwise, generate a name.
	var pvc *corev1.PersistentVolumeClaim
	existingPVCName := getPVCName(clusterVolumes, labels.SelectorFromSet(labelMap))
	if existingPVCName != "" {
		pvc = &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{
			Namespace: cluster.GetNamespace(),
			Name:      existingPVCName,
		}}
	} else {
		pvc = &corev1.PersistentVolumeClaim{ObjectMeta: naming.ClusterDedicatedSnapshotVolume(cluster)}
	}
	pvc.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"))

	// If snapshots are disabled, delete the PVC if it exists and return early.
	// Check the client cache first using Get.
	if cluster.Spec.Backups.Snapshots == nil {
		key := client.ObjectKeyFromObject(pvc)
		err := errors.WithStack(r.Client.Get(ctx, key, pvc))
		if err == nil {
			err = errors.WithStack(r.deleteControlled(ctx, cluster, pvc))
		}
		return nil, client.IgnoreNotFound(err)
	}

	// If we've got this far, snapshots are enabled so we should create/update/get
	// the dedicated snapshot volume
	pvc, err := r.createDedicatedSnapshotVolume(ctx, cluster, labelMap, pvc)
	if err != nil {
		return pvc, err
	}

	// Determine if we need to run a restore job, based on the most recent backup
	// and an annotation on the PVC.

	// Find the most recently completed backup job.
	backupJob, err := r.getLatestCompleteBackupJob(ctx, cluster)
	if err != nil {
		return pvc, err
	}

	// Return early if no complete backup job is found.
	if backupJob == nil {
		return pvc, nil
	}

	// Return early if the pvc is annotated with a timestamp newer or equal to the latest backup job.
	// If the annotation value cannot be parsed, we want to proceed with a restore.
	pvcAnnotationTimestampString := pvc.GetAnnotations()[naming.PGBackRestBackupJobCompletion]
	if pvcAnnotationTime, err := time.Parse(time.RFC3339, pvcAnnotationTimestampString); err == nil {
		if backupJob.Status.CompletionTime.Compare(pvcAnnotationTime) <= 0 {
			return pvc, nil
		}
	}

	// If we've made it here, the pvc has not been restored with latest backup.
	// Find the dedicated snapshot volume restore job if it exists. Since we delete
	// successful restores after we annotate the PVC and stop making restore jobs
	// if a failed DSV restore job exists, there should only ever be one DSV restore
	// job in existence at a time.
	// TODO(snapshots): Should this function throw an error or something if multiple
	// DSV restores somehow exist?
	restoreJob, err := r.getDedicatedSnapshotVolumeRestoreJob(ctx, cluster)
	if err != nil {
		return pvc, err
	}

	// If we don't find a restore job, we run one.
	if restoreJob == nil {
		err = r.dedicatedSnapshotVolumeRestore(ctx, cluster, pvc, backupJob)
		return pvc, err
	}

	// If we've made it here, we have found a restore job. If the restore job was
	// successful, set/update the annotation on the PVC and delete the restore job.
	if restoreJob.Status.Succeeded == 1 {
		if pvc.GetAnnotations() == nil {
			pvc.Annotations = map[string]string{}
		}
		pvc.Annotations[naming.PGBackRestBackupJobCompletion] = restoreJob.GetAnnotations()[naming.PGBackRestBackupJobCompletion]
		annotations := fmt.Sprintf(`{"metadata":{"annotations":{"%s": "%s"}}}`,
			naming.PGBackRestBackupJobCompletion, pvc.Annotations[naming.PGBackRestBackupJobCompletion])

		patch := client.RawPatch(client.Merge.Type(), []byte(annotations))
		err = r.handlePersistentVolumeClaimError(cluster,
			errors.WithStack(r.patch(ctx, pvc, patch)))

		if err != nil {
			return pvc, err
		}

		err = r.Client.Delete(ctx, restoreJob, client.PropagationPolicy(metav1.DeletePropagationBackground))
		return pvc, errors.WithStack(err)
	}

	// If the restore job failed, create a warning event.
	if restoreJob.Status.Failed == 1 {
		r.Recorder.Event(cluster, corev1.EventTypeWarning,
			"DedicatedSnapshotVolumeRestoreJobError", "restore job failed, check the logs")
		return pvc, nil
	}

	// If we made it here, the restore job is still running and we should do nothing.
	return pvc, err
}

// createDedicatedSnapshotVolume creates/updates/gets the dedicated snapshot volume.
// It expects that the volume name and GVK has already been set on the pvc that is passed in.
func (r *Reconciler) createDedicatedSnapshotVolume(ctx context.Context,
	cluster *v1beta1.PostgresCluster, labelMap map[string]string,
	pvc *corev1.PersistentVolumeClaim,
) (*corev1.PersistentVolumeClaim, error) {
	var err error

	// An InstanceSet must be chosen to scale resources for the dedicated snapshot volume.
	// TODO: We've chosen the first InstanceSet for the time being, but might want to consider
	// making the choice configurable.
	instanceSpec := cluster.Spec.InstanceSets[0]

	pvc.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil(),
		instanceSpec.Metadata.GetAnnotationsOrNil())

	pvc.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		instanceSpec.Metadata.GetLabelsOrNil(),
		labelMap,
	)

	err = errors.WithStack(r.setControllerReference(cluster, pvc))
	if err != nil {
		return pvc, err
	}

	pvc.Spec = instanceSpec.DataVolumeClaimSpec.AsPersistentVolumeClaimSpec()

	// Set the snapshot volume to the same size as the pgdata volume. The size should scale with auto-grow.
	r.setVolumeSize(ctx, cluster, &pvc.Spec, "pgData", instanceSpec.Name)

	// Clear any set limit before applying PVC. This is needed to allow the limit
	// value to change later.
	pvc.Spec.Resources.Limits = nil

	err = r.handlePersistentVolumeClaimError(cluster,
		errors.WithStack(r.apply(ctx, pvc)))
	if err != nil {
		return pvc, err
	}

	return pvc, err
}

// dedicatedSnapshotVolumeRestore creates a Job that performs a restore into the dedicated
// snapshot volume.
// This function is very similar to reconcileRestoreJob, but specifically tailored to the
// dedicated snapshot volume.
func (r *Reconciler) dedicatedSnapshotVolumeRestore(ctx context.Context,
	cluster *v1beta1.PostgresCluster, dedicatedSnapshotVolume *corev1.PersistentVolumeClaim,
	backupJob *batchv1.Job,
) error {

	pgdata := postgres.DataDirectory(cluster)
	repoName := backupJob.GetLabels()[naming.LabelPGBackRestRepo]

	opts := []string{
		"--stanza=" + pgbackrest.DefaultStanzaName,
		"--pg1-path=" + pgdata,
		"--repo=" + regexRepoIndex.FindString(repoName),
		"--delta",
	}

	cmd := pgbackrest.DedicatedSnapshotVolumeRestoreCommand(pgdata, strings.Join(opts, " "))

	// Create the volume resources required for the Postgres data directory.
	dataVolumeMount := postgres.DataVolumeMount()
	dataVolume := corev1.Volume{
		Name: dataVolumeMount.Name,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: dedicatedSnapshotVolume.GetName(),
			},
		},
	}
	volumes := []corev1.Volume{dataVolume}
	volumeMounts := []corev1.VolumeMount{dataVolumeMount}

	_, configHash, err := pgbackrest.CalculateConfigHashes(cluster)
	if err != nil {
		return err
	}

	// A DataSource is required to avoid a nil pointer exception.
	fakeDataSource := &v1beta1.PostgresClusterDataSource{RepoName: ""}

	restoreJob := &batchv1.Job{}
	instanceName := cluster.Status.StartupInstance

	if err := r.generateRestoreJobIntent(cluster, configHash, instanceName, cmd,
		volumeMounts, volumes, fakeDataSource, restoreJob); err != nil {
		return errors.WithStack(err)
	}

	// Attempt the restore exactly once. If the restore job fails, we prompt the user to investigate.
	restoreJob.Spec.BackoffLimit = initialize.Int32(0)
	restoreJob.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyNever

	// Add pgBackRest configs to template.
	pgbackrest.AddConfigToRestorePod(cluster, cluster, &restoreJob.Spec.Template.Spec)

	// Add nss_wrapper init container and add nss_wrapper env vars to the pgbackrest restore container.
	addNSSWrapper(
		config.PGBackRestContainerImage(cluster),
		cluster.Spec.ImagePullPolicy,
		&restoreJob.Spec.Template)

	AddTMPEmptyDir(&restoreJob.Spec.Template)

	restoreJob.Annotations[naming.PGBackRestBackupJobCompletion] = backupJob.Status.CompletionTime.Format(time.RFC3339)
	return errors.WithStack(r.apply(ctx, restoreJob))
}

// generateSnapshotOfDedicatedSnapshotVolume will generate a VolumeSnapshot of
// the dedicated snapshot PersistentVolumeClaim and annotate it with the
// provided backup job's UID.
func (r *Reconciler) generateSnapshotOfDedicatedSnapshotVolume(
	postgrescluster *v1beta1.PostgresCluster,
	dedicatedSnapshotVolume *corev1.PersistentVolumeClaim,
) (*volumesnapshotv1.VolumeSnapshot, error) {

	snapshot, err := r.generateVolumeSnapshot(postgrescluster, *dedicatedSnapshotVolume,
		postgrescluster.Spec.Backups.Snapshots.VolumeSnapshotClassName)
	if err == nil {
		if snapshot.Annotations == nil {
			snapshot.Annotations = map[string]string{}
		}
		snapshot.Annotations[naming.PGBackRestBackupJobCompletion] = dedicatedSnapshotVolume.GetAnnotations()[naming.PGBackRestBackupJobCompletion]
	}

	return snapshot, err
}

// generateVolumeSnapshot generates a VolumeSnapshot that will use the supplied
// PersistentVolumeClaim and VolumeSnapshotClassName and will set the provided
// PostgresCluster as the owner.
func (r *Reconciler) generateVolumeSnapshot(postgrescluster *v1beta1.PostgresCluster,
	pvc corev1.PersistentVolumeClaim, volumeSnapshotClassName string,
) (*volumesnapshotv1.VolumeSnapshot, error) {

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

// getDedicatedSnapshotVolumeRestoreJob finds a dedicated snapshot volume (DSV)
// restore job if one exists. Since we delete successful restore jobs and stop
// creating new restore jobs when one fails, there should only ever be one DSV
// restore job present at a time. If a DSV restore cannot be found, we return nil.
func (r *Reconciler) getDedicatedSnapshotVolumeRestoreJob(ctx context.Context,
	postgrescluster *v1beta1.PostgresCluster) (*batchv1.Job, error) {

	// Get all restore jobs for this cluster
	jobs := &batchv1.JobList{}
	selectJobs, err := naming.AsSelector(naming.ClusterRestoreJobs(postgrescluster.Name))
	if err == nil {
		err = errors.WithStack(
			r.Client.List(ctx, jobs,
				client.InNamespace(postgrescluster.Namespace),
				client.MatchingLabelsSelector{Selector: selectJobs},
			))
	}
	if err != nil {
		return nil, err
	}

	// Get restore job that has PGBackRestBackupJobCompletion annotation
	for _, job := range jobs.Items {
		_, annotationExists := job.GetAnnotations()[naming.PGBackRestBackupJobCompletion]
		if annotationExists {
			return &job, nil
		}
	}

	return nil, nil
}

// getLatestCompleteBackupJob finds the most recently completed
// backup job for a cluster
func (r *Reconciler) getLatestCompleteBackupJob(ctx context.Context,
	postgrescluster *v1beta1.PostgresCluster) (*batchv1.Job, error) {

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
		return nil, err
	}

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
		return nil, nil
	}

	return &latestCompleteBackupJob, nil
}

// getSnapshotWithLatestError takes a VolumeSnapshotList and returns a pointer to the
// snapshot that has most recently had an error. If no snapshot errors exist
// then it returns nil.
func getSnapshotWithLatestError(snapshots []*volumesnapshotv1.VolumeSnapshot) *volumesnapshotv1.VolumeSnapshot {
	zeroTime := metav1.NewTime(time.Time{})
	snapshotWithLatestError := &volumesnapshotv1.VolumeSnapshot{
		Status: &volumesnapshotv1.VolumeSnapshotStatus{
			Error: &volumesnapshotv1.VolumeSnapshotError{
				Time: &zeroTime,
			},
		},
	}
	for _, snapshot := range snapshots {
		if snapshot.Status != nil && snapshot.Status.Error != nil &&
			snapshotWithLatestError.Status.Error.Time.Before(snapshot.Status.Error.Time) {
			snapshotWithLatestError = snapshot
		}
	}

	if snapshotWithLatestError.Status.Error.Time.Equal(&zeroTime) {
		return nil
	}

	return snapshotWithLatestError
}

// getSnapshotsForCluster gets all the VolumeSnapshots for a given postgrescluster.
func (r *Reconciler) getSnapshotsForCluster(ctx context.Context, cluster *v1beta1.PostgresCluster) (
	[]*volumesnapshotv1.VolumeSnapshot, error) {

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

	return initialize.Pointers(snapshots.Items...), err
}

// getLatestReadySnapshot takes a VolumeSnapshotList and returns the latest ready VolumeSnapshot.
func getLatestReadySnapshot(snapshots []*volumesnapshotv1.VolumeSnapshot) *volumesnapshotv1.VolumeSnapshot {
	zeroTime := metav1.NewTime(time.Time{})
	latestReadySnapshot := &volumesnapshotv1.VolumeSnapshot{
		Status: &volumesnapshotv1.VolumeSnapshotStatus{
			CreationTime: &zeroTime,
		},
	}
	for _, snapshot := range snapshots {
		if snapshot.Status != nil && snapshot.Status.ReadyToUse != nil && *snapshot.Status.ReadyToUse &&
			latestReadySnapshot.Status.CreationTime.Before(snapshot.Status.CreationTime) {
			latestReadySnapshot = snapshot
		}
	}

	if latestReadySnapshot.Status.CreationTime.Equal(&zeroTime) {
		return nil
	}

	return latestReadySnapshot
}

// deleteSnapshots takes a postgrescluster and a snapshot list and deletes all snapshots
// in the list that are controlled by the provided postgrescluster.
func (r *Reconciler) deleteSnapshots(ctx context.Context,
	postgrescluster *v1beta1.PostgresCluster, snapshots []*volumesnapshotv1.VolumeSnapshot) error {

	for i := range snapshots {
		err := errors.WithStack(client.IgnoreNotFound(
			r.deleteControlled(ctx, postgrescluster, snapshots[i])))
		if err != nil {
			return err
		}
	}
	return nil
}

// tablespaceVolumesInUse determines if the TablespaceVolumes feature is enabled and the given
// cluster has tablespace volumes in place.
func clusterUsingTablespaces(ctx context.Context, postgrescluster *v1beta1.PostgresCluster) bool {
	for _, instanceSet := range postgrescluster.Spec.InstanceSets {
		if len(instanceSet.TablespaceVolumes) > 0 {
			return feature.Enabled(ctx, feature.TablespaceVolumes)
		}
	}
	return false
}
