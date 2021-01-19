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
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"errors"
	"math/big"
	"reflect"
	"testing"
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

	marshalPrivateKey := func(ca *RootCertificateAuthority, password []byte) []byte {
		ca.PrivateKey.Password = password
		data, _ := ca.PrivateKey.MarshalText()
		return data
	}

	ca := generateRootCertificateAuthority()

	t.Run("valid plaintext", func(t *testing.T) {
		privateKey := marshalPrivateKey(ca, []byte{})
		certificate := marshalCertificate(ca)

		rootCA, err := ParseRootCertificateAuthority(privateKey, certificate, []byte{})

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

	t.Run("valid encrypted", func(t *testing.T) {
		password := make([]byte, 16)
		if _, err := rand.Read(password); err != nil {
			t.Fatalf("could not generate password")
		}
		privateKey := marshalPrivateKey(ca, password)
		certificate := marshalCertificate(ca)

		rootCA, err := ParseRootCertificateAuthority(privateKey, certificate, password)

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

			rootCA, err := ParseRootCertificateAuthority(privateKey, certificate, []byte{})

			if err == nil {
				t.Fatalf("expected error")
			}

			if rootCA != nil {
				t.Fatalf("expected CA to be nil")
			}
		})

		t.Run("bad certificate key", func(t *testing.T) {
			privateKey := marshalPrivateKey(ca, []byte{})
			certificate := []byte("bad")

			rootCA, err := ParseRootCertificateAuthority(privateKey, certificate, []byte{})

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

			if x509Certificate.MaxPathLen != 1 {
				t.Fatalf("expected max path len to be 1, actual %d", x509Certificate.MaxPathLen)
			}

			if x509Certificate.Subject.CommonName != rootCAName {
				t.Fatalf("expected subject name to be %s, actual %s", defaultRootCAExpiration, x509Certificate.Subject.CommonName)
			}

			// ensure private key functions are set
			if ca.PrivateKey.encryptPEMBlock == nil {
				t.Fatalf("expected encryptPEMBlock to be set on private key")
			}

			// ensure private key functions are set
			if ca.PrivateKey.marshalECPrivateKey == nil {
				t.Fatalf("expected marshalECPrivateKey to be set on private key")
			}
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
