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
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"time"
)

// LeafCertificate is a certificate and private key pair that can be validated
// by RootCertificateAuthority.
type LeafCertificate struct {
	Certificate Certificate
	PrivateKey  PrivateKey
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
	return !rootCertCA.leafIsValid(leaf)
}

func generateLeafCertificate(
	signer *x509.Certificate, signerPrivate *ecdsa.PrivateKey,
	signeePublic *ecdsa.PublicKey, serialNumber *big.Int,
	commonName string, dnsNames []string,
) (*x509.Certificate, error) {
	const leafExpiration = time.Hour * 24 * 365
	const leafStartValid = time.Hour * -1

	now := currentTime()
	template := &x509.Certificate{
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		NotBefore:             now.Add(leafStartValid),
		NotAfter:              now.Add(leafExpiration),
		SerialNumber:          serialNumber,
		SignatureAlgorithm:    certificateSignatureAlgorithm,
		Subject: pkix.Name{
			CommonName: commonName,
		},
	}

	bytes, err := x509.CreateCertificate(rand.Reader, template, signer,
		signeePublic, signerPrivate)

	parsed, _ := x509.ParseCertificate(bytes)
	return parsed, err
}
