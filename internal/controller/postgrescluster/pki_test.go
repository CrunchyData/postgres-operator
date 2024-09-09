// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/pki"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// TestReconcileCerts tests the proper reconciliation of the root ca certificate
// secret, leaf certificate secrets and the updates that occur when updates are
// made to the cluster certificates generally. For the removal of ownership
// references and deletion of the root CA cert secret, a separate Kuttl test is
// used due to the need for proper garbage collection.
func TestReconcileCerts(t *testing.T) {
	// Garbage collector cleans up test resources before the test completes
	if strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
		t.Skip("USE_EXISTING_CLUSTER: Test fails due to garbage collection")
	}

	_, tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 2)
	ctx := context.Background()
	namespace := setupNamespace(t, tClient).Name

	r := &Reconciler{
		Client: tClient,
		Owner:  ControllerName,
	}

	// set up cluster1
	clusterName1 := "hippocluster1"

	// set up test cluster1
	cluster1 := testCluster()
	cluster1.Name = clusterName1
	cluster1.Namespace = namespace
	if err := tClient.Create(ctx, cluster1); err != nil {
		t.Error(err)
	}

	// set up test cluster2
	cluster2Name := "hippocluster2"

	cluster2 := testCluster()
	cluster2.Name = cluster2Name
	cluster2.Namespace = namespace
	if err := tClient.Create(ctx, cluster2); err != nil {
		t.Error(err)
	}

	primaryService := new(corev1.Service)
	primaryService.Namespace = namespace
	primaryService.Name = "the-primary"

	replicaService := new(corev1.Service)
	replicaService.Namespace = namespace
	replicaService.Name = "the-replicas"

	t.Run("check root certificate reconciliation", func(t *testing.T) {

		initialRoot, err := r.reconcileRootCertificate(ctx, cluster1)
		assert.NilError(t, err)

		rootSecret := &corev1.Secret{}
		rootSecret.Namespace, rootSecret.Name = namespace, naming.RootCertSecret
		rootSecret.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))

		t.Run("check root CA secret first owner reference", func(t *testing.T) {

			err := tClient.Get(ctx, client.ObjectKeyFromObject(rootSecret), rootSecret)
			assert.NilError(t, err)

			assert.Check(t, len(rootSecret.ObjectMeta.OwnerReferences) == 1, "first owner reference not set")

			expectedOR := metav1.OwnerReference{
				APIVersion: "postgres-operator.crunchydata.com/v1beta1",
				Kind:       "PostgresCluster",
				Name:       "hippocluster1",
				UID:        cluster1.UID,
			}

			if len(rootSecret.ObjectMeta.OwnerReferences) > 0 {
				assert.Equal(t, rootSecret.ObjectMeta.OwnerReferences[0], expectedOR)
			}
		})

		t.Run("check root CA secret second owner reference", func(t *testing.T) {

			_, err := r.reconcileRootCertificate(ctx, cluster2)
			assert.NilError(t, err)

			err = tClient.Get(ctx, client.ObjectKeyFromObject(rootSecret), rootSecret)
			assert.NilError(t, err)

			clist := &v1beta1.PostgresClusterList{}
			assert.NilError(t, tClient.List(ctx, clist))

			assert.Check(t, len(rootSecret.ObjectMeta.OwnerReferences) == 2, "second owner reference not set")

			expectedOR := metav1.OwnerReference{
				APIVersion: "postgres-operator.crunchydata.com/v1beta1",
				Kind:       "PostgresCluster",
				Name:       "hippocluster2",
				UID:        cluster2.UID,
			}

			if len(rootSecret.ObjectMeta.OwnerReferences) > 1 {
				assert.Equal(t, rootSecret.ObjectMeta.OwnerReferences[1], expectedOR)
			}
		})

		t.Run("root certificate is returned correctly", func(t *testing.T) {

			fromSecret, err := getCertFromSecret(ctx, tClient, naming.RootCertSecret, namespace, "root.crt")
			assert.NilError(t, err)

			// assert returned certificate matches the one created earlier
			assert.DeepEqual(t, *fromSecret, initialRoot.Certificate)
		})

		t.Run("root certificate changes", func(t *testing.T) {
			// force the generation of a new root cert
			// create an empty secret and apply the change
			emptyRootSecret := &corev1.Secret{}
			emptyRootSecret.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))
			emptyRootSecret.Namespace, emptyRootSecret.Name = namespace, naming.RootCertSecret
			emptyRootSecret.Data = make(map[string][]byte)
			err = errors.WithStack(r.apply(ctx, emptyRootSecret))
			assert.NilError(t, err)

			// reconcile the root cert secret, creating a new root cert
			returnedRoot, err := r.reconcileRootCertificate(ctx, cluster1)
			assert.NilError(t, err)

			fromSecret, err := getCertFromSecret(ctx, tClient, naming.RootCertSecret, namespace, "root.crt")
			assert.NilError(t, err)

			// check that the cert from the secret does not equal the initial certificate
			assert.Assert(t, !fromSecret.Equal(initialRoot.Certificate))

			// check that the returned cert matches the cert from the secret
			assert.DeepEqual(t, *fromSecret, returnedRoot.Certificate)
		})

	})

	t.Run("check leaf certificate reconciliation", func(t *testing.T) {

		initialRoot, err := r.reconcileRootCertificate(ctx, cluster1)
		assert.NilError(t, err)

		// instance with minimal required fields
		instance := &appsv1.StatefulSet{
			TypeMeta: metav1.TypeMeta{
				APIVersion: appsv1.SchemeGroupVersion.String(),
				Kind:       "StatefulSet",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName1,
				Namespace: namespace,
			},
			Spec: appsv1.StatefulSetSpec{
				ServiceName: clusterName1,
			},
		}

		t.Run("check leaf certificate in secret", func(t *testing.T) {
			existing := &corev1.Secret{Data: make(map[string][]byte)}
			intent := &corev1.Secret{Data: make(map[string][]byte)}

			initialLeafCert, err := r.instanceCertificate(ctx, instance, existing, intent, initialRoot)
			assert.NilError(t, err)

			fromSecret := &pki.LeafCertificate{}
			assert.NilError(t, fromSecret.Certificate.UnmarshalText(intent.Data["dns.crt"]))
			assert.NilError(t, fromSecret.PrivateKey.UnmarshalText(intent.Data["dns.key"]))

			assert.DeepEqual(t, fromSecret, initialLeafCert)
		})

		t.Run("check that the leaf certs update when root changes", func(t *testing.T) {

			// force the generation of a new root cert
			// create an empty secret and apply the change
			emptyRootSecret := &corev1.Secret{}
			emptyRootSecret.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))
			emptyRootSecret.Namespace, emptyRootSecret.Name = namespace, naming.RootCertSecret
			emptyRootSecret.Data = make(map[string][]byte)
			err = errors.WithStack(r.apply(ctx, emptyRootSecret))

			// reconcile the root cert secret
			newRootCert, err := r.reconcileRootCertificate(ctx, cluster1)
			assert.NilError(t, err)

			existing := &corev1.Secret{Data: make(map[string][]byte)}
			intent := &corev1.Secret{Data: make(map[string][]byte)}

			initialLeaf, err := r.instanceCertificate(ctx, instance, existing, intent, initialRoot)
			assert.NilError(t, err)

			// reconcile the certificate
			newLeaf, err := r.instanceCertificate(ctx, instance, existing, intent, newRootCert)
			assert.NilError(t, err)

			// assert old leaf cert does not match the newly reconciled one
			assert.Assert(t, !initialLeaf.Certificate.Equal(newLeaf.Certificate))

			// 'reconcile' the certificate when the secret does not change. The returned leaf certificate should not change
			newLeaf2, err := r.instanceCertificate(ctx, instance, intent, intent, newRootCert)
			assert.NilError(t, err)

			// check that the leaf cert did not change after another reconciliation
			assert.DeepEqual(t, newLeaf2, newLeaf)

		})

	})

	t.Run("check cluster certificate secret reconciliation", func(t *testing.T) {
		// example auto-generated secret projection
		testSecretProjection := &corev1.SecretProjection{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: fmt.Sprintf(naming.ClusterCertSecret, cluster1.Name),
			},
			Items: []corev1.KeyToPath{
				{
					Key:  clusterCertFile,
					Path: clusterCertFile,
				},
				{
					Key:  clusterKeyFile,
					Path: clusterKeyFile,
				},
				{
					Key:  rootCertFile,
					Path: rootCertFile,
				},
			},
		}

		// example custom secret projection
		customSecretProjection := &corev1.SecretProjection{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: "customsecret",
			},
			Items: []corev1.KeyToPath{
				{
					Key:  clusterCertFile,
					Path: clusterCertFile,
				},
				{
					Key:  clusterKeyFile,
					Path: clusterKeyFile,
				},
				{
					Key:  rootCertFile,
					Path: rootCertFile,
				},
			},
		}

		cluster2.Spec.CustomTLSSecret = customSecretProjection

		initialRoot, err := r.reconcileRootCertificate(ctx, cluster1)
		assert.NilError(t, err)

		t.Run("check standard secret projection", func(t *testing.T) {
			secretCertProj, err := r.reconcileClusterCertificate(ctx, initialRoot, cluster1, primaryService, replicaService)
			assert.NilError(t, err)

			assert.DeepEqual(t, testSecretProjection, secretCertProj)
		})

		t.Run("check custom secret projection", func(t *testing.T) {
			customSecretCertProj, err := r.reconcileClusterCertificate(ctx, initialRoot, cluster2, primaryService, replicaService)
			assert.NilError(t, err)

			assert.DeepEqual(t, customSecretProjection, customSecretCertProj)
		})

		t.Run("check switch to a custom secret projection", func(t *testing.T) {
			// simulate a new custom secret
			testSecret := &corev1.Secret{}
			testSecret.Namespace, testSecret.Name = namespace, "newcustomsecret"
			// simulate cluster spec update
			cluster2.Spec.CustomTLSSecret.LocalObjectReference.Name = "newcustomsecret"

			// get the expected secret projection
			testSecretProjection := clusterCertSecretProjection(testSecret)

			// reconcile the secret project using the normal process
			customSecretCertProj, err := r.reconcileClusterCertificate(ctx, initialRoot, cluster2, primaryService, replicaService)
			assert.NilError(t, err)

			// results should be the same
			assert.DeepEqual(t, testSecretProjection, customSecretCertProj)
		})

		t.Run("check cluster certificate secret", func(t *testing.T) {
			// get the cluster cert secret
			initialClusterCertSecret := &corev1.Secret{}
			err := tClient.Get(ctx, types.NamespacedName{
				Name:      fmt.Sprintf(naming.ClusterCertSecret, cluster1.Name),
				Namespace: namespace,
			}, initialClusterCertSecret)
			assert.NilError(t, err)

			// force the generation of a new root cert
			// create an empty secret and apply the change
			emptyRootSecret := &corev1.Secret{}
			emptyRootSecret.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))
			emptyRootSecret.Namespace, emptyRootSecret.Name = namespace, naming.RootCertSecret
			emptyRootSecret.Data = make(map[string][]byte)
			err = errors.WithStack(r.apply(ctx, emptyRootSecret))
			assert.NilError(t, err)

			// reconcile the root cert secret, creating a new root cert
			returnedRoot, err := r.reconcileRootCertificate(ctx, cluster1)
			assert.NilError(t, err)

			// pass in the new root, which should result in a new cluster cert
			_, err = r.reconcileClusterCertificate(ctx, returnedRoot, cluster1, primaryService, replicaService)
			assert.NilError(t, err)

			// get the new cluster cert secret
			newClusterCertSecret := &corev1.Secret{}
			err = tClient.Get(ctx, types.NamespacedName{
				Name:      fmt.Sprintf(naming.ClusterCertSecret, cluster1.Name),
				Namespace: namespace,
			}, newClusterCertSecret)
			assert.NilError(t, err)

			assert.Assert(t, !reflect.DeepEqual(initialClusterCertSecret, newClusterCertSecret))

			leaf := &pki.LeafCertificate{}
			assert.NilError(t, leaf.Certificate.UnmarshalText(newClusterCertSecret.Data["tls.crt"]))
			assert.NilError(t, leaf.PrivateKey.UnmarshalText(newClusterCertSecret.Data["tls.key"]))

			assert.Assert(t,
				strings.HasPrefix(leaf.Certificate.CommonName(), "the-primary."+namespace+".svc."),
				"got %q", leaf.Certificate.CommonName())

			if dnsNames := leaf.Certificate.DNSNames(); assert.Check(t, len(dnsNames) > 1) {
				assert.DeepEqual(t, dnsNames[1:4], []string{
					"the-primary." + namespace + ".svc",
					"the-primary." + namespace,
					"the-primary",
				})
				assert.DeepEqual(t, dnsNames[5:8], []string{
					"the-replicas." + namespace + ".svc",
					"the-replicas." + namespace,
					"the-replicas",
				})
			}
		})
	})
}

// getCertFromSecret returns a parsed certificate from the named secret
func getCertFromSecret(
	ctx context.Context, tClient client.Client, name, namespace, dataKey string,
) (*pki.Certificate, error) {
	// get cert secret
	secret := &corev1.Secret{}
	if err := tClient.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, secret); err != nil {
		return nil, err
	}

	// get the cert from the secret
	secretCRT, ok := secret.Data[dataKey]
	if !ok {
		return nil, errors.New(fmt.Sprintf("could not retrieve %s", dataKey))
	}

	// parse the cert from binary encoded data
	fromSecret := &pki.Certificate{}
	return fromSecret, fromSecret.UnmarshalText(secretCRT)
}
