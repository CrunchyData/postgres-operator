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

// reconcileRootCertificate ensures the root certificate, stored
// in the relevant secret, has been created and is not 'bad' due
// to being expired, formatted incorrectly, etc.
// If it is bad for some reason, a new root certificate is
// generated for use.
func (r *Reconciler) reconcileRootCertificate(
	ctx context.Context, namespace string,
) (
	*pki.RootCertificateAuthority, error,
) {
	const keyCertificate, keyPrivateKey = "root.crt", "root.key"

	existing := &v1.Secret{}
	existing.Namespace, existing.Name = namespace, naming.RootCertSecret
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

	intent := &v1.Secret{}
	intent.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Secret"))
	intent.Namespace, intent.Name = namespace, naming.RootCertSecret
	intent.Data = make(map[string][]byte)

	// TODO(cbandy/tjmoore4): Ownership, Controller.
	// The postgrescluster controller is likely incorrect as the owner
	// of the root secrets. Among other reasons, there is the potential
	// that a root certificate secret is in a different namespace than
	// the last postgrescluster to trigger its reconciliation
	// k8s.io/docs/concepts/workloads/controllers/garbage-collection/#owners-and-dependents

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
	existing, intent *v1.Secret, rootCACert *pki.RootCertificateAuthority,
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
