// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package standalone_pgadmin

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestFindPGAdminsForSecret(t *testing.T) {
	ctx := context.Background()
	tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	ns := setupNamespace(t, tClient)
	reconciler := &PGAdminReconciler{Client: tClient}

	secret1 := &corev1.Secret{}
	secret1.Namespace = ns.Name
	secret1.Name = "first-password-secret"

	assert.NilError(t, tClient.Create(ctx, secret1))
	secretObjectKey := client.ObjectKeyFromObject(secret1)

	t.Run("NoPGAdmins", func(t *testing.T) {
		pgadmins := reconciler.findPGAdminsForSecret(ctx, secretObjectKey)

		assert.Equal(t, len(pgadmins), 0)
	})

	t.Run("OnePGAdmin", func(t *testing.T) {
		pgadmin1 := new(v1beta1.PGAdmin)
		pgadmin1.Namespace = ns.Name
		pgadmin1.Name = "first-pgadmin"
		pgadmin1.Spec.Users = []v1beta1.PGAdminUser{
			{
				PasswordRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "first-password-secret",
					},
					Key: "password",
				},
				Username: "testuser",
				Role:     "Administrator",
			},
		}
		assert.NilError(t, tClient.Create(ctx, pgadmin1))

		pgadmins := reconciler.findPGAdminsForSecret(ctx, secretObjectKey)

		assert.Equal(t, len(pgadmins), 1)
		assert.Equal(t, pgadmins[0].Name, "first-pgadmin")
	})

	t.Run("TwoPGAdmins", func(t *testing.T) {
		pgadmin2 := new(v1beta1.PGAdmin)
		pgadmin2.Namespace = ns.Name
		pgadmin2.Name = "second-pgadmin"
		pgadmin2.Spec.Users = []v1beta1.PGAdminUser{
			{
				PasswordRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "first-password-secret",
					},
					Key: "password",
				},
				Username: "testuser2",
				Role:     "Administrator",
			},
		}
		assert.NilError(t, tClient.Create(ctx, pgadmin2))

		pgadmins := reconciler.findPGAdminsForSecret(ctx, secretObjectKey)

		assert.Equal(t, len(pgadmins), 2)
		pgadminCount := map[string]int{}
		for _, pgadmin := range pgadmins {
			pgadminCount[pgadmin.Name] += 1
		}
		assert.Equal(t, pgadminCount["first-pgadmin"], 1)
		assert.Equal(t, pgadminCount["second-pgadmin"], 1)
	})

	t.Run("PGAdminWithDifferentSecretNameNotIncluded", func(t *testing.T) {
		pgadmin3 := new(v1beta1.PGAdmin)
		pgadmin3.Namespace = ns.Name
		pgadmin3.Name = "third-pgadmin"
		pgadmin3.Spec.Users = []v1beta1.PGAdminUser{
			{
				PasswordRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "other-password-secret",
					},
					Key: "password",
				},
				Username: "testuser2",
				Role:     "Administrator",
			},
		}
		assert.NilError(t, tClient.Create(ctx, pgadmin3))

		pgadmins := reconciler.findPGAdminsForSecret(ctx, secretObjectKey)

		assert.Equal(t, len(pgadmins), 2)
		pgadminCount := map[string]int{}
		for _, pgadmin := range pgadmins {
			pgadminCount[pgadmin.Name] += 1
		}
		assert.Equal(t, pgadminCount["first-pgadmin"], 1)
		assert.Equal(t, pgadminCount["second-pgadmin"], 1)
		assert.Equal(t, pgadminCount["third-pgadmin"], 0)
	})
}
