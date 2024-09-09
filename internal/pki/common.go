// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pki

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"time"
)

// certificateSignatureAlgorithm is ECDSA with SHA-384, the recommended
// signature algorithm with the P-256 curve.
const certificateSignatureAlgorithm = x509.ECDSAWithSHA384

// currentTime returns the current local time. It is a variable so it can be
// replaced during testing.
var currentTime = time.Now

// generateKey returns a random ECDSA key using a P-256 curve. This curve is
// roughly equivalent to an RSA 3072-bit key but requires less bits to achieve
// the equivalent cryptographic strength. Additionally, ECDSA is FIPS 140-2
// compliant.
func generateKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

// generateSerialNumber returns a random 128-bit integer.
func generateSerialNumber() (*big.Int, error) {
	return rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
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
