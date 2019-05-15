package sshutil

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"golang.org/x/crypto/ssh"
)

type SSHKey struct {
	Private []byte
	Public  []byte
}

func NewPrivatePublicKeyPair(bits int) (SSHKey, error) {
	var keys SSHKey

	privateKey, err := NewPrivateKey(bits)
	if err != nil {
		return keys, err
	}

	keys.Public, err = NewPublicKey(privateKey)
	if err != nil {
		return keys, err
	}

	keys.Private, err = RSAToPEM(privateKey)
	if err != nil {
		return keys, err
	}

	return keys, nil
}

func NewPrivateKey(bits int) (*rsa.PrivateKey, error) {
	pk, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return pk, err
	}

	err = pk.Validate()
	if err != nil {
		return pk, err
	}

	return pk, err
}

func RSAToPEM(privateKey *rsa.PrivateKey) ([]byte, error) {
	pemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	var privateKeyPEM bytes.Buffer
	if err := pem.Encode(&privateKeyPEM, pemBlock); err != nil {
		return nil, err
	}

	return privateKeyPEM.Bytes(), nil
}

func NewPublicKey(privateKey *rsa.PrivateKey) ([]byte, error) {
	publicRSA, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}

	return ssh.MarshalAuthorizedKey(publicRSA), nil
}
