/*
 Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
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
)

func TestDeleteControlled(t *testing.T) {
	ctx := context.Background()
	cc := setupKubernetes(t)
	require.ParallelCapacity(t, 1)

	ns := setupNamespace(t, cc)
	reconciler := PGAdminReconciler{Client: cc}

	pgadmin := testPGAdmin()
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
