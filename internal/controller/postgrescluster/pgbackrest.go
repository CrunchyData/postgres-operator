// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/patroni"
	"github.com/crunchydata/postgres-operator/internal/pgbackrest"
	"github.com/crunchydata/postgres-operator/internal/pki"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

const (
	// ConditionPostgresDataInitialized is the type used in a condition to indicate whether or not the
	// PostgresCluster's PostgreSQL data directory has been initialized (e.g. via a restore)
	ConditionPostgresDataInitialized = "PostgresDataInitialized"

	// ConditionManualBackupSuccessful is the type used in a condition to indicate whether or not
	// the manual backup for the current backup ID (as provided via annotation) was successful
	ConditionManualBackupSuccessful = "PGBackRestManualBackupSuccessful"

	// ConditionReplicaCreate is the type used in a condition to indicate whether or not
	// pgBackRest can be utilized for replica creation
	ConditionReplicaCreate = "PGBackRestReplicaCreate"

	// ConditionReplicaRepoReady is the type used in a condition to indicate whether or not
	// the pgBackRest repository for creating replicas is ready
	ConditionReplicaRepoReady = "PGBackRestReplicaRepoReady"

	// ConditionRepoHostReady is the type used in a condition to indicate whether or not a
	// pgBackRest repository host PostgresCluster is ready
	ConditionRepoHostReady = "PGBackRestRepoHostReady"

	// ConditionPGBackRestRestoreProgressing is the type used in a condition to indicate that
	// and in-place pgBackRest restore is in progress
	ConditionPGBackRestRestoreProgressing = "PGBackRestoreProgressing"

	// EventRepoHostNotFound is used to indicate that a pgBackRest repository was not
	// found when reconciling
	EventRepoHostNotFound = "RepoDeploymentNotFound"

	// EventRepoHostCreated is the event reason utilized when a pgBackRest repository host is
	// created
	EventRepoHostCreated = "RepoHostCreated"

	// EventUnableToCreateStanzas is the event reason utilized when pgBackRest is unable to create
	// stanzas for the repositories in a PostgreSQL cluster
	EventUnableToCreateStanzas = "UnableToCreateStanzas"

	// EventStanzasCreated is the event reason utilized when a pgBackRest stanza create command
	// completes successfully
	EventStanzasCreated = "StanzasCreated"

	// EventUnableToCreatePGBackRestCronJob is the event reason utilized when a pgBackRest backup
	// CronJob fails to create successfully
	EventUnableToCreatePGBackRestCronJob = "UnableToCreatePGBackRestCronJob"

	// ReasonReadyForRestore is the reason utilized within ConditionPGBackRestRestoreProgressing
	// to indicate that the restore Job can proceed because the cluster is now ready to be
	// restored (i.e. it has been properly prepared for a restore).
	ReasonReadyForRestore = "ReadyForRestore"
)

// backup types
const (
	full         = "full"
	differential = "diff"
	incremental  = "incr"
)

// regexRepoIndex is the regex used to obtain the repo index from a pgBackRest repo name
var regexRepoIndex = regexp.MustCompile(`\d+`)

// RepoResources is used to store various resources for pgBackRest repositories and
// repository hosts
type RepoResources struct {
	hosts                   []*appsv1.StatefulSet
	cronjobs                []*batchv1.CronJob
	manualBackupJobs        []*batchv1.Job
	replicaCreateBackupJobs []*batchv1.Job
	pvcs                    []*corev1.PersistentVolumeClaim
	sas                     []*corev1.ServiceAccount
	roles                   []*rbacv1.Role
	rolebindings            []*rbacv1.RoleBinding
}

// applyRepoHostIntent ensures the pgBackRest repository host StatefulSet is synchronized with the
// proper configuration according to the provided PostgresCluster custom resource.  This is done by
// applying the PostgresCluster controller's fully specified intent for the repository host
// StatefulSet.  Any changes to the deployment spec as a result of synchronization will result in a
// rollout of the pgBackRest repository host StatefulSet in accordance with its configured
// strategy.
func (r *Reconciler) applyRepoHostIntent(ctx context.Context, postgresCluster *v1beta1.PostgresCluster,
	repoHostName string, repoResources *RepoResources,
	observedInstances *observedInstances) (*appsv1.StatefulSet, error) {

	repo, err := r.generateRepoHostIntent(ctx, postgresCluster, repoHostName, repoResources, observedInstances)
	if err != nil {
		return nil, err
	}

	// Previous versions of PGO used a StatefulSet Pod Management Policy that could leave the Pod
	// in a failed state. When we see that it has the wrong policy, we will delete the StatefulSet
	// and then recreate it with the correct policy, as this is not a property that can be patched.
	// When we delete the StatefulSet, we will leave its Pods in place. They will be claimed by
	// the StatefulSet that gets created in the next reconcile.
	existing := &appsv1.StatefulSet{}
	if err := errors.WithStack(r.Client.Get(ctx, client.ObjectKeyFromObject(repo), existing)); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
	} else {
		if existing.Spec.PodManagementPolicy != repo.Spec.PodManagementPolicy {
			// We want to delete the STS without affecting the Pods, so we set the PropagationPolicy to Orphan.
			// The orphaned Pods will be claimed by the new StatefulSet that gets created in the next reconcile.
			uid := existing.GetUID()
			version := existing.GetResourceVersion()
			exactly := client.Preconditions{UID: &uid, ResourceVersion: &version}
			propagate := client.PropagationPolicy(metav1.DeletePropagationOrphan)

			return repo, errors.WithStack(r.Client.Delete(ctx, existing, exactly, propagate))
		}
	}

	if err := r.apply(ctx, repo); err != nil {
		return nil, err
	}

	return repo, nil
}

// +kubebuilder:rbac:groups="",resources="persistentvolumeclaims",verbs={create,patch}

// applyRepoVolumeIntent ensures the pgBackRest repository host deployment is synchronized with the
// proper configuration according to the provided PostgresCluster custom resource.  This is done by
// applying the PostgresCluster controller's fully specified intent for the PersistentVolumeClaim
// representing a repository.
func (r *Reconciler) applyRepoVolumeIntent(ctx context.Context,
	postgresCluster *v1beta1.PostgresCluster, spec corev1.PersistentVolumeClaimSpec,
	repoName string, repoResources *RepoResources) (*corev1.PersistentVolumeClaim, error) {

	repo, err := r.generateRepoVolumeIntent(postgresCluster, spec, repoName, repoResources)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if err := r.apply(ctx, repo); err != nil {
		return nil, r.handlePersistentVolumeClaimError(postgresCluster,
			errors.WithStack(err))
	}

	return repo, nil
}

// +kubebuilder:rbac:groups="apps",resources="statefulsets",verbs={list}
// +kubebuilder:rbac:groups="batch",resources="cronjobs",verbs={list}
// +kubebuilder:rbac:groups="batch",resources="jobs",verbs={list}
// +kubebuilder:rbac:groups="",resources="configmaps",verbs={list}
// +kubebuilder:rbac:groups="",resources="persistentvolumeclaims",verbs={list}
// +kubebuilder:rbac:groups="",resources="secrets",verbs={list}
// +kubebuilder:rbac:groups="",resources="serviceaccounts",verbs={list}
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources="roles",verbs={list}
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources="rolebindings",verbs={list}

// getPGBackRestResources returns the existing pgBackRest resources that should utilized by the
// PostgresCluster controller during reconciliation.  Any items returned are verified to be owned
// by the PostgresCluster controller and still applicable per the current PostgresCluster spec.
// Additionally, and resources identified that no longer correspond to any current configuration
// are deleted.
func (r *Reconciler) getPGBackRestResources(ctx context.Context,
	postgresCluster *v1beta1.PostgresCluster,
	backupsSpecFound bool,
) (*RepoResources, error) {

	repoResources := &RepoResources{}

	gvks := []schema.GroupVersionKind{{
		Group:   appsv1.SchemeGroupVersion.Group,
		Version: appsv1.SchemeGroupVersion.Version,
		Kind:    "StatefulSetList",
	}, {
		Group:   batchv1.SchemeGroupVersion.Group,
		Version: batchv1.SchemeGroupVersion.Version,
		Kind:    "CronJobList",
	}, {
		Group:   batchv1.SchemeGroupVersion.Group,
		Version: batchv1.SchemeGroupVersion.Version,
		Kind:    "JobList",
	}, {
		Group:   corev1.SchemeGroupVersion.Group,
		Version: corev1.SchemeGroupVersion.Version,
		Kind:    "ConfigMapList",
	}, {
		Group:   corev1.SchemeGroupVersion.Group,
		Version: corev1.SchemeGroupVersion.Version,
		Kind:    "PersistentVolumeClaimList",
	}, {
		Group:   corev1.SchemeGroupVersion.Group,
		Version: corev1.SchemeGroupVersion.Version,
		Kind:    "SecretList",
	}, {
		Group:   corev1.SchemeGroupVersion.Group,
		Version: corev1.SchemeGroupVersion.Version,
		Kind:    "ServiceAccountList",
	}, {
		Group:   rbacv1.SchemeGroupVersion.Group,
		Version: rbacv1.SchemeGroupVersion.Version,
		Kind:    "RoleList",
	}, {
		Group:   rbacv1.SchemeGroupVersion.Group,
		Version: rbacv1.SchemeGroupVersion.Version,
		Kind:    "RoleBindingList",
	}}

	selector := naming.PGBackRestSelector(postgresCluster.GetName())
	for _, gvk := range gvks {
		uList := &unstructured.UnstructuredList{}
		uList.SetGroupVersionKind(gvk)
		if err := r.Client.List(ctx, uList,
			client.InNamespace(postgresCluster.GetNamespace()),
			client.MatchingLabelsSelector{Selector: selector}); err != nil {
			return nil, errors.WithStack(err)
		}
		if len(uList.Items) == 0 {
			continue
		}

		owned, err := r.cleanupRepoResources(ctx, postgresCluster, uList.Items, backupsSpecFound)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		uList.Items = owned
		if err := unstructuredToRepoResources(gvk.Kind, repoResources,
			uList); err != nil {
			return nil, errors.WithStack(err)
		}

		// if the current objects are Jobs, update the status for the Jobs
		// created by the pgBackRest scheduled backup CronJobs
		if gvk.Kind == "JobList" {
			r.setScheduledJobStatus(ctx, postgresCluster, uList.Items)
		}

	}

	return repoResources, nil
}

// +kubebuilder:rbac:groups="",resources="persistentvolumeclaims",verbs={delete}
// +kubebuilder:rbac:groups="",resources="serviceaccounts",verbs={delete}
// +kubebuilder:rbac:groups="apps",resources="statefulsets",verbs={delete}
// +kubebuilder:rbac:groups="batch",resources="cronjobs",verbs={delete}
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources="roles",verbs={delete}
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources="rolebindings",verbs={delete}

