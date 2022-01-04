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
	"errors"
	"fmt"
	"math/big"
	"net"
	"reflect"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestLeafCertificate(t *testing.T) {
	t.Run("Generate", func(t *testing.T) {
		namespace := "pgo-test"
		commonName := "hippo." + namespace
		dnsNames := []string{commonName, "hippo." + namespace + ".svc"}
		ipAddresses := []net.IP{net.ParseIP("127.0.0.1")}
		// run generate on rootCA to ensure it sets valid values
		rootCA := NewRootCertificateAuthority()
		if err := rootCA.Generate(); err != nil {
			t.Fatalf("root certificate authority could not be generated")
		}

		// see if certificate can be parsed
		x509RootCA, err := x509.ParseCertificate(rootCA.Certificate.Certificate)
		if err != nil {
			t.Fatalf("expected valid x509 root certificate, actual %s", err.Error())
		}

		t.Run("valid", func(t *testing.T) {
			cert := &LeafCertificate{
				CommonName:           commonName,
				DNSNames:             dnsNames,
				IPAddresses:          ipAddresses,
				generateCertificate:  generateLeafCertificate,
				generateKey:          generateKey,
				generateSerialNumber: generateSerialNumber,
			}

			// run generate to ensure it sets valid values
			if err := cert.Generate(rootCA); err != nil {
				t.Fatalf("expected generate to return no errors, got: %s", err.Error())
			}

			// ensure private key and certificate are set
			if cert.PrivateKey == nil {
				t.Fatalf("expected private key to be set")
			}

			if cert.Certificate == nil {
				t.Fatalf("expected certificate to be set")
			}

			if cert.PrivateKey.PrivateKey == nil {
				t.Fatalf("expected private key to be set, got nil")
			}

			if len(cert.Certificate.Certificate) == 0 {
				t.Fatalf("expected certificate to be generated")
			}

			x509Certificate, err := x509.ParseCertificate(cert.Certificate.Certificate)
			if err != nil {
				t.Fatalf("expected valid x509 ceriticate, actual %s", err.Error())
			}

			if !cert.PrivateKey.PrivateKey.PublicKey.Equal(x509Certificate.PublicKey) {
				t.Fatalf("expected public key from stored key to match public key on certificate")
			}

			// check certain attributes
			if x509Certificate.IsCA {
				t.Fatalf("expected certificate to be a leaf certificate")
			}

			if x509Certificate.Issuer.CommonName != x509RootCA.Subject.CommonName {
				t.Fatalf("expected issuer common name to be %s, actual %s",
					x509RootCA.Subject.CommonName, x509Certificate.Issuer.CommonName)
			}

			if x509Certificate.Subject.CommonName != commonName {
				t.Fatalf("expected subject name to be %s, actual %s", commonName, x509Certificate.Subject.CommonName)
			}

			if !reflect.DeepEqual(x509Certificate.DNSNames, dnsNames) {
				t.Fatalf("expected SAN DNS names to be %v, actual %v", dnsNames, x509Certificate.DNSNames)
			}

			// check IP addresses...inefficiently, as we cannot use a DeepEqual on
			// net.IP slices.
			if len(x509Certificate.IPAddresses) != len(ipAddresses) {
				t.Fatalf("expected SAN IP addresses to be &v, actual &v")
			}

			for _, ip := range x509Certificate.IPAddresses {
				ok := false
				for _, knownIP := range ipAddresses {
					ok = ok || (ip.Equal(knownIP))
				}

				if !ok {
					t.Fatalf("expected SAN IP addresses to be %v, actual %v", ipAddresses, x509Certificate.IPAddresses)
				}
			}

			// ensure private key functions are set
			assertConstructed(t, cert.PrivateKey)
		})

		t.Run("invalid", func(t *testing.T) {
			t.Run("generate certificate not set", func(t *testing.T) {
				cert := &LeafCertificate{
					CommonName: commonName,
				}
				cert.generateCertificate = nil
				cert.generateKey = generateKey
				cert.generateSerialNumber = generateSerialNumber

				if err := cert.Generate(rootCA); !errors.Is(err, ErrFunctionNotImplemented) {
					t.Fatalf("expected function not implemented error")
				}
			})

			t.Run("generate key not set", func(t *testing.T) {
				cert := &LeafCertificate{
					CommonName: commonName,
				}
				cert.generateCertificate = generateLeafCertificate
				cert.generateKey = nil
				cert.generateSerialNumber = generateSerialNumber

				if err := cert.Generate(rootCA); !errors.Is(err, ErrFunctionNotImplemented) {
					t.Fatalf("expected function not implemented error")
				}
			})

			t.Run("generate serial number not set", func(t *testing.T) {
				cert := &LeafCertificate{
					CommonName: commonName,
				}
				cert.generateCertificate = generateLeafCertificate
				cert.generateKey = generateKey
				cert.generateSerialNumber = nil

				if err := cert.Generate(rootCA); !errors.Is(err, ErrFunctionNotImplemented) {
					t.Fatalf("expected function not implemented error")
				}
			})

			t.Run("CommonName not set", func(t *testing.T) {
				cert := &LeafCertificate{
					generateCertificate:  generateLeafCertificate,
					generateKey:          generateKey,
					generateSerialNumber: generateSerialNumber,
				}

				if err := cert.Generate(rootCA); !errors.Is(err, ErrMissingRequired) {
					t.Fatalf("expected missing required error")
				}
			})

			t.Run("root certificate authority is nil", func(t *testing.T) {
				cert := &LeafCertificate{
					CommonName: commonName,
				}
				cert.generateCertificate = generateLeafCertificate
				cert.generateKey = generateKey
				cert.generateSerialNumber = generateSerialNumber

				if err := cert.Generate(nil); !errors.Is(err, ErrInvalidCertificateAuthority) {
					t.Log(err)
				}
			})

			t.Run("root certificate authority has no private key", func(t *testing.T) {
				cert := &LeafCertificate{
					CommonName:           commonName,
					generateCertificate:  generateLeafCertificate,
					generateKey:          generateKey,
					generateSerialNumber: generateSerialNumber,
				}
				rootCA := NewRootCertificateAuthority()
				rootCA.PrivateKey = nil

				if err := cert.Generate(rootCA); !errors.Is(err, ErrInvalidCertificateAuthority) {
					t.Fatalf("expected invalid certificate authority")
				}
			})

			t.Run("root certificate authority has no certificate", func(t *testing.T) {
				cert := &LeafCertificate{
					CommonName:           commonName,
					generateCertificate:  generateLeafCertificate,
					generateKey:          generateKey,
					generateSerialNumber: generateSerialNumber,
				}
				rootCA := NewRootCertificateAuthority()
				if err := rootCA.Generate(); err != nil {
					t.Fatalf("root certificate authority could not be generated")
				}
				rootCA.Certificate = nil

				if err := cert.Generate(rootCA); !errors.Is(err, ErrInvalidCertificateAuthority) {
					t.Fatalf("expected invalid certificate authority")
				}
			})

			t.Run("root certificate authority has invalid certificate", func(t *testing.T) {
				cert := &LeafCertificate{
					CommonName:           commonName,
					generateCertificate:  generateLeafCertificate,
					generateKey:          generateKey,
					generateSerialNumber: generateSerialNumber,
				}
				rootCA := NewRootCertificateAuthority()
				if err := rootCA.Generate(); err != nil {
					t.Fatalf("root certificate authority could not be generated")
				}
				rootCA.Certificate.Certificate = []byte{}

				if err := cert.Generate(rootCA); err == nil {
					t.Fatalf("expected certificate parsing error")
				}
			})

			t.Run("cannot generate private key", func(t *testing.T) {
				msg := "cannot generate private key"
				cert := &LeafCertificate{
					CommonName:           commonName,
					generateCertificate:  generateLeafCertificate,
					generateKey:          func() (*ecdsa.PrivateKey, error) { return nil, errors.New(msg) },
					generateSerialNumber: generateSerialNumber,
				}

				if err := cert.Generate(rootCA); err.Error() != msg {
					t.Fatalf("expected error: %s", msg)
				}
			})

			t.Run("cannot generate serial number", func(t *testing.T) {
				msg := "cannot generate serial number"
				cert := &LeafCertificate{
					CommonName:           commonName,
					generateCertificate:  generateLeafCertificate,
					generateKey:          generateKey,
					generateSerialNumber: func() (*big.Int, error) { return nil, errors.New(msg) },
				}

				if err := cert.Generate(rootCA); err.Error() != msg {
					t.Fatalf("expected error: %s", msg)
				}
			})

			t.Run("cannot generate certificate", func(t *testing.T) {
				msg := "cannot generate certificate"
				cert := &LeafCertificate{
					CommonName: commonName,
					generateCertificate: func(*ecdsa.PrivateKey, *big.Int, *RootCertificateAuthority, string, []string, []net.IP) ([]byte, error) {
						return nil, errors.New(msg)
					},
					generateKey:          generateKey,
					generateSerialNumber: generateSerialNumber,
				}

				if err := cert.Generate(rootCA); err.Error() != msg {
					t.Fatalf("expected error: %s", msg)
				}
			})
		})
	})
}

