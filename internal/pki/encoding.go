/*
 Copyright 2021 - 2023 Crunchy Data Solutions, Inc.
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
	"crypto/x509"
	"encoding"
	"encoding/pem"
	"fmt"
)

const (
	// pemLabelCertificate is the textual encoding label for an X.509 certificate
	// according to RFC 7468. See https://tools.ietf.org/html/rfc7468.
	pemLabelCertificate = "CERTIFICATE"

	// pemLabelECDSAKey is the textual encoding label for an elliptic curve private key
	// according to RFC 5915. See https://tools.ietf.org/html/rfc5915.
	pemLabelECDSAKey = "EC PRIVATE KEY"
)

var (
	_ encoding.TextMarshaler   = Certificate{}
	_ encoding.TextMarshaler   = (*Certificate)(nil)
	_ encoding.TextUnmarshaler = (*Certificate)(nil)
)

// MarshalText returns a PEM encoding of c that OpenSSL understands.
func (c Certificate) MarshalText() ([]byte, error) {
	if c.x509 == nil || len(c.x509.Raw) == 0 {
		_, err := x509.ParseCertificate(nil)
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  pemLabelCertificate,
		Bytes: c.x509.Raw,
	}), nil
}

// UnmarshalText populates c from its PEM encoding.
func (c *Certificate) UnmarshalText(data []byte) error {
	block, _ := pem.Decode(data)

	if block == nil || block.Type != pemLabelCertificate {
		return fmt.Errorf("not a PEM-encoded certificate")
	}

	parsed, err := x509.ParseCertificate(block.Bytes)
	if err == nil {
		c.x509 = parsed
	}
	return err
}

var (
	_ encoding.TextMarshaler   = PrivateKey{}
	_ encoding.TextMarshaler   = (*PrivateKey)(nil)
	_ encoding.TextUnmarshaler = (*PrivateKey)(nil)
)

// MarshalText returns a PEM encoding of k that OpenSSL understands.
func (k PrivateKey) MarshalText() ([]byte, error) {
	if k.ecdsa == nil {
		k.ecdsa = new(ecdsa.PrivateKey)
	}

	der, err := x509.MarshalECPrivateKey(k.ecdsa)
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  pemLabelECDSAKey,
		Bytes: der,
	}), nil
}

// UnmarshalText populates k from its PEM encoding.
func (k *PrivateKey) UnmarshalText(data []byte) error {
	block, _ := pem.Decode(data)

	if block == nil || block.Type != pemLabelECDSAKey {
		return fmt.Errorf("not a PEM-encoded private key")
	}

	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err == nil {
		k.ecdsa = key
	}
	return err
}
