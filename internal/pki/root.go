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
	"math/big"
	"time"
)

const (
	// defaultRootCAExpiration sets the default time for the root CA, which is
	// placed far enough into the future
	defaultRootCAExpiration = 10 * 365 * 24 * time.Hour

	// rootCAName is the name of the root CA
	rootCAName = "postgres-operator-ca"
)

// RootCertificateAuthority contains the ability to generate the necessary
// components of a root certificate authority (root CA). This includes the
// private key for the root CA as well as its certificate, which is self-signed
// (as is the nature of a root CA).
//
// In the context of the Operator, this is available "cluster wide" and should
// reside in the same namespace that the Operator is deployed into.
type RootCertificateAuthority struct {
	// Certificate is the certificate of this certificate authority
	Certificate *Certificate

	// PrivateKey is the private key portion of the certificate authority
	PrivateKey *PrivateKey

	// generateKey generates an ECDSA keypair
	generateKey func() (*ecdsa.PrivateKey, error)

	// generateCertificate generates a X509 certificate return in DER format
	generateCertificate func(*ecdsa.PrivateKey, *big.Int) ([]byte, error)

	// generateSerialNumber creates a unique serial number to assign to the
	// certificate
	generateSerialNumber func() (*big.Int, error)
}

// Generate creates a new root certificate authority
func (ca *RootCertificateAuthority) Generate() error {
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
	if certificate, err := ca.generateCertificate(ca.PrivateKey.PrivateKey, serialNumber); err != nil {
		return err
	} else {
		ca.Certificate = &Certificate{Certificate: certificate}
	}

	return nil
}

// NewRootCertificateAuthority generates a new root certificate authority that
// can be used to issue intermediate certificate authoritities
func NewRootCertificateAuthority() *RootCertificateAuthority {
	return &RootCertificateAuthority{
		generateCertificate:  generateRootCertificate,
		generateKey:          generateKey,
		generateSerialNumber: generateSerialNumber,
	}
}

// ParseRootCertificateAuthority takes a PEM encoded private key and certificate
// representation and attempts to parse it.
func ParseRootCertificateAuthority(privateKey, certificate []byte) (*RootCertificateAuthority, error) {
	var err error
	ca := NewRootCertificateAuthority()

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

// RootCAIsBad checks that the root CA has been generated, is not expired
// and has been issued by the Postgres Operator
func RootCAIsBad(root *RootCertificateAuthority) bool {
	// if the certificate or the private key are nil, the root CA is bad
	if root.Certificate == nil || root.PrivateKey == nil {
		return true
	}

	var certs []*x509.Certificate
	var err error
	// if there is an error parsing the certificate or if the number of certificates returned
	// is not one, the certificate is bad
	if certs, err = x509.ParseCertificates(root.Certificate.Certificate); err != nil && len(certs) != 1 {
		return true
	}

	// find our root cert in the returned slice
	var rootCert *x509.Certificate
	for _, cert := range certs {
		if cert.Issuer.CommonName == "postgres-operator-ca" {
			rootCert = cert
		}
	}

	// if our root cert was not found, return so new cert can be generated
	if rootCert == nil {
		return true
	}

	// root cert is bad if it is not a CA
	if !rootCert.IsCA {
		return true
	}

	// finally, if it is outside of the certs configured valid time range or the common name
	// does not match ours, return true
	return time.Now().After(rootCert.NotAfter) || time.Now().Before(rootCert.NotBefore)

}

// generateRootCertificate creates a x509 certificate with a ECDSA signature using
// the SHA-384 algorithm
func generateRootCertificate(privateKey *ecdsa.PrivateKey, serialNumber *big.Int) ([]byte, error) {
	// prepare the certificate. set the validity time to the predefined range
	now := time.Now()
	template := &x509.Certificate{
		BasicConstraintsValid: true,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		MaxPathLen:            1, // as each namespace will have an intermediate CA
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
