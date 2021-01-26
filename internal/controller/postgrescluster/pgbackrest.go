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
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
)

const (
	// ConditionRepoHostsReady is the type used in a condition to indicate whether or not all
	// pgBackRest repository hosts for the PostgresCluster are ready
	ConditionRepoHostsReady = "PGBackRestRepoHostsReady"

	// EventRepoHostNotFound is used to indicate that a pgBackRest repository was not
	// found when reconciling
	EventRepoHostNotFound = "RepoDeploymentNotFound"

	// EventRepoHostCreated is the event reason utilized when a pgBackRest repository host is
	// created
	EventRepoHostCreated = "RepoHostCreated"

	// EventInvalidRepoHostCount is the event created when an invalid number of repo hosts are
	// detected in the Kubernetes cluster
	EventInvalidRepoHostCount = "InvalidRepoHostCount"

	// EventRepoHostInvalid is the event created when an existing pgBackRest repository host is
	// invalid
	EventRepoHostInvalid = "RepoHostInvalid"

	// PGBackRestRepoContainerName is the name assigned to the container used to run the repository
	// host
	PGBackRestRepoContainerName = "repohost"

	// PGBackRestRepoName is the name used for a pgbackrest repository
	PGBackRestRepoName = "%s-pgbackrest-repo"
)

// applyRepoHostIntent ensure the pgBackRest repository host deployment is synchronized with the
// proper configuration according to the provided PostgresCluster custom resource.  This is done by
// applying the PostgresCluster controller's fully specified intent for the repo host deployment.
// Any changes to the deployment spec as a result of synchronization will result in a rollout of
// the pgBackRest repo host deployment in accordance with its configured strategy.
func (r *Reconciler) applyRepoHostIntent(ctx context.Context,
	postgresCluster *v1alpha1.PostgresCluster) (*appsv1.StatefulSet, error) {

	repo, err := generateRepoHostIntent(postgresCluster, r.Client.Scheme())
	if err != nil {
		return nil, err
	}

	if err := r.apply(ctx, repo, client.ForceOwnership); err != nil {
		return nil, err
	}

	return repo, nil
}

// calculatePGBackRestConditions is responsible for calculating any pgBackRest conditions
// based on the current pgBackRest status, and then updating the conditions array within
// the PostgresCluster status with those conditions as needed.
func calculatePGBackRestConditions(
	status *v1alpha1.PGBackRestStatus,
	postgresClusterGeneration int64) []metav1.Condition {

	// if we ever allow more than 1 repo host, we would check them all before setting this
	// condition to true
	repoHostsReady := metav1.Condition{
		ObservedGeneration: postgresClusterGeneration,
		Type:               ConditionRepoHostsReady,
	}
	if status.RepoHost == nil {
		repoHostsReady.Status = metav1.ConditionUnknown
		repoHostsReady.Reason = "RepoHostStatusMissing"
		repoHostsReady.Message = "pgBackRest repository host status is missing"
	} else if status.RepoHost.Ready {
		repoHostsReady.Status = metav1.ConditionTrue
		repoHostsReady.Reason = "AllRepoHostsReady"
		repoHostsReady.Message = "All pgBackRest repository hosts are ready"
	} else {
		repoHostsReady.Status = metav1.ConditionFalse
		repoHostsReady.Reason = "AllRepoHostsNotReady"
		repoHostsReady.Message = "All pgBackRest repository hosts are not ready"
	}

	return []metav1.Condition{repoHostsReady}
}

// generateRepoHostIntent creates and populates StatefulSet with the PostgresCluster's full intent
// as needed to create and reconcile a pgBackRest dedicated repository host within the kubernetes
// cluster.
func generateRepoHostIntent(postgresCluster *v1alpha1.PostgresCluster,
	scheme *runtime.Scheme) (*appsv1.StatefulSet, error) {

	clusterName := postgresCluster.GetName()
	replicas := int32(1)
	labels := getRepoLabels(clusterName)

	configVolumes := postgresCluster.Spec.Archive.PGBackRest.Configuration
	// TODO append any operator-generated volumes containing required & default pgbackrest configs

	repoDeployment := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Name:      fmt.Sprintf(PGBackRestRepoName, clusterName),
			Namespace: postgresCluster.GetNamespace(),
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Command: []string{"tail", "-f", "/dev/null"},
							Env: []v1.EnvVar{
								{
									Name:  "MODE",
									Value: "pgbackrest-repo",
								},
							},
							Image: postgresCluster.Spec.Archive.PGBackRest.Image,
							Name:  PGBackRestRepoContainerName,
							VolumeMounts: []v1.VolumeMount{{
								Name:      "config",
								MountPath: "/etc/pgbackrest/conf.d",
							}},
						},
					},
					Volumes: []v1.Volume{{
						Name: "config",
						VolumeSource: v1.VolumeSource{
							Projected: &v1.ProjectedVolumeSource{
								Sources: configVolumes,
							},
						},
					}},
				},
			},
		},
	}

	// set ownership references
	if err := controllerutil.SetControllerReference(postgresCluster, repoDeployment,
		scheme); err != nil {
		return nil, err
	}

	return repoDeployment, nil
}

