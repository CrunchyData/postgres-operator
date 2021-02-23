// +build go1.15

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
	"time"
)

const (
	// defaultIntermediateCAExpiration sets the default expiration time for the
	// intermediate CA, which is placed far enough into the future
	defaultIntermediateCAExpiration = 365 * 24 * time.Hour
)

// IntermediateCertificateAuthority contains the ability to generate the
// necessary components of an intermediate certificate authority (intermediate
// CA). This includes the private key for the root CA as well as its
// certificate, which is signed by a root certificate authority.
//
// For the purposes of the Operator, this is available in an individual watched
// Namespace. One Operator deployment can have multiple intermediate CAs, so
// long as each intermediate CA is scoped to a particular namespace
type IntermediateCertificateAuthority struct {
	// Certificate is the certificate of this certificate authority
	Certificate *Certificate

	// Namespace is the Kubernetes namespace that this certificate authority will
	// be responsible for.
	Namespace string

	// PrivateKey is the private key portion of the certificate authority
	PrivateKey *PrivateKey

	// generateKey generates an ECDSA keypair
	generateKey func() (*ecdsa.PrivateKey, error)

	// generateCertificate generates a X509 certificate return in DER format
	generateCertificate func(*ecdsa.PrivateKey, *big.Int, *RootCertificateAuthority, string) ([]byte, error)

	// generateSerialNumber creates a unique serial number to assign to the
	// certificate
	generateSerialNumber func() (*big.Int, error)
}

// Generate creates a new intermediate certificate authority
func (ca *IntermediateCertificateAuthority) Generate(rootCA *RootCertificateAuthority) error {
	// ensure functions are defined
	if ca.generateKey == nil || ca.generateCertificate == nil || ca.generateSerialNumber == nil {
		return ErrFunctionNotImplemented
	}

	// generate a private key
	if privateKey, err := ca.generateKey(); err != nil {
		return err
	} else {
		ca.PrivateKey = NewPrivateKey(privateKey)
	}

	// generate a serial number
	serialNumber, err := ca.generateSerialNumber()

	if err != nil {
		return err
	}

	// generate a certificate
	if certificate, err := ca.generateCertificate(ca.PrivateKey.PrivateKey, serialNumber, rootCA, ca.Namespace); err != nil {
		return err
	} else {
		ca.Certificate = &Certificate{Certificate: certificate}
	}

	return nil
}

// IntermediateCAIsBad checks that the intermediate CA has been generated, is not expired,
// has been issued by the Postgres Operator and its authority key matches the
// subject key of the current root CA
func IntermediateCAIsBad(intermediate *IntermediateCertificateAuthority, root *RootCertificateAuthority) bool {
	// if the certificate or the private key are nil, the intermediate CA is bad
	if intermediate.Certificate == nil || intermediate.PrivateKey == nil {
		return true
	}

	var intCerts []*x509.Certificate
	var intErr error
	// if there is an error parsing the intermediate certificate or if the number of certificates
	// returned is not one, the certificate is bad
	if intCerts, intErr = x509.ParseCertificates(intermediate.Certificate.Certificate); intErr != nil && len(intCerts) < 1 {
		return true
	}

	// find our intermediate cert in the returned slice
	var intermediateCert *x509.Certificate
	for _, cert := range intCerts {
		if cert.Issuer.CommonName == "postgres-operator-ca" {
			intermediateCert = cert
		}
	}

	// if our intermediate cert was not found, return so new cert can be generated
	if intermediateCert == nil {
		return true
	}

	// intermediate cert is bad if it is not a CA
	if !intermediateCert.IsCA {
		return true
	}

	// if it is outside of the certs configured valid time range, return true
	if time.Now().After(intermediateCert.NotAfter) || time.Now().Before(intermediateCert.NotBefore) {
		return true
	}

	var rootCerts []*x509.Certificate
	var rootErr error
	// if there is an error parsing the root certificate or if the number of certificates returned
	// is not one, the certificate is bad
	if rootCerts, rootErr = x509.ParseCertificates(root.Certificate.Certificate); rootErr != nil && len(rootCerts) < 1 {
		return true
	}

	// find our root cert in the returned slice
	var rootCert *x509.Certificate
	for _, cert := range rootCerts {
		if cert.Issuer.CommonName == "postgres-operator-ca" {
			rootCert = cert
		}
	}

	// if our root cert was not found, return so new cert can be generated
	if rootCert == nil {
		return true
	}

	// set up root cert pool
	roots := x509.NewCertPool()
	roots.AddCert(rootCert)

	// verify intermediate cert
	_, err := intermediateCert.Verify(x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	})

	// finally, if Verify returns an error, intermediate cert is bad
	return err != nil
}

// NewIntermediateCertificateAuthority generates a new intermdiate certificate
// authority that can be used to issue intermediate certificate authoritities.
//
// Accepts an argument for "namespace" which should be set to the namespace that
// this CA will cover.
func NewIntermediateCertificateAuthority(namespace string) *IntermediateCertificateAuthority {
	return &IntermediateCertificateAuthority{
		Namespace:            namespace,
		generateCertificate:  generateIntermediateCertificate,
		generateKey:          generateKey,
		generateSerialNumber: generateSerialNumber,
	}
}

// ParseIntermediateCertificateAuthority takes a PEM encoded private key and
// certificate representation and attempts to parse it.
func ParseIntermediateCertificateAuthority(namespace string, privateKey, certificate []byte) (*IntermediateCertificateAuthority, error) {
	var err error
	ca := NewIntermediateCertificateAuthority(namespace)

	// attempt to parse the private key
	if ca.PrivateKey, err = ParsePrivateKey(privateKey); err != nil {
		return nil, err
	}

	// attempt to parse the certificate
	if ca.Certificate, err = ParseCertificate(certificate); err != nil {
		return nil, err
	}

	return ca, nil
}

// generateIntermediateCertificate creates a x509 certificate with a ECDSA
// signature using the SHA-384 algorithm
func generateIntermediateCertificate(privateKey *ecdsa.PrivateKey, serialNumber *big.Int,
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
		IsCA:                  true,
		// This certificate should only be signing leaf certificates.
		// MaxPathLen is intentionally zero, forbidding any further  intermediates
		// in the chain.
		MaxPathLenZero:     true,
		KeyUsage:           x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		NotBefore:          now.Add(beforeInterval),
		NotAfter:           now.Add(defaultIntermediateCAExpiration),
		SerialNumber:       serialNumber,
		SignatureAlgorithm: certificateSignatureAlgorithm,
		Subject: pkix.Name{
			CommonName: fmt.Sprintf("%s.%s", namespace, rootCAName),
		},
	}

	// create the intermediate certificate and sign it using the root CA
	return x509.CreateCertificate(rand.Reader, template, parent,
		privateKey.Public(), rootCA.PrivateKey.PrivateKey)
}
