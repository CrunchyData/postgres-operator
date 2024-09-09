// Copyright 2023 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package standalone_pgadmin

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pkg/errors"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// reconcilePGAdminStatefulSet writes the StatefulSet that runs pgAdmin.
func (r *PGAdminReconciler) reconcilePGAdminStatefulSet(
	ctx context.Context, pgadmin *v1beta1.PGAdmin,
	configmap *corev1.ConfigMap, dataVolume *corev1.PersistentVolumeClaim,
) error {
	sts := statefulset(r, pgadmin, configmap, dataVolume)

	// Previous versions of PGO used a StatefulSet Pod Management Policy that could leave the Pod
	// in a failed state. When we see that it has the wrong policy, we will delete the StatefulSet
	// and then recreate it with the correct policy, as this is not a property that can be patched.
	// When we delete the StatefulSet, we will leave its Pods in place. They will be claimed by
	// the StatefulSet that gets created in the next reconcile.
	existing := &appsv1.StatefulSet{}
	if err := errors.WithStack(r.Client.Get(ctx, client.ObjectKeyFromObject(sts), existing)); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	} else {
		if existing.Spec.PodManagementPolicy != sts.Spec.PodManagementPolicy {
			// We want to delete the STS without affecting the Pods, so we set the PropagationPolicy to Orphan.
			// The orphaned Pods will be claimed by the StatefulSet that will be created in the next reconcile.
			uid := existing.GetUID()
			version := existing.GetResourceVersion()
			exactly := client.Preconditions{UID: &uid, ResourceVersion: &version}
			propagate := client.PropagationPolicy(metav1.DeletePropagationOrphan)

			return errors.WithStack(client.IgnoreNotFound(r.Client.Delete(ctx, existing, exactly, propagate)))
		}
	}

	if err := errors.WithStack(r.setControllerReference(pgadmin, sts)); err != nil {
		return err
	}
	return errors.WithStack(r.apply(ctx, sts))
}

// statefulset defines the StatefulSet needed to run pgAdmin.
func statefulset(
	r *PGAdminReconciler,
	pgadmin *v1beta1.PGAdmin,
	configmap *corev1.ConfigMap,
	dataVolume *corev1.PersistentVolumeClaim,
) *appsv1.StatefulSet {
	sts := &appsv1.StatefulSet{ObjectMeta: naming.StandalonePGAdmin(pgadmin)}
	sts.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("StatefulSet"))

	sts.Annotations = pgadmin.Spec.Metadata.GetAnnotationsOrNil()
	sts.Labels = naming.Merge(
		pgadmin.Spec.Metadata.GetLabelsOrNil(),
		naming.StandalonePGAdminDataLabels(pgadmin.Name),
	)
	sts.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: naming.StandalonePGAdminLabels(pgadmin.Name),
	}
	sts.Spec.Template.Annotations = pgadmin.Spec.Metadata.GetAnnotationsOrNil()
	sts.Spec.Template.Labels = naming.Merge(
		pgadmin.Spec.Metadata.GetLabelsOrNil(),
		naming.StandalonePGAdminDataLabels(pgadmin.Name),
	)

	// Don't clutter the namespace with extra ControllerRevisions.
	sts.Spec.RevisionHistoryLimit = initialize.Int32(0)

	// Use StatefulSet's "RollingUpdate" strategy and "Parallel" policy to roll
	// out changes to pods even when not Running or not Ready.
	// - https://docs.k8s.io/concepts/workloads/controllers/statefulset/#rolling-updates
	// - https://docs.k8s.io/concepts/workloads/controllers/statefulset/#forced-rollback
	// - https://kep.k8s.io/3541
	sts.Spec.PodManagementPolicy = appsv1.ParallelPodManagement
	sts.Spec.UpdateStrategy.Type = appsv1.RollingUpdateStatefulSetStrategyType

	// Use scheduling constraints from the cluster spec.
	sts.Spec.Template.Spec.Affinity = pgadmin.Spec.Affinity
	sts.Spec.Template.Spec.Tolerations = pgadmin.Spec.Tolerations

	if pgadmin.Spec.PriorityClassName != nil {
		sts.Spec.Template.Spec.PriorityClassName = *pgadmin.Spec.PriorityClassName
	}

	// Restart containers any time they stop, die, are killed, etc.
	// - https://docs.k8s.io/concepts/workloads/pods/pod-lifecycle/#restart-policy
	sts.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyAlways

	// pgAdmin does not make any Kubernetes API calls. Use the default
	// ServiceAccount and do not mount its credentials.
	sts.Spec.Template.Spec.AutomountServiceAccountToken = initialize.Bool(false)

	// Do not add environment variables describing services in this namespace.
	sts.Spec.Template.Spec.EnableServiceLinks = initialize.Bool(false)

	// set the image pull secrets, if any exist
	sts.Spec.Template.Spec.ImagePullSecrets = pgadmin.Spec.ImagePullSecrets

	sts.Spec.Template.Spec.SecurityContext = podSecurityContext(r)

	pod(pgadmin, configmap, &sts.Spec.Template.Spec, dataVolume)

	return sts
}
