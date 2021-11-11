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
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
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
