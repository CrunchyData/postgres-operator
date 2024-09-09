// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/pgbackrest"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// +kubebuilder:rbac:groups="",resources="persistentvolumeclaims",verbs={list}

// observePersistentVolumeClaims reads all PVCs for cluster from the Kubernetes
// API and sets the PersistentVolumeResizing condition as appropriate.
func (r *Reconciler) observePersistentVolumeClaims(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
) ([]corev1.PersistentVolumeClaim, error) {
	volumes := &corev1.PersistentVolumeClaimList{}

	selector, err := naming.AsSelector(naming.Cluster(cluster.Name))
	if err == nil {
		err = errors.WithStack(
			r.Client.List(ctx, volumes,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selector},
			))
	}

	resizing := metav1.Condition{
		Type:    v1beta1.PersistentVolumeResizing,
		Message: "One or more volumes are changing size",

		ObservedGeneration: cluster.Generation,
	}

	minNotZero := func(a, b metav1.Time) metav1.Time {
		if b.IsZero() || (a.Before(&b) && !a.IsZero()) {
			return a
		}
		return b
	}

	for _, pvc := range volumes.Items {
		for _, condition := range pvc.Status.Conditions {
			switch condition.Type {
			case
				// When the resize controller sees `spec.resources != status.capacity`,
				// it sets a "Resizing" condition and invokes the storage provider.
				// NOTE: The oldest KEP talks about "ResizeStarted", but that
				// changed to "Resizing" during the merge to Kubernetes v1.8.
				// - https://git.k8s.io/enhancements/keps/sig-storage/284-enable-volume-expansion
				// - https://pr.k8s.io/49727#discussion_r136678508
				corev1.PersistentVolumeClaimResizing,

				// Kubernetes v1.10 added the "FileSystemResizePending" condition
				// to indicate when the storage provider has finished its work.
				// When a CSI implementation indicates that it performed the
				// *entire* resize, this condition does not appear.
				// - https://git.k8s.io/enhancements/keps/sig-storage/556-csi-volume-resizing
				// - https://pr.k8s.io/58415
				//
				// Kubernetes v1.15 ("ExpandInUsePersistentVolumes" feature gate)
				// finishes the resize of mounted and writable PVCs that have
				// the "FileSystemResizePending" condition. When the work is done,
				// the condition is removed and `spec.resources == status.capacity`.
				// - https://git.k8s.io/enhancements/keps/sig-storage/531-online-pv-resizing
				corev1.PersistentVolumeClaimFileSystemResizePending:

				// Initialize from the first condition.
				if resizing.Status == "" {
					resizing.Status = metav1.ConditionStatus(condition.Status)
					resizing.Reason = condition.Reason
					resizing.LastTransitionTime = condition.LastTransitionTime

					// corev1.PersistentVolumeClaimCondition.Reason is optional
					// while metav1.Condition.Reason is required.
					if resizing.Reason == "" {
						resizing.Reason = string(condition.Type)
					}
				}

				// Use most things from an adverse condition.
				if condition.Status != corev1.ConditionTrue {
					resizing.Status = metav1.ConditionStatus(condition.Status)
					resizing.Reason = condition.Reason
					resizing.Message = condition.Message
					resizing.LastTransitionTime = condition.LastTransitionTime

					// corev1.PersistentVolumeClaimCondition.Reason is optional
					// while metav1.Condition.Reason is required.
					if resizing.Reason == "" {
						resizing.Reason = string(condition.Type)
					}
				}

				// Use the oldest transition time of healthy conditions.
				if resizing.Status == metav1.ConditionTrue &&
					condition.Status == corev1.ConditionTrue {
					resizing.LastTransitionTime = minNotZero(
						resizing.LastTransitionTime, condition.LastTransitionTime)
				}

			case
				// The "ModifyingVolume" and "ModifyVolumeError" conditions occur
				// when the attribute class of a PVC is changing. These attributes
				// do not affect the size of a volume, so there's nothing to do.
				// See the "VolumeAttributesClass" feature gate.
				// - https://git.k8s.io/enhancements/keps/sig-storage/3751-volume-attributes-class
				corev1.PersistentVolumeClaimVolumeModifyingVolume,
				corev1.PersistentVolumeClaimVolumeModifyVolumeError:
			}
		}
	}

	if resizing.Status != "" {
		meta.SetStatusCondition(&cluster.Status.Conditions, resizing)
	} else {
		// NOTE(cbandy): This clears the condition, but it may immediately
		// return with a new LastTransitionTime when a PVC spec is invalid.
		meta.RemoveStatusCondition(&cluster.Status.Conditions, resizing.Type)
	}

	return volumes.Items, err
}

