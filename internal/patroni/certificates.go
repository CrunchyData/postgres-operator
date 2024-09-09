// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package patroni

import (
	"encoding"

	corev1 "k8s.io/api/core/v1"
)

const (
	certAuthorityConfigPath = "~postgres-operator/patroni.ca-roots"
	certServerConfigPath    = "~postgres-operator/patroni.crt+key"

	certAuthorityFileKey = "patroni.ca-roots"
	certServerFileKey    = "patroni.crt-combined"
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
