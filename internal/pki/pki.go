/*
 Copyright 2021 - 2023 Crunchy Data Solutions, Inc.
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

package pki

import (
	"crypto/ecdsa"
	"crypto/x509"
	"math/big"
	"time"
)

const renewalRatio = 3

// Certificate represents an X.509 certificate that conforms to the Internet
// PKI Profile, RFC 5280.
type Certificate struct{ x509 *x509.Certificate }

// PrivateKey represents the private key of a Certificate.
type PrivateKey struct{ ecdsa *ecdsa.PrivateKey }

// Equal reports whether c and other have the same value.
func (c Certificate) Equal(other Certificate) bool {
	return c.x509.Equal(other.x509)
}

// CommonName returns a copy of the certificate common name (ASN.1 OID 2.5.4.3).
func (c Certificate) CommonName() string {
	if c.x509 == nil {
		return ""
	}
	return c.x509.Subject.CommonName
}

// DNSNames returns a copy of the certificate subject alternative names
// (ASN.1 OID 2.5.29.17) that are DNS names.
func (c Certificate) DNSNames() []string {
	if c.x509 == nil || len(c.x509.DNSNames) == 0 {
		return nil
	}
	return append([]string{}, c.x509.DNSNames...)
}

// hasSubject checks that c has these values in its subject.
func (c Certificate) hasSubject(commonName string, dnsNames []string) bool {
	ok := c.x509 != nil &&
		c.x509.Subject.CommonName == commonName &&
		len(c.x509.DNSNames) == len(dnsNames)

	for i := range dnsNames {
		ok = ok && c.x509.DNSNames[i] == dnsNames[i]
	}

	return ok
}

// Equal reports whether k and other have the same value.
func (k PrivateKey) Equal(other PrivateKey) bool {
	if k.ecdsa == nil || other.ecdsa == nil {
		return k.ecdsa == other.ecdsa
	}
	return k.ecdsa.Equal(other.ecdsa)
}

// LeafCertificate is a certificate and private key pair that can be validated
// by RootCertificateAuthority.
type LeafCertificate struct {
	Certificate Certificate
	PrivateKey  PrivateKey
}

// RootCertificateAuthority is a certificate and private key pair that can
// generate other certificates.
type RootCertificateAuthority struct {
	Certificate Certificate
	PrivateKey  PrivateKey
}

// NewRootCertificateAuthority generates a new key and self-signed certificate
// for issuing other certificates.
func NewRootCertificateAuthority() (*RootCertificateAuthority, error) {
	var root RootCertificateAuthority
	var serial *big.Int

	key, err := generateKey()
	if err == nil {
		serial, err = generateSerialNumber()
	}
	if err == nil {
		root.PrivateKey.ecdsa = key
		root.Certificate.x509, err = generateRootCertificate(key, serial)
	}

	return &root, err
}

// RootIsValid checks if root is valid according to this package's policies.
func RootIsValid(root *RootCertificateAuthority) bool {
	if root == nil || root.Certificate.x509 == nil {
		return false
	}

	trusted := x509.NewCertPool()
	trusted.AddCert(root.Certificate.x509)

	// Verify the certificate expiration, basic constraints, key usages, and
	// critical extensions. Trust the certificate as an authority so it is not
	// compared to system roots or sent to the platform certificate verifier.
	_, err := root.Certificate.x509.Verify(x509.VerifyOptions{
		Roots: trusted,
	})

	// Its expiration, key usages, and critical extensions are good.
	ok := err == nil

	// It is an authority with the Subject Key Identifier extension.
	// The "crypto/x509" package adds the extension automatically since Go 1.15.
	// - https://tools.ietf.org/html/rfc5280#section-4.2.1.2
	// - https://go.dev/doc/go1.15#crypto/x509
	ok = ok &&
		root.Certificate.x509.BasicConstraintsValid &&
		root.Certificate.x509.IsCA &&
		len(root.Certificate.x509.SubjectKeyId) > 0

	// It is signed by this private key.
	ok = ok &&
		root.PrivateKey.ecdsa != nil &&
		root.PrivateKey.ecdsa.PublicKey.Equal(root.Certificate.x509.PublicKey)

	return ok
}

// GenerateLeafCertificate generates a new key and certificate signed by root.
func (root *RootCertificateAuthority) GenerateLeafCertificate(
	commonName string, dnsNames []string,
) (*LeafCertificate, error) {
	var leaf LeafCertificate
	var serial *big.Int

	key, err := generateKey()
	if err == nil {
		serial, err = generateSerialNumber()
	}
	if err == nil {
		leaf.PrivateKey.ecdsa = key
		leaf.Certificate.x509, err = generateLeafCertificate(
			root.Certificate.x509, root.PrivateKey.ecdsa, &key.PublicKey, serial,
			commonName, dnsNames)
	}

	return &leaf, err
}

// leafIsValid checks if leaf is valid according to this package's policies and
// is signed by root.
func (root *RootCertificateAuthority) leafIsValid(leaf *LeafCertificate) bool {
	if root == nil || root.Certificate.x509 == nil {
		return false
	}
	if leaf == nil || leaf.Certificate.x509 == nil {
		return false
	}

	trusted := x509.NewCertPool()
	trusted.AddCert(root.Certificate.x509)

	// Go 1.10 enforces name constraints for all names in the certificate.
	// Go 1.15 does not enforce name constraints on the CommonName field.
	// - https://go.dev/doc/go1.10#crypto/x509
	// - https://go.dev/doc/go1.15#commonname
	_, err := leaf.Certificate.x509.Verify(x509.VerifyOptions{
		Roots: trusted,
	})

	// Its expiration, name constraints, key usages, and critical extensions are good.
	ok := err == nil

	// It is not an authority.
	ok = ok &&
		leaf.Certificate.x509.BasicConstraintsValid &&
		!leaf.Certificate.x509.IsCA

	// It is signed by this private key.
	ok = ok &&
		leaf.PrivateKey.ecdsa != nil &&
		leaf.PrivateKey.ecdsa.PublicKey.Equal(leaf.Certificate.x509.PublicKey)

	// It is not yet past the "renewal by" time,
	// as defined by the before and after times of the certificate's expiration
	// and the default ratio
	ok = ok && isBeforeRenewalTime(leaf.Certificate.x509.NotBefore,
		leaf.Certificate.x509.NotAfter)

	return ok
}

// isBeforeRenewalTime checks if the result of `currentTime`
// is after the default renewal time of
// 1/3rds before the certificate's expiry
func isBeforeRenewalTime(before, after time.Time) bool {
	renewalDuration := after.Sub(before) / renewalRatio
	renewalTime := after.Add(-1 * renewalDuration)
	return currentTime().Before(renewalTime)
}

// RegenerateLeafWhenNecessary returns leaf when it is valid according to this
// package's policies, signed by root, and has commonName and dnsNames in its
// subject. Otherwise, it returns a new key and certificate signed by root.
func (root *RootCertificateAuthority) RegenerateLeafWhenNecessary(
	leaf *LeafCertificate, commonName string, dnsNames []string,
) (*LeafCertificate, error) {
	ok := root.leafIsValid(leaf) &&
		leaf.Certificate.hasSubject(commonName, dnsNames)

	if ok {
		return leaf, nil
	}
	return root.GenerateLeafCertificate(commonName, dnsNames)
}