// configureExistingPVCs configures the defined pgData, pg_wal and pgBackRest
// repo volumes to be used by the PostgresCluster. In the case of existing
// pgData volumes, an appropriate instance set name is defined that will be
// used for the PostgresCluster. Existing pg_wal volumes MUST be defined along
// with existing pgData volumes to ensure consistent naming and proper
// bootstrapping.
func (r *Reconciler) configureExistingPVCs(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
	volumes []corev1.PersistentVolumeClaim,
) ([]corev1.PersistentVolumeClaim, error) {

	var err error

	if cluster.Spec.DataSource != nil &&
		cluster.Spec.DataSource.Volumes != nil &&
		cluster.Spec.DataSource.Volumes.PGDataVolume != nil {
		// If the startup instance name isn't set, use the instance set defined at position zero.
		if cluster.Status.StartupInstance == "" {
			set := &cluster.Spec.InstanceSets[0]
			cluster.Status.StartupInstanceSet = set.Name
			cluster.Status.StartupInstance = naming.GenerateStartupInstance(cluster, set).Name
		}
		volumes, err = r.configureExistingPGVolumes(ctx, cluster, volumes,
			cluster.Status.StartupInstance)

		// existing WAL volume must be paired with an existing pgData volume
		if cluster.Spec.DataSource != nil &&
			cluster.Spec.DataSource.Volumes != nil &&
			cluster.Spec.DataSource.Volumes.PGWALVolume != nil &&
			err == nil {
			volumes, err = r.configureExistingPGWALVolume(ctx, cluster, volumes,
				cluster.Status.StartupInstance)
		}
	}

	if cluster.Spec.DataSource != nil &&
		cluster.Spec.DataSource.Volumes != nil &&
		cluster.Spec.DataSource.Volumes.PGBackRestVolume != nil &&
		err == nil {

		volumes, err = r.configureExistingRepoVolumes(ctx, cluster, volumes)
	}
	return volumes, err
}

// +kubebuilder:rbac:groups="",resources="persistentvolumeclaims",verbs={create,patch}

// configureExistingPGVolumes first searches the observed volumes list to see
// if the existing pgData volume defined in the spec is already updated. If not,
// this sets the appropriate labels and ownership for the volume to be used in
// the PostgresCluster.
func (r *Reconciler) configureExistingPGVolumes(
	ctx context.Context,
	cluster *v1beta1.PostgresCluster,
	volumes []corev1.PersistentVolumeClaim,
	instanceName string,
) ([]corev1.PersistentVolumeClaim, error) {

	// if the volume is already in the list, move on
	for i := range volumes {
		if cluster.Spec.DataSource.Volumes.PGDataVolume.
			PVCName == volumes[i].Name {
			return volumes, nil
		}
	}

	if len(cluster.Spec.InstanceSets) > 0 {
		if volName := cluster.Spec.DataSource.Volumes.
			PGDataVolume.PVCName; volName != "" {
			volume := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      volName,
					Namespace: cluster.Namespace,
				},
				Spec: cluster.Spec.InstanceSets[0].DataVolumeClaimSpec,
			}

			volume.ObjectMeta.Labels = map[string]string{
				naming.LabelCluster:     cluster.Name,
				naming.LabelInstanceSet: cluster.Spec.InstanceSets[0].Name,
				naming.LabelInstance:    instanceName,
				naming.LabelRole:        naming.RolePostgresData,
				naming.LabelData:        naming.DataPostgres,
			}
			volume.SetGroupVersionKind(corev1.SchemeGroupVersion.
				WithKind("PersistentVolumeClaim"))
			if err := r.setControllerReference(cluster, volume); err != nil {
				return volumes, err
			}
			if err := errors.WithStack(r.apply(ctx, volume)); err != nil {
				return volumes, err
			}
			volumes = append(volumes, *volume)
		}
	}
	return volumes, nil
}

// +kubebuilder:rbac:groups="",resources="persistentvolumeclaims",verbs={create,patch}

