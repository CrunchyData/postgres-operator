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

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/pki"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

const (
	// https://www.postgresql.org/docs/current/ssl-tcp.html
	clusterCertFile = "tls.crt"
	clusterKeyFile  = "tls.key"
	rootCertFile    = "ca.crt"
)

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get
// +kubebuilder:rbac:groups="",resources=secrets,verbs=create;patch

// reconcileRootCertificate ensures the root certificate, stored
// in the relevant secret, has been created and is not 'bad' due
// to being expired, formatted incorrectly, etc.
// If it is bad for some reason, a new root certificate is
// generated for use.
func (r *Reconciler) reconcileRootCertificate(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
) (
	*pki.RootCertificateAuthority, error,
) {
	const keyCertificate, keyPrivateKey = "root.crt", "root.key"

	existing := &corev1.Secret{}
	existing.Namespace, existing.Name = cluster.Namespace, naming.RootCertSecret
	err := errors.WithStack(client.IgnoreNotFound(
		r.Client.Get(ctx, client.ObjectKeyFromObject(existing), existing)))

	root := pki.NewRootCertificateAuthority()

	if data, ok := existing.Data[keyCertificate]; err == nil && ok {
		root.Certificate, err = pki.ParseCertificate(data)
		err = errors.WithStack(err)
	}
	if data, ok := existing.Data[keyPrivateKey]; err == nil && ok {
		root.PrivateKey, err = pki.ParsePrivateKey(data)
		err = errors.WithStack(err)
	}

	// if there is an error or the root CA is bad, generate a new one
	if err != nil || pki.RootCAIsBad(root) {
		err = errors.WithStack(root.Generate())
	}

	intent := &corev1.Secret{}
	intent.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))
	intent.Namespace, intent.Name = cluster.Namespace, naming.RootCertSecret
	intent.Data = make(map[string][]byte)
	intent.ObjectMeta.OwnerReferences = existing.ObjectMeta.OwnerReferences

	// A root secret is scoped to the namespace where postgrescluster(s)
	// are deployed. For operator deployments with postgresclusters in more than
	// one namespace, there will be one root per namespace.
	// During reconciliation, the owner reference block of the root secret is
	// updated to include the postgrescluster as an owner.
	// However, unlike the leaf certificate, the postgrescluster will not be
	// set as the controller. This allows for multiple owners to guide garbage
	// collection, but avoids any errors related to setting multiple controllers.
	// https://docs.k8s.io/concepts/workloads/controllers/garbage-collection/#owners-and-dependents
	if err == nil {
		err = errors.WithStack(r.setOwnerReference(cluster, intent))
	}
	if err == nil {
		intent.Data[keyCertificate], err = root.Certificate.MarshalText()
		err = errors.WithStack(err)
	}
	if err == nil {
		intent.Data[keyPrivateKey], err = root.PrivateKey.MarshalText()
		err = errors.WithStack(err)
	}
	if err == nil {
		err = errors.WithStack(r.apply(ctx, intent))
	}

	return root, err
}

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get
// +kubebuilder:rbac:groups="",resources=secrets,verbs=create;patch

