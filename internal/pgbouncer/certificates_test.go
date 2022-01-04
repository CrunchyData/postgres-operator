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

package pgbouncer

import (
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestBackendAuthority(t *testing.T) {
	// No items; assume Key matches Path.
	projection := &corev1.SecretProjection{
		LocalObjectReference: corev1.LocalObjectReference{Name: "some-name"},
	}
	assert.Assert(t, marshalMatches(backendAuthority(projection), `
secret:
  items:
  - key: ca.crt
    path: ~postgres-operator/backend-ca.crt
  name: some-name
	`))

	// Some items; use only the CA Path.
	projection.Items = []corev1.KeyToPath{
		{Key: "some-crt-key", Path: "tls.crt"},
		{Key: "some-ca-key", Path: "ca.crt"},
	}
	assert.Assert(t, marshalMatches(backendAuthority(projection), `
secret:
  items:
  - key: some-ca-key
    path: ~postgres-operator/backend-ca.crt
  name: some-name
	`))
}

func TestFrontendCertificate(t *testing.T) {
	secret := new(corev1.Secret)
	secret.Name = "op-secret"

	t.Run("Generated", func(t *testing.T) {
		assert.Assert(t, marshalMatches(frontendCertificate(nil, secret), `
secret:
  items:
  - key: pgbouncer-frontend.ca-roots
    path: ~postgres-operator/frontend-ca.crt
  - key: pgbouncer-frontend.key
    path: ~postgres-operator/frontend-tls.key
  - key: pgbouncer-frontend.crt
    path: ~postgres-operator/frontend-tls.crt
  name: op-secret
		`))
	})

	t.Run("Custom", func(t *testing.T) {
		custom := new(corev1.SecretProjection)
		custom.Name = "some-other"

		// No items; assume Key matches Path.
		assert.Assert(t, marshalMatches(frontendCertificate(custom, secret), `
secret:
  items:
  - key: ca.crt
    path: ~postgres-operator/frontend-ca.crt
  - key: tls.key
    path: ~postgres-operator/frontend-tls.key
  - key: tls.crt
    path: ~postgres-operator/frontend-tls.crt
  name: some-other
		`))

		// Some items; use only the TLS Paths.
		custom.Items = []corev1.KeyToPath{
			{Key: "any", Path: "thing"},
			{Key: "some-ca-key", Path: "ca.crt"},
			{Key: "some-cert-key", Path: "tls.crt"},
			{Key: "some-key-key", Path: "tls.key"},
		}
		assert.Assert(t, marshalMatches(frontendCertificate(custom, secret), `
secret:
  items:
  - key: some-ca-key
    path: ~postgres-operator/frontend-ca.crt
  - key: some-cert-key
    path: ~postgres-operator/frontend-tls.crt
  - key: some-key-key
    path: ~postgres-operator/frontend-tls.key
  name: some-other
		`))
	})
}
