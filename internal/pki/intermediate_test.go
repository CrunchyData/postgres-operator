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
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestIntermediateCertificateAuthority(t *testing.T) {
	t.Run("Generate", func(t *testing.T) {
		namespace := "pgo-test"
		// run generate on rootCA to ensure it sets valid values
		rootCA := NewRootCertificateAuthority()
		if err := rootCA.Generate(); err != nil {
			t.Fatalf("root certificate authority could not be generated")
		}

		t.Run("valid", func(t *testing.T) {
			ca := &IntermediateCertificateAuthority{
				Namespace:            namespace,
				generateCertificate:  generateIntermediateCertificate,
				generateKey:          generateKey,
				generateSerialNumber: generateSerialNumber,
			}

			// run generate to ensure it sets valid values
			if err := ca.Generate(rootCA); err != nil {
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
				t.Fatalf("expected public key from stored key to match public key on certificate")
			}

			// check certain attributes
			if !x509Certificate.IsCA {
				t.Fatalf("expected certificate to be CA")
			}

			if !x509Certificate.MaxPathLenZero || x509Certificate.MaxPathLen != 0 {
				t.Fatalf("expected max path length to be 0")
			}

			if x509Certificate.Issuer.CommonName != rootCAName {
				t.Fatalf("expected issuer common name to be %s, actual %s",
					rootCAName, x509Certificate.Issuer.CommonName)
			}

			commonName := fmt.Sprintf("%s.%s", namespace, rootCAName)
			if x509Certificate.Subject.CommonName != commonName {
				t.Fatalf("expected subject name to be %s, actual %s", commonName, x509Certificate.Subject.CommonName)
			}

			// ensure private key functions are set
			assertConstructed(t, ca.PrivateKey)
		})

		t.Run("invalid", func(t *testing.T) {
			t.Run("generate certificate not set", func(t *testing.T) {
				ca := &IntermediateCertificateAuthority{}
				ca.generateCertificate = nil
				ca.generateKey = generateKey
				ca.generateSerialNumber = generateSerialNumber

				if err := ca.Generate(rootCA); !errors.Is(err, ErrFunctionNotImplemented) {
					t.Fatalf("expected function not implemented error")
				}
			})

			t.Run("generate key not set", func(t *testing.T) {
				ca := &IntermediateCertificateAuthority{}
				ca.generateCertificate = generateIntermediateCertificate
				ca.generateKey = nil
				ca.generateSerialNumber = generateSerialNumber

				if err := ca.Generate(rootCA); !errors.Is(err, ErrFunctionNotImplemented) {
					t.Fatalf("expected function not implemented error")
				}
			})

			t.Run("generate serial number not set", func(t *testing.T) {
				ca := &IntermediateCertificateAuthority{}
				ca.generateCertificate = generateIntermediateCertificate
				ca.generateKey = generateKey
				ca.generateSerialNumber = nil

				if err := ca.Generate(rootCA); !errors.Is(err, ErrFunctionNotImplemented) {
					t.Fatalf("expected function not implemented error")
				}
			})

			t.Run("root certificate authority is nil", func(t *testing.T) {
				ca := &IntermediateCertificateAuthority{}
				ca.generateCertificate = generateIntermediateCertificate
				ca.generateKey = generateKey
				ca.generateSerialNumber = generateSerialNumber

				if err := ca.Generate(nil); !errors.Is(err, ErrInvalidCertificateAuthority) {
					t.Fatalf("expected invalid certificate authority")
				}
			})

			t.Run("root certificate authority has no private key", func(t *testing.T) {
				ca := &IntermediateCertificateAuthority{
					Namespace:            namespace,
					generateCertificate:  generateIntermediateCertificate,
					generateKey:          generateKey,
					generateSerialNumber: generateSerialNumber,
				}
				rootCA := NewRootCertificateAuthority()
				if err := rootCA.Generate(); err != nil {
					t.Fatalf("root certificate authority could not be generated")
				}
				rootCA.PrivateKey = nil

				if err := ca.Generate(rootCA); !errors.Is(err, ErrInvalidCertificateAuthority) {
					t.Fatalf("expected invalid certificate authority")
				}
			})

			t.Run("root certificate authority has no certificate", func(t *testing.T) {
				ca := &IntermediateCertificateAuthority{
					Namespace:            namespace,
					generateCertificate:  generateIntermediateCertificate,
					generateKey:          generateKey,
					generateSerialNumber: generateSerialNumber,
				}
				rootCA := NewRootCertificateAuthority()
				if err := rootCA.Generate(); err != nil {
					t.Fatalf("root certificate authority could not be generated")
				}
				rootCA.Certificate = nil

				if err := ca.Generate(rootCA); !errors.Is(err, ErrInvalidCertificateAuthority) {
					t.Fatalf("expected invalid certificate authority")
				}
			})

			t.Run("root certificate authority has invalid certificate", func(t *testing.T) {
				ca := &IntermediateCertificateAuthority{
					Namespace:            namespace,
					generateCertificate:  generateIntermediateCertificate,
					generateKey:          generateKey,
					generateSerialNumber: generateSerialNumber,
				}
				rootCA := NewRootCertificateAuthority()
				if err := rootCA.Generate(); err != nil {
					t.Fatalf("root certificate authority could not be generated")
				}
				rootCA.Certificate.Certificate = []byte{}

				if err := ca.Generate(rootCA); err == nil {
					t.Fatalf("expected certificate parsing error")
				}
			})

			t.Run("cannot generate private key", func(t *testing.T) {
				msg := "cannot generate private key"
				ca := &IntermediateCertificateAuthority{
					Namespace:            namespace,
					generateCertificate:  generateIntermediateCertificate,
					generateKey:          func() (*ecdsa.PrivateKey, error) { return nil, errors.New(msg) },
					generateSerialNumber: generateSerialNumber,
				}

				if err := ca.Generate(rootCA); err.Error() != msg {
					t.Fatalf("expected error: %s", msg)
				}
			})

			t.Run("cannot generate serial number", func(t *testing.T) {
				msg := "cannot generate serial number"
				ca := &IntermediateCertificateAuthority{
					Namespace:            namespace,
					generateCertificate:  generateIntermediateCertificate,
					generateKey:          generateKey,
					generateSerialNumber: func() (*big.Int, error) { return nil, errors.New(msg) },
				}

				if err := ca.Generate(rootCA); err.Error() != msg {
					t.Fatalf("expected error: %s", msg)
				}
			})

			t.Run("cannot generate certificate", func(t *testing.T) {
				msg := "cannot generate certificate"
				ca := &IntermediateCertificateAuthority{
					Namespace: namespace,
					generateCertificate: func(*ecdsa.PrivateKey, *big.Int, *RootCertificateAuthority, string) ([]byte, error) {
						return nil, errors.New(msg)
					},
					generateKey:          generateKey,
					generateSerialNumber: generateSerialNumber,
				}

				if err := ca.Generate(rootCA); err.Error() != msg {
					t.Fatalf("expected error: %s", msg)
				}
			})
		})
	})
}

