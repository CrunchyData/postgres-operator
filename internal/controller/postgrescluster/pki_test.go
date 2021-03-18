// +build envtest

/*
 Copyright 2021 Crunchy Data Solutions, Inc.
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
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/pki"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
)

func TestReconcileCerts(t *testing.T) {
	// setup the test environment and ensure a clean teardown
	tEnv, tClient, cfg := setupTestEnv(t, ControllerName)
	ctx := context.Background()
	// set namespace name
	ns := &v1.Namespace{}
	ns.GenerateName = "postgres-operator-test-"
	assert.NilError(t, tClient.Create(ctx, ns))
	t.Cleanup(func() {
		assert.Check(t, tClient.Delete(ctx, ns))
		teardownTestEnv(t, tEnv)
	})
	namespace := ns.Name

	testScheme := runtime.NewScheme()
	scheme.AddToScheme(testScheme)
	v1alpha1.AddToScheme(testScheme)

	// set up a non-cached client
	newClient, err := client.New(cfg, client.Options{Scheme: testScheme})
	assert.NilError(t, err)

	r := &Reconciler{
		Client: newClient,
		Owner:  ControllerName,
	}

	// set up cluster1
	clusterName1 := "hippocluster1"

	// set up test cluster1
	cluster1 := &v1alpha1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName1,
			Namespace: namespace,
		},
		Spec: v1alpha1.PostgresClusterSpec{
			PostgresVersion: 12,
			InstanceSets:    []v1alpha1.PostgresInstanceSetSpec{},
		},
	}
	cluster1.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("postgrescluster"))
	if err := tClient.Create(ctx, cluster1); err != nil {
		t.Error(err)
	}

	// set up test cluster2
	cluster2Name := "hippocluster2"

	cluster2 := &v1alpha1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster2Name,
			Namespace: namespace,
		},
		Spec: v1alpha1.PostgresClusterSpec{
			PostgresVersion: 12,
			InstanceSets:    []v1alpha1.PostgresInstanceSetSpec{},
		},
	}
	cluster2.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("postgrescluster"))
	if err := tClient.Create(ctx, cluster2); err != nil {
		t.Error(err)
	}

	t.Run("check root certificate reconciliation", func(t *testing.T) {

		initialRoot, err := r.reconcileRootCertificate(ctx, cluster1, namespace)
		assert.NilError(t, err)

		rootSecret := &v1.Secret{}
		rootSecret.Namespace, rootSecret.Name = namespace, naming.RootCertSecret
		rootSecret.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Secret"))

		t.Run("check root CA secret first owner reference", func(t *testing.T) {

			err := tClient.Get(ctx, client.ObjectKeyFromObject(rootSecret), rootSecret)
			assert.NilError(t, err)

			assert.Check(t, len(rootSecret.ObjectMeta.OwnerReferences) == 1, "first owner reference not set")

			expectedOR := metav1.OwnerReference{
				APIVersion: "postgres-operator.crunchydata.com/v1alpha1",
				Kind:       "PostgresCluster",
				Name:       "hippocluster1",
				UID:        cluster1.UID,
			}

			if len(rootSecret.ObjectMeta.OwnerReferences) > 0 {
				assert.Equal(t, rootSecret.ObjectMeta.OwnerReferences[0], expectedOR)
			}
		})

		t.Run("check root CA secret second owner reference", func(t *testing.T) {

			_, err := r.reconcileRootCertificate(ctx, cluster2, namespace)
			assert.NilError(t, err)

			err = tClient.Get(ctx, client.ObjectKeyFromObject(rootSecret), rootSecret)
			assert.NilError(t, err)

			clist := &v1alpha1.PostgresClusterList{}
			tClient.List(ctx, clist)

			assert.Check(t, len(rootSecret.ObjectMeta.OwnerReferences) == 2, "second owner reference not set")

			expectedOR := metav1.OwnerReference{
				APIVersion: "postgres-operator.crunchydata.com/v1alpha1",
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

			err = wait.Poll(time.Second/2, time.Second*15, func() (bool, error) {
				if err := tClient.Get(ctx,
					client.ObjectKeyFromObject(rootSecret), rootSecret); len(rootSecret.ObjectMeta.OwnerReferences) == 1 {
					return true, err
				}
				return false, nil
			})
			assert.NilError(t, err)

			assert.Check(t, len(rootSecret.ObjectMeta.OwnerReferences) == 1, "owner reference not removed")

			expectedOR := metav1.OwnerReference{
				APIVersion: "postgres-operator.crunchydata.com/v1alpha1",
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
			assert.Assert(t, bytes.Equal(fromSecret.Certificate, initialRoot.Certificate.Certificate))
		})

		t.Run("root certificate changes", func(t *testing.T) {
			// force the generation of a new root cert
			// create an empty secret and apply the change
			emptyRootSecret := &v1.Secret{}
			emptyRootSecret.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Secret"))
			emptyRootSecret.Namespace, emptyRootSecret.Name = namespace, naming.RootCertSecret
			emptyRootSecret.Data = make(map[string][]byte)
			err = errors.WithStack(r.apply(ctx, emptyRootSecret))
			assert.NilError(t, err)

			// reconcile the root cert secret, creating a new root cert
			returnedRoot, err := r.reconcileRootCertificate(ctx, cluster1, namespace)
			assert.NilError(t, err)

			fromSecret, err := getCertFromSecret(ctx, tClient, naming.RootCertSecret, namespace, "root.crt")
			assert.NilError(t, err)

			// check that the cert from the secret does not equal the initial certificate
			assert.Assert(t, !bytes.Equal(fromSecret.Certificate, initialRoot.Certificate.Certificate))

			// check that the returned cert matches the cert from the secret
			assert.Assert(t, bytes.Equal(fromSecret.Certificate, returnedRoot.Certificate.Certificate))
		})

		t.Run("root CA secret is deleted after final cluster is deleted", func(t *testing.T) {

			if !strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
				t.Skip("requires a running garbage collection controller")
			}

			err = tClient.Get(ctx, client.ObjectKeyFromObject(cluster2), cluster2)
			assert.NilError(t, err)

			err = tClient.Delete(ctx, cluster2)
			assert.NilError(t, err)

			err = wait.Poll(time.Second/2, time.Second*15, func() (bool, error) {
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

		initialRoot, err := r.reconcileRootCertificate(ctx, cluster1, namespace)
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

		intent, existing, err := createInstanceSecrets(ctx, tClient, instance, initialRoot)

		// apply the secret changes
		err = errors.WithStack(r.apply(ctx, existing))
		assert.NilError(t, err)

		initialLeafCert, err := r.instanceCertificate(ctx, instance, existing, intent, initialRoot)
		assert.NilError(t, err)

		t.Run("check leaf certificate in secret", func(t *testing.T) {

			fromSecret, err := getCertFromSecret(ctx, tClient, instance.GetName()+"-certs", namespace, "dns.crt")
			assert.NilError(t, err)

			// assert returned certificate matches the one created earlier
			assert.Assert(t, bytes.Equal(fromSecret.Certificate, initialLeafCert.Certificate.Certificate))
		})

		t.Run("check that the leaf certs update when root changes", func(t *testing.T) {

			// force the generation of a new root cert
			// create an empty secret and apply the change
			emptyRootSecret := &v1.Secret{}
			emptyRootSecret.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Secret"))
			emptyRootSecret.Namespace, emptyRootSecret.Name = namespace, naming.RootCertSecret
			emptyRootSecret.Data = make(map[string][]byte)
			err = errors.WithStack(r.apply(ctx, emptyRootSecret))

			// reconcile the root cert secret
			newRootCert, err := r.reconcileRootCertificate(ctx, cluster1, namespace)
			assert.NilError(t, err)

			// get the existing leaf/instance secret which will receive a new certificate during reconciliation
			existingInstanceSecret := &v1.Secret{}
			err = tClient.Get(ctx, types.NamespacedName{
				Name:      instance.GetName() + "-certs",
				Namespace: namespace,
			}, existingInstanceSecret)

			// create an empty 'intent' secret for the reconcile function
			instanceIntentSecret := &v1.Secret{ObjectMeta: naming.InstanceCertificates(instance)}
			instanceIntentSecret.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Secret"))
			instanceIntentSecret.Type = v1.SecretTypeOpaque
			instanceIntentSecret.Data = make(map[string][]byte)

			// save a copy of the 'pre-reconciled' certificate
			oldLeafFromSecret, err := getCertFromSecret(ctx, tClient, instance.GetName()+"-certs", namespace, "dns.crt")
			assert.NilError(t, err)

			// reconcile the certificate
			newLeaf, err := r.instanceCertificate(ctx, instance, existingInstanceSecret, instanceIntentSecret, newRootCert)
			assert.NilError(t, err)

			// assert old leaf cert does not match the newly reconciled one
			assert.Assert(t, !bytes.Equal(oldLeafFromSecret.Certificate, newLeaf.Certificate.Certificate))

			// 'reconcile' the certificate when the secret does not change. The returned leaf certificate should not change
			newLeaf2, err := r.instanceCertificate(ctx, instance, instanceIntentSecret, instanceIntentSecret, newRootCert)
			assert.NilError(t, err)

			// check that the leaf cert did not change after another reconciliation
			assert.Assert(t, bytes.Equal(newLeaf2.Certificate.Certificate, newLeaf.Certificate.Certificate))

		})

	})
}

// getCertFromSecret returns a parsed certificate from the named secret
func getCertFromSecret(
	ctx context.Context, tClient client.Client, name, namespace, dataKey string,
) (*pki.Certificate, error) {
	// get cert secret
	secret := &v1.Secret{}
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

// createInstanceSecrets creates the two initial leaf instance secrets for use when
// testing the leaf cert reconciliation
func createInstanceSecrets(
	ctx context.Context, tClient client.Client, instance *appsv1.StatefulSet,
	rootCA *pki.RootCertificateAuthority,
) (*v1.Secret, *v1.Secret, error) {
	// create two secret structs for reconciliation
	intent := &v1.Secret{ObjectMeta: naming.InstanceCertificates(instance)}
	existing := &v1.Secret{ObjectMeta: naming.InstanceCertificates(instance)}

	// populate the 'intent' secret
	err := errors.WithStack(client.IgnoreNotFound(
		tClient.Get(ctx, client.ObjectKeyFromObject(intent), intent)))
	intent.Data = make(map[string][]byte)
	if err != nil {
		return intent, existing, err
	}

	// generate a leaf cert for the 'existing' secret
	leafCert := pki.NewLeafCertificate("", nil, nil)
	leafCert.DNSNames = naming.InstancePodDNSNames(ctx, instance)
	leafCert.CommonName = leafCert.DNSNames[0] // FQDN
	err = errors.WithStack(leafCert.Generate(rootCA))
	if err != nil {
		return intent, existing, err
	}

	// populate the 'existing' secret
	existing.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Secret"))
	existing.Data = make(map[string][]byte)

	if err == nil {
		existing.Data["dns.crt"], err = leafCert.Certificate.MarshalText()
		err = errors.WithStack(err)
	}
	if err != nil {
		return intent, existing, err
	}

	if err == nil {
		existing.Data["dns.key"], err = leafCert.PrivateKey.MarshalText()
		err = errors.WithStack(err)
	}

	return intent, existing, err
}
