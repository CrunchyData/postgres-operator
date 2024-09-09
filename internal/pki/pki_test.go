// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pki

import (
	"crypto/ecdsa"
	"crypto/x509"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/internal/testing/require"
)

type StringSet map[string]struct{}

func (s StringSet) Has(item string) bool { _, ok := s[item]; return ok }
func (s StringSet) Insert(item string)   { s[item] = struct{}{} }

func TestCertificateCommonName(t *testing.T) {
	zero := Certificate{}
	assert.Assert(t, zero.CommonName() == "")
}

func TestCertificateDNSNames(t *testing.T) {
	zero := Certificate{}
	assert.Assert(t, zero.DNSNames() == nil)
}

func TestCertificateHasSubject(t *testing.T) {
	zero := Certificate{}

	// The zero value has no subject.
	for _, cn := range []string{"", "any"} {
		for _, dns := range [][]string{nil, {}, {"any"}} {
			assert.Assert(t, !zero.hasSubject(cn, dns), "for (%q, %q)", cn, dns)
		}
	}
}

func TestCertificateEqual(t *testing.T) {
	zero := Certificate{}
	assert.Assert(t, zero.Equal(zero))

	root, err := NewRootCertificateAuthority()
	assert.NilError(t, err)
	assert.Assert(t, root.Certificate.Equal(root.Certificate))

	assert.Assert(t, !root.Certificate.Equal(zero))
	assert.Assert(t, !zero.Equal(root.Certificate))

	other, err := NewRootCertificateAuthority()
	assert.NilError(t, err)
	assert.Assert(t, !root.Certificate.Equal(other.Certificate))

	// DeepEqual calls the Equal method, so no cmp.Option are necessary.
	assert.DeepEqual(t, zero, zero)
	assert.DeepEqual(t, root.Certificate, root.Certificate)
}

func TestPrivateKeyEqual(t *testing.T) {
	zero := PrivateKey{}
	assert.Assert(t, zero.Equal(zero))

	root, err := NewRootCertificateAuthority()
	assert.NilError(t, err)
	assert.Assert(t, root.PrivateKey.Equal(root.PrivateKey))

	assert.Assert(t, !root.PrivateKey.Equal(zero))
	assert.Assert(t, !zero.Equal(root.PrivateKey))

	other, err := NewRootCertificateAuthority()
	assert.NilError(t, err)
	assert.Assert(t, !root.PrivateKey.Equal(other.PrivateKey))

	// DeepEqual calls the Equal method, so no cmp.Option are necessary.
	assert.DeepEqual(t, zero, zero)
	assert.DeepEqual(t, root.PrivateKey, root.PrivateKey)
}

