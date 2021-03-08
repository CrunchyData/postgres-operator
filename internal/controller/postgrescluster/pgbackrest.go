package postgrescluster

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

import (
	"context"
	"fmt"
	"sort"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/pgbackrest"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
)

const (
	// ConditionRepoHostReady is the type used in a condition to indicate whether or not a
	// pgBackRest repository host PostgresCluster is ready
	ConditionRepoHostReady = "PGBackRestRepoHostReady"

	// EventRepoHostNotFound is used to indicate that a pgBackRest repository was not
	// found when reconciling
	EventRepoHostNotFound = "RepoDeploymentNotFound"

	// EventRepoHostCreated is the event reason utilized when a pgBackRest repository host is
	// created
	EventRepoHostCreated = "RepoHostCreated"
)

// RepoResources is used to store various resources for pgBackRest repositories
type RepoResources struct {
	hosts     []*appsv1.StatefulSet
	pvcs      []*v1.PersistentVolumeClaim
	sshConfig *v1.ConfigMap
	sshSecret *v1.Secret
}

// applyRepoHostIntent ensures the pgBackRest repository host StatefulSet is synchronized with the
// proper configuration according to the provided PostgresCluster custom resource.  This is done by
// applying the PostgresCluster controller's fully specified intent for the repository host
// StatefulSet.  Any changes to the deployment spec as a result of synchronization will result in a
// rollout of the pgBackRest repository host StatefulSet in accordance with its configured
// strategy.
func (r *Reconciler) applyRepoHostIntent(ctx context.Context, postgresCluster *v1alpha1.PostgresCluster,
	repoHostName string) (*appsv1.StatefulSet, error) {

	repo, err := r.generateRepoHostIntent(postgresCluster, repoHostName)
	if err != nil {
		return nil, err
	}

	if err := r.apply(ctx, repo); err != nil {
		return nil, err
	}

	return repo, nil
}

// applyRepoVolumeIntent ensures the pgBackRest repository host deployment is synchronized with the
// proper configuration according to the provided PostgresCluster custom resource.  This is done by
// applying the PostgresCluster controller's fully specified intent for the PersistentVolumeClaim
// representing a repository.
func (r *Reconciler) applyRepoVolumeIntent(ctx context.Context,
	postgresCluster *v1alpha1.PostgresCluster, spec *v1.PersistentVolumeClaimSpec,
	repoName string) (*v1.PersistentVolumeClaim, error) {

	repo, err := r.generateRepoVolumeIntent(postgresCluster, spec, repoName)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if err := r.apply(ctx, repo); err != nil {
		return nil, errors.WithStack(err)
	}

	return repo, nil
}

// calculatePGBackRestConditions is responsible for calculating any pgBackRest conditions
// based on the current pgBackRest status, and then updating the conditions array within
// the PostgresCluster status with those conditions as needed.
func calculatePGBackRestConditions(status *v1alpha1.PGBackRestStatus,
	postgresClusterGeneration int64, dedicatedEnabled bool) []metav1.Condition {

	conditions := []metav1.Condition{}

	// right now only a single condition exists for repo hosts
	if !dedicatedEnabled {
		return conditions
	}

	repoHostReady := metav1.Condition{
		ObservedGeneration: postgresClusterGeneration,
		Type:               ConditionRepoHostReady,
	}
	if status.RepoHost == nil {
		repoHostReady.Status = metav1.ConditionUnknown
		repoHostReady.Reason = "RepoHostStatusMissing"
		repoHostReady.Message = "pgBackRest dedicated repository host status is missing"
	} else if status.RepoHost.Ready {
		repoHostReady.Status = metav1.ConditionTrue
		repoHostReady.Reason = "RepoHostReady"
		repoHostReady.Message = "pgBackRest dedicated repository host is ready"
	} else {
		repoHostReady.Status = metav1.ConditionFalse
		repoHostReady.Reason = "RepoHostNotReady"
		repoHostReady.Message = "pgBackRest dedicated repository host is not ready"
	}

	return append(conditions, repoHostReady)
}