func TestNewLeafCertificate(t *testing.T) {
	namespace := "pgo-test"
	commonName := "hippo." + namespace
	dnsNames := []string{commonName}
	cert := NewLeafCertificate(commonName, dnsNames, []net.IP{})

	if cert.CommonName != commonName {
		t.Fatalf("expected commonName to be %s, actual %s", commonName, cert.CommonName)
	}

	if !reflect.DeepEqual(cert.DNSNames, dnsNames) {
		t.Fatalf("expected dnsNames to be %v, actual %v", dnsNames, cert.DNSNames)
	}

	if cert.generateCertificate == nil {
		t.Fatalf("expected generateCertificate to be set, got nil")
	}

	if cert.generateKey == nil {
		t.Fatalf("expected generateKey to be set, got nil")
	}

	if cert.generateSerialNumber == nil {
		t.Fatalf("expected generateSerialNumber to be set, got nil")
	}

	// run generate to ensure it sets valid values...which means
	// generating a root certificate
	rootCA := NewRootCertificateAuthority()
	if err := rootCA.Generate(); err != nil {
		t.Fatalf("root certificate authority could not be generated")
	}

	// ok...let's see if this works
	if err := cert.Generate(rootCA); err != nil {
		t.Fatalf("expected generate to return no errors, got: %s", err.Error())
	}

	// ensure private key and certificate are set
	if cert.PrivateKey == nil {
		t.Fatalf("expected private key to be set")
	}

	if cert.Certificate == nil {
		t.Fatalf("expected certificate to be set")
	}
}