func TestRootCertificateAuthority(t *testing.T) {
	root, err := NewRootCertificateAuthority()
	assert.NilError(t, err)
	assert.Assert(t, root != nil)

	cert := root.Certificate.x509
	assert.Assert(t, RootIsValid(root), "got %#v", cert)

	assert.DeepEqual(t, cert.Issuer, cert.Subject)            // self-signed
	assert.Assert(t, cert.BasicConstraintsValid && cert.IsCA) // authority
	assert.Assert(t, time.Now().After(cert.NotBefore), "early, got %v", cert.NotBefore)
	assert.Assert(t, time.Now().Before(cert.NotAfter), "expired, got %v", cert.NotAfter)

	assert.Equal(t, cert.MaxPathLen, 0)
	assert.Equal(t, cert.PublicKeyAlgorithm, x509.ECDSA)
	assert.Equal(t, cert.SignatureAlgorithm, x509.ECDSAWithSHA384)
	assert.Equal(t, cert.Subject.CommonName, "postgres-operator-ca")
	assert.Equal(t, cert.KeyUsage, x509.KeyUsageCertSign|x509.KeyUsageCRLSign)

	assert.Assert(t, cert.DNSNames == nil)
	assert.Assert(t, cert.EmailAddresses == nil)
	assert.Assert(t, cert.IPAddresses == nil)
	assert.Assert(t, cert.URIs == nil)

	// The Subject Key Identifier extension is necessary on CAs.
	// The "crypto/x509" package adds it automatically since Go 1.15.
	// - https://tools.ietf.org/html/rfc5280#section-4.2.1.2
	// - https://go.dev/doc/go1.15#crypto/x509
	assert.Assert(t, len(cert.SubjectKeyId) > 0)

	// The Subject field must be populated on CAs.
	// - https://tools.ietf.org/html/rfc5280#section-4.1.2.6
	assert.Assert(t, len(cert.Subject.Names) > 0)

	root2, err := NewRootCertificateAuthority()
	assert.NilError(t, err)
	assert.Assert(t, root2 != nil)

	cert2 := root2.Certificate.x509
	assert.Assert(t, RootIsValid(root2), "got %#v", cert2)

	assert.Assert(t, cert2.SerialNumber.Cmp(cert.SerialNumber) != 0, "new serial")
	assert.Assert(t, !cert2.PublicKey.(*ecdsa.PublicKey).Equal(cert.PublicKey), "new key")

	// The root certificate cannot be verified independently by OpenSSL because
	// it is self-signed. OpenSSL does perform some checks when it is part of
	// a proper chain in [TestLeafCertificate].
}

func TestRootIsInvalid(t *testing.T) {
	t.Run("NoCertificate", func(t *testing.T) {
		assert.Assert(t, !RootIsValid(nil))
		assert.Assert(t, !RootIsValid(&RootCertificateAuthority{}))

		root, err := NewRootCertificateAuthority()
		assert.NilError(t, err)

		root.Certificate = Certificate{}
		assert.Assert(t, !RootIsValid(root))
	})

	t.Run("NoPrivateKey", func(t *testing.T) {
		root, err := NewRootCertificateAuthority()
		assert.NilError(t, err)

		root.PrivateKey = PrivateKey{}
		assert.Assert(t, !RootIsValid(root))
	})

	t.Run("WrongPrivateKey", func(t *testing.T) {
		root, err := NewRootCertificateAuthority()
		assert.NilError(t, err)

		other, err := NewRootCertificateAuthority()
		assert.NilError(t, err)

		root.PrivateKey = other.PrivateKey
		assert.Assert(t, !RootIsValid(root))
	})

	t.Run("NotAuthority", func(t *testing.T) {
		root, err := NewRootCertificateAuthority()
		assert.NilError(t, err)

		leaf, err := root.GenerateLeafCertificate("", nil)
		assert.NilError(t, err)

		assert.Assert(t, !RootIsValid((*RootCertificateAuthority)(leaf)))
	})

	t.Run("TooEarly", func(t *testing.T) {
		original := currentTime
		t.Cleanup(func() { currentTime = original })

		currentTime = func() time.Time {
			return time.Now().Add(time.Hour * 24) // tomorrow
		}

		root, err := NewRootCertificateAuthority()
		assert.NilError(t, err)

		assert.Assert(t, !RootIsValid(root))
	})

	t.Run("Expired", func(t *testing.T) {
		original := currentTime
		t.Cleanup(func() { currentTime = original })

		currentTime = func() time.Time {
			return time.Date(2010, time.January, 1, 0, 0, 0, 0, time.Local)
		}

		root, err := NewRootCertificateAuthority()
		assert.NilError(t, err)

		assert.Assert(t, !RootIsValid(root))
	})
}

