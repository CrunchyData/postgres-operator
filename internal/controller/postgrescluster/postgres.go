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

package postgrescluster

import (
	"context"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=create;patch

// reconcilePostgresWALVolume writes the PersistentVolumeClaim for instance's
// PostgreSQL WAL volume.
func (r *Reconciler) reconcilePostgresWALVolume(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
	instanceSpec *v1beta1.PostgresInstanceSetSpec, instance *appsv1.StatefulSet,
) (*corev1.PersistentVolumeClaim, error) {
	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: naming.InstancePostgresWALVolume(instance)}
	pvc.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"))

	if instanceSpec.WALVolumeClaimSpec == nil {
		// TODO(cbandy): delete safely
		return nil, nil
	}

	err := errors.WithStack(r.setControllerReference(cluster, pvc))

	pvc.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil(),
		instanceSpec.Metadata.GetAnnotationsOrNil())

	pvc.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		instanceSpec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster:     cluster.Name,
			naming.LabelInstanceSet: instanceSpec.Name,
			naming.LabelInstance:    instance.Name,
			naming.LabelRole:        naming.RolePostgresWAL,
		})

	pvc.Spec = *instanceSpec.WALVolumeClaimSpec

	if err == nil {
		err = r.handlePersistentVolumeClaimError(cluster,
			errors.WithStack(r.apply(ctx, pvc)))
	}

	return pvc, err
}
