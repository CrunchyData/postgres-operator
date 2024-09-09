// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package standalone_pgadmin

import (
	"context"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestDeleteControlled(t *testing.T) {
	ctx := context.Background()
	cc := setupKubernetes(t)
	require.ParallelCapacity(t, 1)

	ns := setupNamespace(t, cc)
	reconciler := PGAdminReconciler{Client: cc}

	pgadmin := new(v1beta1.PGAdmin)
	pgadmin.Namespace = ns.Name
	pgadmin.Name = strings.ToLower(t.Name())
	assert.NilError(t, cc.Create(ctx, pgadmin))

	t.Run("NoOwnership", func(t *testing.T) {
		secret := &corev1.Secret{}
		secret.Namespace = ns.Name
		secret.Name = "solo"

		assert.NilError(t, cc.Create(ctx, secret))

		// No-op when there's no ownership
		assert.NilError(t, reconciler.deleteControlled(ctx, pgadmin, secret))
		assert.NilError(t, cc.Get(ctx, client.ObjectKeyFromObject(secret), secret))
	})

	// We aren't currently using setOwnerReference in the pgAdmin controller
	// If that changes we can uncomment this code
	// t.Run("Owned", func(t *testing.T) {
	// 	secret := &corev1.Secret{}
	// 	secret.Namespace = ns.Name
	// 	secret.Name = "owned"

	// 	assert.NilError(t, reconciler.setOwnerReference(pgadmin, secret))
	// 	assert.NilError(t, cc.Create(ctx, secret))

	// 	// No-op when not controlled by cluster.
	// 	assert.NilError(t, reconciler.deleteControlled(ctx, pgadmin, secret))
	// 	assert.NilError(t, cc.Get(ctx, client.ObjectKeyFromObject(secret), secret))
	// })

	t.Run("Controlled", func(t *testing.T) {
		secret := &corev1.Secret{}
		secret.Namespace = ns.Name
		secret.Name = "controlled"

		assert.NilError(t, reconciler.setControllerReference(pgadmin, secret))
		assert.NilError(t, cc.Create(ctx, secret))

		// Deletes when controlled by cluster.
		assert.NilError(t, reconciler.deleteControlled(ctx, pgadmin, secret))

		err := cc.Get(ctx, client.ObjectKeyFromObject(secret), secret)
		assert.Assert(t, apierrors.IsNotFound(err), "expected NotFound, got %#v", err)
	})
}
