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

package pki

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"time"
)

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

// RootCAIsBad checks that at least one root CA has been generated and that
// all returned certs are CAs and not expired
//
// TODO(tjmoore4): Currently this will return 'true' if any of the parsed certs
// fail a given check. For scenarios where multiple certs may be returned, such
// as in a BYOC/BYOCA, this will need to be handled so we only generate a new
// certificate for our cert if it is the one that fails.
func RootCAIsBad(root *RootCertificateAuthority) bool {
	return !RootIsValid(root)
}

func generateRootCertificate(
	privateKey *ecdsa.PrivateKey, serialNumber *big.Int,
) (*x509.Certificate, error) {
	const rootCommonName = "postgres-operator-ca"
	const rootExpiration = time.Hour * 24 * 365 * 10
	const rootStartValid = time.Hour * -1

	now := currentTime()
	template := &x509.Certificate{
		BasicConstraintsValid: true,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		MaxPathLenZero:        true, // there are no intermediate certificates
		NotBefore:             now.Add(rootStartValid),
		NotAfter:              now.Add(rootExpiration),
		SerialNumber:          serialNumber,
		SignatureAlgorithm:    certificateSignatureAlgorithm,
		Subject: pkix.Name{
			CommonName: rootCommonName,
		},
	}

	// A root certificate is self-signed, so pass in the template twice.
	bytes, err := x509.CreateCertificate(rand.Reader, template, template,
		privateKey.Public(), privateKey)

	parsed, _ := x509.ParseCertificate(bytes)
	return parsed, err
}