// configureExistingPGWALVolume first searches the observed volumes list to see
// if the existing pg_wal volume defined in the spec is already updated. If not,
// this sets the appropriate labels and ownership for the volume to be used in
// the PostgresCluster.
func (r *Reconciler) configureExistingPGWALVolume(
	ctx context.Context,
	cluster *v1beta1.PostgresCluster,
	volumes []corev1.PersistentVolumeClaim,
	instanceName string,
) ([]corev1.PersistentVolumeClaim, error) {

	// if the volume is already in the list, move on
	for i := range volumes {
		if cluster.Spec.DataSource.Volumes.PGWALVolume.
			PVCName == volumes[i].Name {
			return volumes, nil
		}
	}

	if volName := cluster.Spec.DataSource.Volumes.PGWALVolume.
		PVCName; volName != "" {

		volume := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      volName,
				Namespace: cluster.Namespace,
			},
			Spec: cluster.Spec.InstanceSets[0].DataVolumeClaimSpec,
		}

		volume.ObjectMeta.Labels = map[string]string{
			naming.LabelCluster:     cluster.Name,
			naming.LabelInstanceSet: cluster.Spec.InstanceSets[0].Name,
			naming.LabelInstance:    instanceName,
			naming.LabelRole:        naming.RolePostgresWAL,
			naming.LabelData:        naming.DataPostgres,
		}
		volume.SetGroupVersionKind(corev1.SchemeGroupVersion.
			WithKind("PersistentVolumeClaim"))
		if err := r.setControllerReference(cluster, volume); err != nil {
			return volumes, err
		}
		if err := errors.WithStack(r.apply(ctx, volume)); err != nil {
			return volumes, err
		}
		volumes = append(volumes, *volume)
	}
	return volumes, nil
}

// +kubebuilder:rbac:groups="",resources="persistentvolumeclaims",verbs={create,patch}

// configureExistingRepoVolumes first searches the observed volumes list to see
// if the existing pgBackRest repo volume defined in the spec is already updated.
// If not, this sets the appropriate labels and ownership for the volume to be
// used in the PostgresCluster.
func (r *Reconciler) configureExistingRepoVolumes(
	ctx context.Context,
	cluster *v1beta1.PostgresCluster,
	volumes []corev1.PersistentVolumeClaim,
) ([]corev1.PersistentVolumeClaim, error) {

	// if the volume is already in the list, move on
	for i := range volumes {
		if cluster.Spec.DataSource.Volumes.PGBackRestVolume.
			PVCName == volumes[i].Name {
			return volumes, nil
		}
	}

	if len(cluster.Spec.Backups.PGBackRest.Repos) > 0 {
		// there must be at least on pgBackrest repo defined
		if volName := cluster.Spec.DataSource.Volumes.
			PGBackRestVolume.PVCName; volName != "" {
			volume := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      volName,
					Namespace: cluster.Namespace,
					Labels: naming.PGBackRestRepoVolumeLabels(cluster.Name,
						cluster.Spec.Backups.PGBackRest.Repos[0].Name),
				},
				Spec: cluster.Spec.Backups.PGBackRest.Repos[0].Volume.
					VolumeClaimSpec,
			}

			//volume.ObjectMeta = naming.PGBackRestRepoVolume(cluster, cluster.Spec.Backups.PGBackRest.Repos[0].Name)
			volume.SetGroupVersionKind(corev1.SchemeGroupVersion.
				WithKind("PersistentVolumeClaim"))
			if err := r.setControllerReference(cluster, volume); err != nil {
				return volumes, err
			}
			if err := errors.WithStack(r.apply(ctx, volume)); err != nil {
				return volumes, err
			}
			volumes = append(volumes, *volume)
		}
	}
	return volumes, nil
}

// +kubebuilder:rbac:groups="batch",resources="jobs",verbs={list}

