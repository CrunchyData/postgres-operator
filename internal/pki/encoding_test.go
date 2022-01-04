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
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"reflect"
	"testing"
	"time"
)

// assertConstructed ensures that private key functions are set.
func assertConstructed(t testing.TB, key *PrivateKey) {
	t.Helper()

	if key.marshalECPrivateKey == nil {
		t.Fatalf("expected marshalECPrivateKey to be set on private key")
	}
}

func TestCertificate(t *testing.T) {
	// generateCertificate is a helper function that generates a random private key
	// and ignore any errors. creates a self-signed certificate as we don't need
	// much
	generateCertificate := func() *Certificate {
		key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		now := time.Now()
		template := &x509.Certificate{
			BasicConstraintsValid: true,
			KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
			NotBefore:             now,
			NotAfter:              now.Add(12 * time.Hour),
			SerialNumber:          big.NewInt(1234),
			SignatureAlgorithm:    certificateSignatureAlgorithm,
			Subject: pkix.Name{
				CommonName: "*",
			},
		}

		certificate, _ := x509.CreateCertificate(rand.Reader, template, template, key.Public(), key)

		return &Certificate{Certificate: certificate}
	}

	t.Run("MarshalText", func(t *testing.T) {
		certificate := generateCertificate()

		encoded, err := certificate.MarshalText()

		if err != nil {
			t.Fatalf("something went horribly wrong")
		}

		// test that it matches the value of certificate
		block, _ := pem.Decode(encoded)

		// ensure it's the valid pem type
		if block.Type != pemCertificateType {
			t.Fatalf("expected pem type %q actual %q", block.Type, pemCertificateType)
		}

		// ensure the certificates match
		if !bytes.Equal(certificate.Certificate, block.Bytes) {
			t.Fatalf("pem encoded certificate does not match certificate")
		}
	})

	t.Run("UnmarshalText", func(t *testing.T) {
		expected := generateCertificate()

		t.Run("valid", func(t *testing.T) {
			// manually marshal the certificate
			encoded := pem.EncodeToMemory(&pem.Block{Bytes: expected.Certificate, Type: pemCertificateType})
			c := &Certificate{}

			if err := c.UnmarshalText(encoded); err != nil {
				t.Fatalf("expected no error, got %s", err.Error())
			}

			if !reflect.DeepEqual(expected.Certificate, c.Certificate) {
				t.Fatalf("expected encoded certificate to be unmarshaled in identical format")
			}
		})

		t.Run("invalid", func(t *testing.T) {
			t.Run("not pem", func(t *testing.T) {
				c := &Certificate{}

				if err := c.UnmarshalText([]byte("this is very invalid")); !errors.Is(err, ErrInvalidPEM) {
					t.Fatalf("expected invalid PEM error")
				}
			})

			t.Run("not a certificate", func(t *testing.T) {
				encoded := pem.EncodeToMemory(&pem.Block{Bytes: expected.Certificate, Type: "CEREAL"})
				c := &Certificate{}

				if err := c.UnmarshalText(encoded); !errors.Is(err, ErrInvalidPEM) {
					t.Fatalf("expected invalid PEM error")
				}
			})
		})
	})
}

func TestNewPrivateKey(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	privateKey := NewPrivateKey(key)

	if reflect.TypeOf(privateKey).String() != "*pki.PrivateKey" {
		t.Fatalf("expected *pki.PrivateKey in return")
	}
}

func TestParseCertificate(t *testing.T) {
	// generateCertificate is a helper function that generates a random private key
	// and ignore any errors. creates a self-signed certificate as we don't need
	// much
	generateCertificate := func() *Certificate {
		key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		now := time.Now()
		template := &x509.Certificate{
			BasicConstraintsValid: true,
			KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
			NotBefore:             now,
			NotAfter:              now.Add(12 * time.Hour),
			SerialNumber:          big.NewInt(1234),
			SignatureAlgorithm:    certificateSignatureAlgorithm,
			Subject: pkix.Name{
				CommonName: "*",
			},
		}

		certificate, _ := x509.CreateCertificate(rand.Reader, template, template, key.Public(), key)

		return &Certificate{Certificate: certificate}
	}

	t.Run("valid", func(t *testing.T) {
		expected := generateCertificate()
		encoded := pem.EncodeToMemory(&pem.Block{Bytes: expected.Certificate, Type: pemCertificateType})

		certificate, err := ParseCertificate(encoded)

		if err != nil {
			t.Fatalf("expected no error, actual %s", err.Error())
		}

		if !reflect.DeepEqual(expected.Certificate, certificate.Certificate) {
			t.Fatalf("expected parsed certificate to match expected")
		}
	})

	t.Run("invalid", func(t *testing.T) {
		data := []byte("bad")

		certificate, err := ParseCertificate(data)

		if err == nil {
			t.Fatalf("expected error")
		}

		if certificate != nil {
			t.Fatalf("expected certificate to be nil")
		}
	})
}

