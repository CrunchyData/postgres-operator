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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"math/big"
	"time"
)

const (
	// beforeInterval sets a starting time for the issuance of a certificate,
	// which is defaulted to an hour earlier
	beforeInterval = -1 * time.Hour

	// certificateSignatureAlgorithm sets the default signature algorithm to use
	// for our certificates, which is the ECDSA with SHA-384. This is the
	// recommended signature algorithm with the P-256 curve.
	certificateSignatureAlgorithm = x509.ECDSAWithSHA384

	// serialNumberBits is the number of bits to allow in the generation of a
	// random serial number
	serialNumberBits = 128
)

// generateKey generates a ECDSA keypair using a P-256 curve. This curve is
// roughly equivalent to a RSA 3072 bit key, but requires less bits to achieve
// the equivalent cryptographic strength. Additionally, ECDSA is FIPS 140-2
// compliant.
func generateKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

// generateSerialNumber generates a random serial number that can be used for
// uniquely identifying a certificate
func generateSerialNumber() (*big.Int, error) {
	return rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), serialNumberBits))
}
