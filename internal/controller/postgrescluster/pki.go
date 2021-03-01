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
	"context"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/pki"
)

// +kubebuilder:rbac:resources=secrets,verbs=get;patch

// reconcileRootCertificate. TODO(cbandy): This belongs somewhere else.
func (r *Reconciler) reconcileRootCertificate(
	ctx context.Context, namespace string, // FIXME
) (
	*pki.RootCertificateAuthority, error,
) {
	const keyCertificate, keyPrivateKey = "root.crt", "root.key"

	existing := &v1.Secret{}
	existing.Namespace, existing.Name = namespace, "root" // FIXME
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

	// TODO(cbandy): check that the certificate is still good or something.
	if err == nil && (root.Certificate == nil || root.PrivateKey == nil) {
		err = errors.WithStack(root.Generate())
	}

	intent := &v1.Secret{}
	intent.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Secret"))
	intent.Namespace, intent.Name = namespace, "root" // FIXME
	intent.Data = make(map[string][]byte)

	// TODO(cbandy): Ownership, Controller.

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

// +kubebuilder:rbac:resources=secrets,verbs=get;patch

// reconcileNamespaceCertificate. TODO(cbandy): This belongs somewhere else.
func (r *Reconciler) reconcileNamespaceCertificate(
	ctx context.Context, namespace string, root *pki.RootCertificateAuthority,
) (
	*pki.IntermediateCertificateAuthority, error,
) {
	const keyCertificate, keyPrivateKey = "intermediate.crt", "intermediate.key"

	existing := &v1.Secret{}
	existing.Namespace, existing.Name = namespace, "intermediate" // FIXME
	err := errors.WithStack(client.IgnoreNotFound(
		r.Client.Get(ctx, client.ObjectKeyFromObject(existing), existing)))

	ca := pki.NewIntermediateCertificateAuthority(namespace)

	if data, ok := existing.Data[keyCertificate]; err == nil && ok {
		ca.Certificate, err = pki.ParseCertificate(data)
		err = errors.WithStack(err)
	}
	if data, ok := existing.Data[keyPrivateKey]; err == nil && ok {
		ca.PrivateKey, err = pki.ParsePrivateKey(data)
		err = errors.WithStack(err)
	}

	// TODO(cbandy): check that the certificate is still good or something.
	// - names, expiration, issuer.
	if err == nil && (ca.Certificate == nil || ca.PrivateKey == nil) {
		err = errors.WithStack(ca.Generate(root))
	}

	intent := &v1.Secret{}
	intent.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Secret"))
	intent.Namespace, intent.Name = namespace, "intermediate" // FIXME
	intent.Data = make(map[string][]byte)

	// TODO(cbandy): Ownership, Controller.

	if err == nil {
		intent.Data[keyCertificate], err = ca.Certificate.MarshalText()
		err = errors.WithStack(err)
	}
	if err == nil {
		intent.Data[keyPrivateKey], err = ca.PrivateKey.MarshalText()
		err = errors.WithStack(err)
	}
	if err == nil {
		err = errors.WithStack(r.apply(ctx, intent))
	}

	return ca, err
}

// +kubebuilder:rbac:resources=secrets,verbs=get;patch

// instanceCertificate populates intent with the DNS leaf certificate and
// returns it.
func (*Reconciler) instanceCertificate(
	ctx context.Context, instance *appsv1.StatefulSet,
	existing, intent *v1.Secret, ca *pki.IntermediateCertificateAuthority,
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

	// TODO(cbandy): check that the certificate is still good or something.
	// - names, expiration, issuer.
	if err == nil && (leaf.Certificate == nil || leaf.PrivateKey == nil) {
		err = errors.WithStack(leaf.Generate(ca))
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
