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
	"encoding/pem"
	"fmt"
)

const (
	// pemCertificateType is part of the PEM header that identifies it as a x509
	// certificate
	pemCertificateType = "CERTIFICATE"

	// pemPrivateKeyType is part of the PEM header that identifies the private
	// key. This is presently hard coded to ECDSA keys
	pemPrivateKeyType = "EC PRIVATE KEY"
)

type Certificate struct {
	// Certifiate is the byte encoded value for the certificate
	Certificate []byte
}

// MarshalText encodes a x509 certificate into PEM format
func (c *Certificate) MarshalText() ([]byte, error) {
	block := &pem.Block{
		Type:  pemCertificateType,
		Bytes: c.Certificate,
	}

	return pem.EncodeToMemory(block), nil
}

// UnmarshalText decodes a x509 certificate from PEM format
func (c *Certificate) UnmarshalText(data []byte) error {
	block, _ := pem.Decode(data)

	// if block is nil, that means it is invalid PEM
	if block == nil {
		return fmt.Errorf("%w: malformed data", ErrInvalidPEM)
	}

	// if the type of the PEM block is not a certificate, return an error
	if block.Type != pemCertificateType {
		return fmt.Errorf("%w: not type %s", ErrInvalidPEM, pemCertificateType)
	}

	// everything checks out, at least in terms of PEM. Place encoded bytes in
	// object
	c.Certificate = block.Bytes

	return nil
}

type PrivateKey struct {
	// Password, if set, will encrypt the PrivateKey when converted to PEM format
	Password []byte

	// PrivateKey is the private key
	PrivateKey *ecdsa.PrivateKey

	// encryptPEMBlock allows for the encryption of a PEM Block into the
	// appropriate format
	encryptPEMBlock func(password []byte, der []byte) (*pem.Block, error)

	// marshalECPrivateKey turns a ECDSA private key into DER format, which is an
	// intermediate form prior to turning it into a PEM block
	marshalECPrivateKey func(*ecdsa.PrivateKey) ([]byte, error)
}

// MarshalText encodes the private key in PEM format
func (c *PrivateKey) MarshalText() ([]byte, error) {
	if c.marshalECPrivateKey == nil {
		return []byte{}, fmt.Errorf("%w: marshalECPrivateKey", ErrFunctionNotImplemented)
	}

	// first, convert private key to DER format
	der, err := c.marshalECPrivateKey(c.PrivateKey)

	if err != nil {
		return []byte{}, err
	}

	// encode the PEM block based on whether or not the password is present
	// if the password is nonempty, then encrypt with the password
	if len(c.Password) > 0 {
		return c.marshalEncryptedPrivateKey(der)
	}

	// otherwise, this is a boring encoding
	return c.marshalPrivateKey(der), nil
}

// UnmarshalText decodes a private key from PEM format
func (c *PrivateKey) UnmarshalText(data []byte) error {
	block, _ := pem.Decode(data)

	// if block is nil, that means it is invalid PEM
	if block == nil {
		return fmt.Errorf("%w: malformed data", ErrInvalidPEM)
	}

	// if the type of the PEM block is not private key, return an error
	if block.Type != pemPrivateKeyType {
		return fmt.Errorf("%w: not type %s", ErrInvalidPEM, pemPrivateKeyType)
	}

	// determine if this the PEM block is encrypted and if so, attempt to decrypt
	// it. if not, just read the bytes directly from the PEM block.
	var der []byte

	if len(c.Password) > 0 {
		var err error
		der, err = x509.DecryptPEMBlock(block, c.Password)
		if err != nil {
			return fmt.Errorf("%w: could not decrypt private key", ErrInvalidPEM)
		}
	} else {
		der = block.Bytes
	}

	// determine if the data actually represents a ECDSA private key
	privateKey, err := x509.ParseECPrivateKey(der)

	if err != nil {
		return fmt.Errorf("%w: not a valid ECDSA private key", ErrInvalidPEM)
	}

	// everything checks out, we have a ECDSA private key
	c.PrivateKey = privateKey

	return nil
}

// marshalPrivateKey encodes a private key in PEM format
func (c *PrivateKey) marshalPrivateKey(der []byte) []byte {
	block := &pem.Block{
		Type:  pemPrivateKeyType,
		Bytes: der,
	}

	return pem.EncodeToMemory(block)
}

// marshalEncryptedPrivateKey encodes a private key with encryption in PEM format
func (c *PrivateKey) marshalEncryptedPrivateKey(der []byte) ([]byte, error) {
	if c.encryptPEMBlock == nil {
		return []byte{}, fmt.Errorf("%w: encryptPEMBlock", ErrFunctionNotImplemented)
	}

	if block, err := c.encryptPEMBlock(c.Password, der); err != nil {
		return []byte{}, err
	} else {
		return pem.EncodeToMemory(block), nil
	}
}

// NewPrivateKey performs the setup for creating a new private key, including
// any functions that need to be created
func NewPrivateKey(key *ecdsa.PrivateKey) *PrivateKey {
	return &PrivateKey{
		PrivateKey:          key,
		encryptPEMBlock:     encryptPEMBlock,
		marshalECPrivateKey: marshalECPrivateKey,
	}
}

// ParseCertificate accepts binary encoded data to parse a certificate
func ParseCertificate(data []byte) (*Certificate, error) {
	certificate := &Certificate{}

	if err := certificate.UnmarshalText(data); err != nil {
		return nil, err
	}

	return certificate, nil
}

// ParsePrivateKey accepts binary encoded data with an optional password for
// decryption and attempts to parse a private key
func ParsePrivateKey(data, password []byte) (*PrivateKey, error) {
	privateKey := NewPrivateKey(nil)
	privateKey.Password = password

	if err := privateKey.UnmarshalText(data); err != nil {
		return nil, err
	}

	return privateKey, nil
}

// encryptPEMBlock is a wrapper around the "x509.encryptPEMBlock" function. This
// sets the encryption cipher to use AES-256 as this is the best-in-class at the
// time of this writing.
//
// Returns an encrypted PEM block and/or an error
func encryptPEMBlock(password, der []byte) (*pem.Block, error) {
	return x509.EncryptPEMBlock(rand.Reader, pemPrivateKeyType,
		der, password, x509.PEMCipherAES256)
}

// marshalECPrivateKey is a wrapper function around the
// "x509.MarshalECPrivateKey" function that converts a private key
func marshalECPrivateKey(privateKey *ecdsa.PrivateKey) ([]byte, error) {
	return x509.MarshalECPrivateKey(privateKey)
}
