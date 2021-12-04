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
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/internal/pki"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
)

const rootPEM = `-----BEGIN CERTIFICATE-----
MIIBgTCCASigAwIBAgIRAO0NXdQ5ZtvI26doDvj9Dx8wCgYIKoZIzj0EAwMwHzEd
MBsGA1UEAxMUcG9zdGdyZXMtb3BlcmF0b3ItY2EwHhcNMjEwMTI3MjEyNTU0WhcN
MzEwMTI1MjIyNTU0WjAfMR0wGwYDVQQDExRwb3N0Z3Jlcy1vcGVyYXRvci1jYTBZ
MBMGByqGSM49AgEGCCqGSM49AwEHA0IABL0xD8B6ZQHPscklofw2hpEN1F8h06Ys
IRhK2xoy8ASkiKOkzXVs22R/Wnv/+jAMVf9rit0vhblZlvn2yP7e29WjRTBDMA4G
A1UdDwEB/wQEAwIBBjASBgNVHRMBAf8ECDAGAQH/AgECMB0GA1UdDgQWBBQjfqdS
Ynr3rFHMLd3fHO79tH3w5DAKBggqhkjOPQQDAwNHADBEAiA41LbQXeC0G/AyOHgs
gaUp3fzHKSsrTGhzA8+dK2mnSgIgEKnv1FquJBJuXRBAxzrmnt0nJPiTWB926iNE
BY8V4Ag=
-----END CERTIFICATE-----`

func TestCertAuthorities(t *testing.T) {
	root, err := pki.ParseCertificate([]byte(rootPEM))
	assert.NilError(t, err)

	data, err := certAuthorities(root)
	assert.NilError(t, err)

	// PEM-encoded certificates.
	assert.DeepEqual(t, string(data), rootPEM+"\n")
}

func TestCertFile(t *testing.T) {
	root := pki.NewRootCertificateAuthority()
	assert.NilError(t, root.Generate())

	instance := pki.NewLeafCertificate("instance.pod-dns", nil, nil)
	assert.NilError(t, instance.Generate(root))

	data, err := certFile(instance.PrivateKey, instance.Certificate)
	assert.NilError(t, err)

	// PEM-encoded key followed by the certificate
	// - https://docs.python.org/3/library/ssl.html#combined-key-and-certificate
	// - https://docs.python.org/3/library/ssl.html#certificate-chains
	assert.Assert(t,
		cmp.Regexp(`^`+
			`-----BEGIN [^ ]+ PRIVATE KEY-----\n`+
			`([^-]+\n)+`+
			`-----END [^ ]+ PRIVATE KEY-----\n`+
			`-----BEGIN CERTIFICATE-----\n`+
			`([^-]+\n)+`+
			`-----END CERTIFICATE-----\n`+
			`$`,
			string(data),
		))
}

func TestInstanceCertificates(t *testing.T) {
	certs := new(corev1.Secret)
	certs.Name = "some-name"

	projections := instanceCertificates(certs)

	assert.Assert(t, cmp.MarshalMatches(projections, `
- secret:
    items:
    - key: patroni.ca-roots
      path: ~postgres-operator/patroni.ca-roots
    - key: patroni.crt-combined
      path: ~postgres-operator/patroni.crt+key
    name: some-name
	`))
}