func TestLeafCertificate(t *testing.T) {
	serials := StringSet{}
	root, err := NewRootCertificateAuthority()
	assert.NilError(t, err)

	for _, tt := range []struct {
		test       string
		commonName string
		dnsNames   []string
	}{
		{
			test: "OnlyCommonName", commonName: "some-cn",
		},
		{
			test: "OnlyDNSNames", dnsNames: []string{"local-name", "sub.domain"},
		},
	} {
		t.Run(tt.test, func(t *testing.T) {
			leaf, err := root.GenerateLeafCertificate(tt.commonName, tt.dnsNames)
			assert.NilError(t, err)
			assert.Assert(t, leaf != nil)

			cert := leaf.Certificate.x509
			assert.Assert(t, root.leafIsValid(leaf), "got %#v", cert)

			number := cert.SerialNumber.String()
			assert.Assert(t, !serials.Has(number))
			serials.Insert(number)

			assert.Equal(t, cert.Issuer.CommonName, "postgres-operator-ca")
			assert.Assert(t, cert.BasicConstraintsValid && !cert.IsCA)
			assert.Assert(t, time.Now().After(cert.NotBefore), "early, got %v", cert.NotBefore)
			assert.Assert(t, time.Now().Before(cert.NotAfter), "expired, got %v", cert.NotAfter)

			assert.Equal(t, cert.PublicKeyAlgorithm, x509.ECDSA)
			assert.Equal(t, cert.SignatureAlgorithm, x509.ECDSAWithSHA384)
			assert.Equal(t, cert.KeyUsage, x509.KeyUsageDigitalSignature|x509.KeyUsageKeyEncipherment)

			assert.Equal(t, cert.Subject.CommonName, tt.commonName)
			assert.DeepEqual(t, cert.DNSNames, tt.dnsNames)
			assert.Assert(t, cert.EmailAddresses == nil)
			assert.Assert(t, cert.IPAddresses == nil)
			assert.Assert(t, cert.URIs == nil)

			// CAs must include the Authority Key Identifier on new certificates.
			// The "crypto/x509" package adds it automatically since Go 1.15.
			// - https://tools.ietf.org/html/rfc5280#section-4.2.1.1
			// - https://go.dev/doc/go1.15#crypto/x509
			assert.DeepEqual(t,
				leaf.Certificate.x509.AuthorityKeyId,
				root.Certificate.x509.SubjectKeyId)

			// CAs must include their entire Subject on new certificates.
			// - https://tools.ietf.org/html/rfc5280#section-4.1.2.6
			assert.DeepEqual(t,
				leaf.Certificate.x509.Issuer,
				root.Certificate.x509.Subject)

			t.Run("OpenSSLVerify", func(t *testing.T) {
				openssl := require.OpenSSL(t)

				t.Run("Basic", func(t *testing.T) {
					basicOpenSSLVerify(t, openssl, root.Certificate, leaf.Certificate)
				})

				t.Run("Strict", func(t *testing.T) {
					strictOpenSSLVerify(t, openssl, root.Certificate, leaf.Certificate)
				})
			})

			t.Run("Subject", func(t *testing.T) {
				assert.Equal(t,
					leaf.Certificate.CommonName(), tt.commonName)
				assert.DeepEqual(t,
					leaf.Certificate.DNSNames(), tt.dnsNames)
				assert.Assert(t,
					leaf.Certificate.hasSubject(tt.commonName, tt.dnsNames))

				for _, other := range []struct {
					test       string
					commonName string
					dnsNames   []string
				}{
					{
						test:       "DifferentCommonName",
						commonName: "other",
						dnsNames:   tt.dnsNames,
					},
					{
						test:       "DifferentDNSNames",
						commonName: tt.commonName,
						dnsNames:   []string{"other"},
					},
					{
						test:       "DNSNameSubset",
						commonName: tt.commonName,
						dnsNames:   []string{"local-name"},
					},
				} {
					assert.Assert(t,
						!leaf.Certificate.hasSubject(other.commonName, other.dnsNames))
				}
			})
		})
	}
}