// cleanupRepoResources cleans up pgBackRest repository resources that should no longer be
// reconciled by deleting them.  This includes deleting repos (i.e. PersistentVolumeClaims) that
// are no longer associated with any repository configured within the PostgresCluster spec, or any
// pgBackRest repository host resources if a repository host is no longer configured.
func (r *Reconciler) cleanupRepoResources(ctx context.Context,
	postgresCluster *v1beta1.PostgresCluster,
	ownedResources []unstructured.Unstructured,
	backupsSpecFound bool,
) ([]unstructured.Unstructured, error) {

	// stores the resources that should not be deleted
	ownedNoDelete := []unstructured.Unstructured{}
	for i, owned := range ownedResources {
		delete := true

		// helper to determine if a label is present in the PostgresCluster
		hasLabel := func(label string) bool { _, ok := owned.GetLabels()[label]; return ok }

		// this switch identifies the type of pgBackRest resource via its labels, and then
		// determines whether or not it should be deleted according to the current PostgresCluster
		// spec
		switch {
		case hasLabel(naming.LabelPGBackRestConfig):
			if !backupsSpecFound {
				break
			}
			// Simply add the things we never want to delete (e.g. the pgBackRest configuration)
			// to the slice and do not delete
			ownedNoDelete = append(ownedNoDelete, owned)
			delete = false
		case hasLabel(naming.LabelPGBackRestDedicated):
			if !backupsSpecFound {
				break
			}
			// Any resources from before 5.1 that relate to the previously required
			// SSH configuration should be deleted.
			// TODO(tjmoore4): This can be removed once 5.0 is EOL.
			if owned.GetName() != naming.PGBackRestSSHConfig(postgresCluster).Name &&
				owned.GetName() != naming.PGBackRestSSHSecret(postgresCluster).Name {
				// If a dedicated repo host resource and a dedicated repo host is enabled, then
				// add to the slice and do not delete.
				ownedNoDelete = append(ownedNoDelete, owned)
				delete = false
			}
		case hasLabel(naming.LabelPGBackRestRepoVolume):
			if !backupsSpecFound {
				break
			}
			// If a volume (PVC) is identified for a repo that no longer exists in the
			// spec then delete it.  Otherwise add it to the slice and continue.
			for _, repo := range postgresCluster.Spec.Backups.PGBackRest.Repos {
				// we only care about cleaning up local repo volumes (PVCs), and ignore other repo
				// types (e.g. for external Azure, GCS or S3 repositories)
				if repo.Volume != nil &&
					(repo.Name == owned.GetLabels()[naming.LabelPGBackRestRepo]) {
					ownedNoDelete = append(ownedNoDelete, owned)
					delete = false
				}
			}
		case hasLabel(naming.LabelPGBackRestBackup):
			if !backupsSpecFound {
				break
			}
			// If a Job is identified for a repo that no longer exists in the spec then
			// delete it.  Otherwise add it to the slice and continue.
			for _, repo := range postgresCluster.Spec.Backups.PGBackRest.Repos {
				if repo.Name == owned.GetLabels()[naming.LabelPGBackRestRepo] {
					ownedNoDelete = append(ownedNoDelete, owned)
					delete = false
				}
			}
		case hasLabel(naming.LabelPGBackRestCronJob):
			if !backupsSpecFound {
				break
			}
			for _, repo := range postgresCluster.Spec.Backups.PGBackRest.Repos {
				if repo.Name == owned.GetLabels()[naming.LabelPGBackRestRepo] {
					if backupScheduleFound(repo,
						owned.GetLabels()[naming.LabelPGBackRestCronJob]) {
						delete = false
						ownedNoDelete = append(ownedNoDelete, owned)
					}
					break
				}
			}
		case hasLabel(naming.LabelPGBackRestRestore):
			if !backupsSpecFound {
				break
			}
			// When a cluster is prepared for restore, the system identifier is removed from status
			// and the cluster is therefore no longer bootstrapped.  Only once the restore Job is
			// complete will the cluster then be bootstrapped again, which means by the time we
			// detect a restore Job here and a bootstrapped cluster, the Job and any associated
			// configuration resources can be safely removed.
			if !patroni.ClusterBootstrapped(postgresCluster) {
				ownedNoDelete = append(ownedNoDelete, owned)
				delete = false
			}
		case hasLabel(naming.LabelPGBackRest):
			if !backupsSpecFound {
				break
			}
			ownedNoDelete = append(ownedNoDelete, owned)
			delete = false
		}

		// If nothing has specified that the resource should not be deleted, then delete
		if delete {
			if err := r.Client.Delete(ctx, &ownedResources[i],
				client.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil {
				return []unstructured.Unstructured{}, errors.WithStack(err)
			}
		}
	}

	// return the remaining resources after properly cleaning up any that should no longer exist
	return ownedNoDelete, nil
}

// backupScheduleFound returns true if the CronJob in question should be created as
// defined by the postgrescluster CRD, otherwise it returns false.
func backupScheduleFound(repo v1beta1.PGBackRestRepo, backupType string) bool {
	if repo.BackupSchedules != nil {
		switch backupType {
		case full:
			return repo.BackupSchedules.Full != nil
		case differential:
			return repo.BackupSchedules.Differential != nil
		case incremental:
			return repo.BackupSchedules.Incremental != nil
		default:
			return false
		}
	}
	return false
}

// unstructuredToRepoResources converts unstructured pgBackRest repository resources (specifically
// unstructured StatefulSetLists and PersistentVolumeClaimList) into their structured equivalent.
func unstructuredToRepoResources(kind string, repoResources *RepoResources,
	uList *unstructured.UnstructuredList) error {

	switch kind {
	case "StatefulSetList":
		var stsList appsv1.StatefulSetList
		if err := runtime.DefaultUnstructuredConverter.
			FromUnstructured(uList.UnstructuredContent(), &stsList); err != nil {
			return errors.WithStack(err)
		}
		for i := range stsList.Items {
			repoResources.hosts = append(repoResources.hosts, &stsList.Items[i])
		}
	case "CronJobList":
		var cronList batchv1.CronJobList
		if err := runtime.DefaultUnstructuredConverter.
			FromUnstructured(uList.UnstructuredContent(), &cronList); err != nil {
			return errors.WithStack(err)
		}
		for i := range cronList.Items {
			repoResources.cronjobs = append(repoResources.cronjobs, &cronList.Items[i])
		}
	case "JobList":
		var jobList batchv1.JobList
		if err := runtime.DefaultUnstructuredConverter.
			FromUnstructured(uList.UnstructuredContent(), &jobList); err != nil {
			return errors.WithStack(err)
		}
		// we care about replica create backup jobs and manual backup jobs
		for i, job := range jobList.Items {
			switch job.GetLabels()[naming.LabelPGBackRestBackup] {
			case string(naming.BackupReplicaCreate):
				repoResources.replicaCreateBackupJobs =
					append(repoResources.replicaCreateBackupJobs, &jobList.Items[i])
			case string(naming.BackupManual):
				repoResources.manualBackupJobs =
					append(repoResources.manualBackupJobs, &jobList.Items[i])
			}
		}
	case "ConfigMapList":
		// Repository host now uses mTLS for encryption, authentication, and authorization.
		// Configmaps for SSHD are no longer managed here.
	case "PersistentVolumeClaimList":
		var pvcList corev1.PersistentVolumeClaimList
		if err := runtime.DefaultUnstructuredConverter.
			FromUnstructured(uList.UnstructuredContent(), &pvcList); err != nil {
			return errors.WithStack(err)
		}
		for i := range pvcList.Items {
			repoResources.pvcs = append(repoResources.pvcs, &pvcList.Items[i])
		}
	case "SecretList":
		// Repository host now uses mTLS for encryption, authentication, and authorization.
		// Secrets for SSHD are no longer managed here.
		// TODO(tjmoore4): Consider adding all pgBackRest secrets to RepoResources to
		// observe all pgBackRest secrets in one place.
	case "ServiceAccountList":
		var saList corev1.ServiceAccountList
		if err := runtime.DefaultUnstructuredConverter.
			FromUnstructured(uList.UnstructuredContent(), &saList); err != nil {
			return errors.WithStack(err)
		}
		for i := range saList.Items {
			repoResources.sas = append(repoResources.sas, &saList.Items[i])
		}
	case "RoleList":
		var roleList rbacv1.RoleList
		if err := runtime.DefaultUnstructuredConverter.
			FromUnstructured(uList.UnstructuredContent(), &roleList); err != nil {
			return errors.WithStack(err)
		}
		for i := range roleList.Items {
			repoResources.roles = append(repoResources.roles, &roleList.Items[i])
		}
	case "RoleBindingList":
		var rb rbacv1.RoleBindingList
		if err := runtime.DefaultUnstructuredConverter.
			FromUnstructured(uList.UnstructuredContent(), &rb); err != nil {
			return errors.WithStack(err)
		}
		for i := range rb.Items {
			repoResources.rolebindings = append(repoResources.rolebindings, &rb.Items[i])
		}
	default:
		return fmt.Errorf("unexpected kind %q", kind)
	}

	return nil
}

// setScheduledJobStatus sets the status of the scheduled pgBackRest backup Jobs
// on the postgres cluster CRD
func (r *Reconciler) setScheduledJobStatus(ctx context.Context,
	postgresCluster *v1beta1.PostgresCluster,
	items []unstructured.Unstructured) {
	log := logging.FromContext(ctx)

	uList := &unstructured.UnstructuredList{Items: items}
	var jobList batchv1.JobList
	if err := runtime.DefaultUnstructuredConverter.
		FromUnstructured(uList.UnstructuredContent(), &jobList); err != nil {
		// as this is only setting a status that is not otherwise used
		// by the Operator, simply log an error and return rather than
		// bubble this up to the other functions
		log.Error(err, "unable to convert unstructured objects to jobs, "+
			"unable to set scheduled backup status")
		return
	}

	// TODO(tjmoore4): PGBackRestScheduledBackupStatus can likely be combined with
	// PGBackRestJobStatus as they both contain most of the same information
	scheduledStatus := []v1beta1.PGBackRestScheduledBackupStatus{}
	for _, job := range jobList.Items {
		// we only care about the scheduled backup Jobs created by the
		// associated CronJobs
		sbs := v1beta1.PGBackRestScheduledBackupStatus{}
		if job.GetLabels()[naming.LabelPGBackRestCronJob] != "" {
			if len(job.OwnerReferences) > 0 {
				sbs.CronJobName = job.OwnerReferences[0].Name
			}
			sbs.RepoName = job.GetLabels()[naming.LabelPGBackRestRepo]
			sbs.Type = job.GetLabels()[naming.LabelPGBackRestCronJob]
			sbs.StartTime = job.Status.StartTime
			sbs.CompletionTime = job.Status.CompletionTime
			sbs.Active = job.Status.Active
			sbs.Succeeded = job.Status.Succeeded
			sbs.Failed = job.Status.Failed

			scheduledStatus = append(scheduledStatus, sbs)
		}
	}

	// if nil, create the pgBackRest status
	if postgresCluster.Status.PGBackRest == nil {
		postgresCluster.Status.PGBackRest = &v1beta1.PGBackRestStatus{}
	}
	postgresCluster.Status.PGBackRest.ScheduledBackups = scheduledStatus
}

// generateRepoHostIntent creates and populates StatefulSet with the PostgresCluster's full intent
// as needed to create and reconcile a pgBackRest dedicated repository host within the kubernetes
// cluster.
func (r *Reconciler) generateRepoHostIntent(ctx context.Context, postgresCluster *v1beta1.PostgresCluster,
	repoHostName string, repoResources *RepoResources, observedInstances *observedInstances,
) (*appsv1.StatefulSet, error) {

	annotations := naming.Merge(
		postgresCluster.Spec.Metadata.GetAnnotationsOrNil(),
		postgresCluster.Spec.Backups.PGBackRest.Metadata.GetAnnotationsOrNil())
	labels := naming.Merge(
		postgresCluster.Spec.Metadata.GetLabelsOrNil(),
		postgresCluster.Spec.Backups.PGBackRest.Metadata.GetLabelsOrNil(),
		naming.PGBackRestDedicatedLabels(postgresCluster.GetName()),
		map[string]string{
			naming.LabelData: naming.DataPGBackRest,
		})

	repo := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        repoHostName,
			Namespace:   postgresCluster.GetNamespace(),
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: naming.PGBackRestDedicatedLabels(postgresCluster.GetName()),
			},
			ServiceName: naming.ClusterPodService(postgresCluster).Name,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
			},
		},
	}

	if repoHost := postgresCluster.Spec.Backups.PGBackRest.RepoHost; repoHost != nil {
		repo.Spec.Template.Spec.Affinity = repoHost.Affinity
		repo.Spec.Template.Spec.Tolerations = repoHost.Tolerations
		repo.Spec.Template.Spec.TopologySpreadConstraints = repoHost.TopologySpreadConstraints
		if repoHost.PriorityClassName != nil {
			repo.Spec.Template.Spec.PriorityClassName = *repoHost.PriorityClassName
		}
	}

	// if default pod scheduling is not explicitly disabled, add the default
	// pod topology spread constraints
	if postgresCluster.Spec.DisableDefaultPodScheduling == nil ||
		(postgresCluster.Spec.DisableDefaultPodScheduling != nil &&
			!*postgresCluster.Spec.DisableDefaultPodScheduling) {
		repo.Spec.Template.Spec.TopologySpreadConstraints = append(
			repo.Spec.Template.Spec.TopologySpreadConstraints,
			defaultTopologySpreadConstraints(
				naming.ClusterDataForPostgresAndPGBackRest(postgresCluster.Name),
			)...)
	}

	// Set the image pull secrets, if any exist.
	// This is set here rather than using the service account due to the lack
	// of propagation to existing pods when the CRD is updated:
	// https://github.com/kubernetes/kubernetes/issues/88456
	repo.Spec.Template.Spec.ImagePullSecrets = postgresCluster.Spec.ImagePullSecrets

	// determine if any PG Pods still exist
	var instancePodExists bool
	for _, instance := range observedInstances.forCluster {
		if len(instance.Pods) > 0 {
			instancePodExists = true
			break
		}
	}

	// if the cluster is set to be shutdown and no instance Pods remain, stop the repohost pod
	if postgresCluster.Spec.Shutdown != nil && *postgresCluster.Spec.Shutdown &&
		!instancePodExists {
		repo.Spec.Replicas = initialize.Int32(0)
	} else {
		// the cluster should not be shutdown, set this value to 1
		repo.Spec.Replicas = initialize.Int32(1)
	}

	// Use StatefulSet's "RollingUpdate" strategy and "Parallel" policy to roll
	// out changes to pods even when not Running or not Ready.
	// - https://docs.k8s.io/concepts/workloads/controllers/statefulset/#rolling-updates
	// - https://docs.k8s.io/concepts/workloads/controllers/statefulset/#forced-rollback
	// - https://kep.k8s.io/3541
	repo.Spec.PodManagementPolicy = appsv1.ParallelPodManagement
	repo.Spec.UpdateStrategy.Type = appsv1.RollingUpdateStatefulSetStrategyType

	// Restart containers any time they stop, die, are killed, etc.
	// - https://docs.k8s.io/concepts/workloads/pods/pod-lifecycle/#restart-policy
	repo.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyAlways

	// When ShareProcessNamespace is enabled, Kubernetes' pause process becomes
	// PID 1 and reaps those processes when they complete.
	// - https://github.com/kubernetes/kubernetes/commit/81d27aa23969b77f
	//
	// The pgBackRest TLS server must be signaled when its configuration or
	// certificates change. Let containers see each other's processes.
	// - https://docs.k8s.io/tasks/configure-pod-container/share-process-namespace/
	repo.Spec.Template.Spec.ShareProcessNamespace = initialize.Bool(true)

	// pgBackRest does not make any Kubernetes API calls. Use the default
	// ServiceAccount and do not mount its credentials.
	repo.Spec.Template.Spec.AutomountServiceAccountToken = initialize.Bool(false)

	// Do not add environment variables describing services in this namespace.
	repo.Spec.Template.Spec.EnableServiceLinks = initialize.Bool(false)

	repo.Spec.Template.Spec.SecurityContext = postgres.PodSecurityContext(postgresCluster)

	pgbackrest.AddServerToRepoPod(ctx, postgresCluster, &repo.Spec.Template.Spec)

	if pgbackrest.RepoHostVolumeDefined(postgresCluster) {
		// add the init container to make the pgBackRest repo volume log directory
		pgbackrest.MakePGBackrestLogDir(&repo.Spec.Template, postgresCluster)

		// add pgBackRest repo volumes to pod
		if err := pgbackrest.AddRepoVolumesToPod(postgresCluster, &repo.Spec.Template,
			getRepoPVCNames(postgresCluster, repoResources.pvcs),
			naming.PGBackRestRepoContainerName); err != nil {
			return nil, errors.WithStack(err)
		}
	}
	// add configs to pod
	pgbackrest.AddConfigToRepoPod(postgresCluster, &repo.Spec.Template.Spec)

	// add nss_wrapper init container and add nss_wrapper env vars to the pgbackrest
	// container
	addNSSWrapper(
		config.PGBackRestContainerImage(postgresCluster),
		postgresCluster.Spec.ImagePullPolicy,
		&repo.Spec.Template)

	addTMPEmptyDir(&repo.Spec.Template)

	// set ownership references
	if err := controllerutil.SetControllerReference(postgresCluster, repo,
		r.Client.Scheme()); err != nil {
		return nil, err
	}

	return repo, nil
}

func (r *Reconciler) generateRepoVolumeIntent(postgresCluster *v1beta1.PostgresCluster,
	spec corev1.PersistentVolumeClaimSpec, repoName string,
	repoResources *RepoResources) (*corev1.PersistentVolumeClaim, error) {

	annotations := naming.Merge(
		postgresCluster.Spec.Metadata.GetAnnotationsOrNil(),
		postgresCluster.Spec.Backups.PGBackRest.Metadata.GetAnnotationsOrNil())
	labels := naming.Merge(
		postgresCluster.Spec.Metadata.GetLabelsOrNil(),
		postgresCluster.Spec.Backups.PGBackRest.Metadata.GetLabelsOrNil(),
		naming.PGBackRestRepoVolumeLabels(postgresCluster.GetName(), repoName),
	)

	// generate the default metadata
	meta := naming.PGBackRestRepoVolume(postgresCluster, repoName)

	// but if there is an existing volume for this PVC, use it
	repoPVCNames := getRepoPVCNames(postgresCluster, repoResources.pvcs)
	if repoPVCNames[repoName] != "" {
		meta = metav1.ObjectMeta{
			Name:      repoPVCNames[repoName],
			Namespace: postgresCluster.GetNamespace(),
		}
	}

	meta.Labels = labels
	meta.Annotations = annotations

	repoVol := &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "PersistentVolumeClaim",
		},
		ObjectMeta: meta,
		Spec:       spec,
	}

	// set ownership references
	if err := controllerutil.SetControllerReference(postgresCluster, repoVol,
		r.Client.Scheme()); err != nil {
		return nil, err
	}

	return repoVol, nil
}

// generateBackupJobSpecIntent generates a JobSpec for a pgBackRest backup job
func generateBackupJobSpecIntent(postgresCluster *v1beta1.PostgresCluster,
	repo v1beta1.PGBackRestRepo, serviceAccountName string,
	labels, annotations map[string]string, opts ...string) *batchv1.JobSpec {

	repoIndex := regexRepoIndex.FindString(repo.Name)
	cmdOpts := []string{
		"--stanza=" + pgbackrest.DefaultStanzaName,
		"--repo=" + repoIndex,
	}
	cmdOpts = append(cmdOpts, opts...)

	container := corev1.Container{
		Command: []string{"/opt/crunchy/bin/pgbackrest"},
		Env: []corev1.EnvVar{
			{Name: "COMMAND", Value: "backup"},
			{Name: "COMMAND_OPTS", Value: strings.Join(cmdOpts, " ")},
			{Name: "COMPARE_HASH", Value: "true"},
			{Name: "CONTAINER", Value: naming.PGBackRestRepoContainerName},
			{Name: "NAMESPACE", Value: postgresCluster.GetNamespace()},
			{Name: "SELECTOR", Value: naming.PGBackRestDedicatedSelector(postgresCluster.GetName()).String()},
		},
		Image:           config.PGBackRestContainerImage(postgresCluster),
		ImagePullPolicy: postgresCluster.Spec.ImagePullPolicy,
		Name:            naming.PGBackRestRepoContainerName,
		SecurityContext: initialize.RestrictedSecurityContext(),
	}

	if postgresCluster.Spec.Backups.PGBackRest.Jobs != nil {
		container.Resources = postgresCluster.Spec.Backups.PGBackRest.Jobs.Resources
	}

	jobSpec := &batchv1.JobSpec{
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: labels, Annotations: annotations},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{container},

				// Disable environment variables for services other than the Kubernetes API.
				// - https://docs.k8s.io/concepts/services-networking/connect-applications-service/#accessing-the-service
				// - https://releases.k8s.io/v1.23.0/pkg/kubelet/kubelet_pods.go#L553-L563
				EnableServiceLinks: initialize.Bool(false),

				// Set RestartPolicy to "Never" since we want a new Pod to be created by the Job
				// controller when there is a failure (instead of the container simply restarting).
				// This will ensure the Job always has the latest configs mounted following a
				// failure as needed to successfully verify config hashes and run the Job.
				RestartPolicy:      corev1.RestartPolicyNever,
				SecurityContext:    initialize.PodSecurityContext(),
				ServiceAccountName: serviceAccountName,
			},
		},
	}

	if jobs := postgresCluster.Spec.Backups.PGBackRest.Jobs; jobs != nil {
		jobSpec.TTLSecondsAfterFinished = jobs.TTLSecondsAfterFinished
	}

	// set the priority class name, tolerations, and affinity, if they exist
	if postgresCluster.Spec.Backups.PGBackRest.Jobs != nil {
		if postgresCluster.Spec.Backups.PGBackRest.Jobs.PriorityClassName != nil {
			jobSpec.Template.Spec.PriorityClassName =
				*postgresCluster.Spec.Backups.PGBackRest.Jobs.PriorityClassName
		}
		jobSpec.Template.Spec.Tolerations = postgresCluster.Spec.Backups.PGBackRest.Jobs.Tolerations
		jobSpec.Template.Spec.Affinity = postgresCluster.Spec.Backups.PGBackRest.Jobs.Affinity
	}

	// Set the image pull secrets, if any exist.
	// This is set here rather than using the service account due to the lack
	// of propagation to existing pods when the CRD is updated:
	// https://github.com/kubernetes/kubernetes/issues/88456
	jobSpec.Template.Spec.ImagePullSecrets = postgresCluster.Spec.ImagePullSecrets

	// add pgBackRest configs to template
	pgbackrest.AddConfigToRepoPod(postgresCluster, &jobSpec.Template.Spec)

	return jobSpec
}

