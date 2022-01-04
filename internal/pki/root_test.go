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
	"errors"
	"math/big"
	"reflect"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestNewRootCertificateAuthority(t *testing.T) {
	ca := NewRootCertificateAuthority()

	if ca.generateCertificate == nil {
		t.Fatalf("expected generateCertificate to be set, got nil")
	}

	if ca.generateKey == nil {
		t.Fatalf("expected generateKey to be set, got nil")
	}

	if ca.generateSerialNumber == nil {
		t.Fatalf("expected generateSerialNumber to be set, got nil")
	}

	// run generate to ensure it sets valid values
	if err := ca.Generate(); err != nil {
		t.Fatalf("expected generate to return no errors, got: %s", err.Error())
	}

	// ensure private key and certificate are set
	if ca.PrivateKey == nil {
		t.Fatalf("expected private key to be set")
	}

	if ca.Certificate == nil {
		t.Fatalf("expected certificate to be set")
	}
}

func TestParseRootCertificateAuthority(t *testing.T) {
	generateRootCertificateAuthority := func() *RootCertificateAuthority {
		ca := NewRootCertificateAuthority()
		_ = ca.Generate()
		return ca
	}

	marshalCertificate := func(ca *RootCertificateAuthority) []byte {
		data, _ := ca.Certificate.MarshalText()
		return data
	}

	marshalPrivateKey := func(ca *RootCertificateAuthority) []byte {
		data, _ := ca.PrivateKey.MarshalText()
		return data
	}

	ca := generateRootCertificateAuthority()

	t.Run("valid plaintext", func(t *testing.T) {
		privateKey := marshalPrivateKey(ca)
		certificate := marshalCertificate(ca)

		rootCA, err := ParseRootCertificateAuthority(privateKey, certificate)

		if err != nil {
			t.Fatalf("expected no error, actual %s", err.Error())
		}

		if !reflect.DeepEqual(ca.PrivateKey.PrivateKey, rootCA.PrivateKey.PrivateKey) {
			t.Fatalf("expected private keys to match")
		}

		if !reflect.DeepEqual(ca.Certificate.Certificate, rootCA.Certificate.Certificate) {
			t.Fatalf("expected certificates to match")
		}
	})

	t.Run("invalid", func(t *testing.T) {
		t.Run("bad private key", func(t *testing.T) {
			privateKey := []byte("bad")
			certificate := marshalCertificate(ca)

			rootCA, err := ParseRootCertificateAuthority(privateKey, certificate)

			if err == nil {
				t.Fatalf("expected error")
			}

			if rootCA != nil {
				t.Fatalf("expected CA to be nil")
			}
		})

		t.Run("bad certificate key", func(t *testing.T) {
			privateKey := marshalPrivateKey(ca)
			certificate := []byte("bad")

			rootCA, err := ParseRootCertificateAuthority(privateKey, certificate)

			if err == nil {
				t.Fatalf("expected error")
			}

			if rootCA != nil {
				t.Fatalf("expected CA to be nil")
			}
		})
	})
}

func TestRootCertificateAuthority(t *testing.T) {
	t.Run("Generate", func(t *testing.T) {
		t.Run("valid", func(t *testing.T) {
			ca := &RootCertificateAuthority{}
			ca.generateCertificate = generateRootCertificate
			ca.generateKey = generateKey
			ca.generateSerialNumber = generateSerialNumber

			// run generate to ensure it sets valid values
			if err := ca.Generate(); err != nil {
				t.Fatalf("expected generate to return no errors, got: %s", err.Error())
			}

			// ensure private key and certificate are set
			if ca.PrivateKey == nil {
				t.Fatalf("expected private key to be set")
			}

			if ca.Certificate == nil {
				t.Fatalf("expected certificate to be set")
			}

			if ca.PrivateKey.PrivateKey == nil {
				t.Fatalf("expected private key to be set, got nil")
			}

			if len(ca.Certificate.Certificate) == 0 {
				t.Fatalf("expected certificate to be generated")
			}

			// see if certificate can be parsed
			x509Certificate, err := x509.ParseCertificate(ca.Certificate.Certificate)

			if err != nil {
				t.Fatalf("expected valid x509 ceriticate, actual %s", err.Error())
			}

			if !ca.PrivateKey.PrivateKey.PublicKey.Equal(x509Certificate.PublicKey) {
				t.Fatalf("expected public keys to match")
			}

			// check certain attributes
			if !x509Certificate.IsCA {
				t.Fatalf("expected certificate to be CA")
			}

			if x509Certificate.MaxPathLenZero == false {
				t.Fatalf("expected MaxPathLenZero to be set to 'true', actual %t", x509Certificate.MaxPathLenZero)
			}

			if x509Certificate.Subject.CommonName != rootCAName {
				t.Fatalf("expected subject name to be %s, actual %s", defaultRootCAExpiration, x509Certificate.Subject.CommonName)
			}

			// ensure private key functions are set
			assertConstructed(t, ca.PrivateKey)
		})

		t.Run("invalid", func(t *testing.T) {
			t.Run("generate certificate not set", func(t *testing.T) {
				ca := &RootCertificateAuthority{}
				ca.generateCertificate = nil
				ca.generateKey = generateKey
				ca.generateSerialNumber = generateSerialNumber

				if err := ca.Generate(); !errors.Is(err, ErrFunctionNotImplemented) {
					t.Fatalf("expected function not implemented error")
				}
			})

			t.Run("generate key not set", func(t *testing.T) {
				ca := &RootCertificateAuthority{}
				ca.generateCertificate = generateRootCertificate
				ca.generateKey = nil
				ca.generateSerialNumber = generateSerialNumber

				if err := ca.Generate(); !errors.Is(err, ErrFunctionNotImplemented) {
					t.Fatalf("expected function not implemented error")
				}
			})

			t.Run("generate serial number not set", func(t *testing.T) {
				ca := &RootCertificateAuthority{}
				ca.generateCertificate = generateRootCertificate
				ca.generateKey = generateKey
				ca.generateSerialNumber = nil

				if err := ca.Generate(); !errors.Is(err, ErrFunctionNotImplemented) {
					t.Fatalf("expected function not implemented error")
				}
			})

			t.Run("cannot generate private key", func(t *testing.T) {
				msg := "cannot generate private key"
				ca := &RootCertificateAuthority{}
				ca.generateCertificate = generateRootCertificate
				ca.generateKey = func() (*ecdsa.PrivateKey, error) { return nil, errors.New(msg) }
				ca.generateSerialNumber = generateSerialNumber

				if err := ca.Generate(); err.Error() != msg {
					t.Fatalf("expected error: %s", msg)
				}
			})

			t.Run("cannot generate serial number", func(t *testing.T) {
				msg := "cannot generate serial number"
				ca := &RootCertificateAuthority{}
				ca.generateCertificate = generateRootCertificate
				ca.generateKey = generateKey
				ca.generateSerialNumber = func() (*big.Int, error) { return nil, errors.New(msg) }

				if err := ca.Generate(); err.Error() != msg {
					t.Fatalf("expected error: %s", msg)
				}
			})

			t.Run("cannot generate certificate", func(t *testing.T) {
				msg := "cannot generate certificate"
				ca := &RootCertificateAuthority{}
				ca.generateCertificate = func(*ecdsa.PrivateKey, *big.Int) ([]byte, error) { return nil, errors.New(msg) }
				ca.generateKey = generateKey
				ca.generateSerialNumber = generateSerialNumber

				if err := ca.Generate(); err.Error() != msg {
					t.Fatalf("expected error: %s", msg)
				}
			})
		})
	})
}

