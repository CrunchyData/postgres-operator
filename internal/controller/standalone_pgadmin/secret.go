// Copyright 2023 - 2024 Crunchy Data Solutions, Inc.
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
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/util"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// +kubebuilder:rbac:groups="",resources="secrets",verbs={get}
// +kubebuilder:rbac:groups="",resources="secrets",verbs={create,delete,patch}

// reconcilePGAdminSecret reconciles the secret containing authentication
// for the pgAdmin administrator account
func (r *PGAdminReconciler) reconcilePGAdminSecret(
	ctx context.Context,
	pgadmin *v1beta1.PGAdmin) (*corev1.Secret, error) {

	existing := &corev1.Secret{ObjectMeta: naming.StandalonePGAdmin(pgadmin)}
	err := errors.WithStack(
		r.Client.Get(ctx, client.ObjectKeyFromObject(existing), existing))
	if client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	secret, err := secret(pgadmin, existing)

	if err == nil {
		err = errors.WithStack(r.setControllerReference(pgadmin, secret))
	}

	if err == nil {
		err = errors.WithStack(r.apply(ctx, secret))
	}

	return secret, err
}

func secret(pgadmin *v1beta1.PGAdmin, existing *corev1.Secret) (*corev1.Secret, error) {

	intent := &corev1.Secret{ObjectMeta: naming.StandalonePGAdmin(pgadmin)}
	intent.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))

	intent.Annotations = naming.Merge(
		pgadmin.Spec.Metadata.GetAnnotationsOrNil(),
	)
	intent.Labels = naming.Merge(
		pgadmin.Spec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelStandalonePGAdmin: pgadmin.Name,
			naming.LabelRole:              naming.RolePGAdmin,
		})

	intent.Data = make(map[string][]byte)

	// The username format is hardcoded,
	// but append the full username to the secret for visibility
	intent.Data["username"] = []byte(fmt.Sprintf("admin@%s.%s.svc",
		pgadmin.Name, pgadmin.Namespace))

	// Copy existing password into the intent
	if existing.Data != nil {
		intent.Data["password"] = existing.Data["password"]
	}

	// When password is unset, generate a new one
	if len(intent.Data["password"]) == 0 {
		password, err := util.GenerateASCIIPassword(util.DefaultGeneratedPasswordLength)
		if err != nil {
			return nil, err
		}
		intent.Data["password"] = []byte(password)
	}

	// Copy existing user data into the intent
	if existing.Data["users.json"] != nil {
		intent.Data["users.json"] = existing.Data["users.json"]
	}

	return intent, nil
}