// getPGBackRestResources returns the existing pgBackRest resources that should utilized by the
// PostgresCluster controller during reconciliation.  Any items returned are verified to be owned
// by the PostgresCluster controller and still applicable per the current PostgresCluster spec.
// Additionally, and resources identified that no longer correspond to any current configuration
// are deleted.
func (r *Reconciler) getPGBackRestResources(ctx context.Context,
	postgresCluster *v1alpha1.PostgresCluster) (*RepoResources, error) {

	repoResources := &RepoResources{}

	gvks := []schema.GroupVersionKind{{
		Group:   v1.SchemeGroupVersion.Group,
		Version: v1.SchemeGroupVersion.Version,
		Kind:    "ConfigMapList",
	}, {
		Group:   v1.SchemeGroupVersion.Group,
		Version: v1.SchemeGroupVersion.Version,
		Kind:    "PersistentVolumeClaimList",
	}, {
		Group:   v1.SchemeGroupVersion.Group,
		Version: v1.SchemeGroupVersion.Version,
		Kind:    "SecretList",
	}, {
		Group:   appsv1.SchemeGroupVersion.Group,
		Version: appsv1.SchemeGroupVersion.Version,
		Kind:    "StatefulSetList",
	}}

	selector := naming.PGBackRestSelector(postgresCluster.GetName())
	for _, gvk := range gvks {
		uList := &unstructured.UnstructuredList{}
		uList.SetGroupVersionKind(gvk)
		if err := r.Client.List(context.Background(), uList,
			client.InNamespace(postgresCluster.GetNamespace()),
			client.MatchingLabelsSelector{Selector: selector}); err != nil {
			return nil, errors.WithStack(err)
		}
		if len(uList.Items) == 0 {
			continue
		}

		owned := []unstructured.Unstructured{}
		for i, u := range uList.Items {
			if metav1.IsControlledBy(&uList.Items[i], postgresCluster) {
				owned = append(owned, u)
			}
		}

		owned = r.cleanupRepoResources(ctx, postgresCluster, owned)
		uList.Items = owned
		if err := unstructuredToRepoResources(postgresCluster, gvk.Kind,
			repoResources, uList); err != nil {
			return nil, errors.WithStack(err)
		}
	}

	return repoResources, nil
}

// cleanupRepoResources cleans up pgBackRest repository resources that should no longer be
// reconciled by deleting them.  This includes deleting repos (i.e. PersistentVolumeClaims) that
// are no longer associated with any repository configured within the PostgresCluster spec, or any
// pgBackRest repository host resources if a repository host is no longer configured.
func (r *Reconciler) cleanupRepoResources(ctx context.Context,
	postgresCluster *v1alpha1.PostgresCluster,
	owned []unstructured.Unstructured) []unstructured.Unstructured {

	log := logging.FromContext(ctx)

	ownedNoDelete := []unstructured.Unstructured{}
	for i, ownedRepo := range owned {
		var found bool

		_, isRepoVolume := ownedRepo.GetLabels()[naming.LabelPGBackRestRepoVolume]
		if isRepoVolume {
			for _, repo := range postgresCluster.Spec.Archive.PGBackRest.Repos {
				if repo.Name == ownedRepo.GetLabels()[naming.LabelPGBackRestRepo] {
					found = true
					ownedNoDelete = append(ownedNoDelete, ownedRepo)
					break
				}
			}
		} else {
			_, isRepoHost := ownedRepo.GetLabels()[naming.LabelPGBackRestRepoHost]
			repoHostEnabled := pgbackrest.RepoHostEnabled(postgresCluster)
			_, isDedicatedRepoHost := ownedRepo.GetLabels()[naming.LabelPGBackRestDedicated]
			dedicatedRepoEnabled := pgbackrest.DedicatedRepoHostEnabled(postgresCluster)
			if (!isDedicatedRepoHost && isRepoHost && repoHostEnabled) ||
				(isDedicatedRepoHost && dedicatedRepoEnabled) {
				found = true
				ownedNoDelete = append(ownedNoDelete, ownedRepo)
			}
		}

		if !found {
			if err := r.Client.Delete(ctx, &owned[i]); err != nil {
				log.Error(err, "deleting resource during cleanup attempt")
			}
		}
	}

	return ownedNoDelete
}

