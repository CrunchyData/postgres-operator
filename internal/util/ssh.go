package util

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
	"bytes"
	"crypto/ed25519"
	"encoding/pem"
	"math/rand"

	"golang.org/x/crypto/ssh"
)

// SSHKey stores byte slices that represent private and public ssh keys
type SSHKey struct {
	Private []byte
	Public  []byte
}

// NewPrivatePublicKeyPair generates a an ed25519 ssh private and public key
func NewPrivatePublicKeyPair() (SSHKey, error) {
	var keys SSHKey

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return SSHKey{}, err
	}

	keys.Public, err = newPublicKey(pub)
	if err != nil {
		return SSHKey{}, err
	}

	keys.Private, err = newPrivateKey(priv)
	if err != nil {
		return SSHKey{}, err
	}

	return keys, nil
}

// newPublicKey generates a byte slice containing an public key that can be used
// to ssh. This key is based off of the ed25519.PublicKey type The function is
// only used by NewPrivatePublicKeyPair
func newPublicKey(key ed25519.PublicKey) ([]byte, error) {
	pubKey, err := ssh.NewPublicKey(key)
	if err != nil {
		return nil, err
	}
	return ssh.MarshalAuthorizedKey(pubKey), nil
}

// newPrivateKey generates a byte slice containing an OpenSSH private ssh key.
// This key is based off of the ed25519.PrivateKey type. The function is only
// used by NewPrivatePublicKeyPair
func newPrivateKey(key ed25519.PrivateKey) ([]byte, error) {
	// The following link describes the private key format for OpenSSH. It
	// oulines the structs that are used to generate the OpenSSH private key
	// from the ed25519 private key
	// https://anongit.mindrot.org/openssh.git/tree/PROTOCOL.key?h=V_8_1_P1

	const authMagic = "openssh-key-v1"
	const noneCipherBlockSize = 8

	private := struct {
		Check1  uint32
		Check2  uint32
		KeyType string
		Public  []byte
		Private []byte
		Comment string
		Pad     []byte `ssh:"rest"`
	}{
		KeyType: ssh.KeyAlgoED25519,
		Public:  key.Public().(ed25519.PublicKey),
		Private: key,
	}

	// check fields should match to easily verify
	// that a decryption was successful
	// #nosec: G404
	private.Check1 = rand.Uint32()
	private.Check2 = private.Check1

	{
		bsize := noneCipherBlockSize
		plen := len(ssh.Marshal(private))
		private.Pad = make([]byte, bsize-(plen%bsize))
	}

	// The list of privatekey/comment pairs is padded with the
	// bytes 1, 2, 3, ... until the total length is a multiple
	// of the cipher block size.
	for i := range private.Pad {
		private.Pad[i] = byte(i) + 1
	}

	public := struct {
		Keytype string
		Public  []byte
	}{
		Keytype: ssh.KeyAlgoED25519,
		Public:  private.Public,
	}

	// The overall key consists of a header, a list of public keys, and
	// an encrypted list of matching private keys.
	overall := struct {
		CipherName   string
		KDFName      string
		KDFOpts      string
		NumKeys      uint32
		PubKey       []byte
		PrivKeyBlock []byte
	}{
		CipherName: "none", KDFName: "none", // unencrypted
		NumKeys:      1,
		PubKey:       ssh.Marshal(public),
		PrivKeyBlock: ssh.Marshal(private),
	}

	pemBlock := &pem.Block{
		Type:  "OPENSSH PRIVATE KEY",
		Bytes: append(append([]byte(authMagic), 0), ssh.Marshal(overall)...),
	}

	var privateKeyPEM bytes.Buffer
	if err := pem.Encode(&privateKeyPEM, pemBlock); err != nil {
		return nil, err
	}

	return privateKeyPEM.Bytes(), nil
}
