package pki

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

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestLeafCertIsBad(t *testing.T) {
	ctx := context.Background()
	testRoot, err := NewRootCertificateAuthority()
	assert.NilError(t, err)

	namespace := "pgo-test"
	commonName := "hippo." + namespace
	dnsNames := []string{commonName, "hippo." + namespace + ".svc"}
	ipAddresses := []net.IP{net.ParseIP("127.0.0.1")}

	testLeaf, err := testRoot.GenerateLeafCertificate(commonName, dnsNames)
	assert.NilError(t, err)

	t.Run("leaf cert is good", func(t *testing.T) {

		assert.Assert(t, !LeafCertIsBad(ctx, testLeaf, testRoot, namespace))
	})

	t.Run("leaf cert is empty", func(t *testing.T) {

		emptyLeaf := &LeafCertificate{}
		assert.Assert(t, LeafCertIsBad(ctx, emptyLeaf, testRoot, namespace))
	})

	t.Run("leaf with invalid constraint", func(t *testing.T) {

		testRoot3, err := NewRootCertificateAuthority()
		assert.NilError(t, err)

		key, err := generateKey()
		assert.NilError(t, err)

		id, err := generateSerialNumber()
		assert.NilError(t, err)

		badLeaf := &LeafCertificate{}
		badLeaf.PrivateKey.ecdsa = key
		badLeaf.Certificate.x509, err = generateLeafCertificateInvalidConstraint(
			key, id, testRoot3, commonName, dnsNames, ipAddresses)
		assert.NilError(t, err)

		assert.Assert(t, LeafCertIsBad(ctx, badLeaf, testRoot3, namespace))

	})

	t.Run("leaf is a expired", func(t *testing.T) {

		testRoot3, err := NewRootCertificateAuthority()
		assert.NilError(t, err)

		key, err := generateKey()
		assert.NilError(t, err)

		id, err := generateSerialNumber()
		assert.NilError(t, err)

		badLeaf := &LeafCertificate{}
		badLeaf.PrivateKey.ecdsa = key
		badLeaf.Certificate.x509, err = generateLeafCertificateExpired(
			key, id, testRoot3, commonName, dnsNames, ipAddresses)
		assert.NilError(t, err)

		assert.Assert(t, LeafCertIsBad(ctx, badLeaf, testRoot3, namespace))

	})
}

// generateLeafCertificateInvalidConstraint creates a x509 certificate with BasicConstraintsValid set to false
func generateLeafCertificateInvalidConstraint(privateKey *ecdsa.PrivateKey, serialNumber *big.Int,
	rootCA *RootCertificateAuthority, commonName string, dnsNames []string, ipAddresses []net.IP,
) (*x509.Certificate, error) {
	parent := rootCA.Certificate.x509

	// prepare the certificate. set the validity time to the predefined range
	now := time.Now()
	template := &x509.Certificate{
		BasicConstraintsValid: false,
		DNSNames:              dnsNames,
		IPAddresses:           ipAddresses,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(time.Hour),
		SerialNumber:          serialNumber,
		SignatureAlgorithm:    certificateSignatureAlgorithm,
		Subject: pkix.Name{
			CommonName: commonName,
		},
	}

	// create the leaf certificate and sign it using the root CA
	bytes, err := x509.CreateCertificate(rand.Reader, template, parent,
		privateKey.Public(), rootCA.PrivateKey.ecdsa)

	parsed, _ := x509.ParseCertificate(bytes)
	return parsed, err
}

// generateLeafCertificateExpired creates a x509 certificate that is expired
func generateLeafCertificateExpired(privateKey *ecdsa.PrivateKey, serialNumber *big.Int,
	rootCA *RootCertificateAuthority, commonName string, dnsNames []string, ipAddresses []net.IP,
) (*x509.Certificate, error) {
	parent := rootCA.Certificate.x509

	// prepare the certificate. set the validity time to the predefined range
	now := time.Now()
	template := &x509.Certificate{
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
		IPAddresses:           ipAddresses,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(-time.Hour), // not after an hour ago, i.e. expired
		SerialNumber:          serialNumber,
		SignatureAlgorithm:    certificateSignatureAlgorithm,
		Subject: pkix.Name{
			CommonName: commonName,
		},
	}

	// create the leaf certificate and sign it using the root CA
	bytes, err := x509.CreateCertificate(rand.Reader, template, parent,
		privateKey.Public(), rootCA.PrivateKey.ecdsa)

	parsed, _ := x509.ParseCertificate(bytes)
	return parsed, err
}