func TestNewIntermediateCertificateAuthority(t *testing.T) {
	namespace := "pgo-test"
	ca := NewIntermediateCertificateAuthority(namespace)

	if ca.Namespace != namespace {
		t.Fatalf("expected namespace to be %q, actual %q", namespace, ca.Namespace)
	}

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
	rootCA := NewRootCertificateAuthority()
	if err := rootCA.Generate(); err != nil {
		t.Fatalf("root certificate authority could not be generated")
	}

	if err := ca.Generate(rootCA); err != nil {
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

func TestParseIntermediateCertificateAuthority(t *testing.T) {
	namespace := "pgo-test"
	generateIntermediateCertificateAuthority := func(namespace string) *IntermediateCertificateAuthority {
		root := NewRootCertificateAuthority()
		_ = root.Generate()
		ca := NewIntermediateCertificateAuthority(namespace)
		_ = ca.Generate(root)
		return ca
	}

	marshalCertificate := func(ca *IntermediateCertificateAuthority) []byte {
		data, _ := ca.Certificate.MarshalText()
		return data
	}

	marshalPrivateKey := func(ca *IntermediateCertificateAuthority) []byte {
		data, _ := ca.PrivateKey.MarshalText()
		return data
	}

	ca := generateIntermediateCertificateAuthority(namespace)

	t.Run("valid plaintext", func(t *testing.T) {
		privateKey := marshalPrivateKey(ca)
		certificate := marshalCertificate(ca)

		intermediateCA, err := ParseIntermediateCertificateAuthority(namespace, privateKey, certificate)

		if err != nil {
			t.Fatalf("expected no error, actual %s", err.Error())
		}

		if !reflect.DeepEqual(ca.PrivateKey.PrivateKey, intermediateCA.PrivateKey.PrivateKey) {
			t.Fatalf("expected private keys to match")
		}

		if !reflect.DeepEqual(ca.Certificate.Certificate, intermediateCA.Certificate.Certificate) {
			t.Fatalf("expected certificates to match")
		}
	})

	t.Run("invalid", func(t *testing.T) {
		t.Run("bad private key", func(t *testing.T) {
			privateKey := []byte("bad")
			certificate := marshalCertificate(ca)

			intermediateCA, err := ParseIntermediateCertificateAuthority(namespace, privateKey, certificate)

			if err == nil {
				t.Fatalf("expected error")
			}

			if intermediateCA != nil {
				t.Fatalf("expected CA to be nil")
			}
		})

		t.Run("bad certificate", func(t *testing.T) {
			privateKey := marshalPrivateKey(ca)
			certificate := []byte("bad")

			intermediateCA, err := ParseIntermediateCertificateAuthority(namespace, privateKey, certificate)

			if err == nil {
				t.Fatalf("expected error")
			}

			if intermediateCA != nil {
				t.Fatalf("expected CA to be nil")
			}
		})
	})
}

func TestIntermediateCAIsBad(t *testing.T) {
	ctx := context.Background()
	testRoot, err := newTestRoot()
	assert.NilError(t, err)

	testIntermediate, err := newTestIntermediate(testRoot)
	assert.NilError(t, err)

	t.Run("intermediate cert is good", func(t *testing.T) {

		assert.Assert(t, !IntermediateCAIsBad(ctx, testIntermediate, testRoot))
	})

	t.Run("intermediate cert is empty", func(t *testing.T) {

		emptyIntermediate := &IntermediateCertificateAuthority{}
		assert.Assert(t, IntermediateCAIsBad(ctx, emptyIntermediate, testRoot))
	})

	t.Run("error parsing root certificate", func(t *testing.T) {
		testRoot.Certificate = &Certificate{
			Certificate: []byte("notacert"),
		}

		assert.Assert(t, IntermediateCAIsBad(ctx, testIntermediate, testRoot))
	})

	t.Run("error parsing intermediate certificate", func(t *testing.T) {

		testRoot2, err := newTestRoot()
		assert.NilError(t, err)

		testIntermediate.Certificate = &Certificate{
			Certificate: []byte("notacert"),
		}

		assert.Assert(t, IntermediateCAIsBad(ctx, testIntermediate, testRoot2))
	})

	t.Run("intermediate is not a CA", func(t *testing.T) {

		testRoot3, err := newTestRoot()
		assert.NilError(t, err)

		badIntermediate := &IntermediateCertificateAuthority{}
		badIntermediate.generateCertificate = generateIntermediateCertificateBadCA
		badIntermediate.generateKey = generateKey
		badIntermediate.generateSerialNumber = generateSerialNumber

		// run generate to ensure it sets valid values
		if err := badIntermediate.Generate(testRoot3); err != nil {
			t.Fatalf("expected generate to return no errors, got: %s", err.Error())
		}

		assert.Assert(t, IntermediateCAIsBad(ctx, badIntermediate, testRoot3))

	})

	t.Run("intermediate verify fails", func(t *testing.T) {
		testRoot4, err := newTestRoot()
		assert.NilError(t, err)

		testIntermediate5 := &IntermediateCertificateAuthority{}
		testIntermediate5.generateCertificate = generateIntermediateCertificate
		testIntermediate5.generateKey = generateKey
		testIntermediate5.generateSerialNumber = generateSerialNumber

		// run generate to ensure it sets valid values
		if err := testRoot.Generate(); err != nil {
			t.Fatalf("expected generate to return no errors, got: %s", err.Error())
		}

		if err := testIntermediate5.Generate(testRoot); err != nil {
			t.Fatalf("expected generate to return no errors, got: %s", err.Error())
		}

		// wrong root
		assert.Assert(t, IntermediateCAIsBad(context.Background(), testIntermediate5, testRoot4))

	})
}

// generateIntermediateCertificateBadCA creates an intermediate certificate that is not
// configured as a CA
func generateIntermediateCertificateBadCA(privateKey *ecdsa.PrivateKey, serialNumber *big.Int,
	rootCA *RootCertificateAuthority, namespace string) ([]byte, error) {
	// first, ensure that the root certificate can be turned into a x509
	// Certificate object so it can be used as the parent certificate when
	// generating
	if rootCA == nil || rootCA.Certificate == nil || rootCA.PrivateKey == nil {
		return nil, fmt.Errorf("%w: root certificate authority needs to be generated",
			ErrInvalidCertificateAuthority)
	}

	parent, err := x509.ParseCertificate(rootCA.Certificate.Certificate)

	if err != nil {
		return nil, err
	}

	// prepare the certificate. set the validity time to the predefined range
	now := time.Now()
	template := &x509.Certificate{
		BasicConstraintsValid: true,
		IsCA:                  false,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		NotBefore:             now.Add(beforeInterval),
		NotAfter:              now.Add(defaultIntermediateCAExpiration),
		SerialNumber:          serialNumber,
		SignatureAlgorithm:    certificateSignatureAlgorithm,
		Subject: pkix.Name{
			CommonName: fmt.Sprintf("%s.%s", namespace, rootCAName),
		},
	}

	// create the intermediate certificate and sign it using the root CA
	return x509.CreateCertificate(rand.Reader, template, parent,
		privateKey.Public(), rootCA.PrivateKey.PrivateKey)
}

// newTestIntermediate creates a new test intermediate certificate
func newTestIntermediate(testRoot *RootCertificateAuthority) (*IntermediateCertificateAuthority, error) {

	testIntermediate := &IntermediateCertificateAuthority{}
	testIntermediate.generateCertificate = generateIntermediateCertificate
	testIntermediate.generateKey = generateKey
	testIntermediate.generateSerialNumber = generateSerialNumber

	err := testIntermediate.Generate(testRoot)

	return testIntermediate, err
}