func TestParsePrivateKey(t *testing.T) {
	// generatePrivateKey is a helper function that generates a random private key
	// and ignore any errors.
	generatePrivateKey := func() *PrivateKey {
		key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		privateKey := &PrivateKey{PrivateKey: key}
		privateKey.marshalECPrivateKey = marshalECPrivateKey
		return privateKey
	}

	t.Run("valid", func(t *testing.T) {
		expected := generatePrivateKey()

		t.Run("plaintext", func(t *testing.T) {
			b, _ := x509.MarshalECPrivateKey(expected.PrivateKey)
			encoded := pem.EncodeToMemory(&pem.Block{Bytes: b, Type: pemPrivateKeyType})

			privateKey, err := ParsePrivateKey(encoded)

			if err != nil {
				t.Fatalf("expected no error, actual %s", err.Error())
			}

			if !reflect.DeepEqual(expected.PrivateKey, privateKey.PrivateKey) {
				t.Fatalf("expected parsed key to match expected")
			}

			// ensure private key functions are set
			assertConstructed(t, privateKey)
		})
	})

	t.Run("invalid", func(t *testing.T) {
		t.Run("plaintext", func(t *testing.T) {
			data := []byte("bad")

			privateKey, err := ParsePrivateKey(data)

			if err == nil {
				t.Fatalf("expected error")
			}

			if privateKey != nil {
				t.Fatalf("expected private key to be nil")
			}
		})
	})
}

func TestPrivateKey(t *testing.T) {
	// generatePrivateKey is a helper function that generates a random private key
	// and ignore any errors.
	generatePrivateKey := func() *PrivateKey {
		key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		privateKey := &PrivateKey{PrivateKey: key}
		privateKey.marshalECPrivateKey = marshalECPrivateKey
		return privateKey
	}

	t.Run("MarshalText", func(t *testing.T) {
		t.Run("plaintext", func(t *testing.T) {
			t.Run("valid", func(t *testing.T) {
				privateKey := generatePrivateKey()

				encoded, err := privateKey.MarshalText()

				if err != nil {
					t.Fatalf("expected no error, actual: %s", err)
				}

				block, _ := pem.Decode(encoded)

				if block.Type != pemPrivateKeyType {
					t.Fatalf("expected pem type %q, actual %q", pemPrivateKeyType, block.Type)
				}

				decodedKey, err := x509.ParseECPrivateKey(block.Bytes)

				if err != nil {
					t.Fatalf("expected valid ECDSA key, got error: %s", err.Error())
				}

				if !privateKey.PrivateKey.Equal(decodedKey) {
					t.Fatalf("expected private key to match pem encoded key")
				}
			})

			t.Run("invalid", func(t *testing.T) {
				t.Run("ec marshal function not set", func(t *testing.T) {
					privateKey := generatePrivateKey()
					privateKey.marshalECPrivateKey = nil

					_, err := privateKey.MarshalText()

					if !errors.Is(err, ErrFunctionNotImplemented) {
						t.Fatalf("expected function not implemented error")
					}
				})

				t.Run("cannot marshal elliptical curve key", func(t *testing.T) {
					msg := "marshal failed"
					privateKey := generatePrivateKey()
					privateKey.marshalECPrivateKey = func(*ecdsa.PrivateKey) ([]byte, error) {
						return []byte{}, errors.New(msg)
					}

					_, err := privateKey.MarshalText()

					if err.Error() != msg {
						t.Fatalf("expected error: %s", msg)
					}
				})
			})
		})
	})

	t.Run("UnmarshalText", func(t *testing.T) {
		expected := generatePrivateKey()

		t.Run("plaintext", func(t *testing.T) {
			t.Run("valid", func(t *testing.T) {
				// manually marshal the private key
				b, _ := x509.MarshalECPrivateKey(expected.PrivateKey)
				encoded := pem.EncodeToMemory(&pem.Block{Bytes: b, Type: pemPrivateKeyType})
				pk := &PrivateKey{}

				if err := pk.UnmarshalText(encoded); err != nil {
					t.Fatalf("expected no error, got %s", err.Error())
				}

				if !reflect.DeepEqual(expected.PrivateKey, pk.PrivateKey) {
					t.Fatalf("expected encoded private key to be unmarshaled in identical format")
				}
			})

			t.Run("invalid", func(t *testing.T) {
				t.Run("not pem", func(t *testing.T) {
					pk := &PrivateKey{}

					if err := pk.UnmarshalText([]byte("this is very invalid")); !errors.Is(err, ErrInvalidPEM) {
						t.Fatalf("expected invalid PEM error")
					}
				})

				t.Run("not labeled private key", func(t *testing.T) {
					encoded := pem.EncodeToMemory(&pem.Block{Bytes: []byte("bad key"), Type: "CEREAL"})
					pk := &PrivateKey{}

					if err := pk.UnmarshalText(encoded); !errors.Is(err, ErrInvalidPEM) {
						t.Fatalf("expected invalid PEM error")
					}
				})

				t.Run("not a valid private key", func(t *testing.T) {
					encoded := pem.EncodeToMemory(&pem.Block{Bytes: []byte("bad key"), Type: pemPrivateKeyType})
					pk := &PrivateKey{}

					if err := pk.UnmarshalText(encoded); !errors.Is(err, ErrInvalidPEM) {
						t.Fatalf("expected invalid PEM error")
					}
				})
			})
		})
	})
}
