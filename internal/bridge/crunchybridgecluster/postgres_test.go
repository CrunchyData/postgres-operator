// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package crunchybridgecluster

import (
	"context"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/internal/bridge"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestGeneratePostgresRoleSecret(t *testing.T) {
	tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	reconciler := &CrunchyBridgeClusterReconciler{
		Client: tClient,
		Owner:  "crunchybridgecluster-controller",
	}

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
	tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	apiKey := "9012"
	ns := setupNamespace(t, tClient).Name

	reconciler := &CrunchyBridgeClusterReconciler{
		Client: tClient,
		Owner:  "crunchybridgecluster-controller",
	}

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
		assert.NilError(t, tClient.Create(ctx, secret))

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

	t.Run("UnusedSecretsGetRemoved", func(t *testing.T) {
		applicationRoleInBridge := testClusterRoleApiResource()
		postgresRoleInBridge := testClusterRoleApiResource()
		postgresRoleInBridge.Name = "postgres"
		postgresRoleInBridge.Password = "postgres-password"
		reconciler.NewClient = func() bridge.ClientInterface {
			return &TestBridgeClient{
				ApiKey:       apiKey,
				TeamId:       "5678",
				ClusterRoles: []*bridge.ClusterRoleApiResource{applicationRoleInBridge, postgresRoleInBridge},
			}
		}

		applicationSpec := &v1beta1.CrunchyBridgeClusterRoleSpec{
			Name:       "application",
			SecretName: "application-role-secret",
		}
		postgresSpec := &v1beta1.CrunchyBridgeClusterRoleSpec{
			Name:       "postgres",
			SecretName: "postgres-role-secret",
		}

		cluster := testCluster()
		cluster.Namespace = ns
		cluster.Status.ID = "1234"
		// Add one role to cluster spec
		cluster.Spec.Roles = append(cluster.Spec.Roles, applicationSpec)
		assert.NilError(t, tClient.Create(ctx, cluster))

		applicationRole := &bridge.ClusterRoleApiResource{
			Name:     "application",
			Password: "application-password",
			URI:      "connection-string",
		}
		postgresRole := &bridge.ClusterRoleApiResource{
			Name:     "postgres",
			Password: "postgres-password",
			URI:      "connection-string",
		}

		// Generate secrets
		applicationSecret, err := reconciler.generatePostgresRoleSecret(cluster, applicationSpec, applicationRole)
		assert.NilError(t, err)
		postgresSecret, err := reconciler.generatePostgresRoleSecret(cluster, postgresSpec, postgresRole)
		assert.NilError(t, err)

		// Create secrets in k8s
		assert.NilError(t, tClient.Create(ctx, applicationSecret))
		assert.NilError(t, tClient.Create(ctx, postgresSecret))

		roleSpecSlice, secretMap, err := reconciler.reconcilePostgresRoleSecrets(ctx, apiKey, cluster)
		assert.Check(t, roleSpecSlice != nil)
		assert.Check(t, secretMap != nil)
		assert.NilError(t, err)

		// Assert that postgresSecret was deleted since its associated role is not in the spec
		err = tClient.Get(ctx, client.ObjectKeyFromObject(postgresSecret), postgresSecret)
		assert.Assert(t, apierrors.IsNotFound(err), "expected NotFound, got %#v", err)

		// Assert that applicationSecret is still there
		err = tClient.Get(ctx, client.ObjectKeyFromObject(applicationSecret), applicationSecret)
		assert.NilError(t, err)
	})

	t.Run("SecretsGetUpdated", func(t *testing.T) {
		clusterRoleInBridge := testClusterRoleApiResource()
		clusterRoleInBridge.Password = "different-password"
		reconciler.NewClient = func() bridge.ClientInterface {
			return &TestBridgeClient{
				ApiKey:       apiKey,
				TeamId:       "5678",
				ClusterRoles: []*bridge.ClusterRoleApiResource{clusterRoleInBridge},
			}
		}

		cluster := testCluster()
		cluster.Namespace = ns
		err := tClient.Get(ctx, client.ObjectKeyFromObject(cluster), cluster)
		assert.NilError(t, err)
		cluster.Status.ID = "1234"

		spec1 := &v1beta1.CrunchyBridgeClusterRoleSpec{
			Name:       "application",
			SecretName: "application-role-secret",
		}
		role1 := &bridge.ClusterRoleApiResource{
			Name:     "application",
			Password: "test",
			URI:      "connection-string",
		}
		// Generate secret
		secret1, err := reconciler.generatePostgresRoleSecret(cluster, spec1, role1)
		assert.NilError(t, err)

		roleSpecSlice, secretMap, err := reconciler.reconcilePostgresRoleSecrets(ctx, apiKey, cluster)
		assert.Check(t, roleSpecSlice != nil)
		assert.Check(t, secretMap != nil)
		assert.NilError(t, err)

		// Assert that secret1 was updated
		err = tClient.Get(ctx, client.ObjectKeyFromObject(secret1), secret1)
		assert.NilError(t, err)
		assert.Equal(t, string(secret1.Data["password"]), "different-password")
	})
}
