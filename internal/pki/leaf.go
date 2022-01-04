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
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/crunchydata/postgres-operator/internal/logging"
)

const (
	// defaultCertificateExpiration sets the default expiration time for a leaf
	// certificate
	defaultCertificateExpiration = 365 * 24 * time.Hour
)

// LeafCertificate contains the ability to generate the necessary components of
// a leaf certificate that can be used to identify a PostgreSQL cluster, or
// a pgBouncer instance, etc. A leaf certificate is signed by a root
// certificate authority.
type LeafCertificate struct {
	// Certificate is the certificate of this certificate authority
	Certificate *Certificate

	// CommonName represents the "common name" (CN) of the certificate.
	CommonName string

	// DNSNames is a list of DNS names that are represented by this certificate.
	DNSNames []string

	// IPAddresses is an optional list of IP addresses that can be represented by
	// this certificate
	IPAddresses []net.IP

	// PrivateKey is the private key portion of the leaf certificate
	PrivateKey *PrivateKey

	// generateKey generates an ECDSA keypair
	generateKey func() (*ecdsa.PrivateKey, error)

	// generateCertificate generates a X509 certificate return in DER format
	generateCertificate func(*ecdsa.PrivateKey, *big.Int, *RootCertificateAuthority, string, []string, []net.IP) ([]byte, error)

	// generateSerialNumber creates a unique serial number to assign to the
	// certificate
	generateSerialNumber func() (*big.Int, error)
}

// Generate creates a new leaf certificate!
func (c *LeafCertificate) Generate(rootCA *RootCertificateAuthority) error {
	// ensure functions are defined
	if c.generateKey == nil || c.generateCertificate == nil || c.generateSerialNumber == nil {
		return ErrFunctionNotImplemented
	}

	// ensure there is a Common NAme
	if c.CommonName == "" {
		return fmt.Errorf("%w: common name is required", ErrMissingRequired)
	}

	// generate a private key
	privateKey, err := c.generateKey()

	if err != nil {
		return err
	}

	c.PrivateKey = NewPrivateKey(privateKey)

	// generate a serial number
	serialNumber, err := c.generateSerialNumber()

	if err != nil {
		return err
	}

	// generate a certificate
	certificate, err := c.generateCertificate(c.PrivateKey.PrivateKey,
		serialNumber, rootCA, c.CommonName, c.DNSNames, c.IPAddresses)

	if err != nil {
		return err
	}

	c.Certificate = &Certificate{Certificate: certificate}

	return nil
}

// LeafCertIsBad checks at least one leaf cert has been generated, the basic constraints
// are valid and it has been verified with the root certpool
//
// TODO(tjmoore4): Currently this will return 'true' if any of the parsed certs
// fail a given check. For scenarios where multiple certs may be returned, such
// as in a BYOC/BYOCA, this will need to be handled so we only generate a new
// certificate for our cert if it is the one that fails.
func LeafCertIsBad(
	ctx context.Context, leaf *LeafCertificate, rootCertCA *RootCertificateAuthority,
	namespace string,
) bool {
	log := logging.FromContext(ctx)

	// if the certificate or the private key are nil, the leaf cert is bad
	if leaf.Certificate == nil || leaf.PrivateKey == nil {
		return true
	}

	// set up root cert pool for leaf cert verification
	var rootCerts []*x509.Certificate
	var rootErr error

	// set up root cert pool
	roots := x509.NewCertPool()

	// if there is an error parsing the root certificate or if there is not at least one certificate,
	// the RootCertificateAuthority is bad
	if rootCerts, rootErr = x509.ParseCertificates(rootCertCA.Certificate.Certificate); rootErr != nil && len(rootCerts) < 1 {
		return true
	}

	// add all the root certs returned to the root pool
	for _, cert := range rootCerts {
		roots.AddCert(cert)
	}

	var leafCerts []*x509.Certificate
	var leafErr error
	// if there is an error parsing the leaf certificate or if the number of certificates
	// returned is not one, the certificate is bad
	if leafCerts, leafErr = x509.ParseCertificates(leaf.Certificate.Certificate); leafErr != nil && len(leafCerts) < 1 {
		return true
	}

	// go through the returned leaf certs and check
	// that they are not CAs and Verify them
	for _, cert := range leafCerts {
		// a leaf cert is bad if it is a CA, or if
		// the MaxPathLen or MaxPathLenZero are invalid
		if !cert.BasicConstraintsValid {
			return true
		}

		// verify leaf cert
		_, verifyError := cert.Verify(x509.VerifyOptions{
			DNSName: cert.DNSNames[0],
			Roots:   roots,
		})
		//log verify error if not nil
		if verifyError != nil {
			log.Error(verifyError, "verify failed for leaf cert")
			return true
		}
	}

	// finally, if no check failed, return false
	return false
}

// NewLeafCertificate generates a new leaf certificate that can be used for the
// identity of a particular instance
//
// Accepts arguments for the common name (CN), the DNS names and the IP
// Addresses that will be represented by this certificate
func NewLeafCertificate(commonName string, dnsNames []string, ipAddresses []net.IP) *LeafCertificate {
	return &LeafCertificate{
		CommonName:           commonName,
		DNSNames:             dnsNames,
		IPAddresses:          ipAddresses,
		generateCertificate:  generateLeafCertificate,
		generateKey:          generateKey,
		generateSerialNumber: generateSerialNumber,
	}
}

// generateLeafCertificate creates a x509 certificate with a ECDSA
// signature using the SHA-384 algorithm
func generateLeafCertificate(privateKey *ecdsa.PrivateKey, serialNumber *big.Int,
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