// unstructuredToRepoResources converts unstructred pgBackRest repository resources (specifically
// unstructured StatefulSetLists and PersistentVolumeClaimList) into their structured equivalent.
func unstructuredToRepoResources(postgresCluster *v1alpha1.PostgresCluster, kind string,
	repoResources *RepoResources, uList *unstructured.UnstructuredList) error {

	switch kind {
	case "ConfigMapList":
		var cmList v1.ConfigMapList
		if err := runtime.DefaultUnstructuredConverter.
			FromUnstructured(uList.UnstructuredContent(), &cmList); err != nil {
			return errors.WithStack(err)
		}
		// we only care about ConfigMaps with the proper names
		for i, cm := range cmList.Items {
			if cm.GetName() == naming.PGBackRestSSHConfig(postgresCluster).Name {
				repoResources.sshConfig = &cmList.Items[i]
				break
			}
		}
	case "PersistentVolumeClaimList":
		var pvcList v1.PersistentVolumeClaimList
		if err := runtime.DefaultUnstructuredConverter.
			FromUnstructured(uList.UnstructuredContent(), &pvcList); err != nil {
			return errors.WithStack(err)
		}
		for i := range pvcList.Items {
			repoResources.pvcs = append(repoResources.pvcs, &pvcList.Items[i])
		}
	case "SecretList":
		var secretList v1.SecretList
		if err := runtime.DefaultUnstructuredConverter.
			FromUnstructured(uList.UnstructuredContent(), &secretList); err != nil {
			return errors.WithStack(err)
		}
		// we only care about Secret with the proper names
		for i, secret := range secretList.Items {
			if secret.GetName() == naming.PGBackRestSSHSecret(postgresCluster).Name {
				repoResources.sshSecret = &secretList.Items[i]
				break
			}
		}
	case "StatefulSetList":
		var stsList appsv1.StatefulSetList
		if err := runtime.DefaultUnstructuredConverter.
			FromUnstructured(uList.UnstructuredContent(), &stsList); err != nil {
			return errors.WithStack(err)
		}
		for i := range stsList.Items {
			repoResources.hosts = append(repoResources.hosts, &stsList.Items[i])
		}
	default:
		return fmt.Errorf("unexpected kind %q", kind)
	}

	return nil
}

// generateRepoHostIntent creates and populates StatefulSet with the PostgresCluster's full intent
// as needed to create and reconcile a pgBackRest dedicated repository host within the kubernetes
// cluster.
func (r *Reconciler) generateRepoHostIntent(postgresCluster *v1alpha1.PostgresCluster,
	repoHostName string) (*appsv1.StatefulSet, error) {

	labels := naming.PGBackRestDedicatedLabels(postgresCluster.GetName())

	repo := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      repoHostName,
			Namespace: postgresCluster.GetNamespace(),
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			ServiceName: naming.ClusterPodService(postgresCluster).Name,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
			},
		},
	}

	podSecurityContext := &v1.PodSecurityContext{SupplementalGroups: []int64{65534}}
	// set fsGroups if not OpenShift
	if postgresCluster.Spec.OpenShift == nil || !*postgresCluster.Spec.OpenShift {
		fsGroup := int64(26)
		podSecurityContext.FSGroup = &fsGroup
	}
	repo.Spec.Template.Spec.SecurityContext = podSecurityContext

	// add ssh pod info
	if err := pgbackrest.AddSSHToPod(postgresCluster, &repo.Spec.Template); err != nil {
		return nil, errors.WithStack(err)
	}
	if err := pgbackrest.AddRepoVolumesToPod(postgresCluster, &repo.Spec.Template,
		naming.PGBackRestRepoContainerName); err != nil {
		return nil, errors.WithStack(err)
	}
	// add configs to pod
	if err := pgbackrest.AddConfigsToPod(postgresCluster, &repo.Spec.Template,
		pgbackrest.CMRepoKey, naming.PGBackRestRepoContainerName); err != nil {
		return nil, errors.WithStack(err)
	}
	addTMPEmptyDir(&repo.Spec.Template)

	// set ownership references
	if err := controllerutil.SetControllerReference(postgresCluster, repo,
		r.Client.Scheme()); err != nil {
		return nil, err
	}

	return repo, nil
}

