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

package patroni

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/internal/pki"
)

const (
	certAuthorityConfigPath = "~postgres-operator/patroni.ca-roots"
	certServerConfigPath    = "~postgres-operator/patroni.crt+key"

	certAuthorityFileKey = "patroni.ca-roots"
	certServerFileKey    = "patroni.crt-combined"
)

// certAuthorities encodes roots in a format suitable for Patroni's TLS verification.
func certAuthorities(roots ...*pki.Certificate) ([]byte, error) {
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

// certFile encodes cert and key as a combination suitable for
// Patroni's TLS identification. It can be used by both the client and the server.
func certFile(key *pki.PrivateKey, cert *pki.Certificate) ([]byte, error) {
	var out []byte

	if b, err := key.MarshalText(); err == nil {
		out = append(out, b...)
	} else {
		return nil, err
	}

	if b, err := cert.MarshalText(); err == nil {
		out = append(out, b...)
	} else {
		return nil, err
	}

	return out, nil
}

// instanceCertificates returns projections of Patroni's CAs, keys, and
// certificates to include in the instance configuration volume.
func instanceCertificates(certificates *corev1.Secret) []corev1.VolumeProjection {
	return []corev1.VolumeProjection{{
		Secret: &corev1.SecretProjection{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: certificates.Name,
			},
			Items: []corev1.KeyToPath{
				{
					Key:  certAuthorityFileKey,
					Path: certAuthorityConfigPath,
				},
				{
					Key:  certServerFileKey,
					Path: certServerConfigPath,
				},
			},
		},
	}}
}
