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
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/internal/bridge"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestGeneratePostgresRoleSecret(t *testing.T) {
	_, tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	reconciler := &CrunchyBridgeClusterReconciler{Client: tClient}

	cluster := testCluster()
	cluster.Namespace = setupNamespace(t, tClient).Name

	spec := &v1beta1.CrunchyBridgeClusterRoleSpec{
		Name:       "application",
		SecretName: "application-role-secret",
	}
	role := &bridge.ClusterRoleApiResource{
		Name:     "application",
		Password: "password",
		URI:      "postgres://application:password@example.com:5432/postgres",
	}
	t.Run("ObjectMeta", func(t *testing.T) {
		secret, err := reconciler.generatePostgresRoleSecret(cluster, spec, role)
		assert.NilError(t, err)

		if assert.Check(t, secret != nil) {
			assert.Equal(t, secret.Namespace, cluster.Namespace)
			assert.Assert(t, metav1.IsControlledBy(secret, cluster))
			assert.DeepEqual(t, secret.Labels, map[string]string{
				"postgres-operator.crunchydata.com/cluster":    "hippo-cr",
				"postgres-operator.crunchydata.com/role":       "cbc-pgrole",
				"postgres-operator.crunchydata.com/cbc-pgrole": "application",
			})
		}
	})

	t.Run("Data", func(t *testing.T) {
		secret, err := reconciler.generatePostgresRoleSecret(cluster, spec, role)
		assert.NilError(t, err)

		if assert.Check(t, secret != nil) {
			assert.Equal(t, secret.StringData["name"], "application")
			assert.Equal(t, secret.StringData["password"], "password")
			assert.Equal(t, secret.StringData["uri"],
				"postgres://application:password@example.com:5432/postgres")
		}
	})
}

func TestReconcilePostgresRoleSecrets(t *testing.T) {
	ctx := context.Background()
	_, tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	reconciler := &CrunchyBridgeClusterReconciler{Client: tClient}

	apiKey := "1234567890"
	ns := setupNamespace(t, tClient).Name

	t.Run("DuplicateSecretNameInSpec", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns

		spec1 := &v1beta1.CrunchyBridgeClusterRoleSpec{
			Name:       "application",
			SecretName: "role-secret",
		}
		spec2 := &v1beta1.CrunchyBridgeClusterRoleSpec{
			Name:       "postgres",
			SecretName: "role-secret",
		}
		cluster.Spec.Roles = append(cluster.Spec.Roles, spec1, spec2)

		roleSpecSlice, secretMap, err := reconciler.reconcilePostgresRoleSecrets(ctx, apiKey, cluster)
		assert.Check(t, roleSpecSlice == nil)
		assert.Check(t, secretMap == nil)
		assert.ErrorContains(t, err, "Two or more of the Roles in the CrunchyBridgeCluster spec have "+
			"the same SecretName. Role SecretNames must be unique.", "expected duplicate secret name error")
	})

	t.Run("DuplicateSecretNameInNamespace", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "role-secret",
				Namespace: ns,
			},
			StringData: map[string]string{
				"path": "stuff",
			},
		}
		assert.NilError(t, tClient.Create(ctx, secret.DeepCopy()))

		cluster := testCluster()
		cluster.Namespace = ns

		spec1 := &v1beta1.CrunchyBridgeClusterRoleSpec{
			Name:       "application",
			SecretName: "role-secret",
		}

		cluster.Spec.Roles = append(cluster.Spec.Roles, spec1)

		roleSpecSlice, secretMap, err := reconciler.reconcilePostgresRoleSecrets(ctx, apiKey, cluster)
		assert.Check(t, roleSpecSlice == nil)
		assert.Check(t, secretMap == nil)
		assert.ErrorContains(t, err, "There is already an existing Secret in this namespace with the name role-secret. "+
			"Please choose a different name for this role's Secret.", "expected duplicate secret name error")
	})
}