func (r *Reconciler) generateRepoVolumeIntent(postgresCluster *v1alpha1.PostgresCluster,
	spec *v1.PersistentVolumeClaimSpec, repoName string) (*v1.PersistentVolumeClaim, error) {

	labels := naming.PGBackRestRepoVolumeLabels(postgresCluster.GetName(), repoName)

	// generate metadata
	meta := naming.PGBackRestRepoVolume(postgresCluster, repoName)
	meta.Labels = labels

	repoVol := &v1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       "PersistentVolumeClaim",
		},
		ObjectMeta: meta,
		Spec:       *spec,
	}

	// set ownership references
	if err := controllerutil.SetControllerReference(postgresCluster, repoVol,
		r.Client.Scheme()); err != nil {
		return nil, err
	}

	return repoVol, nil
}

// reconcilePGBackRest is responsible for reconciling any/all pgBackRest resources owned by a
// specific PostgresCluster (e.g. Deployments, ConfigMaps, Secrets, etc.).  This function will
// ensure various reconciliation logic is run as needed for each pgBackRest resource, while then
// also generating the proper Result as needed to ensure proper event requeuing according to
// the results of any attempts to properly reconcile these resources.
func (r *Reconciler) reconcilePGBackRest(ctx context.Context,
	postgresCluster *v1alpha1.PostgresCluster, instanceNames []string) (reconcile.Result, error) {

	// add some additional context about what component is being reconciled
	log := logging.FromContext(ctx).WithValues("reconciler", "pgBackRest")

	// create the pgBackRest status that will be updated when reconciling various pgBackRest
	// resources
	postgresCluster.Status.PGBackRest = new(v1alpha1.PGBackRestStatus)
	pgBackRestStatus := postgresCluster.Status.PGBackRest

	repoResources, err := r.getPGBackRestResources(ctx, postgresCluster)
	if err != nil {
		log.Error(err, "unable to get repository hosts for PostgresCluster")
		return reconcile.Result{
			Requeue: true,
		}, err
	}

	var repoHost *appsv1.StatefulSet
	var repoHostName string
	dedicatedEnabled := (postgresCluster.Spec.Archive.PGBackRest.RepoHost != nil) &&
		(postgresCluster.Spec.Archive.PGBackRest.RepoHost.Dedicated != nil)
	if dedicatedEnabled {
		// reconcile the pgbackrest repository host
		repoHost, err = r.reconcileDedicatedRepoHost(ctx, postgresCluster, pgBackRestStatus,
			repoResources)
		if err != nil {
			log.Error(err, "unable to reconcile pgBackRest repo host")
			return reconcile.Result{
				Requeue: true,
			}, err
		}
		repoHostName = repoHost.GetName()
	}

	// reconcile all pgbackrest repository volumes
	if err := r.reconcileRepoVolumes(ctx, postgresCluster, pgBackRestStatus); err != nil {
		log.Error(err, "unable to reconcile pgBackRest repo host")
		return reconcile.Result{
			Requeue: true,
		}, err
	}

	// reconcile all pgbackrest configuration and secrets
	if err := r.reconcilePGBackRestConfig(ctx, postgresCluster, repoHostName, instanceNames,
		repoResources.sshSecret); err != nil {
		log.Error(err, "unable to reconcile pgBackRest configuration")
		return reconcile.Result{
			Requeue: true,
		}, err
	}

	// remove the dedicated repo host status if a dedicated host is not enabled
	if len(postgresCluster.Status.Conditions) > 0 && !dedicatedEnabled {
		meta.RemoveStatusCondition(&postgresCluster.Status.Conditions, ConditionRepoHostReady)
	}
	// ensure conditions are updated before returning
	setStatusConditions(&postgresCluster.Status, calculatePGBackRestConditions(
		pgBackRestStatus, postgresCluster.GetGeneration(), dedicatedEnabled)...)

	return reconcile.Result{}, nil
}