// +kubebuilder:rbac:groups="",resources="configmaps",verbs={delete,list}
// +kubebuilder:rbac:groups="",resources="secrets",verbs={list,delete}
// +kubebuilder:rbac:groups="",resources="endpoints",verbs={get}
// +kubebuilder:rbac:groups="batch",resources="jobs",verbs={list}

// observeRestoreEnv observes the current Kubernetes environment to obtain any resources applicable
// to performing pgBackRest restores (e.g. when initializing a new cluster using an existing
// pgBackRest backup, or when restoring in-place).  This includes finding any existing Endpoints
// created by Patroni (i.e. DCS, leader and failover Endpoints), while then also finding any existing
// restore Jobs and then updating pgBackRest restore status accordingly.
func (r *Reconciler) observeRestoreEnv(ctx context.Context,
	cluster *v1beta1.PostgresCluster) ([]corev1.Endpoints, *batchv1.Job, error) {

	// lookup the various patroni endpoints
	leaderEP, dcsEP, failoverEP := corev1.Endpoints{}, corev1.Endpoints{}, corev1.Endpoints{}
	currentEndpoints := []corev1.Endpoints{}
	if err := r.Client.Get(ctx, naming.AsObjectKey(naming.PatroniLeaderEndpoints(cluster)),
		&leaderEP); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, nil, errors.WithStack(err)
		}
	} else {
		currentEndpoints = append(currentEndpoints, leaderEP)
	}
	if err := r.Client.Get(ctx, naming.AsObjectKey(naming.PatroniDistributedConfiguration(cluster)),
		&dcsEP); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, nil, errors.WithStack(err)
		}
	} else {
		currentEndpoints = append(currentEndpoints, dcsEP)
	}
	if err := r.Client.Get(ctx, naming.AsObjectKey(naming.PatroniTrigger(cluster)),
		&failoverEP); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, nil, errors.WithStack(err)
		}
	} else {
		currentEndpoints = append(currentEndpoints, failoverEP)
	}

	restoreJobs := &batchv1.JobList{}
	if err := r.Client.List(ctx, restoreJobs, &client.ListOptions{
		Namespace:     cluster.Namespace,
		LabelSelector: naming.PGBackRestRestoreJobSelector(cluster.GetName()),
	}); err != nil {
		return nil, nil, errors.WithStack(err)
	}
	var restoreJob *batchv1.Job
	if len(restoreJobs.Items) > 1 {
		return nil, nil, errors.WithStack(
			errors.New("invalid number of restore Jobs found when attempting to reconcile a " +
				"pgBackRest data source"))
	} else if len(restoreJobs.Items) == 1 {
		restoreJob = &restoreJobs.Items[0]
	}

	if restoreJob != nil {

		completed := jobCompleted(restoreJob)
		failed := jobFailed(restoreJob)

		if cluster.Status.PGBackRest != nil && cluster.Status.PGBackRest.Restore != nil {
			cluster.Status.PGBackRest.Restore.StartTime = restoreJob.Status.StartTime
			cluster.Status.PGBackRest.Restore.CompletionTime = restoreJob.Status.CompletionTime
			cluster.Status.PGBackRest.Restore.Succeeded = restoreJob.Status.Succeeded
			cluster.Status.PGBackRest.Restore.Failed = restoreJob.Status.Failed
			cluster.Status.PGBackRest.Restore.Active = restoreJob.Status.Active
			if completed || failed {
				cluster.Status.PGBackRest.Restore.Finished = true
			}
		}

		// update the data source initialized condition if the Job has finished running, and is
		// therefore in a completed or failed
		if completed {
			meta.SetStatusCondition(&cluster.Status.Conditions, metav1.Condition{
				ObservedGeneration: cluster.GetGeneration(),
				Type:               ConditionPostgresDataInitialized,
				Status:             metav1.ConditionTrue,
				Reason:             "PGBackRestRestoreComplete",
				Message:            "pgBackRest restore completed successfully",
			})
			meta.RemoveStatusCondition(&cluster.Status.Conditions,
				ConditionPGBackRestRestoreProgressing)

			// The clone process used to create resources that were used only
			// by the restore job. Clean them up if they still exist.
			selector := naming.PGBackRestRestoreConfigSelector(cluster.GetName())
			restoreConfigMaps := &corev1.ConfigMapList{}
			if err := r.Client.List(ctx, restoreConfigMaps, &client.ListOptions{
				Namespace:     cluster.Namespace,
				LabelSelector: selector,
			}); err != nil {
				return nil, nil, errors.WithStack(err)
			}
			for i := range restoreConfigMaps.Items {
				if err := r.Client.Delete(ctx, &restoreConfigMaps.Items[i]); err != nil {
					return nil, nil, errors.WithStack(err)
				}
			}
			restoreSecrets := &corev1.SecretList{}
			if err := r.Client.List(ctx, restoreSecrets, &client.ListOptions{
				Namespace:     cluster.Namespace,
				LabelSelector: selector,
			}); err != nil {
				return nil, nil, errors.WithStack(err)
			}
			for i := range restoreSecrets.Items {
				if err := r.Client.Delete(ctx, &restoreSecrets.Items[i]); err != nil {
					return nil, nil, errors.WithStack(err)
				}
			}
		} else if failed {
			meta.SetStatusCondition(&cluster.Status.Conditions, metav1.Condition{
				ObservedGeneration: cluster.GetGeneration(),
				Type:               ConditionPostgresDataInitialized,
				Status:             metav1.ConditionFalse,
				Reason:             "PGBackRestRestoreFailed",
				Message:            "pgBackRest restore failed",
			})
		}
	}

	return currentEndpoints, restoreJob, nil
}

// +kubebuilder:rbac:groups="",resources="endpoints",verbs={delete}
// +kubebuilder:rbac:groups="apps",resources="statefulsets",verbs={delete}
// +kubebuilder:rbac:groups="batch",resources="jobs",verbs={delete}

// prepareForRestore is responsible for reconciling an in place restore for the PostgresCluster.
// This includes setting a "PreparingForRestore" condition, and then removing all existing
// instance runners, as well as any Endpoints created by Patroni.  And once the cluster is no
// longer running, the "PostgresDataInitialized" condition is removed, which will cause the
// cluster to re-bootstrap using a restored data directory.
func (r *Reconciler) prepareForRestore(ctx context.Context,
	cluster *v1beta1.PostgresCluster, observed *observedInstances,
	currentEndpoints []corev1.Endpoints, restoreJob *batchv1.Job, restoreID string) error {

	setPreparingClusterCondition := func(resource string) {
		meta.SetStatusCondition(&cluster.Status.Conditions, metav1.Condition{
			ObservedGeneration: cluster.GetGeneration(),
			Type:               ConditionPGBackRestRestoreProgressing,
			Status:             metav1.ConditionTrue,
			Reason:             "RestoreInPlaceRequested",
			Message: fmt.Sprintf("Preparing cluster to restore in-place: %s",
				resource),
		})
	}

	cluster.Status.PGBackRest = &v1beta1.PGBackRestStatus{}
	cluster.Status.PGBackRest.Restore = &v1beta1.PGBackRestJobStatus{
		ID: restoreID,
	}

	// find all runners, the primary, and determine if the cluster is still running
	var clusterRunning bool
	runners := []*appsv1.StatefulSet{}
	var primary *Instance
	for i, instance := range observed.forCluster {
		if !clusterRunning {
			clusterRunning, _ = instance.IsRunning(naming.ContainerDatabase)
		}
		if instance.Runner != nil {
			runners = append(runners, instance.Runner)
		}
		if isPrimary, _ := instance.IsPrimary(); isPrimary {
			primary = observed.forCluster[i]
		}
	}

	// Set the proper startup instance for the restore.  This specifically enables a delta
	// restore by attempting to find an existing instance whose PVC (if it exists, e.g. as
	// in the case of an in-place restore where all PVCs are kept in place) can be utilized
	// for the restore.  The primary is preferred, but otherwise we will just grab the first
	// runner we find.  If no runner can be identified, then a new instance name is
	// generated, which means a non-delta restore will occur into an empty data volume (note that
	// a new name/empty volume is always used when the restore is to bootstrap a new cluster).
	if cluster.Status.StartupInstance == "" {
		if primary != nil {
			cluster.Status.StartupInstance = primary.Name
			cluster.Status.StartupInstanceSet = primary.Spec.Name
		} else if len(runners) > 0 {
			cluster.Status.StartupInstance = runners[0].GetName()
			cluster.Status.StartupInstanceSet =
				runners[0].GetLabels()[naming.LabelInstanceSet]
		} else if len(cluster.Spec.InstanceSets) > 0 {
			// Generate a hash that will be used make sure that the startup
			// instance is named consistently
			cluster.Status.StartupInstance = naming.GenerateStartupInstance(cluster,
				&cluster.Spec.InstanceSets[0]).Name
			cluster.Status.StartupInstanceSet = cluster.Spec.InstanceSets[0].Name
		} else {
			return errors.New("unable to determine startup instance for restore")
		}
	}

	// remove any existing restore Jobs
	if restoreJob != nil {
		setPreparingClusterCondition("removing restore job")
		if err := r.Client.Delete(ctx, restoreJob,
			client.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil {
			return errors.WithStack(err)
		}
		return nil
	}

	if clusterRunning {
		setPreparingClusterCondition("removing runners")
		for _, runner := range runners {
			err := r.Client.Delete(ctx, runner,
				client.PropagationPolicy(metav1.DeletePropagationForeground))
			if client.IgnoreNotFound(err) != nil {
				return errors.WithStack(err)
			}
		}
		return nil
	}

	// if everything is gone, proceed with re-bootstrapping the cluster via an in-place restore
	if len(currentEndpoints) == 0 {
		meta.RemoveStatusCondition(&cluster.Status.Conditions, ConditionPostgresDataInitialized)
		meta.SetStatusCondition(&cluster.Status.Conditions, metav1.Condition{
			ObservedGeneration: cluster.GetGeneration(),
			Type:               ConditionPGBackRestRestoreProgressing,
			Status:             metav1.ConditionTrue,
			Reason:             ReasonReadyForRestore,
			Message:            "Restoring cluster in-place",
		})
		// the cluster is no longer bootstrapped
		cluster.Status.Patroni.SystemIdentifier = ""
		// the restore will change the contents of the database, so the pgbouncer and exporter hashes
		// are no longer valid
		cluster.Status.Proxy.PGBouncer.PostgreSQLRevision = ""
		cluster.Status.Monitoring.ExporterConfiguration = ""
		return nil
	}

	setPreparingClusterCondition("removing DCS")
	// delete any Endpoints
	for i := range currentEndpoints {
		if err := r.Client.Delete(ctx, &currentEndpoints[i]); client.IgnoreNotFound(err) != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// +kubebuilder:rbac:groups="batch",resources="jobs",verbs={patch}

// reconcileRestoreJob is responsible for reconciling a Job that performs a pgBackRest restore in
// order to populate a PGDATA directory.
func (r *Reconciler) reconcileRestoreJob(ctx context.Context,
	cluster *v1beta1.PostgresCluster, sourceCluster *v1beta1.PostgresCluster,
	pgdataVolume, pgwalVolume *corev1.PersistentVolumeClaim,
	pgtablespaceVolumes []*corev1.PersistentVolumeClaim,
	dataSource *v1beta1.PostgresClusterDataSource,
	instanceName, instanceSetName, configHash, stanzaName string) error {

	repoName := dataSource.RepoName
	options := dataSource.Options

	// ensure options are properly set
	// TODO (andrewlecuyer): move validation logic to a webhook
	for _, opt := range options {
		var msg string
		switch {
		// Since '--repo' can be set with or without an equals ('=') sign, we check for both
		// usage patterns.
		case strings.Contains(opt, "--repo=") || strings.Contains(opt, "--repo "):
			msg = "Option '--repo' is not allowed: please use the 'repoName' field instead."
		case strings.Contains(opt, "--stanza"):
			msg = "Option '--stanza' is not allowed: the operator will automatically set this " +
				"option"
		case strings.Contains(opt, "--pg1-path"):
			msg = "Option '--pg1-path' is not allowed: the operator will automatically set this " +
				"option"
		case strings.Contains(opt, "--target-action"):
			msg = "Option '--target-action' is not allowed: the operator will automatically set this " +
				"option "
		case strings.Contains(opt, "--link-map"):
			msg = "Option '--link-map' is not allowed: the operator will automatically set this " +
				"option "
		}
		if msg != "" {
			r.Recorder.Eventf(cluster, corev1.EventTypeWarning, "InvalidDataSource", msg, repoName)
			return nil
		}
	}

	pgdata := postgres.DataDirectory(cluster)
	// combine options provided by user in the spec with those populated by the operator for a
	// successful restore
	opts := append(options, []string{
		"--stanza=" + stanzaName,
		"--pg1-path=" + pgdata,
		"--repo=" + regexRepoIndex.FindString(repoName)}...)

	var deltaOptFound, foundTarget bool
	for _, opt := range opts {
		switch {
		case strings.Contains(opt, "--target"):
			foundTarget = true
		case strings.Contains(opt, "--delta"):
			deltaOptFound = true
		}
	}
	if !deltaOptFound {
		opts = append(opts, "--delta")
	}

	// Note on the pgBackRest option `--target-action` in the restore job:
	// (a) `--target-action` is only allowed if `--target` and `type` are set;
	// TODO(benjaminjb): ensure that `type` is set as well before accepting `target-action`
	// (b) our restore job assumes the `hot_standby: on` default, which is true of Postgres >= 10;
	// (c) pgBackRest passes the `--target-action` setting as `recovery-target-action`
	// in PostgreSQL versions >=9.5 and as `pause_at_recovery_target` on earlier 9.x versions.
	// But note, pgBackRest may assume a default action of `pause` and may not pass any setting
	// - https://pgbackrest.org/command.html#command-restore/category-command/option-type
	// - https://www.postgresql.org/docs/14/runtime-config-wal.html#RUNTIME-CONFIG-WAL-RECOVERY-TARGET
	// - https://github.com/pgbackrest/pgbackrest/blob/bb03b3f41942d0b781931092a76877ad309001ef/src/command/restore/restore.c#L1623
	// - https://github.com/pgbackrest/pgbackrest/issues/1314
	// - https://github.com/pgbackrest/pgbackrest/issues/987
	if foundTarget {
		opts = append(opts, "--target-action=promote")
	}

	for i, instanceSpec := range cluster.Spec.InstanceSets {
		if instanceSpec.Name == instanceSetName {
			opts = append(opts, "--link-map=pg_wal="+postgres.WALDirectory(cluster,
				&cluster.Spec.InstanceSets[i]))
		}
	}

	// Check to see if huge pages have been requested in the spec. If they have, include 'huge_pages = try'
	// in the restore command. If they haven't, include 'huge_pages = off'.
	hugePagesSetting := "off"
	if postgres.HugePagesRequested(cluster) {
		hugePagesSetting = "try"
	}

	// NOTE (andrewlecuyer): Forcing users to put each argument separately might prevent the need
	// to do any escaping or use eval.
	cmd := pgbackrest.RestoreCommand(pgdata, hugePagesSetting, config.FetchKeyCommand(&cluster.Spec),
		pgtablespaceVolumes, strings.Join(opts, " "))

	// create the volume resources required for the postgres data directory
	dataVolumeMount := postgres.DataVolumeMount()
	dataVolume := corev1.Volume{
		Name: dataVolumeMount.Name,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: pgdataVolume.GetName(),
			},
		},
	}
	volumes := []corev1.Volume{dataVolume}
	volumeMounts := []corev1.VolumeMount{dataVolumeMount}

	if pgwalVolume != nil {
		walVolumeMount := postgres.WALVolumeMount()
		walVolume := corev1.Volume{
			Name: walVolumeMount.Name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pgwalVolume.GetName(),
				},
			},
		}
		volumes = append(volumes, walVolume)
		volumeMounts = append(volumeMounts, walVolumeMount)
	}

	for _, pgtablespaceVolume := range pgtablespaceVolumes {
		tablespaceVolumeMount := postgres.TablespaceVolumeMount(
			pgtablespaceVolume.Labels[naming.LabelData])
		tablespaceVolume := corev1.Volume{
			Name: tablespaceVolumeMount.Name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pgtablespaceVolume.GetName(),
				},
			},
		}
		volumes = append(volumes, tablespaceVolume)
		volumeMounts = append(volumeMounts, tablespaceVolumeMount)
	}

	restoreJob := &batchv1.Job{}
	if err := r.generateRestoreJobIntent(cluster, configHash, instanceName, cmd,
		volumeMounts, volumes, dataSource, restoreJob); err != nil {
		return errors.WithStack(err)
	}

	// add pgBackRest configs to template
	pgbackrest.AddConfigToRestorePod(cluster, sourceCluster, &restoreJob.Spec.Template.Spec)

	// add nss_wrapper init container and add nss_wrapper env vars to the pgbackrest restore
	// container
	addNSSWrapper(
		config.PGBackRestContainerImage(cluster),
		cluster.Spec.ImagePullPolicy,
		&restoreJob.Spec.Template)

	addTMPEmptyDir(&restoreJob.Spec.Template)

	return errors.WithStack(r.apply(ctx, restoreJob))
}