// reconcileDirMoveJobs creates the existing volume move Jobs as defined in
// the PostgresCluster spec. A boolean value is return to indicate whether
// the main control loop should return early.
func (r *Reconciler) reconcileDirMoveJobs(ctx context.Context,
	cluster *v1beta1.PostgresCluster) (bool, error) {

	if cluster.Spec.DataSource != nil &&
		cluster.Spec.DataSource.Volumes != nil {

		moveJobs := &batchv1.JobList{}
		if err := r.Client.List(ctx, moveJobs, &client.ListOptions{
			Namespace:     cluster.Namespace,
			LabelSelector: naming.DirectoryMoveJobLabels(cluster.Name).AsSelector(),
		}); err != nil {
			return false, errors.WithStack(err)
		}

		var err error
		var pgDataReturn, pgWALReturn, repoReturn bool

		if cluster.Spec.DataSource.Volumes.PGDataVolume != nil &&
			cluster.Spec.DataSource.Volumes.PGDataVolume.
				Directory != "" &&
			cluster.Spec.DataSource.Volumes.PGDataVolume.
				PVCName != "" {
			pgDataReturn, err = r.reconcileMovePGDataDir(ctx, cluster, moveJobs)
		}

		if err == nil &&
			cluster.Spec.DataSource.Volumes.PGWALVolume != nil &&
			cluster.Spec.DataSource.Volumes.PGWALVolume.
				Directory != "" &&
			cluster.Spec.DataSource.Volumes.PGWALVolume.
				PVCName != "" {
			pgWALReturn, err = r.reconcileMoveWALDir(ctx, cluster, moveJobs)
		}

		if err == nil &&
			cluster.Spec.DataSource.Volumes.PGBackRestVolume != nil &&
			cluster.Spec.DataSource.Volumes.PGBackRestVolume.
				Directory != "" &&
			cluster.Spec.DataSource.Volumes.PGBackRestVolume.
				PVCName != "" {
			repoReturn, err = r.reconcileMoveRepoDir(ctx, cluster, moveJobs)
		}
		// if any of the 'return early' values are true, return true
		return pgDataReturn || pgWALReturn || repoReturn, err
	}

	return false, nil
}

// +kubebuilder:rbac:groups="batch",resources="jobs",verbs={create,patch,delete}