// reconcileRepoHosts is responsible for reconciling the pgBackRest ConfigMaps and Secrets.
func (r *Reconciler) reconcilePGBackRestConfig(ctx context.Context,
	postgresCluster *v1alpha1.PostgresCluster, repoHostName string,
	instanceNames []string, sshSecret *v1.Secret) error {

	log := logging.FromContext(ctx).WithValues("reconcileResource", "repoConfig")
	errMsg := "reconciling pgBackRest configuration"

	backrestConfig := pgbackrest.CreatePGBackRestConfigMapIntent(postgresCluster, repoHostName,
		instanceNames)
	if err := r.apply(ctx, backrestConfig); err != nil {
		return errors.WithStack(err)
	}

	repoHostConfigured := (postgresCluster.Spec.Archive.PGBackRest.RepoHost != nil)

	if !repoHostConfigured {
		log.V(1).Info("skipping SSH reconciliation, no repo hosts configured")
		return nil
	}

	sshdConfig := pgbackrest.CreateSSHConfigMapIntent(postgresCluster)
	// set ownership references
	if err := controllerutil.SetControllerReference(postgresCluster, &sshdConfig,
		r.Client.Scheme()); err != nil {
		log.Error(err, errMsg)
		return err
	}
	if err := r.apply(ctx, &sshdConfig); err != nil {
		log.Error(err, errMsg)
		return err
	}

	sshdSecret, err := pgbackrest.CreateSSHSecretIntent(postgresCluster, sshSecret)
	if err != nil {
		log.Error(err, errMsg)
		return err
	}
	if err := controllerutil.SetControllerReference(postgresCluster, &sshdSecret,
		r.Client.Scheme()); err != nil {
		return err
	}
	if err := r.apply(ctx, &sshdSecret); err != nil {
		log.Error(err, errMsg)
		return err
	}

	return nil
}

// reconcileDedicatedRepoHost is responsible for reconciling a pgBackRest dedicated repository host
// StatefulSet according to a specific PostgresCluster custom resource.
func (r *Reconciler) reconcileDedicatedRepoHost(ctx context.Context,
	postgresCluster *v1alpha1.PostgresCluster, status *v1alpha1.PGBackRestStatus,
	repoResources *RepoResources) (*appsv1.StatefulSet, error) {

	log := logging.FromContext(ctx).WithValues("reconcileResource", "repoHost")

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
	repoHost, err := r.applyRepoHostIntent(ctx, postgresCluster, repoHostName)
	if err != nil {
		log.Error(err, "reconciling repository host")
		return nil, err
	}

	status.RepoHost = getRepoHostStatus(repoHost)

	if isCreate {
		r.Recorder.Eventf(postgresCluster, v1.EventTypeNormal, EventRepoHostCreated,
			"created pgBackRest repository host %s/%s", repoHost.TypeMeta.Kind, repoHostName)
	}

	return repoHost, nil
}

func (r *Reconciler) reconcileRepoVolumes(ctx context.Context,
	postgresCluster *v1alpha1.PostgresCluster,
	status *v1alpha1.PGBackRestStatus) error {

	log := logging.FromContext(ctx).WithValues("reconcileResource", "repoVolume")

	errors := []error{}
	errMsg := "reconciling repository volume"
	repos := []*v1.PersistentVolumeClaim{}
	for _, repoVol := range postgresCluster.Spec.Archive.PGBackRest.Repos {
		repo, err := r.applyRepoVolumeIntent(ctx, postgresCluster, &repoVol.VolumeClaimSpec,
			repoVol.Name)
		if err != nil {
			log.Error(err, errMsg)
			errors = append(errors, err)
			continue
		}
		repos = append(repos, repo)
	}

	if len(errors) > 0 {
		return utilerrors.NewAggregate(errors)
	}

	status.Repos = getRepoVolumeStatus(repos...)

	return nil
}

// getRepoHostStatus is responsible for returning the pgBackRest status for the provided pgBackRest
// repository host
func getRepoHostStatus(repoHost *appsv1.StatefulSet) *v1alpha1.RepoHostStatus {

	repoHostStatus := &v1alpha1.RepoHostStatus{}

	repoHostStatus.TypeMeta = repoHost.TypeMeta

	if repoHost.Status.ReadyReplicas == *repoHost.Spec.Replicas {
		repoHostStatus.Ready = true
	} else {
		repoHostStatus.Ready = false
	}

	return repoHostStatus
}

// getRepoVolumeStatus is responsible for updating the pgBackRest status for the provided
// pgBackRest repository volume
func getRepoVolumeStatus(repoVolumes ...*v1.PersistentVolumeClaim) []v1alpha1.RepoVolumeStatus {

	repoVolStatus := []v1alpha1.RepoVolumeStatus{}
	for _, repoVol := range repoVolumes {
		repoVolStatus = append(repoVolStatus, v1alpha1.RepoVolumeStatus{
			Bound:      (repoVol.Status.Phase == v1.ClaimBound),
			Name:       repoVol.GetName(),
			VolumeName: repoVol.Spec.VolumeName,
		})
	}

	return repoVolStatus
}