func TestLeafCertIsBad(t *testing.T) {
	ctx := context.Background()
	testRoot, err := newTestRoot()
	assert.NilError(t, err)

	namespace := "pgo-test"
	commonName := "hippo." + namespace
	dnsNames := []string{commonName, "hippo." + namespace + ".svc"}
	ipAddresses := []net.IP{net.ParseIP("127.0.0.1")}

	testLeaf := &LeafCertificate{
		CommonName:           commonName,
		DNSNames:             dnsNames,
		IPAddresses:          ipAddresses,
		generateCertificate:  generateLeafCertificate,
		generateKey:          generateKey,
		generateSerialNumber: generateSerialNumber,
	}

	// run generate to ensure it sets valid values
	err = testLeaf.Generate(testRoot)
	assert.NilError(t, err)

	t.Run("leaf cert is good", func(t *testing.T) {

		assert.Assert(t, !LeafCertIsBad(ctx, testLeaf, testRoot, namespace))
	})

	t.Run("leaf cert is empty", func(t *testing.T) {

		emptyLeaf := &LeafCertificate{}
		assert.Assert(t, LeafCertIsBad(ctx, emptyLeaf, testRoot, namespace))
	})

	t.Run("error parsing root certificate", func(t *testing.T) {
		testRoot.Certificate = &Certificate{
			Certificate: []byte("notacert"),
		}

		assert.Assert(t, LeafCertIsBad(ctx, testLeaf, testRoot, namespace))
	})

	t.Run("error parsing leaf certificate", func(t *testing.T) {

		testRoot2, err := newTestRoot()
		assert.NilError(t, err)

		testLeaf.Certificate = &Certificate{
			Certificate: []byte("notacert"),
		}

		assert.Assert(t, LeafCertIsBad(ctx, testLeaf, testRoot2, namespace))
	})

	t.Run("leaf with invalid constraint", func(t *testing.T) {

		testRoot3, err := newTestRoot()
		assert.NilError(t, err)

		badLeaf := &LeafCertificate{
			CommonName:           commonName,
			DNSNames:             dnsNames,
			IPAddresses:          ipAddresses,
			generateCertificate:  generateLeafCertificateInvalidConstraint,
			generateKey:          generateKey,
			generateSerialNumber: generateSerialNumber,
		}

		// run generate to ensure it sets valid values
		err = badLeaf.Generate(testRoot3)
		assert.NilError(t, err)

		assert.Assert(t, LeafCertIsBad(ctx, badLeaf, testRoot3, namespace))

	})

	t.Run("leaf is a expired", func(t *testing.T) {

		testRoot3, err := newTestRoot()
		assert.NilError(t, err)

		badLeaf := &LeafCertificate{
			CommonName:           commonName,
			DNSNames:             dnsNames,
			IPAddresses:          ipAddresses,
			generateCertificate:  generateLeafCertificateExpired,
			generateKey:          generateKey,
			generateSerialNumber: generateSerialNumber,
		}

		// run generate to ensure it sets valid values
		err = badLeaf.Generate(testRoot3)
		assert.NilError(t, err)

		assert.Assert(t, LeafCertIsBad(ctx, badLeaf, testRoot3, namespace))

	})
}

