// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

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

// +kubebuilder:rbac:groups="",resources="secrets",verbs={get}
// +kubebuilder:rbac:groups="",resources="secrets",verbs={create,patch}

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

	root := &pki.RootCertificateAuthority{}

	if err == nil {
		// Unmarshal and validate the stored root. These first errors can
		// be ignored because they result in an invalid root which is then
		// correctly regenerated.
		_ = root.Certificate.UnmarshalText(existing.Data[keyCertificate])
		_ = root.PrivateKey.UnmarshalText(existing.Data[keyPrivateKey])

		if !pki.RootIsValid(root) {
			root, err = pki.NewRootCertificateAuthority()
			err = errors.WithStack(err)
		}
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

// +kubebuilder:rbac:groups="",resources="secrets",verbs={get}
// +kubebuilder:rbac:groups="",resources="secrets",verbs={create,patch}

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
	ctx context.Context, root *pki.RootCertificateAuthority,
	cluster *v1beta1.PostgresCluster, primaryService *corev1.Service,
	replicaService *corev1.Service,
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

	leaf := &pki.LeafCertificate{}
	dnsNames := append(naming.ServiceDNSNames(ctx, primaryService), naming.ServiceDNSNames(ctx, replicaService)...)
	dnsFQDN := dnsNames[0]

	if err == nil {
		// Unmarshal and validate the stored leaf. These first errors can
		// be ignored because they result in an invalid leaf which is then
		// correctly regenerated.
		_ = leaf.Certificate.UnmarshalText(existing.Data[keyCertificate])
		_ = leaf.PrivateKey.UnmarshalText(existing.Data[keyPrivateKey])

		leaf, err = root.RegenerateLeafWhenNecessary(leaf, dnsFQDN, dnsNames)
		err = errors.WithStack(err)
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
		intent.Data[rootCA], err = root.Certificate.MarshalText()
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

// +kubebuilder:rbac:groups="",resources="secrets",verbs={get}
// +kubebuilder:rbac:groups="",resources="secrets",verbs={create,patch}

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
	existing, intent *corev1.Secret, root *pki.RootCertificateAuthority,
) (
	*pki.LeafCertificate, error,
) {
	var err error
	const keyCertificate, keyPrivateKey = "dns.crt", "dns.key"

	leaf := &pki.LeafCertificate{}

	// RFC 2818 states that the certificate DNS names must be used to verify
	// HTTPS identity.
	dnsNames := naming.InstancePodDNSNames(ctx, instance)
	dnsFQDN := dnsNames[0]

	if err == nil {
		// Unmarshal and validate the stored leaf. These first errors can
		// be ignored because they result in an invalid leaf which is then
		// correctly regenerated.
		_ = leaf.Certificate.UnmarshalText(existing.Data[keyCertificate])
		_ = leaf.PrivateKey.UnmarshalText(existing.Data[keyPrivateKey])

		leaf, err = root.RegenerateLeafWhenNecessary(leaf, dnsFQDN, dnsNames)
		err = errors.WithStack(err)
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
