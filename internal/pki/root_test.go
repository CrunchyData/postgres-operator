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
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestRootCAIsBad(t *testing.T) {
	rootCA, err := NewRootCertificateAuthority()
	assert.NilError(t, err)

	t.Run("root cert is good", func(t *testing.T) {

		assert.Assert(t, !RootCAIsBad(rootCA))
	})

	t.Run("root cert is empty", func(t *testing.T) {

		emptyRoot := &RootCertificateAuthority{}
		assert.Assert(t, RootCAIsBad(emptyRoot))
	})

	t.Run("error is not a CA", func(t *testing.T) {

		key, err := generateKey()
		assert.NilError(t, err)

		id, err := generateSerialNumber()
		assert.NilError(t, err)

		badCa := &RootCertificateAuthority{}
		badCa.PrivateKey.ecdsa = key
		badCa.Certificate.x509, err = generateRootCertificateBadCA(key, id)
		assert.NilError(t, err)

		assert.Assert(t, RootCAIsBad(badCa))

	})

	t.Run("error expired", func(t *testing.T) {

		key, err := generateKey()
		assert.NilError(t, err)

		id, err := generateSerialNumber()
		assert.NilError(t, err)

		badCa := &RootCertificateAuthority{}
		badCa.PrivateKey.ecdsa = key
		badCa.Certificate.x509, err = generateRootCertificateExpired(key, id)
		assert.NilError(t, err)

		assert.Assert(t, RootCAIsBad(badCa))

	})
}

// generateRootCertificateBadCA creates a root certificate that is not
// configured as a CA
func generateRootCertificateBadCA(privateKey *ecdsa.PrivateKey, serialNumber *big.Int) (*x509.Certificate, error) {
	// prepare the certificate. set the validity time to the predefined range
	now := time.Now()
	template := &x509.Certificate{
		BasicConstraintsValid: true,
		IsCA:                  false,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(time.Hour),
		SerialNumber:          serialNumber,
		SignatureAlgorithm:    certificateSignatureAlgorithm,
		Subject: pkix.Name{
			CommonName: "root-cn",
		},
	}

	// a root certificate has no parent, so pass in the template twice
	bytes, err := x509.CreateCertificate(rand.Reader, template, template,
		privateKey.Public(), privateKey)

	parsed, _ := x509.ParseCertificate(bytes)
	return parsed, err
}

// generateRootCertificateExpired creates a root certificate that is already expired
func generateRootCertificateExpired(privateKey *ecdsa.PrivateKey, serialNumber *big.Int) (*x509.Certificate, error) {
	// prepare the certificate. set the validity time to the predefined range
	now := time.Now()
	template := &x509.Certificate{
		BasicConstraintsValid: true,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		MaxPathLenZero:        true, // there are no intermediate certificates
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(-time.Hour), // not after an hour ago, i.e. expired
		SerialNumber:          serialNumber,
		SignatureAlgorithm:    certificateSignatureAlgorithm,
		Subject: pkix.Name{
			CommonName: "root-cn",
		},
	}

	// a root certificate has no parent, so pass in the template twice
	bytes, err := x509.CreateCertificate(rand.Reader, template, template,
		privateKey.Public(), privateKey)

	parsed, _ := x509.ParseCertificate(bytes)
	return parsed, err
}
