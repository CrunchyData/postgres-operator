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

package pgbouncer

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	certBackendDirectory  = configDirectory + "/~postgres-operator-backend"
	certFrontendDirectory = configDirectory + "/~postgres-operator-frontend"

	certBackendAuthorityAbsolutePath   = certBackendDirectory + "/" + certBackendAuthorityProjectionPath
	certBackendAuthorityProjectionPath = "ca.crt"

	certFrontendAuthorityAbsolutePath  = certFrontendDirectory + "/" + certFrontendAuthorityProjectionPath
	certFrontendPrivateKeyAbsolutePath = certFrontendDirectory + "/" + certFrontendPrivateKeyProjectionPath
	certFrontendAbsolutePath           = certFrontendDirectory + "/" + certFrontendProjectionPath

	certFrontendAuthorityProjectionPath  = "ca.crt"
	certFrontendPrivateKeyProjectionPath = "tls.key"
	certFrontendProjectionPath           = "tls.crt"

	certFrontendAuthoritySecretKey  = "pgbouncer-frontend.ca-roots" // #nosec G101 this is a name, not a credential
	certFrontendPrivateKeySecretKey = "pgbouncer-frontend.key"      // #nosec G101 this is a name, not a credential
	certFrontendSecretKey           = "pgbouncer-frontend.crt"      // #nosec G101 this is a name, not a credential
)

// backendAuthority creates a volume projection of the PostgreSQL server
// certificate authority.
func backendAuthority(postgres *corev1.SecretProjection) corev1.VolumeProjection {
	var items []corev1.KeyToPath
	result := postgres.DeepCopy()

	for i := range result.Items {
		if result.Items[i].Path == certBackendAuthorityProjectionPath {
			items = append(items, result.Items[i])
		}
	}

	result.Items = items
	return corev1.VolumeProjection{Secret: result}
}

// frontendCertificate creates a volume projection of the PgBouncer certificate.
func frontendCertificate(
	custom *corev1.SecretProjection, secret *corev1.Secret,
) corev1.VolumeProjection {
	result := custom

	if result == nil {
		result = &corev1.SecretProjection{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: secret.Name,
			},
			Items: []corev1.KeyToPath{
				{
					Key:  certFrontendAuthoritySecretKey,
					Path: certFrontendAuthorityProjectionPath,
				},
				{
					Key:  certFrontendPrivateKeySecretKey,
					Path: certFrontendPrivateKeyProjectionPath,
				},
				{
					Key:  certFrontendSecretKey,
					Path: certFrontendProjectionPath,
				},
			},
		}
	}

	return corev1.VolumeProjection{Secret: result}
}