func (r *Reconciler) generateRestoreJobIntent(cluster *v1beta1.PostgresCluster,
	configHash, instanceName string, cmd []string,
	volumeMounts []corev1.VolumeMount, volumes []corev1.Volume,
	dataSource *v1beta1.PostgresClusterDataSource, job *batchv1.Job) error {

	meta := naming.PGBackRestRestoreJob(cluster)

	annotations := naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil(),
		cluster.Spec.Backups.PGBackRest.Metadata.GetAnnotationsOrNil(),
		map[string]string{naming.PGBackRestConfigHash: configHash})
	labels := naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		cluster.Spec.Backups.PGBackRest.Metadata.GetLabelsOrNil(),
		naming.PGBackRestRestoreJobLabels(cluster.Name),
		map[string]string{naming.LabelStartupInstance: instanceName},
	)
	meta.Annotations = annotations
	meta.Labels = labels

	job.ObjectMeta = meta
	job.Spec = batchv1.JobSpec{
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: annotations,
				Labels:      labels,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Command:         cmd,
					Image:           config.PostgresContainerImage(cluster),
					ImagePullPolicy: cluster.Spec.ImagePullPolicy,
					Name:            naming.PGBackRestRestoreContainerName,
					VolumeMounts:    volumeMounts,
					Env:             []corev1.EnvVar{{Name: "PGHOST", Value: "/tmp"}},
					SecurityContext: initialize.RestrictedSecurityContext(),
					Resources:       dataSource.Resources,
				}},
				RestartPolicy: corev1.RestartPolicyNever,
				Volumes:       volumes,
				Affinity:      dataSource.Affinity,
				Tolerations:   dataSource.Tolerations,
			},
		},
	}

	// Set the image pull secrets, if any exist.
	// This is set here rather than using the service account due to the lack
	// of propagation to existing pods when the CRD is updated:
	// https://github.com/kubernetes/kubernetes/issues/88456
	job.Spec.Template.Spec.ImagePullSecrets = cluster.Spec.ImagePullSecrets

	// pgBackRest does not make any Kubernetes API calls, but it may interact
	// with a cloud storage provider. Use the instance ServiceAccount for its
	// possible cloud identity without mounting its Kubernetes API credentials.
	// - https://cloud.google.com/kubernetes-engine/docs/concepts/workload-identity
	// - https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html
	job.Spec.Template.Spec.AutomountServiceAccountToken = initialize.Bool(false)
	job.Spec.Template.Spec.ServiceAccountName = naming.ClusterInstanceRBAC(cluster).Name

	// Do not add environment variables describing services in this namespace.
	job.Spec.Template.Spec.EnableServiceLinks = initialize.Bool(false)

	job.Spec.Template.Spec.SecurityContext = postgres.PodSecurityContext(cluster)

	// set the priority class name, if it exists
	if dataSource.PriorityClassName != nil {
		job.Spec.Template.Spec.PriorityClassName = *dataSource.PriorityClassName
	}

	job.SetGroupVersionKind(batchv1.SchemeGroupVersion.WithKind("Job"))
	if err := errors.WithStack(r.setControllerReference(cluster, job)); err != nil {
		return err
	}

	return nil
}

// reconcilePGBackRest is responsible for reconciling any/all pgBackRest resources owned by a
// specific PostgresCluster (e.g. Deployments, ConfigMaps, Secrets, etc.).  This function will
// ensure various reconciliation logic is run as needed for each pgBackRest resource, while then
// also generating the proper Result as needed to ensure proper event requeuing according to
// the results of any attempts to properly reconcile these resources.
func (r *Reconciler) reconcilePGBackRest(ctx context.Context,
	postgresCluster *v1beta1.PostgresCluster,
	instances *observedInstances,
	rootCA *pki.RootCertificateAuthority,
	backupsSpecFound bool,
) (reconcile.Result, error) {

	// add some additional context about what component is being reconciled
	log := logging.FromContext(ctx).WithValues("reconciler", "pgBackRest")

	// if nil, create the pgBackRest status that will be updated when
	// reconciling various pgBackRest resources
	if postgresCluster.Status.PGBackRest == nil {
		postgresCluster.Status.PGBackRest = &v1beta1.PGBackRestStatus{}
	}

	// create the Result that will be updated while reconciling any/all pgBackRest resources
	result := reconcile.Result{}

	// Get all currently owned pgBackRest resources in the environment as needed for
	// reconciliation.  This includes deleting resources that should no longer exist per the
	// current spec (e.g. if repos, repo hosts, etc. have been removed).
	repoResources, err := r.getPGBackRestResources(ctx, postgresCluster, backupsSpecFound)
	if err != nil {
		// exit early if can't get and clean existing resources as needed to reconcile
		return reconcile.Result{}, errors.WithStack(err)
	}

	// At this point, reconciliation is allowed, so if no backups spec is found
	// clear the status and exit
	if !backupsSpecFound {
		postgresCluster.Status.PGBackRest = &v1beta1.PGBackRestStatus{}
		return result, nil
	}

	var repoHost *appsv1.StatefulSet
	var repoHostName string
	// reconcile the pgbackrest repository host
	repoHost, err = r.reconcileDedicatedRepoHost(ctx, postgresCluster, repoResources, instances)
	if err != nil {
		log.Error(err, "unable to reconcile pgBackRest repo host")
		result.Requeue = true
		return result, nil
	}
	repoHostName = repoHost.GetName()

	if err := r.reconcilePGBackRestSecret(ctx, postgresCluster, repoHost, rootCA); err != nil {
		log.Error(err, "unable to reconcile pgBackRest secret")
		result.Requeue = true
	}

	// calculate hashes for the external repository configurations in the spec (e.g. for Azure,
	// GCS and/or S3 repositories) as needed to properly detect changes to external repository
	// configuration (and then execute stanza create commands accordingly)
	configHashes, configHash, err := pgbackrest.CalculateConfigHashes(postgresCluster)
	if err != nil {
		log.Error(err, "unable to calculate config hashes")
		result.Requeue = true
		return result, nil
	}

	// reconcile all pgbackrest repository repos
	replicaCreateRepo, err := r.reconcileRepos(ctx, postgresCluster, configHashes, repoResources)
	if err != nil {
		log.Error(err, "unable to reconcile pgBackRest repo host")
		result.Requeue = true
		return result, nil
	}

	// gather instance names and reconcile all pgbackrest configuration and secrets
	instanceNames := []string{}
	for _, instance := range instances.forCluster {
		instanceNames = append(instanceNames, instance.Name)
	}
	// sort to ensure consistent ordering of hosts when creating pgBackRest configs
	sort.Strings(instanceNames)
	if err := r.reconcilePGBackRestConfig(ctx, postgresCluster, repoHostName,
		configHash, naming.ClusterPodService(postgresCluster).Name,
		postgresCluster.GetNamespace(), instanceNames); err != nil {
		log.Error(err, "unable to reconcile pgBackRest configuration")
		result.Requeue = true
	}

	// reconcile the RBAC required to run pgBackRest Jobs (e.g. for backups)
	sa, err := r.reconcilePGBackRestRBAC(ctx, postgresCluster)
	if err != nil {
		log.Error(err, "unable to create replica creation backup")
		result.Requeue = true
		return result, nil
	}

	// reconcile the pgBackRest stanza for all configuration pgBackRest repos
	configHashMismatch, err := r.reconcileStanzaCreate(ctx, postgresCluster, instances, configHash)
	// If a stanza create error then requeue but don't return the error.  This prevents
	// stanza-create errors from bubbling up to the main Reconcile() function, which would
	// prevent subsequent reconciles from occurring.  Also, this provides a better chance
	// that the pgBackRest status will be updated at the end of the Reconcile() function,
	// e.g. to set the "stanzaCreated" indicator to false for any repos failing stanza creation
	// (assuming no other reconcile errors bubble up to the Reconcile() function and block the
	// status update).  And finally, add some time to each requeue to slow down subsequent
	// stanza create attempts in order to prevent pgBackRest mis-configuration (e.g. due to
	// custom configuration) from spamming the logs, while also ensuring stanza creation is
	// re-attempted until successful (e.g. allowing users to correct mis-configurations in
	// custom configuration and ensure stanzas are still created).
	if err != nil {
		log.Error(err, "unable to create stanza")
		result.RequeueAfter = 10 * time.Second
	}
	// If a config hash mismatch, then log an info message and requeue to try again.  Add some time
	// to the requeue to give the pgBackRest configuration changes a chance to propagate to the
	// container.
	if configHashMismatch {
		log.Info("pgBackRest config hash mismatch detected, requeuing to reattempt stanza create")
		result.RequeueAfter = 10 * time.Second
	}
	// reconcile the pgBackRest backup CronJobs
	requeue := r.reconcileScheduledBackups(ctx, postgresCluster, sa, repoResources.cronjobs)
	// If the pgBackRest backup CronJob reconciliation function has encountered an error, requeue
	// after 10 seconds. The error will not bubble up to allow the reconcile loop to continue.
	// An error is not logged because an event was already created.
	// TODO(tjmoore4): Is this the desired eventing/logging/reconciliation strategy?
	// A potential option to handle this proactively would be to use a webhook:
	// https://book.kubebuilder.io/cronjob-tutorial/webhook-implementation.html
	if requeue {
		result.RequeueAfter = 10 * time.Second
	}

	// Reconcile the initial backup that is needed to enable replica creation using pgBackRest.
	// This is done once stanza creation is successful
	if err := r.reconcileReplicaCreateBackup(ctx, postgresCluster, instances,
		repoResources.replicaCreateBackupJobs, sa, configHash, replicaCreateRepo); err != nil {
		log.Error(err, "unable to reconcile replica creation backup")
		result.Requeue = true
	}

	// Reconcile a manual backup as defined in the spec, and triggered by the end-user via
	// annotation
	if err := r.reconcileManualBackup(ctx, postgresCluster, repoResources.manualBackupJobs,
		sa, instances); err != nil {
		log.Error(err, "unable to reconcile manual backup")
		result.Requeue = true
	}

	return result, nil
}

// +kubebuilder:rbac:groups="",resources="persistentvolumeclaims",verbs={create,patch}
// +kubebuilder:rbac:groups="batch",resources="jobs",verbs={create,patch,delete}