// generateLeafCertificateInvalidConstraint creates a x509 certificate with BasicConstraintsValid set to false
func generateLeafCertificateInvalidConstraint(privateKey *ecdsa.PrivateKey, serialNumber *big.Int,
	rootCA *RootCertificateAuthority, commonName string, dnsNames []string, ipAddresses []net.IP) ([]byte, error) {
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
		BasicConstraintsValid: false,
		DNSNames:              dnsNames,
		IPAddresses:           ipAddresses,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		NotBefore:             now.Add(beforeInterval),
		NotAfter:              now.Add(defaultCertificateExpiration),
		SerialNumber:          serialNumber,
		SignatureAlgorithm:    certificateSignatureAlgorithm,
		Subject: pkix.Name{
			CommonName: commonName,
		},
	}

	// create the leaf certificate and sign it using the root CA
	return x509.CreateCertificate(rand.Reader, template, parent,
		privateKey.Public(), rootCA.PrivateKey.PrivateKey)
}

// generateLeafCertificateExpired creates a x509 certificate that is expired
func generateLeafCertificateExpired(privateKey *ecdsa.PrivateKey, serialNumber *big.Int,
	rootCA *RootCertificateAuthority, commonName string, dnsNames []string, ipAddresses []net.IP) ([]byte, error) {
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
		DNSNames:              dnsNames,
		IPAddresses:           ipAddresses,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		NotBefore:             now.Add(beforeInterval),
		NotAfter:              now.Add(beforeInterval), // not after an hour ago, i.e. expired
		SerialNumber:          serialNumber,
		SignatureAlgorithm:    certificateSignatureAlgorithm,
		Subject: pkix.Name{
			CommonName: commonName,
		},
	}

	// create the leaf certificate and sign it using the root CA
	return x509.CreateCertificate(rand.Reader, template, parent,
		privateKey.Public(), rootCA.PrivateKey.PrivateKey)
}
