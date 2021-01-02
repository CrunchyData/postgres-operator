package tlsutil

/*
Copyright 2019 - 2021 Crunchy Data Solutions, Inc.
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
	"encoding/pem"
	"errors"
	"io"
	"io/ioutil"
	"math"
	"math/big"
	"time"
)

const (
	duration365d = time.Hour * 24 * 365
)

// newPrivateKey returns randomly generated RSA private key.
func NewPrivateKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

// encodePrivateKeyPEM encodes the given private key pem and returns bytes (base64).
func EncodePrivateKeyPEM(key *ecdsa.PrivateKey) []byte {
	raw, _ := x509.MarshalECPrivateKey(key)
	return pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: raw,
	})
}

// encodeCertificatePEM encodes the given certificate pem and returns bytes (base64).
func EncodeCertificatePEM(cert *x509.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})
}

// parsePEMEncodedCert parses a certificate from the given pemdata
func ParsePEMEncodedCert(pemdata []byte) (*x509.Certificate, error) {
	decoded, _ := pem.Decode(pemdata)
	if decoded == nil {
		return nil, errors.New("no PEM data found")
	}
	return x509.ParseCertificate(decoded.Bytes)
}

// parsePEMEncodedPrivateKey parses a private key from given pemdata
func ParsePEMEncodedPrivateKey(pemdata []byte) (*ecdsa.PrivateKey, error) {
	decoded, _ := pem.Decode(pemdata)
	if decoded == nil {
		return nil, errors.New("no PEM data found")
	}
	return x509.ParseECPrivateKey(decoded.Bytes)
}

// newSelfSignedCACertificate returns a self-signed CA certificate based on given configuration and private key.
// The certificate has one-year lease.
func NewSelfSignedCACertificate(key *ecdsa.PrivateKey) (*x509.Certificate, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	now := time.Now()
	tmpl := x509.Certificate{
		SerialNumber:          serial,
		NotBefore:             now.UTC(),
		NotAfter:              now.Add(duration365d).UTC(),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		SignatureAlgorithm:    x509.ECDSAWithSHA384,
	}
	certDERBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, key.Public(), key)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

// ExtendTrust extends the provided certpool with the PEM-encoded certificates
// presented by certSource. If reading from certSource produces an error
// the base pool remains unmodified
func ExtendTrust(base *x509.CertPool, certSource io.Reader) error {
	certs, err := ioutil.ReadAll(certSource)
	if err != nil {
		return err
	}
	base.AppendCertsFromPEM(certs)

	return nil
}
