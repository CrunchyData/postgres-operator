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
