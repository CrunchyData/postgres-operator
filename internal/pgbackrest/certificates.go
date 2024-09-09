// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgbackrest

import (
	"encoding"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/internal/initialize"
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

// certFile concatenates the results of multiple PEM-encoding marshalers.
func certFile(texts ...encoding.TextMarshaler) ([]byte, error) {
	var out []byte

	for i := range texts {
		if b, err := texts[i].MarshalText(); err == nil {
			out = append(out, b...)
		} else {
			return nil, err
		}
	}

	return out, nil
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
