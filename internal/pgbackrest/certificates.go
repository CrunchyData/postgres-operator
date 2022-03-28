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

package pgbackrest

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/pki"
)

const (
	certAuthorityAbsolutePath        = configDirectory + "/" + certAuthorityProjectionPath
	certClientPrivateKeyAbsolutePath = configDirectory + "/" + certClientPrivateKeyProjectionPath
	certClientAbsolutePath           = configDirectory + "/" + certClientProjectionPath
	certServerPrivateKeyAbsolutePath = serverMountPath + "/" + certServerPrivateKeyProjectionPath
	certServerAbsolutePath           = serverMountPath + "/" + certServerProjectionPath

	certAuthorityProjectionPath        = "~postgres-operator/tls-ca.crt"
	certClientPrivateKeyProjectionPath = "~postgres-operator/client-tls.key"
	certClientProjectionPath           = "~postgres-operator/client-tls.crt"
	certServerPrivateKeyProjectionPath = "server-tls.key"
	certServerProjectionPath           = "server-tls.crt"

	certAuthoritySecretKey        = "pgbackrest.ca-roots"   // #nosec G101 this is a name, not a credential
	certClientPrivateKeySecretKey = "pgbackrest-client.key" // #nosec G101 this is a name, not a credential
	certClientSecretKey           = "pgbackrest-client.crt" // #nosec G101 this is a name, not a credential

	certInstancePrivateKeySecretKey = "pgbackrest-server.key"
	certInstanceSecretKey           = "pgbackrest-server.crt"

	certRepoPrivateKeySecretKey = "pgbackrest-repo-host.key" // #nosec G101 this is a name, not a credential
	certRepoSecretKey           = "pgbackrest-repo-host.crt" // #nosec G101 this is a name, not a credential
)

// certAuthorities returns the PEM encoding of roots that can be read by OpenSSL
// for TLS verification.
func certAuthorities(roots ...pki.Certificate) ([]byte, error) {
	var out []byte

	for i := range roots {
		if b, err := roots[i].MarshalText(); err == nil {
			out = append(out, b...)
		} else {
			return nil, err
		}
	}

	return out, nil
}

// certFile returns the PEM encoding of cert that can be read by OpenSSL
// for TLS identification.
func certFile(cert pki.Certificate) ([]byte, error) {
	return cert.MarshalText()
}

// certPrivateKey returns the PEM encoding of key that can be read by OpenSSL
// for TLS identification.
func certPrivateKey(key pki.PrivateKey) ([]byte, error) {
	return key.MarshalText()
}

// clientCertificates returns projections of CAs, keys, and certificates to
// include in a configuration volume from the pgBackRest Secret.
func clientCertificates() []corev1.KeyToPath {
	return []corev1.KeyToPath{
		{
			Key:  certAuthoritySecretKey,
			Path: certAuthorityProjectionPath,
		},
		{
			Key:  certClientSecretKey,
			Path: certClientProjectionPath,
		},
		{
			Key:  certClientPrivateKeySecretKey,
			Path: certClientPrivateKeyProjectionPath,

			// pgBackRest requires that certificate keys not be readable by any
			// other user.
			// - https://github.com/pgbackrest/pgbackrest/blob/release/2.38/src/common/io/tls/common.c#L128
			Mode: initialize.Int32(0o600),
		},
	}
}

// clientCommonName returns a client certificate common name (CN) for cluster.
func clientCommonName(cluster metav1.Object) string {
	// The common name (ASN.1 OID 2.5.4.3) of a certificate must be
	// 64 characters or less. ObjectMeta.UID is a UUID in its 36-character
	// string representation.
	// - https://tools.ietf.org/html/rfc5280#appendix-A
	// - https://docs.k8s.io/concepts/overview/working-with-objects/names/#uids
	// - https://releases.k8s.io/v1.22.0/staging/src/k8s.io/apiserver/pkg/registry/rest/create.go#L111
	// - https://releases.k8s.io/v1.22.0/staging/src/k8s.io/apiserver/pkg/registry/rest/meta.go#L30
	return "pgbackrest@" + string(cluster.GetUID())
}

// instanceServerCertificates returns projections of keys and certificates to
// include in a server volume from an instance Secret.
func instanceServerCertificates() []corev1.KeyToPath {
	return []corev1.KeyToPath{
		{
			Key:  certInstanceSecretKey,
			Path: certServerProjectionPath,
		},
		{
			Key:  certInstancePrivateKeySecretKey,
			Path: certServerPrivateKeyProjectionPath,

			// pgBackRest requires that certificate keys not be readable by any
			// other user.
			// - https://github.com/pgbackrest/pgbackrest/blob/release/2.38/src/common/io/tls/common.c#L128
			Mode: initialize.Int32(0o600),
		},
	}
}

// repositoryServerCertificates returns projections of keys and certificates to
// include in a server volume from the pgBackRest Secret.
func repositoryServerCertificates() []corev1.KeyToPath {
	return []corev1.KeyToPath{
		{
			Key:  certRepoSecretKey,
			Path: certServerProjectionPath,
		},
		{
			Key:  certRepoPrivateKeySecretKey,
			Path: certServerPrivateKeyProjectionPath,

			// pgBackRest requires that certificate keys not be readable by any
			// other user.
			// - https://github.com/pgbackrest/pgbackrest/blob/release/2.38/src/common/io/tls/common.c#L128
			Mode: initialize.Int32(0o600),
		},
	}
}
