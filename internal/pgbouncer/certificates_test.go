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
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestBackendAuthority(t *testing.T) {
	projection := &corev1.SecretProjection{
		LocalObjectReference: corev1.LocalObjectReference{Name: "some-name"},
		Items: []corev1.KeyToPath{
			{Key: "some-crt-key", Path: "tls.crt"},
			{Key: "some-ca-key", Path: "ca.crt"},
		},
	}

	assert.Assert(t, marshalEquals(backendAuthority(projection), strings.Trim(`
secret:
  items:
  - key: some-ca-key
    path: ca.crt
  name: some-name
	`, "\t\n")+"\n"))
}

func TestFrontendCertificate(t *testing.T) {
	secret := new(corev1.Secret)
	secret.Name = "op-secret"

	t.Run("Generated", func(t *testing.T) {
		assert.Assert(t, marshalEquals(frontendCertificate(nil, secret), strings.Trim(`
secret:
  items:
  - key: pgbouncer-frontend.ca-roots
    path: ca.crt
  - key: pgbouncer-frontend.key
    path: tls.key
  - key: pgbouncer-frontend.crt
    path: tls.crt
  name: op-secret
		`, "\t\n")+"\n"))
	})

	t.Run("Custom", func(t *testing.T) {
		custom := new(corev1.SecretProjection)
		custom.Name = "some-other"
		custom.Items = []corev1.KeyToPath{
			{Key: "any", Path: "thing"},
		}

		assert.Assert(t, marshalEquals(frontendCertificate(custom, secret), strings.Trim(`
secret:
  items:
  - key: any
    path: thing
  name: some-other
		`, "\t\n")+"\n"))
	})
}