// reconcilePostgresClusterDataSource is responsible for reconciling a PostgresCluster data source.
// This is specifically done by running a pgBackRest restore to populate a PostgreSQL data volume
// for the PostgresCluster being reconciled using the backups of another PostgresCluster.
func (r *Reconciler) reconcilePostgresClusterDataSource(ctx context.Context,
	cluster *v1beta1.PostgresCluster, dataSource *v1beta1.PostgresClusterDataSource,
	configHash string, clusterVolumes []corev1.PersistentVolumeClaim,
	rootCA *pki.RootCertificateAuthority,
	backupsSpecFound bool,
) error {

	// grab cluster, namespaces and repo name information from the data source
	sourceClusterName := dataSource.ClusterName
	// if the data source name is empty then we're restoring in-place and use the current cluster
	// as the source cluster
	if sourceClusterName == "" {
		sourceClusterName = cluster.GetName()
	}
	// if data source namespace is empty then use the same namespace as the current cluster
	sourceClusterNamespace := dataSource.ClusterNamespace
	if sourceClusterNamespace == "" {
		sourceClusterNamespace = cluster.GetNamespace()
	}
	// repo name is required by the api, so RepoName should be populated
	sourceRepoName := dataSource.RepoName

	// Ensure the proper instance and instance set can be identified via the status.  The
	// StartupInstance and StartupInstanceSet values should be populated when the cluster
	// is being prepared for a restore, and should therefore always exist at this point.
	// Therefore, if either are not found it is treated as an error.
	instanceName := cluster.Status.StartupInstance
	if instanceName == "" {
		return errors.WithStack(
			errors.New("unable to find instance name for pgBackRest restore Job"))
	}
	instanceSetName := cluster.Status.StartupInstanceSet
	if instanceSetName == "" {
		return errors.WithStack(
			errors.New("unable to find instance set name for pgBackRest restore Job"))
	}

	// Ensure an instance set can be found in the current spec that corresponds to the
	// instanceSetName.  A valid instance spec is needed to reconcile and cluster volumes
	// below (e.g. the PGDATA and/or WAL volumes).
	var instanceSet *v1beta1.PostgresInstanceSetSpec
	for i, set := range cluster.Spec.InstanceSets {
		if set.Name == instanceSetName {
			instanceSet = &cluster.Spec.InstanceSets[i]
			break
		}
	}
	if instanceSet == nil {
		return errors.WithStack(
			errors.New("unable to determine the proper instance set for the restore"))
	}

	// If the cluster is already bootstrapped, or if the bootstrap Job is complete, then
	// nothing to do.  However, also ensure the "data sources initialized" condition is set
	// to true if for some reason it doesn't exist (e.g. if it was deleted since the
	// data source for the cluster was initialized).
	if patroni.ClusterBootstrapped(cluster) {
		condition := meta.FindStatusCondition(cluster.Status.Conditions,
			ConditionPostgresDataInitialized)
		if condition == nil || (condition.Status != metav1.ConditionTrue) {
			meta.SetStatusCondition(&cluster.Status.Conditions, metav1.Condition{
				ObservedGeneration: cluster.GetGeneration(),
				Type:               ConditionPostgresDataInitialized,
				Status:             metav1.ConditionTrue,
				Reason:             "ClusterAlreadyBootstrapped",
				Message:            "The cluster is already bootstrapped",
			})
		}
		return nil
	}

	// Identify the proper source cluster.  If the source cluster configured matches the current
	// cluster, then we do not need to lookup a cluster and simply copy the current PostgresCluster.
	// Additionally, pgBackRest is reconciled to ensure any configuration needed to bootstrap the
	// cluster exists (specifically since it may not yet exist, e.g. if we're initializing the
	// data directory for a brand new PostgresCluster using existing backups for that cluster).
	// If the source cluster is not the same as the current cluster, then look it up.
	sourceCluster := &v1beta1.PostgresCluster{}
	if sourceClusterName == cluster.GetName() && sourceClusterNamespace == cluster.GetNamespace() {
		sourceCluster = cluster.DeepCopy()
		instance := &Instance{Name: instanceName}
		// Reconciling pgBackRest here will ensure a pgBackRest instance config file exists (since
		// the cluster hasn't bootstrapped yet, and pgBackRest configs therefore have not yet been
		// reconciled) as needed to properly configure the pgBackRest restore Job.
		// Note that function reconcilePGBackRest only uses forCluster in observedInstances.
		result, err := r.reconcilePGBackRest(ctx, cluster, &observedInstances{
			forCluster: []*Instance{instance},
		}, rootCA, backupsSpecFound)
		if err != nil || result != (reconcile.Result{}) {
			return fmt.Errorf("unable to reconcile pgBackRest as needed to initialize "+
				"PostgreSQL data for the cluster: %w", err)
		}
	} else {
		if err := r.Client.Get(ctx,
			client.ObjectKey{Name: sourceClusterName, Namespace: sourceClusterNamespace},
			sourceCluster); err != nil {
			if apierrors.IsNotFound(err) {
				r.Recorder.Eventf(cluster, corev1.EventTypeWarning, "InvalidDataSource",
					"PostgresCluster %q does not exist", sourceClusterName)
				return nil
			}
			return errors.WithStack(err)
		}

		// Copy repository definitions and credentials from the source cluster.
		// A copy is the only way to get this information across namespaces.
		if err := r.copyRestoreConfiguration(ctx, cluster, sourceCluster); err != nil {
			return err
		}
	}

	// verify the repo defined in the data source exists in the source cluster
	var foundRepo bool
	for _, repo := range sourceCluster.Spec.Backups.PGBackRest.Repos {
		if repo.Name == sourceRepoName {
			foundRepo = true
			break
		}
	}
	if !foundRepo {
		r.Recorder.Eventf(cluster, corev1.EventTypeWarning, "InvalidDataSource",
			"PostgresCluster %q does not have a repo named %q defined",
			sourceClusterName, sourceRepoName)
		return nil
	}

	// Define a fake STS to use when calling the reconcile functions below since when
	// bootstrapping the cluster it will not exist until after the restore is complete.
	fakeSTS := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{
		Name:      instanceName,
		Namespace: cluster.GetNamespace(),
	}}
	// Reconcile the PGDATA and WAL volumes for the restore
	pgdata, err := r.reconcilePostgresDataVolume(ctx, cluster, instanceSet, fakeSTS, clusterVolumes, sourceCluster)
	if err != nil {
		return errors.WithStack(err)
	}
	pgwal, err := r.reconcilePostgresWALVolume(ctx, cluster, instanceSet, fakeSTS, nil, clusterVolumes)
	if err != nil {
		return errors.WithStack(err)
	}

	pgtablespaces, err := r.reconcileTablespaceVolumes(ctx, cluster, instanceSet, fakeSTS, clusterVolumes)
	if err != nil {
		return errors.WithStack(err)
	}

	// reconcile the pgBackRest restore Job to populate the cluster's data directory
	if err := r.reconcileRestoreJob(ctx, cluster, sourceCluster, pgdata, pgwal, pgtablespaces,
		dataSource, instanceName, instanceSetName, configHash, pgbackrest.DefaultStanzaName); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// +kubebuilder:rbac:groups="",resources="persistentvolumeclaims",verbs={create,patch}
// +kubebuilder:rbac:groups="batch",resources="jobs",verbs={create,patch,delete}

// reconcileCloudBasedDataSource is responsible for reconciling a cloud-based PostgresCluster
// data source, i.e., S3, etc.
func (r *Reconciler) reconcileCloudBasedDataSource(ctx context.Context,
	cluster *v1beta1.PostgresCluster, dataSource *v1beta1.PGBackRestDataSource,
	configHash string, clusterVolumes []corev1.PersistentVolumeClaim) error {

	// Ensure the proper instance and instance set can be identified via the status.  The
	// StartupInstance and StartupInstanceSet values should be populated when the cluster
	// is being prepared for a restore, and should therefore always exist at this point.
	// Therefore, if either are not found it is treated as an error.
	instanceName := cluster.Status.StartupInstance
	if instanceName == "" {
		return errors.WithStack(
			errors.New("unable to find instance name for pgBackRest restore Job"))
	}
	instanceSetName := cluster.Status.StartupInstanceSet
	if instanceSetName == "" {
		return errors.WithStack(
			errors.New("unable to find instance set name for pgBackRest restore Job"))
	}

	// Ensure an instance set can be found in the current spec that corresponds to the
	// instanceSetName.  A valid instance spec is needed to reconcile and cluster volumes
	// below (e.g. the PGDATA and/or WAL volumes).
	var instanceSet *v1beta1.PostgresInstanceSetSpec
	for i, set := range cluster.Spec.InstanceSets {
		if set.Name == instanceSetName {
			instanceSet = &cluster.Spec.InstanceSets[i]
			break
		}
	}
	if instanceSet == nil {
		return errors.WithStack(
			errors.New("unable to determine the proper instance set for the restore"))
	}

	// If the cluster is already bootstrapped, or if the bootstrap Job is complete, then
	// nothing to do.  However, also ensure the "data sources initialized" condition is set
	// to true if for some reason it doesn't exist (e.g. if it was deleted since the
	// data source for the cluster was initialized).
	if patroni.ClusterBootstrapped(cluster) {
		condition := meta.FindStatusCondition(cluster.Status.Conditions,
			ConditionPostgresDataInitialized)
		if condition == nil || (condition.Status != metav1.ConditionTrue) {
			meta.SetStatusCondition(&cluster.Status.Conditions, metav1.Condition{
				ObservedGeneration: cluster.GetGeneration(),
				Type:               ConditionPostgresDataInitialized,
				Status:             metav1.ConditionTrue,
				Reason:             "ClusterAlreadyBootstrapped",
				Message:            "The cluster is already bootstrapped",
			})
		}
		return nil
	}

	if err := r.createRestoreConfig(ctx, cluster, configHash); err != nil {
		return err
	}

	// TODO(benjaminjb): Is there a way to check that a repo exists outside of spinning
	// up a pod with pgBackRest and checking?

	// Define a fake STS to use when calling the reconcile functions below since when
	// bootstrapping the cluster it will not exist until after the restore is complete.
	fakeSTS := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{
		Name:      instanceName,
		Namespace: cluster.GetNamespace(),
	}}
	// Reconcile the PGDATA and WAL volumes for the restore
	pgdata, err := r.reconcilePostgresDataVolume(ctx, cluster, instanceSet, fakeSTS, clusterVolumes, nil)
	if err != nil {
		return errors.WithStack(err)
	}
	pgwal, err := r.reconcilePostgresWALVolume(ctx, cluster, instanceSet, fakeSTS, nil, clusterVolumes)
	if err != nil {
		return errors.WithStack(err)
	}

	// TODO(benjaminjb): do we really need this for cloud-based datasources?
	pgtablespaces, err := r.reconcileTablespaceVolumes(ctx, cluster, instanceSet, fakeSTS, clusterVolumes)
	if err != nil {
		return errors.WithStack(err)
	}

	// The `reconcileRestoreJob` was originally designed to take a PostgresClusterDataSource
	// and rather than reconfigure that func's signature, we translate the PGBackRestDataSource
	tmpDataSource := &v1beta1.PostgresClusterDataSource{
		RepoName:          dataSource.Repo.Name,
		Options:           dataSource.Options,
		Resources:         dataSource.Resources,
		Affinity:          dataSource.Affinity,
		Tolerations:       dataSource.Tolerations,
		PriorityClassName: dataSource.PriorityClassName,
	}

	// reconcile the pgBackRest restore Job to populate the cluster's data directory
	// Note that the 'source cluster' is nil as this is not used by this restore type.
	if err := r.reconcileRestoreJob(ctx, cluster, nil, pgdata, pgwal, pgtablespaces, tmpDataSource,
		instanceName, instanceSetName, configHash, dataSource.Stanza); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// createRestoreConfig creates a configmap struct with pgBackRest pgbackrest.conf settings
// in the data field, for use with restoring from cloud-based data sources
func (r *Reconciler) createRestoreConfig(ctx context.Context, postgresCluster *v1beta1.PostgresCluster,
	configHash string) error {

	postgresClusterWithMockedBackups := postgresCluster.DeepCopy()
	postgresClusterWithMockedBackups.Spec.Backups.PGBackRest.Global = postgresCluster.Spec.
		DataSource.PGBackRest.Global
	postgresClusterWithMockedBackups.Spec.Backups.PGBackRest.Repos = []v1beta1.PGBackRestRepo{
		postgresCluster.Spec.DataSource.PGBackRest.Repo,
	}

	return r.reconcilePGBackRestConfig(ctx, postgresClusterWithMockedBackups,
		"", configHash, "", "", []string{})
}

// copyRestoreConfiguration copies pgBackRest configuration from another cluster for use by
// the current PostgresCluster (e.g. when restoring across namespaces, and the configuration
// for the source cluster needs to be copied into the PostgresCluster's local namespace).
func (r *Reconciler) copyRestoreConfiguration(ctx context.Context,
	cluster, sourceCluster *v1beta1.PostgresCluster,
) error {
	var err error

	sourceConfig := &corev1.ConfigMap{ObjectMeta: naming.PGBackRestConfig(sourceCluster)}
	if err == nil {
		err = errors.WithStack(
			r.Client.Get(ctx, client.ObjectKeyFromObject(sourceConfig), sourceConfig))
	}

	// Retrieve the pgBackRest Secret of the source cluster if it has one. When
	// it does not, indicate that with a nil pointer.
	sourceSecret := &corev1.Secret{ObjectMeta: naming.PGBackRestSecret(sourceCluster)}
	if err == nil {
		err = errors.WithStack(
			r.Client.Get(ctx, client.ObjectKeyFromObject(sourceSecret), sourceSecret))

		if apierrors.IsNotFound(err) {
			sourceSecret, err = nil, nil
		}
	}

	// See also [pgbackrest.CreatePGBackRestConfigMapIntent].
	config := &corev1.ConfigMap{ObjectMeta: naming.PGBackRestConfig(cluster)}
	config.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))

	config.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil(),
		cluster.Spec.Backups.PGBackRest.Metadata.GetAnnotationsOrNil(),
	)
	config.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		cluster.Spec.Backups.PGBackRest.Metadata.GetLabelsOrNil(),
		naming.PGBackRestConfigLabels(cluster.GetName()),
	)
	if err == nil {
		err = r.setControllerReference(cluster, config)
	}

	// See also [Reconciler.reconcilePGBackRestSecret].
	secret := &corev1.Secret{ObjectMeta: naming.PGBackRestSecret(cluster)}
	secret.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))
	secret.Type = corev1.SecretTypeOpaque

	secret.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil(),
		cluster.Spec.Backups.PGBackRest.Metadata.GetAnnotationsOrNil(),
	)
	secret.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		cluster.Spec.Backups.PGBackRest.Metadata.GetLabelsOrNil(),
		naming.PGBackRestConfigLabels(cluster.Name),
	)
	if err == nil {
		err = r.setControllerReference(cluster, secret)
	}
	if err == nil {
		pgbackrest.RestoreConfig(
			sourceConfig, config,
			sourceSecret, secret,
		)
	}
	if err == nil {
		err = errors.WithStack(r.apply(ctx, config))
	}

	// Write the Secret when there is something we want to keep in it.
	if err == nil && len(secret.Data) != 0 {
		err = errors.WithStack(r.apply(ctx, secret))
	}

	// copy any needed projected Secrets or ConfigMaps
	if err == nil {
		err = r.copyConfigurationResources(ctx, cluster, sourceCluster)
	}

	return err
}

// copyConfigurationResources copies all pgBackRest configuration ConfigMaps and
// Secrets used by the source cluster when bootstrapping the new cluster using
// pgBackRest restore. This ensures those configuration resources mounted as
// VolumeProjections by the source cluster can be used by the new cluster during
// bootstrapping.
func (r *Reconciler) copyConfigurationResources(ctx context.Context, cluster,
	sourceCluster *v1beta1.PostgresCluster) error {

	for i := range sourceCluster.Spec.Backups.PGBackRest.Configuration {
		// While all volume projections from .Configuration will be carried over to
		// the pgBackRest restore Job, we only explicitly copy the relevant ConfigMaps
		// and Secrets. Any DownwardAPI or ServiceAccountToken projections will need
		// to be handled manually.
		// - https://kubernetes.io/docs/concepts/storage/projected-volumes/
		if sourceCluster.Spec.Backups.PGBackRest.Configuration[i].Secret != nil {
			secretProjection := sourceCluster.Spec.Backups.PGBackRest.Configuration[i].Secret
			secretCopy := &corev1.Secret{}
			secretName := types.NamespacedName{
				Name:      secretProjection.Name,
				Namespace: sourceCluster.Namespace,
			}
			// Get the existing Secret for the copy, if it exists. It **must**
			// exist if not configured as optional.
			if secretProjection.Optional != nil && *secretProjection.Optional {
				if err := errors.WithStack(r.Client.Get(ctx, secretName,
					secretCopy)); apierrors.IsNotFound(err) {
					continue
				} else {
					return err
				}
			} else {
				if err := errors.WithStack(
					r.Client.Get(ctx, secretName, secretCopy)); err != nil {
					return err
				}
			}
			// Set a unique name for the Secret copy using the original Secret
			// name and the Secret projection index number.
			secretCopyName := fmt.Sprintf(naming.RestoreConfigCopySuffix, secretProjection.Name, i)

			// set the new name and namespace
			secretCopy.ObjectMeta = metav1.ObjectMeta{
				Name:      secretCopyName,
				Namespace: cluster.Namespace,
			}
			secretCopy.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))
			secretCopy.Annotations = naming.Merge(
				cluster.Spec.Metadata.GetAnnotationsOrNil(),
				cluster.Spec.Backups.PGBackRest.Metadata.GetAnnotationsOrNil(),
			)
			secretCopy.Labels = naming.Merge(
				cluster.Spec.Metadata.GetLabelsOrNil(),
				cluster.Spec.Backups.PGBackRest.Metadata.GetLabelsOrNil(),
				// this label allows for cleanup when the restore completes
				naming.PGBackRestRestoreJobLabels(cluster.Name),
			)
			if err := r.setControllerReference(cluster, secretCopy); err != nil {
				return err
			}

			if err := errors.WithStack(r.apply(ctx, secretCopy)); err != nil {
				return err
			}
			// update the copy of the source PostgresCluster to add the new Secret
			// projection(s) to the restore Job
			sourceCluster.Spec.Backups.PGBackRest.Configuration[i].Secret.Name = secretCopyName
		}

		if sourceCluster.Spec.Backups.PGBackRest.Configuration[i].ConfigMap != nil {
			configMapProjection := sourceCluster.Spec.Backups.PGBackRest.Configuration[i].ConfigMap
			configMapCopy := &corev1.ConfigMap{}
			configMapName := types.NamespacedName{
				Name:      configMapProjection.Name,
				Namespace: sourceCluster.Namespace,
			}
			// Get the existing ConfigMap for the copy, if it exists. It **must**
			// exist if not configured as optional.
			if configMapProjection.Optional != nil && *configMapProjection.Optional {
				if err := errors.WithStack(r.Client.Get(ctx, configMapName,
					configMapCopy)); apierrors.IsNotFound(err) {
					continue
				} else {
					return err
				}
			} else {
				if err := errors.WithStack(
					r.Client.Get(ctx, configMapName, configMapCopy)); err != nil {
					return err
				}
			}
			// Set a unique name for the ConfigMap copy using the original ConfigMap
			// name and the ConfigMap projection index number.
			configMapCopyName := fmt.Sprintf(naming.RestoreConfigCopySuffix, configMapProjection.Name, i)

			// set the new name and namespace
			configMapCopy.ObjectMeta = metav1.ObjectMeta{
				Name:      configMapCopyName,
				Namespace: cluster.Namespace,
			}
			configMapCopy.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))
			configMapCopy.Annotations = naming.Merge(
				cluster.Spec.Metadata.GetAnnotationsOrNil(),
				cluster.Spec.Backups.PGBackRest.Metadata.GetAnnotationsOrNil(),
			)
			configMapCopy.Labels = naming.Merge(
				cluster.Spec.Metadata.GetLabelsOrNil(),
				cluster.Spec.Backups.PGBackRest.Metadata.GetLabelsOrNil(),
				// this label allows for cleanup when the restore completes
				naming.PGBackRestRestoreJobLabels(cluster.Name),
			)
			if err := r.setControllerReference(cluster, configMapCopy); err != nil {
				return err
			}
			if err := errors.WithStack(r.apply(ctx, configMapCopy)); err != nil {
				return err
			}
			// update the copy of the source PostgresCluster to add the new ConfigMap
			// projection(s) to the restore Job
			sourceCluster.Spec.Backups.PGBackRest.Configuration[i].ConfigMap.Name = configMapCopyName
		}
	}
	return nil
}

