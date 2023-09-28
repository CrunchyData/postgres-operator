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

	corev1 "k8s.io/api/core/v1"

	"github.com/pkg/errors"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// +kubebuilder:rbac:groups="",resources="configmaps",verbs={get}
// +kubebuilder:rbac:groups="",resources="configmaps",verbs={create,delete,patch}

// reconcilePGAdminConfigMap writes the ConfigMap for pgAdmin.
func (r *PGAdminReconciler) reconcilePGAdminConfigMap(
	ctx context.Context, pgadmin *v1beta1.PGAdmin,
) (*corev1.ConfigMap, error) {
	configmap := configmap(pgadmin)

	err := errors.WithStack(r.setControllerReference(pgadmin, configmap))

	if err == nil {
		err = errors.WithStack(r.apply(ctx, configmap))
	}

	return configmap, err
}

// configmap returns a v1.ConfigMap for pgAdmin.
func configmap(pgadmin *v1beta1.PGAdmin) *corev1.ConfigMap {
	configmap := &corev1.ConfigMap{ObjectMeta: naming.StandalonePGAdmin(pgadmin)}
	configmap.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))

	configmap.Annotations = pgadmin.Spec.Metadata.GetAnnotationsOrNil()
	configmap.Labels = naming.Merge(
		pgadmin.Spec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelStandalonePGAdmin: pgadmin.Name,
			naming.LabelRole:              naming.RolePGAdmin,
		})

	// TODO(tjmoore4): Populate configuration details.
	initialize.StringMap(&configmap.Data)
	configmap.Data[settingsConfigMapKey] = "config data"

	return configmap
}