// getReposForPostgresCluster returns any pgbackrest repository StatefulSets that should be
// managed by the provided PostgresCluster.
func (r *Reconciler) getReposForPostgresCluster(ctx context.Context,
	postgresCluster *v1alpha1.PostgresCluster) ([]appsv1.StatefulSet, error) {

	namespace := postgresCluster.GetNamespace()
	clusterName := postgresCluster.GetName()
	owned := []appsv1.StatefulSet{}

	repos := &appsv1.StatefulSetList{}
	if err := r.Client.List(ctx, repos, client.InNamespace(namespace)); err != nil {
		return nil, err
	}

	repoSelector, err := metav1.LabelSelectorAsSelector(
		metav1.SetAsLabelSelector(getRepoLabels(clusterName)))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	for i, sts := range repos.Items {
		if metav1.IsControlledBy(&repos.Items[i], postgresCluster) &&
			repoSelector.Matches(labels.Set(repos.Items[i].GetLabels())) {
			owned = append(owned, sts)
		}
	}

	return owned, nil
}

// reconcilePGBackRest is responsible for reconciling any/all pgBackRest resources owned by a
// specific PostgresCluster (e.g. Deployments, ConfigMaps, Secrets, etc.).  This function will
// ensure various reconciliation logic is run as needed for each pgBackRest resource, while then
// also generating the proper Result as needed to ensure proper event requeuing according to
// the results of any attempts to properly reconcile these resources.
func (r *Reconciler) reconcilePGBackRest(ctx context.Context,
	postgresCluster *v1alpha1.PostgresCluster,
	status *v1alpha1.PostgresClusterStatus) (reconcile.Result, error) {

	// add some additional context about what component is being reconciled
	log := logging.FromContext(ctx).WithValues("reconciler", "pgBackRest")

	// create the pgBackRest status that will be updated when reconciling various pgBackRest
	// resources
	if status.PGBackRest == nil {
		status.PGBackRest = new(v1alpha1.PGBackRestStatus)
	}
	pgBackRestStatus := status.PGBackRest

	// reconcile the pgbackrest repository host Deployment
	if err := r.reconcileRepoHost(ctx, postgresCluster, pgBackRestStatus); err != nil {
		log.Error(err, "unable to reconcile pgBackRest repo host")
		return reconcile.Result{
			Requeue: true,
		}, err
	}

	// ensure conditions are updated before returning
	setStatusConditions(status,
		calculatePGBackRestConditions(pgBackRestStatus, postgresCluster.GetGeneration())...)

	return reconcile.Result{}, nil
}

// reconcileRepoHost is responsible for reconciling the pgBackRest repository host deployment
// according to a specific PostgresCluster custom resource.
func (r *Reconciler) reconcileRepoHost(ctx context.Context,
	postgresCluster *v1alpha1.PostgresCluster, status *v1alpha1.PGBackRestStatus) error {

	log := logging.FromContext(ctx).WithValues("reconcileResource", "pgbBackRestRepo")

	repos, err := r.getReposForPostgresCluster(ctx, postgresCluster)
	if err != nil {
		log.Error(err, "unable to get repository hosts for PostgresCluster")
		return err
	}

	create := (len(repos) == 0)
	log.V(1).Info("fetched pgBackRest repos", "repoCount", len(repos))

	// Make sure there isn't an existing resource with the same name to ensure we don't Server-Side
	// apply a resource we do not own, as needed to adhere to the second law of The Three Laws of
	// Controllers: https://github.com/kubernetes/community/blob/master/contributors/
	// design-proposals/api-machinery/controller-ref.md#the-three-laws-of-controllers.
	//
	// Would attempting a Create() be safer here, since it would result in a conflict if the
	// resource already exists (e.g. if the cache is out-of-sync invalidating this check)?
	// Another option is a quorum read.
	if create {
		repoHostName := fmt.Sprintf(PGBackRestRepoName, postgresCluster.GetName())
		existing := &appsv1.StatefulSet{}
		err = r.Client.Get(ctx, client.ObjectKey{
			Name:      repoHostName,
			Namespace: postgresCluster.GetNamespace(),
		}, existing)
		if err == nil {
			err = fmt.Errorf("pgBackRest repo repository detected with missing %s ownership",
				ControllerName)
			r.Recorder.Eventf(postgresCluster, v1.EventTypeWarning, EventRepoHostInvalid,
				"%v: %s/%s", err.Error(), existing.TypeMeta.Kind, repoHostName)
			return errors.WithStack(err)
		}
		if !kerr.IsNotFound(err) {
			return err
		}
	}

	repo, err := r.applyRepoHostIntent(ctx, postgresCluster)
	if err != nil {
		return err
	}

	setRepoHostStatus(repo, status)

	if create {
		log.Info("created pgBackRest repository host", "name", repo.GetName())
		r.Recorder.Eventf(postgresCluster, v1.EventTypeNormal, EventRepoHostCreated,
			"created pgBackRest repository host %s/%s", repo.TypeMeta.Kind, repo.GetName())

		repoBytes, _ := json.MarshalIndent(repo, "", "    ")
		log.V(1).Info("pgBackRest repo host JSON:\n" + string(repoBytes))
	}

	return nil
}

// setRepoHostStatus is responsible for updating the pgBackRest status provided pgBackRest
// repository host
func setRepoHostStatus(repo *appsv1.StatefulSet, status *v1alpha1.PGBackRestStatus) {

	if status.RepoHost == nil {
		status.RepoHost = new(v1alpha1.RepoHostStatus)
	}

	status.RepoHost.Name = repo.GetName()
	status.RepoHost.TypeMeta = repo.TypeMeta

	if repo.Status.ReadyReplicas == *repo.Spec.Replicas {
		status.RepoHost.Ready = true
	} else {
		status.RepoHost.Ready = false
	}
}

// getRepoLabelsreturns the labels for a pgBackRest repository host.
func getRepoLabels(clusterName string) map[string]string {
	return map[string]string{
		naming.LabelCluster:        clusterName,
		naming.LabelPGBackRest:     "",
		naming.LabelPGBackRestRepo: "",
	}
}