// reconcilePGBackRestConfig is responsible for reconciling the pgBackRest ConfigMaps and Secrets.
func (r *Reconciler) reconcilePGBackRestConfig(ctx context.Context,
	postgresCluster *v1beta1.PostgresCluster,
	repoHostName, configHash, serviceName, serviceNamespace string,
	instanceNames []string) error {

	backrestConfig := pgbackrest.CreatePGBackRestConfigMapIntent(postgresCluster, repoHostName,
		configHash, serviceName, serviceNamespace, instanceNames)
	if err := controllerutil.SetControllerReference(postgresCluster, backrestConfig,
		r.Client.Scheme()); err != nil {
		return err
	}
	if err := r.apply(ctx, backrestConfig); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// +kubebuilder:rbac:groups="",resources="secrets",verbs={get}
// +kubebuilder:rbac:groups="",resources="secrets",verbs={create,delete,patch}

// reconcilePGBackRestSecret reconciles the pgBackRest Secret.
func (r *Reconciler) reconcilePGBackRestSecret(ctx context.Context,
	cluster *v1beta1.PostgresCluster, repoHost *appsv1.StatefulSet,
	rootCA *pki.RootCertificateAuthority) error {

	intent := &corev1.Secret{ObjectMeta: naming.PGBackRestSecret(cluster)}
	intent.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))
	intent.Type = corev1.SecretTypeOpaque

	intent.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil(),
		cluster.Spec.Backups.PGBackRest.Metadata.GetAnnotationsOrNil())
	intent.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		cluster.Spec.Backups.PGBackRest.Metadata.GetLabelsOrNil(),
		naming.PGBackRestConfigLabels(cluster.Name),
	)

	existing := &corev1.Secret{}
	err := errors.WithStack(client.IgnoreNotFound(
		r.Client.Get(ctx, client.ObjectKeyFromObject(intent), existing)))

	if err == nil {
		err = r.setControllerReference(cluster, intent)
	}
	if err == nil {
		err = pgbackrest.Secret(ctx, cluster, repoHost, rootCA, existing, intent)
	}

	// Delete the Secret when it exists and there is nothing we want to keep in it.
	if err == nil && len(existing.UID) != 0 && len(intent.Data) == 0 {
		err = errors.WithStack(client.IgnoreNotFound(
			r.deleteControlled(ctx, cluster, existing)))
	}

	// Write the Secret when there is something we want to keep in it.
	if err == nil && len(intent.Data) != 0 {
		err = errors.WithStack(r.apply(ctx, intent))
	}
	return err
}

// +kubebuilder:rbac:groups="",resources="serviceaccounts",verbs={create,patch}
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources="roles",verbs={create,patch}
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources="rolebindings",verbs={create,patch}

// reconcileInstanceRBAC reconciles the Role, RoleBinding, and ServiceAccount for
// pgBackRest
func (r *Reconciler) reconcilePGBackRestRBAC(ctx context.Context,
	postgresCluster *v1beta1.PostgresCluster) (*corev1.ServiceAccount, error) {

	sa := &corev1.ServiceAccount{ObjectMeta: naming.PGBackRestRBAC(postgresCluster)}
	sa.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ServiceAccount"))

	role := &rbacv1.Role{ObjectMeta: naming.PGBackRestRBAC(postgresCluster)}
	role.SetGroupVersionKind(rbacv1.SchemeGroupVersion.WithKind("Role"))

	binding := &rbacv1.RoleBinding{ObjectMeta: naming.PGBackRestRBAC(postgresCluster)}
	binding.SetGroupVersionKind(rbacv1.SchemeGroupVersion.WithKind("RoleBinding"))

	if err := r.setControllerReference(postgresCluster, sa); err != nil {
		return nil, errors.WithStack(err)
	}
	if err := r.setControllerReference(postgresCluster, binding); err != nil {
		return nil, errors.WithStack(err)
	}
	if err := r.setControllerReference(postgresCluster, role); err != nil {
		return nil, errors.WithStack(err)
	}

	sa.Annotations = naming.Merge(postgresCluster.Spec.Metadata.GetAnnotationsOrNil(),
		postgresCluster.Spec.Backups.PGBackRest.Metadata.GetAnnotationsOrNil())
	sa.Labels = naming.Merge(postgresCluster.Spec.Metadata.GetLabelsOrNil(),
		postgresCluster.Spec.Backups.PGBackRest.Metadata.GetLabelsOrNil(),
		naming.PGBackRestLabels(postgresCluster.GetName()))
	binding.Annotations = naming.Merge(postgresCluster.Spec.Metadata.GetAnnotationsOrNil(),
		postgresCluster.Spec.Backups.PGBackRest.Metadata.GetAnnotationsOrNil())
	binding.Labels = naming.Merge(postgresCluster.Spec.Metadata.GetLabelsOrNil(),
		postgresCluster.Spec.Backups.PGBackRest.Metadata.GetLabelsOrNil(),
		naming.PGBackRestLabels(postgresCluster.GetName()))
	role.Annotations = naming.Merge(postgresCluster.Spec.Metadata.GetAnnotationsOrNil(),
		postgresCluster.Spec.Backups.PGBackRest.Metadata.GetAnnotationsOrNil())
	role.Labels = naming.Merge(postgresCluster.Spec.Metadata.GetLabelsOrNil(),
		postgresCluster.Spec.Backups.PGBackRest.Metadata.GetLabelsOrNil(),
		naming.PGBackRestLabels(postgresCluster.GetName()))

	binding.RoleRef = rbacv1.RoleRef{
		APIGroup: rbacv1.SchemeGroupVersion.Group,
		Kind:     role.Kind,
		Name:     role.Name,
	}
	binding.Subjects = []rbacv1.Subject{{
		Kind: sa.Kind,
		Name: sa.Name,
	}}
	role.Rules = pgbackrest.Permissions(postgresCluster)

	if err := r.apply(ctx, sa); err != nil {
		return nil, errors.WithStack(err)
	}
	if err := r.apply(ctx, role); err != nil {
		return nil, errors.WithStack(err)
	}
	if err := r.apply(ctx, binding); err != nil {
		return nil, errors.WithStack(err)
	}

	return sa, nil
}

// reconcileDedicatedRepoHost is responsible for reconciling a pgBackRest dedicated repository host
// StatefulSet according to a specific PostgresCluster custom resource.
func (r *Reconciler) reconcileDedicatedRepoHost(ctx context.Context,
	postgresCluster *v1beta1.PostgresCluster,
	repoResources *RepoResources,
	observedInstances *observedInstances) (*appsv1.StatefulSet, error) {

	log := logging.FromContext(ctx).WithValues("reconcileResource", "repoHost")

	// ensure conditions are set before returning as needed by subsequent reconcile functions
	defer func() {
		repoHostReady := metav1.Condition{
			ObservedGeneration: postgresCluster.GetGeneration(),
			Type:               ConditionRepoHostReady,
		}
		if postgresCluster.Status.PGBackRest.RepoHost == nil {
			repoHostReady.Status = metav1.ConditionUnknown
			repoHostReady.Reason = "RepoHostStatusMissing"
			repoHostReady.Message = "pgBackRest dedicated repository host status is missing"
		} else if postgresCluster.Status.PGBackRest.RepoHost.Ready {
			repoHostReady.Status = metav1.ConditionTrue
			repoHostReady.Reason = "RepoHostReady"
			repoHostReady.Message = "pgBackRest dedicated repository host is ready"
		} else {
			repoHostReady.Status = metav1.ConditionFalse
			repoHostReady.Reason = "RepoHostNotReady"
			repoHostReady.Message = "pgBackRest dedicated repository host is not ready"
		}
		meta.SetStatusCondition(&postgresCluster.Status.Conditions, repoHostReady)
	}()
	var isCreate bool
	if len(repoResources.hosts) == 0 {
		name := fmt.Sprintf("%s-%s", postgresCluster.GetName(), "repo-host")
		repoResources.hosts = append(repoResources.hosts, &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			}})
		isCreate = true
	} else {
		sort.Slice(repoResources.hosts, func(i, j int) bool {
			return repoResources.hosts[i].CreationTimestamp.Before(
				&repoResources.hosts[j].CreationTimestamp)
		})
	}
	repoHostName := repoResources.hosts[0].Name
	repoHost, err := r.applyRepoHostIntent(ctx, postgresCluster, repoHostName, repoResources,
		observedInstances)
	if err != nil {
		log.Error(err, "reconciling repository host")
		return nil, err
	}

	postgresCluster.Status.PGBackRest.RepoHost = getRepoHostStatus(repoHost)

	if isCreate {
		r.Recorder.Eventf(postgresCluster, corev1.EventTypeNormal, EventRepoHostCreated,
			"created pgBackRest repository host %s/%s", repoHost.TypeMeta.Kind, repoHostName)
	}

	return repoHost, nil
}

// +kubebuilder:rbac:groups="batch",resources="jobs",verbs={create,patch,delete}

