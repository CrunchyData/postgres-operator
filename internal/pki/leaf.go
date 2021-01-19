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
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"time"
)

const (
	// defaultCertificateExpiration sets the default expiration time for a leaf
	// certificate
	defaultCertificateExpiration = 90 * 24 * time.Hour
)

// LeafCertificate contains the ability to generate the necessary components of
// a leaf certificate that can be used to identify a PostgreSQL cluster, or
// a pgBouncer instance, etc. A leaf certificate is signed by an intermediate
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
	generateCertificate func(*ecdsa.PrivateKey, *big.Int, *IntermediateCertificateAuthority, string, []string, []net.IP) ([]byte, error)

	// generateSerialNumber creates a unique serial number to assign to the
	// certificate
	generateSerialNumber func() (*big.Int, error)
}

// Generate creates a new leaf certificate!
func (c *LeafCertificate) Generate(intermediateCA *IntermediateCertificateAuthority) error {
	// ensure functions are defined
	if c.generateKey == nil || c.generateCertificate == nil || c.generateSerialNumber == nil {
		return ErrFunctionNotImplemented
	}

	// ensure there is a Common NAme
	if c.CommonName == "" {
		return fmt.Errorf("%w: common name is required", ErrMissingRequired)
	}

	// generate a private key
	if privateKey, err := c.generateKey(); err != nil {
		return err
	} else {
		c.PrivateKey = NewPrivateKey(privateKey)
	}

	// generate a serial number
	serialNumber, err := c.generateSerialNumber()

	if err != nil {
		return err
	}

	// generate a certificate
	if certificate, err := c.generateCertificate(c.PrivateKey.PrivateKey,
		serialNumber, intermediateCA, c.CommonName, c.DNSNames, c.IPAddresses); err != nil {
		return err
	} else {
		c.Certificate = &Certificate{Certificate: certificate}
	}

	return nil
}

// NewCertificate generates a new leaf certificate that can be used for the
// identity of a particular instance
//
// Accepts arguments for the common name (CN), the DNS names and the IP
// Addresses that will be represented by this certificate
func NewCertificate(commonName string, dnsNames []string, ipAddresses []net.IP) *LeafCertificate {
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
	intermediateCA *IntermediateCertificateAuthority, commonName string, dnsNames []string, ipAddresses []net.IP) ([]byte, error) {
	// first, ensure that the intermediate certificate can be turned into a x509
	// Certificate object so it can be used as the parent certificate when
	// generating
	if intermediateCA == nil || intermediateCA.Certificate == nil || intermediateCA.PrivateKey == nil {
		return nil, fmt.Errorf("%w: intermediate certificate authority needs to be generated",
			ErrInvalidCertificateAuthority)
	}

	parent, err := x509.ParseCertificate(intermediateCA.Certificate.Certificate)

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

	// create the leaf certificate and sign it using the intermediate CA
	return x509.CreateCertificate(rand.Reader, template, parent,
		privateKey.Public(), intermediateCA.PrivateKey.PrivateKey)
}
