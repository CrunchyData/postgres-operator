//go:build envtest
// +build envtest

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

package crunchybridgecluster

import (
	"context"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/testing/require"
)

func TestGetSecretKeys(t *testing.T) {
	ctx := context.Background()
	_, tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	reconciler := &CrunchyBridgeClusterReconciler{Client: tClient}

	ns := setupNamespace(t, tClient).Name
	cluster := testCluster()
	cluster.Namespace = ns

	t.Run("NoSecret", func(t *testing.T) {
		apiKey, team, err := reconciler.GetSecretKeys(ctx, cluster)
		assert.Check(t, apiKey == "")
		assert.Check(t, team == "")
		assert.ErrorContains(t, err, "secrets \"crunchy-bridge-api-key\" not found")
	})

	t.Run("SecretMissingApiKey", func(t *testing.T) {
		cluster.Spec.Secret = "secret-missing-api-key"
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-missing-api-key",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"team": []byte(`jkl;`),
			},
		}
		assert.NilError(t, tClient.Create(ctx, secret))

		apiKey, team, err := reconciler.GetSecretKeys(ctx, cluster)
		assert.Check(t, apiKey == "")
		assert.Check(t, team == "")
		assert.ErrorContains(t, err, "error handling secret; expected to find a key and a team: found key false, found team true")

		assert.NilError(t, tClient.Delete(ctx, secret))
	})

	t.Run("SecretMissingTeamId", func(t *testing.T) {
		cluster.Spec.Secret = "secret-missing-team-id"
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-missing-team-id",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"key": []byte(`asdf`),
			},
		}
		assert.NilError(t, tClient.Create(ctx, secret))

		apiKey, team, err := reconciler.GetSecretKeys(ctx, cluster)
		assert.Check(t, apiKey == "")
		assert.Check(t, team == "")
		assert.ErrorContains(t, err, "error handling secret; expected to find a key and a team: found key true, found team false")
	})

	t.Run("GoodSecret", func(t *testing.T) {
		cluster.Spec.Secret = "crunchy-bridge-api-key"
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "crunchy-bridge-api-key",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"key":  []byte(`asdf`),
				"team": []byte(`jkl;`),
			},
		}
		assert.NilError(t, tClient.Create(ctx, secret))

		apiKey, team, err := reconciler.GetSecretKeys(ctx, cluster)
		assert.Check(t, apiKey == "asdf")
		assert.Check(t, team == "jkl;")
		assert.NilError(t, err)
	})
}

func TestDeleteControlled(t *testing.T) {
	ctx := context.Background()
	_, tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 1)

	ns := setupNamespace(t, tClient)
	reconciler := &CrunchyBridgeClusterReconciler{Client: tClient}

	cluster := testCluster()
	cluster.Namespace = ns.Name
	cluster.Name = strings.ToLower(t.Name())
	assert.NilError(t, tClient.Create(ctx, cluster))

	t.Run("NotControlled", func(t *testing.T) {
		secret := &corev1.Secret{}
		secret.Namespace = ns.Name
		secret.Name = "solo"

		assert.NilError(t, tClient.Create(ctx, secret))

		// No-op when there's no ownership
		assert.NilError(t, reconciler.deleteControlled(ctx, cluster, secret))
		assert.NilError(t, tClient.Get(ctx, client.ObjectKeyFromObject(secret), secret))
	})

	t.Run("Controlled", func(t *testing.T) {
		secret := &corev1.Secret{}
		secret.Namespace = ns.Name
		secret.Name = "controlled"

		assert.NilError(t, reconciler.setControllerReference(cluster, secret))
		assert.NilError(t, tClient.Create(ctx, secret))

		// Deletes when controlled by cluster.
		assert.NilError(t, reconciler.deleteControlled(ctx, cluster, secret))

		err := tClient.Get(ctx, client.ObjectKeyFromObject(secret), secret)
		assert.Assert(t, apierrors.IsNotFound(err), "expected NotFound, got %#v", err)
	})
}

// TODO: add TestReconcileBridgeConnectionSecret once conditions are in place