// reconcileManualBackup is responsible for reconciling pgBackRest backups that are initiated
// manually by the end-user
func (r *Reconciler) reconcileManualBackup(ctx context.Context,
	postgresCluster *v1beta1.PostgresCluster, manualBackupJobs []*batchv1.Job,
	serviceAccount *corev1.ServiceAccount, instances *observedInstances) error {

	manualAnnotation := postgresCluster.GetAnnotations()[naming.PGBackRestBackup]
	manualStatus := postgresCluster.Status.PGBackRest.ManualBackup

	// first update status and cleanup according to any existing manual backup Jobs observed in
	// the environment
	var currentBackupJob *batchv1.Job
	if len(manualBackupJobs) > 0 {

		currentBackupJob = manualBackupJobs[0]
		completed := jobCompleted(currentBackupJob)
		failed := jobFailed(currentBackupJob)
		backupID := currentBackupJob.GetAnnotations()[naming.PGBackRestBackup]

		if manualStatus != nil && manualStatus.ID == backupID {
			if completed {
				meta.SetStatusCondition(&postgresCluster.Status.Conditions, metav1.Condition{
					ObservedGeneration: postgresCluster.GetGeneration(),
					Type:               ConditionManualBackupSuccessful,
					Status:             metav1.ConditionTrue,
					Reason:             "ManualBackupComplete",
					Message:            "Manual backup completed successfully",
				})
			} else if failed {
				meta.SetStatusCondition(&postgresCluster.Status.Conditions, metav1.Condition{
					ObservedGeneration: postgresCluster.GetGeneration(),
					Type:               ConditionManualBackupSuccessful,
					Status:             metav1.ConditionFalse,
					Reason:             "ManualBackupFailed",
					Message:            "Manual backup did not complete successfully",
				})
			}

			// update the manual backup status based on the current status of the manual backup Job
			manualStatus.StartTime = currentBackupJob.Status.StartTime
			manualStatus.CompletionTime = currentBackupJob.Status.CompletionTime
			manualStatus.Succeeded = currentBackupJob.Status.Succeeded
			manualStatus.Failed = currentBackupJob.Status.Failed
			manualStatus.Active = currentBackupJob.Status.Active
			if completed || failed {
				manualStatus.Finished = true
			}
		}

		// If the Job is finished with a "completed" or "failure" condition, and the Job is not
		// annotated per the current value of the "pgbackrest-backup" annotation, then delete it so
		// that a new Job can be generated with the proper (i.e. new) backup ID.  This means any
		// Jobs that are in progress will complete before being deleted to trigger a new backup
		// per a new value for the annotation (unless the user manually deletes the Job).
		if completed || failed {
			if manualAnnotation != "" && backupID != manualAnnotation {
				return errors.WithStack(r.Client.Delete(ctx, currentBackupJob,
					client.PropagationPolicy(metav1.DeletePropagationBackground)))
			}
		}
	}

	// pgBackRest connects to a PostgreSQL instance that is not in recovery to
	// initiate a backup. Similar to "writable" but not exactly.
	clusterWritable := false
	for _, instance := range instances.forCluster {
		writable, known := instance.IsWritable()
		if writable && known {
			clusterWritable = true
			break
		}
	}

	// nothing to reconcile if there is no postgres or if a manual backup has not been
	// requested
	//
	// TODO (andrewlecuyer): Since reconciliation doesn't currently occur when a leader is elected,
	// the operator may not get another chance to create the backup if a writable instance is not
	// detected, and it then returns without requeuing.  To ensure this doesn't occur and that the
	// operator always has a chance to reconcile when an instance becomes writable, we should watch
	// Pods in the cluster for leader election events, and trigger reconciles accordingly.
	if !clusterWritable || manualAnnotation == "" ||
		postgresCluster.Spec.Backups.PGBackRest.Manual == nil {
		return nil
	}

	// if there is an existing status, see if a new backup id has been provided, and if so reset
	// the status and proceed with reconciling a new backup
	if manualStatus == nil || manualStatus.ID != manualAnnotation {
		manualStatus = &v1beta1.PGBackRestJobStatus{
			ID: manualAnnotation,
		}
		// Remove an existing manual backup condition if present.  It will be
		// created again as needed based on the newly reconciled backup Job.
		meta.RemoveStatusCondition(&postgresCluster.Status.Conditions,
			ConditionManualBackupSuccessful)

		postgresCluster.Status.PGBackRest.ManualBackup = manualStatus
	}

	// if the status shows the Job is no longer in progress, then simply exit (which means a Job
	// that has reached a "completed" or "failed" status is no longer reconciled)
	if manualStatus != nil && manualStatus.Finished {
		return nil
	}

	// determine if the dedicated repository host is ready using the repo host ready
	// condition, and return if not
	repoCondition := meta.FindStatusCondition(postgresCluster.Status.Conditions, ConditionRepoHostReady)
	if repoCondition == nil || repoCondition.Status != metav1.ConditionTrue {
		return nil
	}

	// Determine if the replica create backup is complete and return if not. This allows for proper
	// orchestration of backup Jobs since only one backup can be run at a time.
	backupCondition := meta.FindStatusCondition(postgresCluster.Status.Conditions,
		ConditionReplicaCreate)
	if backupCondition == nil || backupCondition.Status != metav1.ConditionTrue {
		return nil
	}

	// Verify that status exists for the repo configured for the manual backup, and that a stanza
	// has been created, before proceeding.  If either conditions are not true, then simply return
	// without requeuing and record and event (subsequent events, e.g. successful stanza creation,
	// writing of the proper repo status, adding a missing repo, etc. will trigger the reconciles
	// needed to try again).
	var statusFound, stanzaCreated bool
	repoName := postgresCluster.Spec.Backups.PGBackRest.Manual.RepoName
	for _, repo := range postgresCluster.Status.PGBackRest.Repos {
		if repo.Name == repoName {
			statusFound = true
			stanzaCreated = repo.StanzaCreated
		}
	}
	if !statusFound {
		r.Recorder.Eventf(postgresCluster, corev1.EventTypeWarning, "InvalidBackupRepo",
			"Unable to find status for %q as configured for a manual backup.  Please ensure "+
				"this repo is defined in the spec.", repoName)
		return nil
	}
	if !stanzaCreated {
		r.Recorder.Eventf(postgresCluster, corev1.EventTypeWarning, "StanzaNotCreated",
			"Stanza not created for %q as specified for a manual backup", repoName)
		return nil
	}

	var repo v1beta1.PGBackRestRepo
	for i := range postgresCluster.Spec.Backups.PGBackRest.Repos {
		if postgresCluster.Spec.Backups.PGBackRest.Repos[i].Name == repoName {
			repo = postgresCluster.Spec.Backups.PGBackRest.Repos[i]
		}
	}
	if repo.Name == "" {
		return errors.Errorf("repo %q is not defined for this cluster", repoName)
	}

	// Users should specify the repo for the command using the "manual.repoName" field in the spec,
	// and not using the "--repo" option in the "manual.options" field.  Therefore, record a
	// warning event and return if a "--repo" option is found.  Reconciliation will then be
	// reattempted when "--repo" is removed from "manual.options" and the spec is updated.
	// Since '--repo' can be set with or without an equals ('=') sign, we check for both
	// usage patterns.
	backupOpts := postgresCluster.Spec.Backups.PGBackRest.Manual.Options
	for _, opt := range backupOpts {
		if strings.Contains(opt, "--repo=") || strings.Contains(opt, "--repo ") {
			r.Recorder.Eventf(postgresCluster, corev1.EventTypeWarning, "InvalidManualBackup",
				"Option '--repo' is not allowed: please use the 'repoName' field instead.",
				repoName)
			return nil
		}
	}

	// create the backup Job
	backupJob := &batchv1.Job{}
	backupJob.ObjectMeta = naming.PGBackRestBackupJob(postgresCluster)
	if currentBackupJob != nil {
		backupJob.ObjectMeta.Name = currentBackupJob.ObjectMeta.Name
	}

	var labels, annotations map[string]string
	labels = naming.Merge(postgresCluster.Spec.Metadata.GetLabelsOrNil(),
		postgresCluster.Spec.Backups.PGBackRest.Metadata.GetLabelsOrNil(),
		naming.PGBackRestBackupJobLabels(postgresCluster.GetName(), repoName,
			naming.BackupManual))
	annotations = naming.Merge(postgresCluster.Spec.Metadata.GetAnnotationsOrNil(),
		postgresCluster.Spec.Backups.PGBackRest.Metadata.GetAnnotationsOrNil(),
		map[string]string{
			naming.PGBackRestBackup: manualAnnotation,
		})
	backupJob.ObjectMeta.Labels = labels
	backupJob.ObjectMeta.Annotations = annotations

	spec := generateBackupJobSpecIntent(postgresCluster, repo,
		serviceAccount.GetName(), labels, annotations, backupOpts...)

	backupJob.Spec = *spec

	// set gvk and ownership refs
	backupJob.SetGroupVersionKind(batchv1.SchemeGroupVersion.WithKind("Job"))
	if err := controllerutil.SetControllerReference(postgresCluster, backupJob,
		r.Client.Scheme()); err != nil {
		return errors.WithStack(err)
	}

	// server-side apply the backup Job intent
	if err := r.apply(ctx, backupJob); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// +kubebuilder:rbac:groups="batch",resources="jobs",verbs={create,patch,delete}

// reconcileReplicaCreateBackup is responsible for reconciling a full pgBackRest backup for the
// cluster as required to create replicas
func (r *Reconciler) reconcileReplicaCreateBackup(ctx context.Context,
	postgresCluster *v1beta1.PostgresCluster, instances *observedInstances,
	replicaCreateBackupJobs []*batchv1.Job,
	serviceAccount *corev1.ServiceAccount, configHash string,
	replicaCreateRepo v1beta1.PGBackRestRepo) error {

	var replicaCreateRepoStatus *v1beta1.RepoStatus
	for i, repo := range postgresCluster.Status.PGBackRest.Repos {
		if repo.Name == replicaCreateRepo.Name {
			replicaCreateRepoStatus = &postgresCluster.Status.PGBackRest.Repos[i]
			break
		}
	}

	// ensure condition is set before returning as needed by subsequent reconcile functions
	defer func() {
		replicaCreate := metav1.Condition{
			ObservedGeneration: postgresCluster.GetGeneration(),
			Type:               ConditionReplicaCreate,
		}
		if replicaCreateRepoStatus == nil {
			replicaCreate.Status = metav1.ConditionUnknown
			replicaCreate.Reason = "RepoStatusMissing"
			replicaCreate.Message = "Status is missing for the replica create repo"
		} else if replicaCreateRepoStatus.ReplicaCreateBackupComplete {
			replicaCreate.Status = metav1.ConditionTrue
			replicaCreate.Reason = "RepoBackupComplete"
			replicaCreate.Message = "pgBackRest replica creation is now possible"
		} else {
			replicaCreate.Status = metav1.ConditionFalse
			replicaCreate.Reason = "RepoBackupNotComplete"
			replicaCreate.Message = "pgBackRest replica creation is not currently " +
				"possible"
		}
		meta.SetStatusCondition(&postgresCluster.Status.Conditions, replicaCreate)
	}()

	// pgBackRest connects to a PostgreSQL instance that is not in recovery to
	// initiate a backup. Similar to "writable" but not exactly.
	clusterWritable := false
	for _, instance := range instances.forCluster {
		writable, known := instance.IsWritable()
		if writable && known {
			clusterWritable = true
			break
		}
	}

	// return early when there is no postgres, no repo, or the backup is already complete.
	//
	// TODO (andrewlecuyer): Since reconciliation doesn't currently occur when a leader is elected,
	// the operator may not get another chance to create the backup if a writable instance is not
	// detected, and it then returns without requeuing.  To ensure this doesn't occur and that the
	// operator always has a chance to reconcile when an instance becomes writable, we should watch
	// Pods in the cluster for leader election events, and trigger reconciles accordingly.
	if !clusterWritable || replicaCreateRepoStatus == nil || replicaCreateRepoStatus.ReplicaCreateBackupComplete {
		return nil
	}

	// determine if the replica create repo is ready using the "PGBackRestReplicaRepoReady" condition
	var replicaRepoReady bool
	condition := meta.FindStatusCondition(postgresCluster.Status.Conditions, ConditionReplicaRepoReady)
	if condition != nil {
		replicaRepoReady = (condition.Status == metav1.ConditionTrue)
	}

	// determine if the dedicated repository host is ready using the repo host ready status
	var dedicatedRepoReady bool
	condition = meta.FindStatusCondition(postgresCluster.Status.Conditions, ConditionRepoHostReady)
	if condition != nil {
		dedicatedRepoReady = (condition.Status == metav1.ConditionTrue)
	}

	// grab the current job if one exists, and perform any required Job cleanup or update the
	// PostgresCluster status as required
	var job *batchv1.Job
	if len(replicaCreateBackupJobs) > 0 {
		job = replicaCreateBackupJobs[0]

		failed := jobFailed(job)
		completed := jobCompleted(job)

		// determine if the replica creation repo has changed
		replicaCreateRepoChanged := true
		if replicaCreateRepo.Name == job.GetLabels()[naming.LabelPGBackRestRepo] {
			replicaCreateRepoChanged = false
		}

		// Delete an existing Job (whether running or not) under the following conditions:
		// - The job has failed.  The Job will be deleted and recreated to try again.
		// - The replica creation repo has changed since the Job was created.  Delete and recreate
		//   with the Job with the proper repo configured.
		// - The "config hash" annotation has changed, indicating a configuration change has been
		//   made in the spec (specifically a change to the config for an external repo).  Delete
		//   and recreate the Job with proper hash per the current config.
		if failed || replicaCreateRepoChanged ||
			(job.GetAnnotations()[naming.PGBackRestConfigHash] != configHash) {
			if err := r.Client.Delete(ctx, job,
				client.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil {
				return errors.WithStack(err)
			}
			return nil
		}

		// if the Job completed then update status and return
		if completed {
			replicaCreateRepoStatus.ReplicaCreateBackupComplete = true
			return nil
		}
	}

	// return if no job has been created and the replica repo or the dedicated
	// repo host is not ready
	if job == nil && (!dedicatedRepoReady || !replicaRepoReady) {
		return nil
	}

	// create the backup Job, and populate ObjectMeta based on whether or not a Job already exists
	backupJob := &batchv1.Job{}
	backupJob.ObjectMeta = naming.PGBackRestBackupJob(postgresCluster)
	if job != nil {
		backupJob.ObjectMeta.Name = job.ObjectMeta.Name
	}

	var labels, annotations map[string]string
	labels = naming.Merge(postgresCluster.Spec.Metadata.GetLabelsOrNil(),
		postgresCluster.Spec.Backups.PGBackRest.Metadata.GetLabelsOrNil(),
		naming.PGBackRestBackupJobLabels(postgresCluster.GetName(),
			postgresCluster.Spec.Backups.PGBackRest.Repos[0].Name, naming.BackupReplicaCreate))
	annotations = naming.Merge(postgresCluster.Spec.Metadata.GetAnnotationsOrNil(),
		postgresCluster.Spec.Backups.PGBackRest.Metadata.GetAnnotationsOrNil(),
		map[string]string{
			naming.PGBackRestConfigHash: configHash,
		})
	backupJob.ObjectMeta.Labels = labels
	backupJob.ObjectMeta.Annotations = annotations

	spec := generateBackupJobSpecIntent(postgresCluster, replicaCreateRepo,
		serviceAccount.GetName(), labels, annotations)

	backupJob.Spec = *spec

	// set gvk and ownership refs
	backupJob.SetGroupVersionKind(batchv1.SchemeGroupVersion.WithKind("Job"))
	if err := controllerutil.SetControllerReference(postgresCluster, backupJob,
		r.Client.Scheme()); err != nil {
		return errors.WithStack(err)
	}

	if err := r.apply(ctx, backupJob); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// reconcileRepos is responsible for reconciling any pgBackRest repositories configured
// for the cluster
func (r *Reconciler) reconcileRepos(ctx context.Context,
	postgresCluster *v1beta1.PostgresCluster, extConfigHashes map[string]string,
	repoResources *RepoResources) (v1beta1.PGBackRestRepo, error) {

	log := logging.FromContext(ctx).WithValues("reconcileResource", "repoVolume")

	errors := []error{}
	errMsg := "reconciling repository volume"
	repoVols := []*corev1.PersistentVolumeClaim{}
	var replicaCreateRepo v1beta1.PGBackRestRepo
	for i, repo := range postgresCluster.Spec.Backups.PGBackRest.Repos {
		// the repo at index 0 is the replica creation repo
		if i == 0 {
			replicaCreateRepo = postgresCluster.Spec.Backups.PGBackRest.Repos[i]
		}
		// we only care about reconciling repo volumes, so ignore everything else
		if repo.Volume == nil {
			continue
		}
		repo, err := r.applyRepoVolumeIntent(ctx, postgresCluster, repo.Volume.VolumeClaimSpec,
			repo.Name, repoResources)
		if err != nil {
			log.Error(err, errMsg)
			errors = append(errors, err)
			continue
		}
		if repo != nil {
			repoVols = append(repoVols, repo)
		}
	}

	postgresCluster.Status.PGBackRest.Repos =
		getRepoVolumeStatus(postgresCluster.Status.PGBackRest.Repos, repoVols, extConfigHashes,
			replicaCreateRepo.Name)

	return replicaCreateRepo, utilerrors.NewAggregate(errors)
}

// +kubebuilder:rbac:groups="",resources="pods",verbs={get,list}
// +kubebuilder:rbac:groups="",resources="pods/exec",verbs={create}

// reconcileStanzaCreate is responsible for ensuring stanzas are properly created for the
// pgBackRest repositories configured for a PostgresCluster.  If the bool returned from this
// function is false, this indicates that a pgBackRest config hash mismatch was identified that
// prevented the "pgbackrest stanza-create" command from running (with a config has mismatch
// indicating that pgBackRest configuration as stored in the pgBackRest ConfigMap has not yet
// propagated to the Pod).
func (r *Reconciler) reconcileStanzaCreate(ctx context.Context,
	postgresCluster *v1beta1.PostgresCluster,
	instances *observedInstances, configHash string) (bool, error) {

	// ensure conditions are set before returning as needed by subsequent reconcile functions
	defer func() {
		var replicaCreateRepoStatus *v1beta1.RepoStatus
		if len(postgresCluster.Spec.Backups.PGBackRest.Repos) == 0 {
			return
		}
		replicaCreateRepoName := postgresCluster.Spec.Backups.PGBackRest.Repos[0].Name
		for i, repo := range postgresCluster.Status.PGBackRest.Repos {
			if repo.Name == replicaCreateRepoName {
				replicaCreateRepoStatus = &postgresCluster.Status.PGBackRest.Repos[i]
				break
			}
		}

		replicaCreateRepoReady := metav1.Condition{
			ObservedGeneration: postgresCluster.GetGeneration(),
			Type:               ConditionReplicaRepoReady,
		}
		if replicaCreateRepoStatus == nil {
			replicaCreateRepoReady.Status = metav1.ConditionUnknown
			replicaCreateRepoReady.Reason = "RepoStatusMissing"
			replicaCreateRepoReady.Message = "Status is missing for the replica creation repo"
		} else if replicaCreateRepoStatus.StanzaCreated {
			replicaCreateRepoReady.Status = metav1.ConditionTrue
			replicaCreateRepoReady.Reason = "StanzaCreated"
			replicaCreateRepoReady.Message = "pgBackRest replica create repo is ready for " +
				"backups"
		} else {
			replicaCreateRepoReady.Status = metav1.ConditionFalse
			replicaCreateRepoReady.Reason = "StanzaNotCreated"
			replicaCreateRepoReady.Message = "pgBackRest replica create repo is not ready " +
				"for backups"
		}
		meta.SetStatusCondition(&postgresCluster.Status.Conditions, replicaCreateRepoReady)
	}()

	// determine if the cluster has been initialized. pgBackRest compares the
	// local PostgreSQL data directory to information it sees in a PostgreSQL
	// instance that is not in recovery. Similar to "writable" but not exactly.
	//
	// also, capture the name of the writable instance, since that instance (i.e.
	// the primary) is where the stanza create command will always be run.  This
	// is possible as of the following change in pgBackRest v2.33:
	// https://github.com/pgbackrest/pgbackrest/pull/1326.
	clusterWritable := false
	var writableInstanceName string
	for _, instance := range instances.forCluster {
		writable, known := instance.IsWritable()
		if writable && known {
			clusterWritable = true
			writableInstanceName = instance.Name + "-0"
			break
		}
	}

	stanzasCreated := true
	for _, repoStatus := range postgresCluster.Status.PGBackRest.Repos {
		if !repoStatus.StanzaCreated {
			stanzasCreated = false
			break
		}
	}

	// returns if the cluster is not yet writable, or if it has been initialized and
	// all stanzas have already been created successfully
	//
	// TODO (andrewlecuyer): Since reconciliation doesn't currently occur when a leader is elected,
	// the operator may not get another chance to create the stanza if a writable instance is not
	// detected, and it then returns without requeuing.  To ensure this doesn't occur and that the
	// operator always has a chance to reconcile when an instance becomes writable, we should watch
	// Pods in the cluster for leader election events, and trigger reconciles accordingly.
	if !clusterWritable || stanzasCreated {
		return false, nil
	}

	// create a pgBackRest executor and attempt stanza creation
	exec := func(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer,
		command ...string) error {
		return r.PodExec(ctx, postgresCluster.GetNamespace(), writableInstanceName,
			naming.ContainerDatabase, stdin, stdout, stderr, command...)
	}

	// Always attempt to create pgBackRest stanza first
	configHashMismatch, err := pgbackrest.Executor(exec).StanzaCreateOrUpgrade(ctx, configHash, postgresCluster)
	if err != nil {
		// record and log any errors resulting from running the stanza-create command
		r.Recorder.Event(postgresCluster, corev1.EventTypeWarning, EventUnableToCreateStanzas,
			err.Error())

		return false, errors.WithStack(err)
	}
	// Don't record event or return an error if configHashMismatch is true, since this just means
	// configuration changes in ConfigMaps/Secrets have not yet propagated to the container.
	// Therefore, just log an an info message and return an error to requeue and try again.
	if configHashMismatch {

		return true, nil
	}

	// record an event indicating successful stanza creation
	r.Recorder.Event(postgresCluster, corev1.EventTypeNormal, EventStanzasCreated,
		"pgBackRest stanza creation completed successfully")

	// if no errors then stanza(s) created successfully
	for i := range postgresCluster.Status.PGBackRest.Repos {
		postgresCluster.Status.PGBackRest.Repos[i].StanzaCreated = true
	}

	return false, nil
}

// getRepoHostStatus is responsible for returning the pgBackRest status for the
// provided pgBackRest repository host
func getRepoHostStatus(repoHost *appsv1.StatefulSet) *v1beta1.RepoHostStatus {

	repoHostStatus := &v1beta1.RepoHostStatus{}

	repoHostStatus.TypeMeta = repoHost.TypeMeta

	if repoHost.Status.ReadyReplicas > 0 {
		repoHostStatus.Ready = true
	} else {
		repoHostStatus.Ready = false
	}

	return repoHostStatus
}

// getRepoVolumeStatus is responsible for creating an array of repo statuses based on the
// existing/current status for any repos in the cluster, the repository volumes
// (i.e. PVCs) reconciled  for the cluster, and the hashes calculated for the configuration for any
// external repositories defined for the cluster.
func getRepoVolumeStatus(repoStatus []v1beta1.RepoStatus, repoVolumes []*corev1.PersistentVolumeClaim,
	configHashes map[string]string, replicaCreateRepoName string) []v1beta1.RepoStatus {

	// the new repository status that will be generated and returned
	updatedRepoStatus := []v1beta1.RepoStatus{}

	// Update the repo status based on the repo volumes (PVCs) that were reconciled.  This includes
	// updating the status for any existing repository volumes, and adding status for any new
	// repository volumes.
	for _, rv := range repoVolumes {
		newRepoVolStatus := true
		repoName := rv.Labels[naming.LabelPGBackRestRepo]
		for _, rs := range repoStatus {
			// treat as new status if contains properties of a cloud (s3, gcr or azure) repo
			if rs.Name == repoName && rs.RepoOptionsHash == "" {
				newRepoVolStatus = false

				// if we find a status with ReplicaCreateBackupComplete set to "true" but the repo name
				// for that status does not match the current replica create repo name, then reset
				// ReplicaCreateBackupComplete and StanzaCreate back to false
				if (rs.ReplicaCreateBackupComplete && (rs.Name != replicaCreateRepoName)) ||
					rs.RepoOptionsHash != "" {
					rs.ReplicaCreateBackupComplete = false
					rs.RepoOptionsHash = ""
				}

				// update binding info if needed
				if rs.Bound != (rv.Status.Phase == corev1.ClaimBound) {
					rs.Bound = (rv.Status.Phase == corev1.ClaimBound)
				}

				// if a different volume is detected, reset the stanza and replica create backup status
				// so that both are run again.
				if rs.VolumeName != "" && rs.VolumeName != rv.Spec.VolumeName {
					rs.StanzaCreated = false
					rs.ReplicaCreateBackupComplete = false
				}
				rs.VolumeName = rv.Spec.VolumeName

				updatedRepoStatus = append(updatedRepoStatus, rs)
				break
			}
		}
		if newRepoVolStatus {
			updatedRepoStatus = append(updatedRepoStatus, v1beta1.RepoStatus{
				Bound:      (rv.Status.Phase == corev1.ClaimBound),
				Name:       repoName,
				VolumeName: rv.Spec.VolumeName,
			})
		}
	}

	// Update the repo status based on the configuration hashes for any external repositories
	// configured for the cluster (e.g. Azure, GCS or S3 repositories).  This includes
	// updating the status for any existing external repositories, and adding status for any new
	// external repositories.
	for repoName, hash := range configHashes {
		newExtRepoStatus := true
		for _, rs := range repoStatus {
			// treat as new status if contains properties of a "volume" repo
			if rs.Name == repoName && !rs.Bound && rs.VolumeName == "" {
				newExtRepoStatus = false

				// if we find a status with ReplicaCreateBackupComplete set to "true" but the repo name
				// for that status does not match the current replica create repo name, then reset
				// ReplicaCreateBackupComplete back to false
				if rs.ReplicaCreateBackupComplete && (rs.Name != replicaCreateRepoName) {
					rs.ReplicaCreateBackupComplete = false
				}

				// Update the hash if needed. Setting StanzaCreated to "false" will force another
				// run of the  pgBackRest stanza-create command, while also setting
				// ReplicaCreateBackupComplete to false (this will result in a new replica creation
				// backup if this is the replica creation repo)
				if rs.RepoOptionsHash != hash {
					rs.RepoOptionsHash = hash
					rs.StanzaCreated = false
					rs.ReplicaCreateBackupComplete = false
				}

				updatedRepoStatus = append(updatedRepoStatus, rs)
				break
			}
		}
		if newExtRepoStatus {
			updatedRepoStatus = append(updatedRepoStatus, v1beta1.RepoStatus{
				Name:            repoName,
				RepoOptionsHash: hash,
			})
		}
	}

	// sort to ensure repo status always displays in a consistent order according to repo name
	sort.Slice(updatedRepoStatus, func(i, j int) bool {
		return updatedRepoStatus[i].Name < updatedRepoStatus[j].Name
	})

	return updatedRepoStatus
}

// reconcileScheduledBackups is responsible for reconciling pgBackRest backup
// schedules configured in the cluster definition
func (r *Reconciler) reconcileScheduledBackups(
	ctx context.Context, cluster *v1beta1.PostgresCluster, sa *corev1.ServiceAccount,
	cronjobs []*batchv1.CronJob,
) bool {
	log := logging.FromContext(ctx).WithValues("reconcileResource", "repoCronJob")
	// requeue if there is an error during creation
	var requeue bool

	for _, repo := range cluster.Spec.Backups.PGBackRest.Repos {
		// if the repo level backup schedules block has not been created,
		// there are no schedules defined
		if repo.BackupSchedules != nil {
			// next if the repo level schedule is not nil, create the CronJob.
			if repo.BackupSchedules.Full != nil {
				if err := r.reconcilePGBackRestCronJob(ctx, cluster, repo,
					full, repo.BackupSchedules.Full, sa, cronjobs); err != nil {
					log.Error(err, "unable to reconcile Full backup for "+repo.Name)
					requeue = true
				}
			}
			if repo.BackupSchedules.Differential != nil {
				if err := r.reconcilePGBackRestCronJob(ctx, cluster, repo,
					differential, repo.BackupSchedules.Differential, sa, cronjobs); err != nil {
					log.Error(err, "unable to reconcile Differential backup for "+repo.Name)
					requeue = true
				}
			}
			if repo.BackupSchedules.Incremental != nil {
				if err := r.reconcilePGBackRestCronJob(ctx, cluster, repo,
					incremental, repo.BackupSchedules.Incremental, sa, cronjobs); err != nil {
					log.Error(err, "unable to reconcile Incremental backup for "+repo.Name)
					requeue = true
				}
			}
		}
	}
	return requeue
}

// +kubebuilder:rbac:groups="batch",resources="cronjobs",verbs={create,patch}

// reconcilePGBackRestCronJob creates the CronJob for the given repo, pgBackRest
// backup type and schedule
func (r *Reconciler) reconcilePGBackRestCronJob(
	ctx context.Context, cluster *v1beta1.PostgresCluster, repo v1beta1.PGBackRestRepo,
	backupType string, schedule *string, serviceAccount *corev1.ServiceAccount,
	cronjobs []*batchv1.CronJob,
) error {

	log := logging.FromContext(ctx).WithValues("reconcileResource", "repoCronJob")

	annotations := naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil(),
		cluster.Spec.Backups.PGBackRest.Metadata.GetAnnotationsOrNil())
	labels := naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		cluster.Spec.Backups.PGBackRest.Metadata.GetLabelsOrNil(),
		naming.PGBackRestCronJobLabels(cluster.Name, repo.Name, backupType),
	)
	objectmeta := naming.PGBackRestCronJob(cluster, backupType, repo.Name)

	// Look for an existing CronJob by the associated Labels. If one exists,
	// update the ObjectMeta accordingly.
	for _, cronjob := range cronjobs {
		// ignore CronJobs that are terminating
		if cronjob.GetDeletionTimestamp() != nil {
			continue
		}

		if cronjob.GetLabels()[naming.LabelCluster] == cluster.Name &&
			cronjob.GetLabels()[naming.LabelPGBackRestCronJob] == backupType &&
			cronjob.GetLabels()[naming.LabelPGBackRestRepo] == repo.Name {
			objectmeta = metav1.ObjectMeta{
				Namespace: cluster.GetNamespace(),
				Name:      cronjob.Name,
			}
		}
	}

	objectmeta.Labels = labels
	objectmeta.Annotations = annotations

	// if the cluster isn't bootstrapped, return
	if !patroni.ClusterBootstrapped(cluster) {
		return nil
	}

	// Determine if the replica create backup is complete and return if not. This allows for proper
	// orchestration of backup Jobs since only one backup can be run at a time.
	condition := meta.FindStatusCondition(cluster.Status.Conditions,
		ConditionReplicaCreate)
	if condition == nil || condition.Status != metav1.ConditionTrue {
		return nil
	}

	// Verify that status exists for the repo configured for the scheduled backup, and that a stanza
	// has been created, before proceeding.  If either conditions are not true, then simply return
	// without requeuing and record and event (subsequent events, e.g. successful stanza creation,
	// writing of the proper repo status, adding a missing reop, etc. will trigger the reconciles
	// needed to try again).
	var statusFound, stanzaCreated bool
	for _, repoStatus := range cluster.Status.PGBackRest.Repos {
		if repoStatus.Name == repo.Name {
			statusFound = true
			stanzaCreated = repoStatus.StanzaCreated
		}
	}
	if !statusFound {
		r.Recorder.Eventf(cluster, corev1.EventTypeWarning, "InvalidBackupRepo",
			"Unable to find status for %q as configured for a scheduled backup.  Please ensure "+
				"this repo is defined in the spec.", repo.Name)
		return nil
	}
	if !stanzaCreated {
		r.Recorder.Eventf(cluster, corev1.EventTypeWarning, "StanzaNotCreated",
			"Stanza not created for %q as specified for a scheduled backup", repo.Name)
		return nil
	}

	// set backup type (i.e. "full", "diff", "incr")
	backupOpts := []string{"--type=" + backupType}

	jobSpec := generateBackupJobSpecIntent(cluster, repo,
		serviceAccount.GetName(), labels, annotations, backupOpts...)

	// Suspend cronjobs when shutdown or read-only. Any jobs that have already
	// started will continue.
	// - https://docs.k8s.io/reference/kubernetes-api/workload-resources/cron-job-v1beta1/#CronJobSpec
	suspend := (cluster.Spec.Shutdown != nil && *cluster.Spec.Shutdown) ||
		(cluster.Spec.Standby != nil && cluster.Spec.Standby.Enabled)

	pgBackRestCronJob := &batchv1.CronJob{
		ObjectMeta: objectmeta,
		Spec: batchv1.CronJobSpec{
			Schedule:          *schedule,
			Suspend:           &suspend,
			ConcurrencyPolicy: batchv1.ForbidConcurrent,
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: annotations,
					Labels:      labels,
				},
				Spec: *jobSpec,
			},
		},
	}

	// Set the image pull secrets, if any exist.
	// This is set here rather than using the service account due to the lack
	// of propagation to existing pods when the CRD is updated:
	// https://github.com/kubernetes/kubernetes/issues/88456
	pgBackRestCronJob.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets =
		cluster.Spec.ImagePullSecrets

	// set metadata
	pgBackRestCronJob.SetGroupVersionKind(batchv1.SchemeGroupVersion.WithKind("CronJob"))
	err := errors.WithStack(r.setControllerReference(cluster, pgBackRestCronJob))

	if err == nil {
		err = r.apply(ctx, pgBackRestCronJob)
	}
	if err != nil {
		// record and log any errors resulting from trying to create the pgBackRest backup CronJob
		r.Recorder.Event(cluster, corev1.EventTypeWarning, EventUnableToCreatePGBackRestCronJob,
			err.Error())
		log.Error(err, "error when attempting to create pgBackRest CronJob")
	}
	return err
}

