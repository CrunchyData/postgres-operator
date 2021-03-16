package pki

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

import (
	"crypto/x509"
	"io/ioutil"
	"net"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

// TestPKI does a full test of generating a valid ceritificate chain
func TestPKI(t *testing.T) {
	// generate the root CA
	rootCA := NewRootCertificateAuthority()
	if err := rootCA.Generate(); err != nil {
		t.Fatalf("root certificate authority could not be generated")
	}

	// generate the leaf CA
	namespace := "pgo-test"
	commonName := "hippo." + namespace
	dnsNames := []string{commonName}
	cert := NewLeafCertificate(commonName, dnsNames, []net.IP{})
	if err := cert.Generate(rootCA); err != nil {
		t.Fatalf("leaf certificate could not be generated")
	}

	// OK, test if we can verify the validity of the leaf certificate
	rootCertificate, err := x509.ParseCertificate(rootCA.Certificate.Certificate)
	if err != nil {
		t.Fatalf("could not parse root certificate: %s", err.Error())
	}

	certificate, err := x509.ParseCertificate(cert.Certificate.Certificate)
	if err != nil {
		t.Fatalf("could not parse leaf certificate: %s", err.Error())
	}

	opts := x509.VerifyOptions{
		DNSName: commonName,
		Roots:   x509.NewCertPool(),
	}
	opts.Roots.AddCert(rootCertificate)

	if _, err := certificate.Verify(opts); err != nil {
		t.Fatalf("could not verify certificate: %s", err.Error())
	}
}

func TestPKIOpenSSL(t *testing.T) {
	openssl, err := exec.LookPath("openssl")
	if err != nil {
		t.Skip(`requires "openssl" executable`)
	} else {
		output, err := exec.Command(openssl, "version", "-a").CombinedOutput()
		assert.NilError(t, err)
		t.Logf("using %q:\n%s", openssl, output)
	}

	rootCA := NewRootCertificateAuthority()
	assert.NilError(t, rootCA.Generate())

	namespace := "pgo-test"
	commonName := "hippo." + namespace
	dnsNames := []string{commonName}
	leaf := NewLeafCertificate(commonName, dnsNames, []net.IP{})
	assert.NilError(t, leaf.Generate(rootCA))

	basicOpenSSLVerify(t, openssl,
		rootCA.Certificate, leaf.Certificate)

	t.Run("strict", func(t *testing.T) {
		output, _ := exec.Command(openssl, "verify", "-help").CombinedOutput()
		if !strings.Contains(string(output), "-x509_strict") {
			t.Skip(`requires "-x509_strict" flag`)
		}

		strictOpenSSLVerify(t, openssl, rootCA.Certificate, leaf.Certificate)
	})
}

func basicOpenSSLVerify(t *testing.T, openssl string, root, leaf *Certificate) {
	verify := func(t testing.TB, args ...string) {
		t.Helper()
		args = append([]string{"verify"}, args...)

		output, err := exec.Command(openssl, args...).CombinedOutput()
		assert.NilError(t, err, "%q\n%s", append([]string{openssl}, args...), output)
	}

	dir := t.TempDir()

	rootFile := filepath.Join(dir, "root.crt")
	rootBytes, err := root.MarshalText()
	assert.NilError(t, err)
	assert.NilError(t, ioutil.WriteFile(rootFile, rootBytes, 0600))

	// The root certificate cannot be verified independently because it is self-signed.
	// It is checked below by being the specified CA.

	leafFile := filepath.Join(dir, "leaf.crt")
	leafBytes, err := leaf.MarshalText()
	assert.NilError(t, err)
	assert.NilError(t, ioutil.WriteFile(leafFile, leafBytes, 0600))

	// Older versions of OpenSSL have fewer options for verifying certificates.
	// When the only flag available is "-CAfile", CAs must be bundled
	// there and are *implicitly trusted*.

	bundleFile := filepath.Join(dir, "ca-chain.crt")
	assert.NilError(t, ioutil.WriteFile(bundleFile, rootBytes, 0600))

	verify(t, "-CAfile", bundleFile, leafFile)
	verify(t, "-CAfile", bundleFile, "-purpose", "sslclient", leafFile)
	verify(t, "-CAfile", bundleFile, "-purpose", "sslserver", leafFile)
}

func strictOpenSSLVerify(t *testing.T, openssl string, root, leaf *Certificate) {
	verify := func(t testing.TB, args ...string) {
		t.Helper()
		args = append([]string{"verify",
			// Do not use the default trusted CAs.
			"-no-CAfile", "-no-CApath",
			// Disable "non-compliant workarounds for broken certificates".
			"-x509_strict",
		}, args...)

		output, err := exec.Command(openssl, args...).CombinedOutput()
		assert.NilError(t, err, "%q\n%s", append([]string{openssl}, args...), output)
	}

	dir := t.TempDir()

	rootFile := filepath.Join(dir, "root.crt")
	rootBytes, err := root.MarshalText()
	assert.NilError(t, err)
	assert.NilError(t, ioutil.WriteFile(rootFile, rootBytes, 0600))

	// The root certificate cannot be verified independently because it is self-signed.
	// Some checks are performed when it is a "trusted" certificate below.

	leafFile := filepath.Join(dir, "leaf.crt")
	leafBytes, err := leaf.MarshalText()
	assert.NilError(t, err)
	assert.NilError(t, ioutil.WriteFile(leafFile, leafBytes, 0600))

	verify(t, "-trusted", rootFile, leafFile)
	verify(t, "-trusted", rootFile, "-purpose", "sslclient", leafFile)
	verify(t, "-trusted", rootFile, "-purpose", "sslserver", leafFile)
}