func TestRootCAIsBad(t *testing.T) {
	rootCA, err := newTestRoot()
	assert.NilError(t, err)

	t.Run("root cert is good", func(t *testing.T) {

		assert.Assert(t, !RootCAIsBad(rootCA))
	})

	t.Run("root cert is empty", func(t *testing.T) {

		emptyRoot := &RootCertificateAuthority{}
		assert.Assert(t, RootCAIsBad(emptyRoot))
	})

	t.Run("error parsing certificate", func(t *testing.T) {
		rootCA.Certificate = &Certificate{
			Certificate: []byte("notacert"),
		}

		assert.Assert(t, RootCAIsBad(rootCA))
	})

	t.Run("error is not a CA", func(t *testing.T) {

		badCa := &RootCertificateAuthority{
			generateCertificate:  generateRootCertificateBadCA,
			generateKey:          generateKey,
			generateSerialNumber: generateSerialNumber,
		}

		// run generate to ensure it sets valid values
		if err := badCa.Generate(); err != nil {
			t.Fatalf("expected generate to return no errors, got: %s", err.Error())
		}

		assert.Assert(t, RootCAIsBad(badCa))

	})

	t.Run("error expired", func(t *testing.T) {

		badCa := &RootCertificateAuthority{
			generateCertificate:  generateRootCertificateExpired,
			generateKey:          generateKey,
			generateSerialNumber: generateSerialNumber,
		}

		// run generate to ensure it sets valid values
		if err := badCa.Generate(); err != nil {
			t.Fatalf("expected generate to return no errors, got: %s", err.Error())
		}

		assert.Assert(t, RootCAIsBad(badCa))

	})
}

// generateRootCertificateBadCA creates a root certificate that is not
// configured as a CA
func generateRootCertificateBadCA(privateKey *ecdsa.PrivateKey, serialNumber *big.Int) ([]byte, error) {
	// prepare the certificate. set the validity time to the predefined range
	now := time.Now()
	template := &x509.Certificate{
		BasicConstraintsValid: true,
		IsCA:                  false,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		NotBefore:             now.Add(beforeInterval),
		NotAfter:              now.Add(defaultRootCAExpiration),
		SerialNumber:          serialNumber,
		SignatureAlgorithm:    certificateSignatureAlgorithm,
		Subject: pkix.Name{
			CommonName: rootCAName,
		},
	}

	// a root certificate has no parent, so pass in the template twice
	return x509.CreateCertificate(rand.Reader, template, template,
		privateKey.Public(), privateKey)
}

// generateRootCertificateExpired creates a root certificate that is already expired
func generateRootCertificateExpired(privateKey *ecdsa.PrivateKey, serialNumber *big.Int) ([]byte, error) {
	// prepare the certificate. set the validity time to the predefined range
	now := time.Now()
	template := &x509.Certificate{
		BasicConstraintsValid: true,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		MaxPathLenZero:        true, // there are no intermediate certificates
		NotBefore:             now.Add(beforeInterval),
		NotAfter:              now.Add(beforeInterval), // not after an hour ago, i.e. expired
		SerialNumber:          serialNumber,
		SignatureAlgorithm:    certificateSignatureAlgorithm,
		Subject: pkix.Name{
			CommonName: rootCAName,
		},
	}

	// a root certificate has no parent, so pass in the template twice
	return x509.CreateCertificate(rand.Reader, template, template,
		privateKey.Public(), privateKey)
}

// newTestRoot creates a new test root certificate
func newTestRoot() (*RootCertificateAuthority, error) {
	testRoot := &RootCertificateAuthority{}
	testRoot.generateCertificate = generateRootCertificate
	testRoot.generateKey = generateKey
	testRoot.generateSerialNumber = generateSerialNumber

	// run generate to ensure it sets valid values
	err := testRoot.Generate()

	return testRoot, err
}
