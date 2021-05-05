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
	"io"
	"sort"
	"time"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/patroni"
	"github.com/crunchydata/postgres-operator/internal/pgbackrest"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
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

	// EventUnableToCreateStanzas is the event reason utilized when pgBackRest is unable to create
	// stanzas for the repositories in a PostgreSQL cluster
	EventUnableToCreateStanzas = "UnableToCreateStanzas"

	// EventStanzasCreated is the event reason utilized when a pgBackRest stanza create command
	// completes successfully
	EventStanzasCreated = "StanzasCreated"
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
func (r *Reconciler) applyRepoHostIntent(ctx context.Context, postgresCluster *v1beta1.PostgresCluster,
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
	postgresCluster *v1beta1.PostgresCluster, spec *v1.PersistentVolumeClaimSpec,
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
func calculatePGBackRestConditions(status *v1beta1.PGBackRestStatus,
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
	postgresCluster *v1beta1.PostgresCluster) (*RepoResources, error) {

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

// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=delete

// cleanupRepoResources cleans up pgBackRest repository resources that should no longer be
// reconciled by deleting them.  This includes deleting repos (i.e. PersistentVolumeClaims) that
// are no longer associated with any repository configured within the PostgresCluster spec, or any
// pgBackRest repository host resources if a repository host is no longer configured.
func (r *Reconciler) cleanupRepoResources(ctx context.Context,
	postgresCluster *v1beta1.PostgresCluster,
	owned []unstructured.Unstructured) []unstructured.Unstructured {

	log := logging.FromContext(ctx)

	ownedNoDelete := []unstructured.Unstructured{}
	for i, ownedRepo := range owned {
		var found bool

		_, isRepoVolume := ownedRepo.GetLabels()[naming.LabelPGBackRestRepoVolume]
		if isRepoVolume {
			for _, repo := range postgresCluster.Spec.Archive.PGBackRest.Repos {
				// we only care about cleaning up local repo volumes (PVCs), and ignore other repo
				// types (e.g. for external Azure, GCS or S3 repositories)
				if repo.Volume != nil &&
					(repo.Name == ownedRepo.GetLabels()[naming.LabelPGBackRestRepo]) {
					found = true
					ownedNoDelete = append(ownedNoDelete, ownedRepo)
					break
				}
			}
		} else {
			_, isRepoHost := ownedRepo.GetLabels()[naming.LabelPGBackRestRepoHost]
			repoHostEnabled := pgbackrest.RepoHostEnabled(postgresCluster)
			_, isPGBackRestConfig := ownedRepo.GetLabels()[naming.LabelPGBackRestConfig]
			_, isDedicatedRepoHost := ownedRepo.GetLabels()[naming.LabelPGBackRestDedicated]
			dedicatedRepoEnabled := pgbackrest.DedicatedRepoHostEnabled(postgresCluster)
			if (!isDedicatedRepoHost && isRepoHost && repoHostEnabled) ||
				(isDedicatedRepoHost && dedicatedRepoEnabled) || isPGBackRestConfig {
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
func unstructuredToRepoResources(postgresCluster *v1beta1.PostgresCluster, kind string,
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
func (r *Reconciler) generateRepoHostIntent(postgresCluster *v1beta1.PostgresCluster,
	repoHostName string) (*appsv1.StatefulSet, error) {

	labels := naming.Merge(postgresCluster.Spec.Metadata.Labels,
		postgresCluster.Spec.Archive.PGBackRest.Metadata.Labels,
		naming.PGBackRestDedicatedLabels(postgresCluster.GetName()),
	)

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
				MatchLabels: naming.PGBackRestDedicatedLabels(postgresCluster.GetName()),
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
		podSecurityContext.FSGroup = initialize.Int64(26)
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

	// add nss_wrapper init container and add nss_wrapper env vars to the pgbackrest
	// container
	addNSSWrapper(postgresCluster, &repo.Spec.Template)
	addTMPEmptyDir(&repo.Spec.Template)

	// set ownership references
	if err := controllerutil.SetControllerReference(postgresCluster, repo,
		r.Client.Scheme()); err != nil {
		return nil, err
	}

	return repo, nil
}

func (r *Reconciler) generateRepoVolumeIntent(postgresCluster *v1beta1.PostgresCluster,
	spec *v1.PersistentVolumeClaimSpec, repoName string) (*v1.PersistentVolumeClaim, error) {

	labels := naming.Merge(postgresCluster.Spec.Metadata.Labels,
		postgresCluster.Spec.Archive.PGBackRest.Metadata.Labels,
		naming.PGBackRestRepoVolumeLabels(postgresCluster.GetName(), repoName),
	)

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
	postgresCluster *v1beta1.PostgresCluster, instanceNames []string) (reconcile.Result, error) {

	// add some additional context about what component is being reconciled
	log := logging.FromContext(ctx).WithValues("reconciler", "pgBackRest")

	// if nil, create the pgBackRest status that will be updated when reconciling various
	// pgBackRest resources
	if postgresCluster.Status.PGBackRest == nil {
		postgresCluster.Status.PGBackRest = &v1beta1.PGBackRestStatus{}
	}

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
		repoHost, err = r.reconcileDedicatedRepoHost(ctx, postgresCluster, repoResources)
		if err != nil {
			log.Error(err, "unable to reconcile pgBackRest repo host")
			return reconcile.Result{
				Requeue: true,
			}, err
		}
		repoHostName = repoHost.GetName()
	}

	// calculate hashes for the external repository configurations in the spec (e.g. for Azure,
	// GCS and/or S3 repositories) as needed to properly detect changes to external repository
	// configuration (and then execute stanza create commands accordingly)
	configHashes, configHash, err := pgbackrest.CalculateConfigHashes(postgresCluster)
	if err != nil {
		log.Error(err, "unable to calculate config hashes")
		return reconcile.Result{
			Requeue: true,
		}, err
	}

	// reconcile all pgbackrest repository volumes
	if err := r.reconcileRepoVolumes(ctx, postgresCluster, configHashes); err != nil {
		log.Error(err, "unable to reconcile pgBackRest repo host")
		return reconcile.Result{
			Requeue: true,
		}, err
	}

	// reconcile all pgbackrest configuration and secrets
	if err := r.reconcilePGBackRestConfig(ctx, postgresCluster, repoHostName,
		configHash, instanceNames, repoResources.sshSecret); err != nil {
		log.Error(err, "unable to reconcile pgBackRest configuration")
		return reconcile.Result{
			Requeue: true,
		}, err
	}

	// reconcile all pgbackrest configuration and secrets
	configHashMismatch, err := r.reconcileStanzaCreate(ctx, postgresCluster, configHash)
	// If a stanza create error then requeue but don't return the error.  This prevents
	// stanza-create errors from bubbling up to the main Reconcile() function, which would
	// prevent subsequent reconciles from occurring.  Also, this provides a better chance
	// that the pgBackRest status will be updated at the end of the Reconcile() function,
	// e.g. to set the "stanzaCreated" indicator to false for any repos failing stanza creation
	// (assuming no other reconcile errors bubble up to the Reconcile() function and block the
	// status update).  And finally, add some time to each requeue to slow down subsequent
	// stanza create attempts in order to prevent pgBackRest mis-configuration (e.g. due to
	// custom confiugration) from spamming the logs, while also ensuring stanza creation is
	// re-attempted until successful (e.g. allowing users to correct mis-configurations in
	// custom configuration and ensure stanzas are still created).
	if err != nil {
		log.Error(err, "unable to create stanza")
		return reconcile.Result{
			RequeueAfter: 10 * time.Second,
		}, nil
	}
	// If a config hash mismatch, then log an info message and requeue to try again.  Add some time
	// to the requeue to give the pgBackRest configuration changes a chance to propagate to the
	// container.
	if configHashMismatch {
		log.Info("pgBackRest config hash mismatch detected, requeuing to reattempt stanza create")
		return reconcile.Result{
			RequeueAfter: 10 * time.Second,
		}, nil
	}

	// remove the dedicated repo host status if a dedicated host is not enabled
	if len(postgresCluster.Status.Conditions) > 0 && !dedicatedEnabled {
		meta.RemoveStatusCondition(&postgresCluster.Status.Conditions, ConditionRepoHostReady)
	}
	// ensure conditions are updated before returning
	setStatusConditions(&postgresCluster.Status, calculatePGBackRestConditions(
		postgresCluster.Status.PGBackRest, postgresCluster.GetGeneration(), dedicatedEnabled)...)

	return reconcile.Result{}, nil
}

// reconcileRepoHosts is responsible for reconciling the pgBackRest ConfigMaps and Secrets.
func (r *Reconciler) reconcilePGBackRestConfig(ctx context.Context,
	postgresCluster *v1beta1.PostgresCluster, repoHostName, configHash string,
	instanceNames []string, sshSecret *v1.Secret) error {

	log := logging.FromContext(ctx).WithValues("reconcileResource", "repoConfig")
	errMsg := "reconciling pgBackRest configuration"

	backrestConfig := pgbackrest.CreatePGBackRestConfigMapIntent(postgresCluster, repoHostName,
		configHash, instanceNames)
	if err := controllerutil.SetControllerReference(postgresCluster, backrestConfig,
		r.Client.Scheme()); err != nil {
		return err
	}
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
	postgresCluster *v1beta1.PostgresCluster,
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

	postgresCluster.Status.PGBackRest.RepoHost = getRepoHostStatus(repoHost)

	if isCreate {
		r.Recorder.Eventf(postgresCluster, v1.EventTypeNormal, EventRepoHostCreated,
			"created pgBackRest repository host %s/%s", repoHost.TypeMeta.Kind, repoHostName)
	}

	return repoHost, nil
}

// reconcileRepoVolumes is responsible for reconciling any pgBackRest repository volumes
// (i.e. PVCs) configured for the cluster
func (r *Reconciler) reconcileRepoVolumes(ctx context.Context,
	postgresCluster *v1beta1.PostgresCluster, extConfigHashes map[string]string) error {

	log := logging.FromContext(ctx).WithValues("reconcileResource", "repoVolume")

	errors := []error{}
	errMsg := "reconciling repository volume"
	repoVols := []*v1.PersistentVolumeClaim{}
	for _, repo := range postgresCluster.Spec.Archive.PGBackRest.Repos {
		// we only care about reconciling repo volumes, so ignore everything else
		if repo.Volume == nil {
			continue
		}
		repo, err := r.applyRepoVolumeIntent(ctx, postgresCluster, &repo.Volume.VolumeClaimSpec,
			repo.Name)
		if err != nil {
			log.Error(err, errMsg)
			errors = append(errors, err)
			continue
		}
		repoVols = append(repoVols, repo)
	}

	postgresCluster.Status.PGBackRest.Repos =
		getRepoVolumeStatus(postgresCluster.Status.PGBackRest.Repos, repoVols, extConfigHashes)

	if len(errors) > 0 {
		return utilerrors.NewAggregate(errors)
	}

	return nil
}

// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list
// +kubebuilder:rbac:groups="",resources=pods/exec,verbs=create

// reconcileStanzaCreate is responsible for ensuring stanzas are properly created for the
// pgBackRest repositories configured for a PostgresCluster.  If the bool returned from this
// function is false, this indicates that a pgBackRest config hash mismatch was identified that
// prevented the "pgbackrest stanza-create" command from running (with a config has mitmatch
// indicating that pgBackRest configuration as stored in the pgBackRest ConfigMap has not yet
// propagated to the Pod).
func (r *Reconciler) reconcileStanzaCreate(ctx context.Context,
	postgresCluster *v1beta1.PostgresCluster, configHash string) (bool, error) {

	clusterName := postgresCluster.GetName()
	namespace := postgresCluster.GetNamespace()

	// determine if the cluster has been initialized
	clusterBootstrapped := patroni.ClusterBootstrapped(postgresCluster)

	// determine if the dedicated repository host is ready using the repo host ready status
	dedicatedRepoReady := true
	condition := meta.FindStatusCondition(postgresCluster.Status.Conditions, ConditionRepoHostReady)
	if condition != nil {
		dedicatedRepoReady = (condition.Status == metav1.ConditionTrue)
	}

	stanzasCreated := true
	for _, repoStatus := range postgresCluster.Status.PGBackRest.Repos {
		if !repoStatus.StanzaCreated {
			stanzasCreated = false
			break
		}
	}

	// return if the cluster has not yet been initialized, or if it has been initialized and
	// all stanzas have already been created successfully
	if !clusterBootstrapped || !dedicatedRepoReady || stanzasCreated {
		return false, nil
	}

	// create the proper pod selector based on whether or not a a dedicated repository host is
	// enabled.  If a dedicated repo host is enabled, then the stanza create command will be
	// run there.  Otherwise it will be run on the current primary.
	dedicatedEnabled := pgbackrest.DedicatedRepoHostEnabled(postgresCluster)
	repoHostEnabled := pgbackrest.RepoHostEnabled(postgresCluster)
	var err error
	var podSelector labels.Selector
	var containerName string
	if dedicatedEnabled {
		podSelector = naming.PGBackRestDedicatedSelector(clusterName)
		containerName = naming.PGBackRestRepoContainerName
	} else {
		primarySelector := naming.ClusterPrimary(clusterName)
		podSelector, err = metav1.LabelSelectorAsSelector(&primarySelector)
		if err != nil {
			return false, nil
		}
		// There will only be a pgBackRest container if using a repo host.  Otherwise
		// the stanza create command will be run in the database container.
		if repoHostEnabled {
			containerName = naming.PGBackRestRepoContainerName
		} else {
			containerName = naming.ContainerDatabase
		}
	}

	pods := &v1.PodList{}
	if err := r.Client.List(ctx, pods, &client.ListOptions{
		LabelSelector: podSelector,
	}); err != nil {
		return false, err
	}

	// TODO(andrewlecuyer): Returning an error to address an out-of-sync cache (e.g, if the
	// expected Pods are not found) is a symptom of a missed event. Consider watching Pods instead
	// instead to ensure the these events are not missed
	if len(pods.Items) != 1 {
		return false, errors.WithStack(
			errors.New("invalid number of Pods found when attempting to create stanzas"))
	}

	// var stdout, stderr bytes.Buffer
	stanzaCreatePodName := pods.Items[0].GetName()

	// create a pgBackRest executor and attempt stanza creation
	exec := func(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer,
		command ...string) error {
		return r.PodExec(namespace, stanzaCreatePodName, containerName, stdin,
			stdout, stderr, command...)
	}
	configHashMismatch, err := pgbackrest.Executor(exec).StanzaCreate(ctx, configHash)
	if err != nil {
		// record and log any errors resulting from running the stanza-create command
		r.Recorder.Event(postgresCluster, v1.EventTypeWarning, EventUnableToCreateStanzas,
			err.Error())

		return false, err
	}
	// Don't record event or return an error if configHashMismatch is true, since this just means
	// configuration changes in ConfigMaps/Secrets have not yet propagated to the container.
	// Therefore, just log an an info message and return an error to requeue and try again.
	if configHashMismatch {
		return true, nil
	}

	// record an event indicating successful stanza creation
	r.Recorder.Event(postgresCluster, v1.EventTypeNormal, EventStanzasCreated,
		"pgBackRest stanza creation completed successfully")

	// if no errors then stanza(s) created successfully
	for i := range postgresCluster.Status.PGBackRest.Repos {
		postgresCluster.Status.PGBackRest.Repos[i].StanzaCreated = true
	}

	return false, nil
}

// getRepoHostStatus is responsible for returning the pgBackRest status for the provided pgBackRest
// repository host
func getRepoHostStatus(repoHost *appsv1.StatefulSet) *v1beta1.RepoHostStatus {

	repoHostStatus := &v1beta1.RepoHostStatus{}

	repoHostStatus.TypeMeta = repoHost.TypeMeta

	if repoHost.Status.ReadyReplicas == *repoHost.Spec.Replicas {
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
func getRepoVolumeStatus(repoStatus []v1beta1.RepoStatus, repoVolumes []*v1.PersistentVolumeClaim,
	configHashes map[string]string) []v1beta1.RepoStatus {

	// the new repository status that will be generated and returned
	updatedRepoStatus := []v1beta1.RepoStatus{}

	// Update the repo status based on the repo volumes (PVCs) that were reconciled.  This includes
	// updating the status for any existing repository volumes, and adding status for any new
	// repository volumes.
	for _, rv := range repoVolumes {
		newRepoVolStatus := true
		repoName := rv.Labels[naming.LabelPGBackRestRepo]
		for _, rs := range repoStatus {
			if rs.Name == repoName {
				newRepoVolStatus = false

				// update binding info if needed
				if rs.Bound != (rv.Status.Phase == v1.ClaimBound) {
					rs.Bound = (rv.Status.Phase == v1.ClaimBound)
				}
				updatedRepoStatus = append(updatedRepoStatus, rs)
				break
			}
		}
		if newRepoVolStatus {
			updatedRepoStatus = append(updatedRepoStatus, v1beta1.RepoStatus{
				Bound:      (rv.Status.Phase == v1.ClaimBound),
				Name:       repoName,
				VolumeName: rv.Spec.VolumeName,
			})
			break
		}
	}

	// Update the repo status based on the configuration hashes for any external repositories
	// configured for the cluster (e.g. Azure, GCS or S3 repositories).  This includes
	// updating the status for any existing external repositories, and adding status for any new
	// external repositories.
	for repoName, hash := range configHashes {
		newExtRepoStatus := true
		for _, rs := range repoStatus {
			if rs.Name == repoName {
				newExtRepoStatus = false
				// Update the hash if needed. Setting StanzaCreated to "false" will force another
				// run of the  pgBackRest stanza-create command
				if rs.RepoOptionsHash != hash {
					rs.RepoOptionsHash = hash
					rs.StanzaCreated = false
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
		return updatedRepoStatus[i].Name > updatedRepoStatus[j].Name
	})

	return updatedRepoStatus
}