func TestLeafIsInvalid(t *testing.T) {
	root, err := NewRootCertificateAuthority()
	assert.NilError(t, err)

	t.Run("ZeroRoot", func(t *testing.T) {
		zero := RootCertificateAuthority{}
		assert.Assert(t, !zero.leafIsValid(nil))

		leaf, err := root.GenerateLeafCertificate("", nil)
		assert.NilError(t, err)

		assert.Assert(t, !zero.leafIsValid(leaf))
	})

	t.Run("NoCertificate", func(t *testing.T) {
		assert.Assert(t, !root.leafIsValid(nil))
		assert.Assert(t, !root.leafIsValid(&LeafCertificate{}))

		leaf, err := root.GenerateLeafCertificate("", nil)
		assert.NilError(t, err)

		leaf.Certificate = Certificate{}
		assert.Assert(t, !root.leafIsValid(leaf))
	})

	t.Run("NoPrivateKey", func(t *testing.T) {
		leaf, err := root.GenerateLeafCertificate("", nil)
		assert.NilError(t, err)

		leaf.PrivateKey = PrivateKey{}
		assert.Assert(t, !root.leafIsValid(leaf))
	})

	t.Run("WrongPrivateKey", func(t *testing.T) {
		leaf, err := root.GenerateLeafCertificate("", nil)
		assert.NilError(t, err)

		other, err := root.GenerateLeafCertificate("", nil)
		assert.NilError(t, err)

		leaf.PrivateKey = other.PrivateKey
		assert.Assert(t, !root.leafIsValid(leaf))
	})

	t.Run("IsAuthority", func(t *testing.T) {
		assert.Assert(t, !root.leafIsValid((*LeafCertificate)(root)))
	})

	t.Run("TooEarly", func(t *testing.T) {
		original := currentTime
		t.Cleanup(func() { currentTime = original })

		currentTime = func() time.Time {
			return time.Now().Add(time.Hour * 24) // tomorrow
		}

		leaf, err := root.GenerateLeafCertificate("", nil)
		assert.NilError(t, err)

		assert.Assert(t, !root.leafIsValid(leaf))
	})

	t.Run("PastRenewalTime", func(t *testing.T) {
		// Generate a cert with the default valid times,
		// e.g., 1 hour before now until 1 year from now
		leaf, err := root.GenerateLeafCertificate("", nil)
		assert.NilError(t, err)

		// set the time now to be over 2/3rds of a year for checking
		original := currentTime
		t.Cleanup(func() { currentTime = original })

		currentTime = func() time.Time {
			return time.Now().Add(time.Hour * 24 * 330)
		}

		assert.Assert(t, !root.leafIsValid(leaf))
	})

	t.Run("Expired", func(t *testing.T) {
		original := currentTime
		t.Cleanup(func() { currentTime = original })

		currentTime = func() time.Time {
			return time.Date(2010, time.January, 1, 0, 0, 0, 0, time.Local)
		}

		leaf, err := root.GenerateLeafCertificate("", nil)
		assert.NilError(t, err)

		assert.Assert(t, !root.leafIsValid(leaf))
	})
}

func TestIsBeforeRenewalTime(t *testing.T) {
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	twoHoursInTheFuture := time.Now().Add(2 * time.Hour)

	assert.Assert(t, isBeforeRenewalTime(oneHourAgo, twoHoursInTheFuture))

	sixHoursAgo := time.Now().Add(-6 * time.Hour)
	assert.Assert(t, !isBeforeRenewalTime(sixHoursAgo, twoHoursInTheFuture))
}

func TestRegenerateLeaf(t *testing.T) {
	root, err := NewRootCertificateAuthority()
	assert.NilError(t, err)

	before, err := root.GenerateLeafCertificate("before", nil)
	assert.NilError(t, err)

	// Leaf is the same when the subject is the same.
	same, err := root.RegenerateLeafWhenNecessary(before, "before", nil)
	assert.NilError(t, err)
	assert.DeepEqual(t, same, before)

	after, err := root.RegenerateLeafWhenNecessary(before, "after", nil)
	assert.NilError(t, err)
	assert.DeepEqual(t, same, before) // Argument does not change.

	assert.Assert(t, after.Certificate.hasSubject("after", nil))
	assert.Assert(t, !after.Certificate.Equal(before.Certificate))
}

