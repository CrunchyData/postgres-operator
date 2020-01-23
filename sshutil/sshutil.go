package sshutil

import (
	"math/rand"
	"crypto/ed25519"
	"encoding/pem"
	"bytes"

	"golang.org/x/crypto/ssh"
)

// SSHKey stores byte slices that represent private and public ssh keys
type SSHKey struct {
	Private []byte
	Public  []byte
}

// NewPrivatePublicKeyPair generates a an ed25519 ssh private and public key
func NewPrivatePublicKeyPair(bits int) (SSHKey, error) {
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

func newPublicKey(key ed25519.PublicKey) ([]byte, error) {
	pubKey, err := ssh.NewPublicKey(key)
	if err != nil {
		return nil, err
	}
	return ssh.MarshalAuthorizedKey(pubKey), nil
}

func newPrivateKey(key ed25519.PrivateKey) ([]byte, error) {
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
		Type: "OPENSSH PRIVATE KEY",
		Bytes: append(append([]byte(authMagic), 0), ssh.Marshal(overall)...),
	}

	var privateKeyPEM bytes.Buffer
	if err := pem.Encode(&privateKeyPEM, pemBlock); err != nil {
		return nil, err
	}

	return privateKeyPEM.Bytes(), nil
}
