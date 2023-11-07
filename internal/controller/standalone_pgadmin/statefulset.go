// Copyright 2023 Crunchy Data Solutions, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package standalone_pgadmin

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
		naming.StandalonePGAdminCommonLabels(pgadmin),
	)
	sts.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			naming.LabelStandalonePGAdmin: pgadmin.Name,
			naming.LabelRole:              naming.RolePGAdmin,
		},
	}
	sts.Spec.Template.Annotations = pgadmin.Spec.Metadata.GetAnnotationsOrNil()
	sts.Spec.Template.Labels = naming.Merge(
		pgadmin.Spec.Metadata.GetLabelsOrNil(),
		naming.StandalonePGAdminCommonLabels(pgadmin),
	)

	// Don't clutter the namespace with extra ControllerRevisions.
	sts.Spec.RevisionHistoryLimit = initialize.Int32(0)

	// Set the StatefulSet update strategy to "RollingUpdate", and the Partition size for the
	// update strategy to 0 (note that these are the defaults for a StatefulSet).  This means
	// every pod of the StatefulSet will be deleted and recreated when the Pod template changes.
	// - https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#rolling-updates
	// - https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#forced-rollback
	sts.Spec.UpdateStrategy.Type = appsv1.RollingUpdateStatefulSetStrategyType
	sts.Spec.UpdateStrategy.RollingUpdate = &appsv1.RollingUpdateStatefulSetStrategy{
		Partition: initialize.Int32(0),
	}

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