// BackupsEnabled checks the state of the backups (i.e., if backups are in the spec,
// if a repo-host StatefulSet exists, if the annotation permitting backup deletion exists)
// and determines whether reconciliation is allowed.
// Reconciliation of backup-related Kubernetes objects is paused if
//   - a user created a cluster with backups;
//   - the cluster is updated to remove backups;
//   - the annotation authorizing that removal is missing.
//
// This function also returns whether the spec has a defined backups or not.
func (r *Reconciler) BackupsEnabled(
	ctx context.Context,
	postgresCluster *v1beta1.PostgresCluster,
) (
	backupsSpecFound bool,
	backupsReconciliationAllowed bool,
	err error,
) {
	specFound, stsNotFound, annotationFound, err := r.ObserveBackupUniverse(ctx, postgresCluster)

	switch {
	case err != nil:
	case specFound:
		backupsSpecFound = true
		backupsReconciliationAllowed = true
	case annotationFound || stsNotFound:
		backupsReconciliationAllowed = true
	case !annotationFound && !stsNotFound:
		// Destroying backups is a two key operation:
		// 1. You must remove the backups section of the spec.
		// 2. You must apply an annotation to the cluster.
		// The existence of a StatefulSet without the backups spec is
		// evidence of key 1 being turned without key 2 being turned
		// -- block reconciliation until the annotation is added.
		backupsReconciliationAllowed = false
	default:
		backupsReconciliationAllowed = false
	}
	return backupsSpecFound, backupsReconciliationAllowed, err
}

// ObserveBackupUniverse returns
//   - whether the spec has backups defined;
//   - whether the repo-host statefulset exists;
//   - whether the cluster has the annotation authorizing backup removal.
func (r *Reconciler) ObserveBackupUniverse(ctx context.Context,
	postgresCluster *v1beta1.PostgresCluster,
) (
	backupsSpecFound bool,
	repoHostStatefulSetNotFound bool,
	backupsRemovalAnnotationFound bool,
	err error,
) {

	// Does the cluster have a blank Backups section
	backupsSpecFound = !reflect.DeepEqual(postgresCluster.Spec.Backups, v1beta1.Backups{PGBackRest: v1beta1.PGBackRestArchive{}})

	// Does the repo-host StatefulSet exist?
	name := fmt.Sprintf("%s-%s", postgresCluster.GetName(), "repo-host")
	existing := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: postgresCluster.Namespace,
			Name:      name,
		},
	}
	err = errors.WithStack(
		r.Client.Get(ctx, client.ObjectKeyFromObject(existing), existing))
	repoHostStatefulSetNotFound = apierrors.IsNotFound(err)

	// If we have an error that is not related to a missing repo-host StatefulSet,
	// we return an error and expect the calling function to correctly stop processing.
	if err != nil && !repoHostStatefulSetNotFound {
		return true, false, false, err
	}

	backupsRemovalAnnotationFound = authorizeBackupRemovalAnnotationPresent(postgresCluster)

	// If we have reached this point, the err is either nil or an IsNotFound error
	// which we do not care about; hence, pass nil rather than the err
	return backupsSpecFound, repoHostStatefulSetNotFound, backupsRemovalAnnotationFound, nil
}

func authorizeBackupRemovalAnnotationPresent(postgresCluster *v1beta1.PostgresCluster) bool {
	annotations := postgresCluster.GetAnnotations()
	for annotation := range annotations {
		if annotation == naming.AuthorizeBackupRemovalAnnotation {
			return annotations[naming.AuthorizeBackupRemovalAnnotation] == "true"
		}
	}
	return false
}