// reconcileMovePGDataDir creates a Job to move the provided pgData directory
// in the given volume to the expected location before the PostgresCluster is
// bootstrapped. It returns any errors and a boolean indicating whether the
// main control loop should continue or return early to allow time for the job
// to complete.
func (r *Reconciler) reconcileMovePGDataDir(ctx context.Context,
	cluster *v1beta1.PostgresCluster, moveJobs *batchv1.JobList) (bool, error) {

	moveDirJob := &batchv1.Job{}
	moveDirJob.ObjectMeta = naming.MovePGDataDirJob(cluster)

	// check for an existing Job
	for i := range moveJobs.Items {
		if moveJobs.Items[i].Name == moveDirJob.Name {
			if jobCompleted(&moveJobs.Items[i]) {
				// if the Job is completed, return as this only needs to run once
				return false, nil
			}
			if !jobFailed(&moveJobs.Items[i]) {
				// if the Job otherwise exists and has not failed, return and
				// give the Job time to finish
				return true, nil
			}
		}
	}

	// at this point, the Job either wasn't found or it has failed, so the it
	// should be created
	moveDirJob.ObjectMeta.Annotations = naming.Merge(cluster.Spec.Metadata.
		GetAnnotationsOrNil())
	labels := naming.Merge(cluster.Spec.Metadata.GetLabelsOrNil(),
		naming.DirectoryMoveJobLabels(cluster.Name),
		map[string]string{
			naming.LabelMovePGDataDir: "",
		})
	moveDirJob.ObjectMeta.Labels = labels

	// `patroni.dynamic.json` holds the previous state of the DCS. Since we are
	// migrating the volumes, we want to clear out any obsolete configuration info.
	script := fmt.Sprintf(`echo "Preparing cluster %s volumes for PGO v5.x"
    echo "pgdata_pvc=%s"
    echo "Current PG data directory volume contents:" 
    ls -lh "/pgdata"
    echo "Now updating PG data directory..."
    [ -d "/pgdata/%s" ] && mv "/pgdata/%s" "/pgdata/pg%s_bootstrap"
    rm -f "/pgdata/pg%s/patroni.dynamic.json"
    echo "Updated PG data directory contents:" 
    ls -lh "/pgdata"
    echo "PG Data directory preparation complete"
    `, cluster.Name,
		cluster.Spec.DataSource.Volumes.PGDataVolume.PVCName,
		cluster.Spec.DataSource.Volumes.PGDataVolume.Directory,
		cluster.Spec.DataSource.Volumes.PGDataVolume.Directory,
		strconv.Itoa(cluster.Spec.PostgresVersion),
		strconv.Itoa(cluster.Spec.PostgresVersion))

	container := corev1.Container{
		Command:         []string{"bash", "-ceu", script},
		Image:           config.PostgresContainerImage(cluster),
		ImagePullPolicy: cluster.Spec.ImagePullPolicy,
		Name:            naming.ContainerJobMovePGDataDir,
		SecurityContext: initialize.RestrictedSecurityContext(),
		VolumeMounts:    []corev1.VolumeMount{postgres.DataVolumeMount()},
	}
	if len(cluster.Spec.InstanceSets) > 0 {
		container.Resources = cluster.Spec.InstanceSets[0].Resources
	}

	jobSpec := &batchv1.JobSpec{
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: labels},
			Spec: corev1.PodSpec{
				// Set the image pull secrets, if any exist.
				// This is set here rather than using the service account due to the lack
				// of propagation to existing pods when the CRD is updated:
				// https://github.com/kubernetes/kubernetes/issues/88456
				ImagePullSecrets: cluster.Spec.ImagePullSecrets,
				Containers:       []corev1.Container{container},
				SecurityContext:  postgres.PodSecurityContext(cluster),
				// Set RestartPolicy to "Never" since we want a new Pod to be
				// created by the Job controller when there is a failure
				// (instead of the container simply restarting).
				RestartPolicy: corev1.RestartPolicyNever,
				// These Jobs don't make Kubernetes API calls, so we can just
				// use the default ServiceAccount and not mount its credentials.
				AutomountServiceAccountToken: initialize.Bool(false),
				EnableServiceLinks:           initialize.Bool(false),
				Volumes: []corev1.Volume{{
					Name: "postgres-data",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: cluster.Spec.DataSource.Volumes.
								PGDataVolume.PVCName,
						},
					}},
				},
			},
		},
	}
	// set the priority class name, if it exists
	if len(cluster.Spec.InstanceSets) > 0 &&
		cluster.Spec.InstanceSets[0].PriorityClassName != nil {
		jobSpec.Template.Spec.PriorityClassName =
			*cluster.Spec.InstanceSets[0].PriorityClassName
	}
	moveDirJob.Spec = *jobSpec

	// set gvk and ownership refs
	moveDirJob.SetGroupVersionKind(batchv1.SchemeGroupVersion.WithKind("Job"))
	if err := controllerutil.SetControllerReference(cluster, moveDirJob,
		r.Client.Scheme()); err != nil {
		return true, errors.WithStack(err)
	}

	// server-side apply the backup Job intent
	if err := r.apply(ctx, moveDirJob); err != nil {
		return true, errors.WithStack(err)
	}

	return true, nil
}

// +kubebuilder:rbac:groups="batch",resources="jobs",verbs={create,patch,delete}

