//go:build envtest
// +build envtest

/*
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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
	"crypto/x509"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/pki"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

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

		t.Run("remove owner references after deleting first cluster", func(t *testing.T) {

			if !strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
				t.Skip("requires a running garbage collection controller")
			}

			err = tClient.Get(ctx, client.ObjectKeyFromObject(cluster1), cluster1)
			assert.NilError(t, err)

			err = tClient.Delete(ctx, cluster1)
			assert.NilError(t, err)

			err = wait.Poll(time.Second/2, Scale(time.Second*15), func() (bool, error) {
				err := tClient.Get(ctx, client.ObjectKeyFromObject(rootSecret), rootSecret)
				return len(rootSecret.ObjectMeta.OwnerReferences) == 1, err
			})
			assert.NilError(t, err)

			assert.Check(t, len(rootSecret.ObjectMeta.OwnerReferences) == 1, "owner reference not removed")

			expectedOR := metav1.OwnerReference{
				APIVersion: "postgres-operator.crunchydata.com/v1beta1",
				Kind:       "PostgresCluster",
				Name:       "hippocluster2",
				UID:        cluster2.UID,
			}

			if len(rootSecret.ObjectMeta.OwnerReferences) > 0 {
				assert.Equal(t, rootSecret.ObjectMeta.OwnerReferences[0], expectedOR)
			}
		})

		t.Run("root certificate is returned correctly", func(t *testing.T) {

			fromSecret, err := getCertFromSecret(ctx, tClient, naming.RootCertSecret, namespace, "root.crt")
			assert.NilError(t, err)

			// assert returned certificate matches the one created earlier
			assert.DeepEqual(t, fromSecret, initialRoot.Certificate)
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
			assert.Assert(t, !fromSecret.Equal(*initialRoot.Certificate))

			// check that the returned cert matches the cert from the secret
			assert.DeepEqual(t, fromSecret, returnedRoot.Certificate)
		})

		t.Run("root CA secret is deleted after final cluster is deleted", func(t *testing.T) {

			if !strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
				t.Skip("requires a running garbage collection controller")
			}

			err = tClient.Get(ctx, client.ObjectKeyFromObject(cluster2), cluster2)
			assert.NilError(t, err)

			err = tClient.Delete(ctx, cluster2)
			assert.NilError(t, err)

			err = wait.Poll(time.Second/2, Scale(time.Second*15), func() (bool, error) {
				if err := tClient.Get(ctx,
					client.ObjectKeyFromObject(rootSecret), rootSecret); apierrors.ReasonForError(err) == metav1.StatusReasonNotFound {
					return true, err
				}
				return false, nil
			})
			assert.Assert(t, apierrors.IsNotFound(err))
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

			certificate, err := pki.ParseCertificate(intent.Data["dns.crt"])
			assert.NilError(t, err)
			privateKey, err := pki.ParsePrivateKey(intent.Data["dns.key"])
			assert.NilError(t, err)

			assert.DeepEqual(t, certificate, initialLeafCert.Certificate)
			assert.DeepEqual(t, privateKey, initialLeafCert.PrivateKey)
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
			assert.Assert(t, !initialLeaf.Certificate.Equal(*newLeaf.Certificate))

			// 'reconcile' the certificate when the secret does not change. The returned leaf certificate should not change
			newLeaf2, err := r.instanceCertificate(ctx, instance, intent, intent, newRootCert)
			assert.NilError(t, err)

			// check that the leaf cert did not change after another reconciliation
			assert.DeepEqual(t, newLeaf2.Certificate, newLeaf.Certificate)

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
			secretCertProj, err := r.reconcileClusterCertificate(ctx, initialRoot, cluster1, primaryService)
			assert.NilError(t, err)

			assert.DeepEqual(t, testSecretProjection, secretCertProj)
		})

		t.Run("check custom secret projection", func(t *testing.T) {
			customSecretCertProj, err := r.reconcileClusterCertificate(ctx, initialRoot, cluster2, primaryService)
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
			customSecretCertProj, err := r.reconcileClusterCertificate(ctx, initialRoot, cluster2, primaryService)
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
			_, err = r.reconcileClusterCertificate(ctx, returnedRoot, cluster1, primaryService)
			assert.NilError(t, err)

			// get the new cluster cert secret
			newClusterCertSecret := &corev1.Secret{}
			err = tClient.Get(ctx, types.NamespacedName{
				Name:      fmt.Sprintf(naming.ClusterCertSecret, cluster1.Name),
				Namespace: namespace,
			}, newClusterCertSecret)
			assert.NilError(t, err)

			assert.Assert(t, !reflect.DeepEqual(initialClusterCertSecret, newClusterCertSecret))

			pkiCert, err := pki.ParseCertificate(newClusterCertSecret.Data["tls.crt"])
			assert.NilError(t, err)

			x509Cert, err := x509.ParseCertificate(pkiCert.Certificate)
			assert.NilError(t, err)
			assert.Assert(t,
				strings.HasPrefix(x509Cert.Subject.CommonName, "the-primary."+namespace+".svc."),
				"got %q", x509Cert.Subject.CommonName)

			if assert.Check(t, len(x509Cert.DNSNames) > 1) {
				assert.DeepEqual(t, x509Cert.DNSNames[1:], []string{
					"the-primary." + namespace + ".svc",
					"the-primary." + namespace,
					"the-primary",
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
	if fromSecret, err := pki.ParseCertificate(secretCRT); fromSecret == nil || err != nil {
		return nil, fmt.Errorf("error parsing %s", dataKey)
	} else {
		return fromSecret, nil
	}
}