// reconcileClusterCertificate first checks if a custom certificate
// secret is configured. If so, that secret projection is returned.
// Otherwise, a secret containing a generated leaf certificate, stored in
// the relevant secret, has been created and is not 'bad' due to being
// expired, formatted incorrectly, etc. If it is bad for any reason, a new
// leaf certificate is generated using the current root certificate.
// In either case, the relevant secret is expected to contain three files:
// tls.crt, tls.key and ca.crt which are the TLS certificate, private key
// and CA certificate, respectively.
func (r *Reconciler) reconcileClusterCertificate(
	ctx context.Context, rootCACert *pki.RootCertificateAuthority,
	cluster *v1beta1.PostgresCluster, primaryService *corev1.Service,
) (
	*corev1.SecretProjection, error,
) {
	// if a custom postgrescluster secret is provided, just return it
	if cluster.Spec.CustomTLSSecret != nil {
		return cluster.Spec.CustomTLSSecret, nil
	}

	const keyCertificate, keyPrivateKey, rootCA = "tls.crt", "tls.key", "ca.crt"

	existing := &corev1.Secret{ObjectMeta: naming.PostgresTLSSecret(cluster)}
	err := errors.WithStack(client.IgnoreNotFound(
		r.Client.Get(ctx, client.ObjectKeyFromObject(existing), existing)))

	leaf := pki.NewLeafCertificate("", nil, nil)
	leaf.DNSNames = naming.ServiceDNSNames(ctx, primaryService)
	leaf.CommonName = leaf.DNSNames[0] // FQDN

	if data, ok := existing.Data[keyCertificate]; err == nil && ok {
		leaf.Certificate, err = pki.ParseCertificate(data)
		err = errors.WithStack(err)
	}
	if data, ok := existing.Data[keyPrivateKey]; err == nil && ok {
		leaf.PrivateKey, err = pki.ParsePrivateKey(data)
		err = errors.WithStack(err)
	}

	// if there is an error or the leaf certificate is bad, generate a new one
	if err != nil || pki.LeafCertIsBad(ctx, leaf, rootCACert, cluster.Namespace) {
		err = errors.WithStack(leaf.Generate(rootCACert))
	}

	intent := &corev1.Secret{ObjectMeta: naming.PostgresTLSSecret(cluster)}
	intent.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))
	intent.Data = make(map[string][]byte)
	intent.ObjectMeta.OwnerReferences = existing.ObjectMeta.OwnerReferences

	intent.Annotations = naming.Merge(cluster.Spec.Metadata.GetAnnotationsOrNil())
	intent.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster:            cluster.Name,
			naming.LabelClusterCertificate: "postgres-tls",
		})

	if err == nil {
		err = errors.WithStack(r.setControllerReference(cluster, intent))
	}

	if err == nil {
		intent.Data[keyCertificate], err = leaf.Certificate.MarshalText()
		err = errors.WithStack(err)
	}
	if err == nil {
		intent.Data[keyPrivateKey], err = leaf.PrivateKey.MarshalText()
		err = errors.WithStack(err)
	}
	if err == nil {
		intent.Data[rootCA], err = rootCACert.Certificate.MarshalText()
		err = errors.WithStack(err)
	}

	// TODO(tjmoore4): The generated postgrescluster secret is only created
	// when a custom secret is not specified. However, if the secret is
	// initially created and a custom secret is later used, the generated
	// secret is currently left in place.
	if err == nil {
		err = errors.WithStack(r.apply(ctx, intent))
	}

	return clusterCertSecretProjection(intent), err
}

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get
// +kubebuilder:rbac:groups="",resources=secrets,verbs=create;patch

// instanceCertificate populates intent with the DNS leaf certificate and
// returns it. It also ensures the leaf certificate, stored in the relevant
// secret, has been created and is not 'bad' due to being expired, formatted
// incorrectly, etc. In addition, a check is made to ensure the leaf cert's
// authority key ID matches the corresponding root cert's subject
// key ID (i.e. the root cert is the 'parent' of the leaf cert).
// If it is bad for any reason, a new leaf certificate is generated
// using the current root certificate
func (*Reconciler) instanceCertificate(
	ctx context.Context, instance *appsv1.StatefulSet,
	existing, intent *corev1.Secret, rootCACert *pki.RootCertificateAuthority,
) (
	*pki.LeafCertificate, error,
) {
	var err error
	const keyCertificate, keyPrivateKey = "dns.crt", "dns.key"

	// RFC 2818 states that the certificate DNS names must be used to verify
	// HTTPS identity.
	leaf := pki.NewLeafCertificate("", nil, nil)
	leaf.DNSNames = naming.InstancePodDNSNames(ctx, instance)
	leaf.CommonName = leaf.DNSNames[0] // FQDN

	if data, ok := existing.Data[keyCertificate]; err == nil && ok {
		leaf.Certificate, err = pki.ParseCertificate(data)
		err = errors.WithStack(err)
	}
	if data, ok := existing.Data[keyPrivateKey]; err == nil && ok {
		leaf.PrivateKey, err = pki.ParsePrivateKey(data)
		err = errors.WithStack(err)
	}

	// if there is an error or the leaf certificate is bad, generate a new one
	if err != nil || pki.LeafCertIsBad(ctx, leaf, rootCACert, instance.Namespace) {
		err = errors.WithStack(leaf.Generate(rootCACert))
	}

	if err == nil {
		intent.Data[keyCertificate], err = leaf.Certificate.MarshalText()
		err = errors.WithStack(err)
	}
	if err == nil {
		intent.Data[keyPrivateKey], err = leaf.PrivateKey.MarshalText()
		err = errors.WithStack(err)
	}

	return leaf, err
}

// clusterCertSecretProjection returns a secret projection of the postgrescluster's
// CA, key, and certificate to include in the instance configuration volume.
func clusterCertSecretProjection(certificate *corev1.Secret) *corev1.SecretProjection {
	return &corev1.SecretProjection{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: certificate.Name,
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
}