// reconcileMoveWalDir creates a Job to move the provided pg_wal directory
// in the given volume to the expected location before the PostgresCluster is
// bootstrapped. It returns any errors and a boolean indicating whether the
// main control loop should continue or return early to allow time for the job
// to complete.
func (r *Reconciler) reconcileMoveWALDir(ctx context.Context,
	cluster *v1beta1.PostgresCluster, moveJobs *batchv1.JobList) (bool, error) {

	moveDirJob := &batchv1.Job{}
	moveDirJob.ObjectMeta = naming.MovePGWALDirJob(cluster)

	// check for an existing Job
	for i := range moveJobs.Items {
		if moveJobs.Items[i].Name == moveDirJob.Name {
			if jobCompleted(&moveJobs.Items[i]) {
				// if the Job is completed, return as this only needs to run once
				return false, nil
			}
			if !jobFailed(&moveJobs.Items[i]) {
				// if the Job otherwise exists and has not failed, return and
				// give the Job time to finish
				return true, nil
			}
		}
	}

	moveDirJob.ObjectMeta.Annotations = naming.Merge(cluster.Spec.Metadata.
		GetAnnotationsOrNil())
	labels := naming.Merge(cluster.Spec.Metadata.GetLabelsOrNil(),
		naming.DirectoryMoveJobLabels(cluster.Name),
		map[string]string{
			naming.LabelMovePGWalDir: "",
		})
	moveDirJob.ObjectMeta.Labels = labels

	script := fmt.Sprintf(`echo "Preparing cluster %s volumes for PGO v5.x"
    echo "pg_wal_pvc=%s"
    echo "Current PG WAL directory volume contents:"
    ls -lh "/pgwal"
    echo "Now updating PG WAL directory..."
    [ -d "/pgwal/%s" ] && mv "/pgwal/%s" "/pgwal/%s-wal"
    echo "Updated PG WAL directory contents:"
    ls -lh "/pgwal"
    echo "PG WAL directory preparation complete"
    `, cluster.Name,
		cluster.Spec.DataSource.Volumes.PGWALVolume.PVCName,
		cluster.Spec.DataSource.Volumes.PGWALVolume.Directory,
		cluster.Spec.DataSource.Volumes.PGWALVolume.Directory,
		cluster.ObjectMeta.Name)

	container := corev1.Container{
		Command:         []string{"bash", "-ceu", script},
		Image:           config.PostgresContainerImage(cluster),
		ImagePullPolicy: cluster.Spec.ImagePullPolicy,
		Name:            naming.ContainerJobMovePGWALDir,
		SecurityContext: initialize.RestrictedSecurityContext(),
		VolumeMounts:    []corev1.VolumeMount{postgres.WALVolumeMount()},
	}
	if len(cluster.Spec.InstanceSets) > 0 {
		container.Resources = cluster.Spec.InstanceSets[0].Resources
	}

	jobSpec := &batchv1.JobSpec{
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: labels},
			Spec: corev1.PodSpec{
				// Set the image pull secrets, if any exist.
				// This is set here rather than using the service account due to the lack
				// of propagation to existing pods when the CRD is updated:
				// https://github.com/kubernetes/kubernetes/issues/88456
				ImagePullSecrets: cluster.Spec.ImagePullSecrets,
				Containers:       []corev1.Container{container},
				SecurityContext:  postgres.PodSecurityContext(cluster),
				// Set RestartPolicy to "Never" since we want a new Pod to be
				// created by the Job controller when there is a failure
				// (instead of the container simply restarting).
				RestartPolicy: corev1.RestartPolicyNever,
				// These Jobs don't make Kubernetes API calls, so we can just
				// use the default ServiceAccount and not mount its credentials.
				AutomountServiceAccountToken: initialize.Bool(false),
				EnableServiceLinks:           initialize.Bool(false),
				Volumes: []corev1.Volume{{
					Name: "postgres-wal",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: cluster.Spec.DataSource.Volumes.
								PGWALVolume.PVCName,
						},
					}},
				},
			},
		},
	}
	// set the priority class name, if it exists
	if len(cluster.Spec.InstanceSets) > 0 &&
		cluster.Spec.InstanceSets[0].PriorityClassName != nil {
		jobSpec.Template.Spec.PriorityClassName =
			*cluster.Spec.InstanceSets[0].PriorityClassName
	}
	moveDirJob.Spec = *jobSpec

	// set gvk and ownership refs
	moveDirJob.SetGroupVersionKind(batchv1.SchemeGroupVersion.WithKind("Job"))
	if err := controllerutil.SetControllerReference(cluster, moveDirJob,
		r.Client.Scheme()); err != nil {
		return true, errors.WithStack(err)
	}

	// server-side apply the backup Job intent
	if err := r.apply(ctx, moveDirJob); err != nil {
		return true, errors.WithStack(err)
	}

	return true, nil
}

// +kubebuilder:rbac:groups="batch",resources="jobs",verbs={create,patch,delete}

