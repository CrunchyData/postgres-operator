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
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"time"

	"github.com/crunchydata/postgres-operator/internal/logging"
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

// IntermediateCAIsBad checks that at least one intermediate CA has been generated and that
// all returned certs are CAs and verified
//
// TODO(tjmoore4): Currently this will return 'true' if any of the parsed certs
// fail a given check. For scenarios where multiple certs may be returned, such
// as in a BYOC/BYOCA, this will need to be handled so we only generate a new
// certificate for our cert if it is the one that fails.
func IntermediateCAIsBad(ctx context.Context, intermediate *IntermediateCertificateAuthority, root *RootCertificateAuthority) bool {
	log := logging.FromContext(ctx)

	// if the certificate or the private key are nil, the intermediate CA is bad
	if intermediate.Certificate == nil || intermediate.PrivateKey == nil {
		return true
	}

	// set up root cert pool for intermediate cert verification
	var rootCerts []*x509.Certificate
	var rootErr error

	// set up root cert pool
	roots := x509.NewCertPool()

	// if there is an error parsing the root certificate or if there is not at least one certificate,
	// the RootCertificateAuthority is bad
	if rootCerts, rootErr = x509.ParseCertificates(root.Certificate.Certificate); rootErr != nil && len(rootCerts) < 1 {
		return true
	}

	// add all the root certs returned to the root pool
	for _, cert := range rootCerts {
		roots.AddCert(cert)
	}

	var intCerts []*x509.Certificate
	var intErr error
	// if there is an error parsing the intermediate certificate or if there is not at least one certificate,
	// the IntermediateCertificateAuthority is bad
	if intCerts, intErr = x509.ParseCertificates(intermediate.Certificate.Certificate); intErr != nil && len(intCerts) < 1 {
		return true
	}

	// go through the returned intermediate certs and check
	// that they are CAs and Verify them
	for _, cert := range intCerts {
		// any cert at this point should be an intermediate
		// if any are not CAs, something went wrong and this
		// IntermediateCertificateAuthority is bad
		if !cert.IsCA || !cert.BasicConstraintsValid {
			return true
		}

		// verify intermediate cert
		_, verifyError := cert.Verify(x509.VerifyOptions{
			Roots: roots,
			// x509.ExtKeyUsageAny effectively ignores the
			// intermediateCert.ExtKeyUsage check.
			// We don't want this default check, x509.ExtKeyUsageServerAuth,
			// to be used against the intermediate CAs.
			KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		})
		//log verify error if not nil
		if verifyError != nil {
			log.Error(verifyError, "verify failed for intermediate cert")
			return true
		}
	}

	// finally, if no check failed, return false
	return false
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