func basicOpenSSLVerify(t *testing.T, openssl string, root, leaf Certificate) {
	verify := func(t testing.TB, args ...string) {
		t.Helper()
		// #nosec G204 -- args from this test
		cmd := exec.Command(openssl, append([]string{"verify"}, args...)...)

		output, err := cmd.CombinedOutput()
		assert.NilError(t, err, "%q\n%s", cmd.Args, output)
	}

	dir := t.TempDir()

	rootFile := filepath.Join(dir, "root.crt")
	rootBytes, err := root.MarshalText()
	assert.NilError(t, err)
	assert.NilError(t, os.WriteFile(rootFile, rootBytes, 0o600))

	// The root certificate cannot be verified independently because it is self-signed.
	// It is checked below by being the specified CA.

	leafFile := filepath.Join(dir, "leaf.crt")
	leafBytes, err := leaf.MarshalText()
	assert.NilError(t, err)
	assert.NilError(t, os.WriteFile(leafFile, leafBytes, 0o600))

	// Older versions of the "openssl verify" command cannot properly verify
	// a certificate chain that contains intermediates. When the only flag
	// available is "-CAfile", intermediates must be bundled there and are
	// *implicitly trusted*. The [strictOpenSSLVerify] function is able to
	// verify the chain properly.
	// - https://mail.python.org/pipermail/cryptography-dev/2016-August/000676.html

	// TODO(cbandy): When we generate intermediate certificates, verify them
	// independently then bundle them with the root to verify the leaf.

	verify(t, "-CAfile", rootFile, leafFile)
	verify(t, "-CAfile", rootFile, "-purpose", "sslclient", leafFile)
	verify(t, "-CAfile", rootFile, "-purpose", "sslserver", leafFile)
}

func strictOpenSSLVerify(t *testing.T, openssl string, root, leaf Certificate) {
	output, _ := exec.Command(openssl, "verify", "-help").CombinedOutput()
	if !strings.Contains(string(output), "-x509_strict") {
		t.Skip(`requires "-x509_strict" flag`)
	}
	if !strings.Contains(string(output), "-no-CAfile") {
		t.Skip(`requires a flag to ignore system certificates`)
	}

	verify := func(t testing.TB, args ...string) {
		t.Helper()
		// #nosec G204 -- args from this test
		cmd := exec.Command(openssl, append([]string{"verify",
			// Do not use the default trusted CAs.
			"-no-CAfile", "-no-CApath",
			// Disable "non-compliant workarounds for broken certificates".
			"-x509_strict",
		}, args...)...)

		output, err := cmd.CombinedOutput()
		assert.NilError(t, err, "%q\n%s", cmd.Args, output)
	}

	dir := t.TempDir()

	rootFile := filepath.Join(dir, "root.crt")
	rootBytes, err := root.MarshalText()
	assert.NilError(t, err)
	assert.NilError(t, os.WriteFile(rootFile, rootBytes, 0o600))

	// The root certificate cannot be verified independently because it is self-signed.
	// Some checks are performed when it is a "trusted" certificate below.

	leafFile := filepath.Join(dir, "leaf.crt")
	leafBytes, err := leaf.MarshalText()
	assert.NilError(t, err)
	assert.NilError(t, os.WriteFile(leafFile, leafBytes, 0o600))

	// TODO(cbandy): When we generate intermediate certificates, verify them
	// independently then pass them via "-untrusted" to verify the leaf.

	verify(t, "-trusted", rootFile, leafFile)
	verify(t, "-trusted", rootFile, "-purpose", "sslclient", leafFile)
	verify(t, "-trusted", rootFile, "-purpose", "sslserver", leafFile)
}
