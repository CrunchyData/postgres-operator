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
	"net"
	"testing"
)

// TestPKI does a full test of generating a valid ceritificate chain
func TestPKI(t *testing.T) {
	// generate the root CA
	rootCA := NewRootCertificateAuthority()
	if err := rootCA.Generate(); err != nil {
		t.Fatalf("root certificate authority could not be generated")
	}

	// generate the intermediate CA
	namespace := "pgo-test"
	intermediateCA := NewIntermediateCertificateAuthority(namespace)
	if err := intermediateCA.Generate(rootCA); err != nil {
		t.Fatalf("intermediate certificate authority could not be generated")
	}

	// generate the leaf CA
	commonName := "hippo." + namespace
	dnsNames := []string{commonName}
	cert := NewLeafCertificate(commonName, dnsNames, []net.IP{})
	if err := cert.Generate(intermediateCA); err != nil {
		t.Fatalf("leaf certificate could not be generated")
	}

	// OK, test if we can verify the validity of the leaf certificate
	rootCertificate, err := x509.ParseCertificate(rootCA.Certificate.Certificate)
	if err != nil {
		t.Fatalf("could not parse root certificate: %s", err.Error())
	}

	intermediateCertificate, err := x509.ParseCertificate(intermediateCA.Certificate.Certificate)
	if err != nil {
		t.Fatalf("could not parse intermediate certificate: %s", err.Error())
	}

	certificate, err := x509.ParseCertificate(cert.Certificate.Certificate)
	if err != nil {
		t.Fatalf("could not parse leaf certificate: %s", err.Error())
	}

	opts := x509.VerifyOptions{
		DNSName:       commonName,
		Intermediates: x509.NewCertPool(),
		Roots:         x509.NewCertPool(),
	}
	opts.Roots.AddCert(rootCertificate)
	opts.Intermediates.AddCert(intermediateCertificate)

	if _, err := certificate.Verify(opts); err != nil {
		t.Fatalf("could not verify certificate: %s", err.Error())
	}
}