// reconcileMoveRepoDir creates a Job to move the provided pgBackRest repo
// directory in the given volume to the expected location before the
// PostgresCluster is bootstrapped. It returns any errors and a boolean
// indicating whether the main control loop should continue or return early
// to allow time for the job to complete.
func (r *Reconciler) reconcileMoveRepoDir(ctx context.Context,
	cluster *v1beta1.PostgresCluster, moveJobs *batchv1.JobList) (bool, error) {

	moveDirJob := &batchv1.Job{}
	moveDirJob.ObjectMeta = naming.MovePGBackRestRepoDirJob(cluster)

	// check for an existing Job
	for i := range moveJobs.Items {
		if moveJobs.Items[i].Name == moveDirJob.Name {
			if jobCompleted(&moveJobs.Items[i]) {
				// if the Job is completed, return as this only needs to run once
				return false, nil
			}
			if !jobFailed(&moveJobs.Items[i]) {
				// if the Job otherwise exists and has not failed, return and
				// give the Job time to finish
				return true, nil
			}
		}
	}

	moveDirJob.ObjectMeta.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil())
	labels := naming.Merge(cluster.Spec.Metadata.GetLabelsOrNil(),
		naming.DirectoryMoveJobLabels(cluster.Name),
		map[string]string{
			naming.LabelMovePGBackRestRepoDir: "",
		})
	moveDirJob.ObjectMeta.Labels = labels

	script := fmt.Sprintf(`echo "Preparing cluster %s pgBackRest repo volume for PGO v5.x"
    echo "repo_pvc=%s"
    echo "pgbackrest directory:"
    ls -lh /pgbackrest
    echo "Current pgBackRest repo directory volume contents:" 
    ls -lh "/pgbackrest/%s"
    echo "Now updating repo directory..."
    [ -d "/pgbackrest/%s" ] && mv -t "/pgbackrest/" "/pgbackrest/%s/archive"
    [ -d "/pgbackrest/%s" ] && mv -t "/pgbackrest/" "/pgbackrest/%s/backup"
    echo "Updated /pgbackrest directory contents:"
    ls -lh "/pgbackrest"
    echo "Repo directory preparation complete"
    `, cluster.Name,
		cluster.Spec.DataSource.Volumes.PGBackRestVolume.PVCName,
		cluster.Spec.DataSource.Volumes.PGBackRestVolume.Directory,
		cluster.Spec.DataSource.Volumes.PGBackRestVolume.Directory,
		cluster.Spec.DataSource.Volumes.PGBackRestVolume.Directory,
		cluster.Spec.DataSource.Volumes.PGBackRestVolume.Directory,
		cluster.Spec.DataSource.Volumes.PGBackRestVolume.Directory)

	container := corev1.Container{
		Command:         []string{"bash", "-ceu", script},
		Image:           config.PGBackRestContainerImage(cluster),
		ImagePullPolicy: cluster.Spec.ImagePullPolicy,
		Name:            naming.ContainerJobMovePGBackRestRepoDir,
		SecurityContext: initialize.RestrictedSecurityContext(),
		VolumeMounts:    []corev1.VolumeMount{pgbackrest.RepoVolumeMount()},
	}
	if cluster.Spec.Backups.PGBackRest.RepoHost != nil {
		container.Resources = cluster.Spec.Backups.PGBackRest.RepoHost.Resources
	}

	jobSpec := &batchv1.JobSpec{
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: labels},
			Spec: corev1.PodSpec{
				// Set the image pull secrets, if any exist.
				// This is set here rather than using the service account due to the lack
				// of propagation to existing pods when the CRD is updated:
				// https://github.com/kubernetes/kubernetes/issues/88456
				ImagePullSecrets: cluster.Spec.ImagePullSecrets,
				Containers:       []corev1.Container{container},
				SecurityContext:  postgres.PodSecurityContext(cluster),
				// Set RestartPolicy to "Never" since we want a new Pod to be created by the Job
				// controller when there is a failure (instead of the container simply restarting).
				RestartPolicy: corev1.RestartPolicyNever,
				// These Jobs don't make Kubernetes API calls, so we can just
				// use the default ServiceAccount and not mount its credentials.
				AutomountServiceAccountToken: initialize.Bool(false),
				EnableServiceLinks:           initialize.Bool(false),
				Volumes: []corev1.Volume{{
					Name: "pgbackrest-repo",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: cluster.Spec.DataSource.Volumes.
								PGBackRestVolume.PVCName,
						},
					}},
				},
			},
		},
	}
	// set the priority class name, if it exists
	if repoHost := cluster.Spec.Backups.PGBackRest.RepoHost; repoHost != nil {
		if repoHost.PriorityClassName != nil {
			jobSpec.Template.Spec.PriorityClassName = *repoHost.PriorityClassName
		}
	}
	moveDirJob.Spec = *jobSpec

	// set gvk and ownership refs
	moveDirJob.SetGroupVersionKind(batchv1.SchemeGroupVersion.WithKind("Job"))
	if err := controllerutil.SetControllerReference(cluster, moveDirJob,
		r.Client.Scheme()); err != nil {
		return true, errors.WithStack(err)
	}

	// server-side apply the backup Job intent
	if err := r.apply(ctx, moveDirJob); err != nil {
		return true, errors.WithStack(err)
	}
	return true, nil
}

// handlePersistentVolumeClaimError inspects err for expected Kubernetes API
// responses to writing a PVC. It turns errors it understands into conditions
// and events. When err is handled it returns nil. Otherwise it returns err.
func (r *Reconciler) handlePersistentVolumeClaimError(
	cluster *v1beta1.PostgresCluster, err error,
) error {
	var status metav1.Status
	if api := apierrors.APIStatus(nil); errors.As(err, &api) {
		status = api.Status()
	}

	cannotResize := func(err error) {
		meta.SetStatusCondition(&cluster.Status.Conditions, metav1.Condition{
			Type:    v1beta1.PersistentVolumeResizing,
			Status:  metav1.ConditionFalse,
			Reason:  string(apierrors.ReasonForError(err)),
			Message: "One or more volumes cannot be resized",

			ObservedGeneration: cluster.Generation,
		})
	}

	volumeError := func(err error) {
		r.Recorder.Event(cluster,
			corev1.EventTypeWarning, "PersistentVolumeError", err.Error())
	}

	// Forbidden means (RBAC is broken or) the API request was rejected by an
	// admission controller. Assume it is the latter and raise the issue as a
	// condition and event.
	// - https://releases.k8s.io/v1.21.0/plugin/pkg/admission/storage/persistentvolume/resize/admission.go
	if apierrors.IsForbidden(err) {
		cannotResize(err)
		volumeError(err)
		return nil
	}

	if apierrors.IsInvalid(err) && status.Details != nil {
		unknownCause := false
		for _, cause := range status.Details.Causes {
			switch {
			// Forbidden "spec" happens when the PVC is waiting to be bound.
			// It should resolve on its own and trigger another reconcile. Raise
			// the issue as an event.
			// - https://releases.k8s.io/v1.21.0/pkg/apis/core/validation/validation.go#L2028
			//
			// TODO(cbandy): This can also happen when changing a field other
			// than requests within the spec (access modes, storage class, etc).
			// That case needs a condition or should be prevented via a webhook.
			case
				cause.Type == metav1.CauseType(field.ErrorTypeForbidden) &&
					cause.Field == "spec":
				volumeError(err)

			// Forbidden "storage" happens when the change is not allowed. Raise
			// the issue as a condition and event.
			// - https://releases.k8s.io/v1.21.0/pkg/apis/core/validation/validation.go#L2028
			case
				cause.Type == metav1.CauseType(field.ErrorTypeForbidden) &&
					cause.Field == "spec.resources.requests.storage":
				cannotResize(err)
				volumeError(err)

			default:
				unknownCause = true
			}
		}

		if len(status.Details.Causes) > 0 && !unknownCause {
			// All the causes were identified and handled.
			return nil
		}
	}

	return err
}

// getRepoPVCNames returns a map containing the names of repo PVCs that have
// the appropriate labels for each defined pgBackRest repo, if found.
func getRepoPVCNames(
	cluster *v1beta1.PostgresCluster,
	currentRepoPVCs []*corev1.PersistentVolumeClaim,
) map[string]string {

	repoPVCs := make(map[string]string)
	for _, repo := range cluster.Spec.Backups.PGBackRest.Repos {
		for _, pvc := range currentRepoPVCs {
			if pvc.Labels[naming.LabelPGBackRestRepo] == repo.Name {
				repoPVCs[repo.Name] = pvc.GetName()
				break
			}
		}
	}

	return repoPVCs
}

// getPGPVCName returns the name of a PVC that has the provided labels, if found.
func getPGPVCName(labelMap map[string]string,
	clusterVolumes []corev1.PersistentVolumeClaim,
) (string, error) {

	selector, err := naming.AsSelector(metav1.LabelSelector{
		MatchLabels: labelMap,
	})
	if err != nil {
		return "", errors.WithStack(err)
	}

	for _, pvc := range clusterVolumes {
		if selector.Matches(labels.Set(pvc.GetLabels())) {
			return pvc.GetName(), nil
		}
	}

	return "", nil
}
