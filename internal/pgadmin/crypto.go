package pgadmin

/*
 Copyright 2018 - 2021 Crunchy Data Solutions, Inc.
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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

// padKey ensures the resultant key is 32 bytes long, using the Procrustes method
func padKey(key []byte) []byte {
	if strLen := len(key); strLen > 32 {
		newKey := make([]byte, 32)
		copy(newKey, key)
		return newKey
	} else if strLen > 8 && strLen%8 == 0 {
		return key
	}

	// 31 bytes of '}', as per PyCrypto impl
	buffer := []byte{
		125, 125, 125, 125, 125, 125, 125, 125, 125, 125,
		125, 125, 125, 125, 125, 125, 125, 125, 125, 125,
		125, 125, 125, 125, 125, 125, 125, 125, 125, 125, 125,
	}

	padded := append(key, buffer...)
	newKey := make([]byte, 32)
	copy(newKey, padded)

	return newKey
}

func encrypt(plaintext, key string) string {
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to initialize AES vector: %v\n", err)
		os.Exit(1)
	}
	return encryptImpl(key, []byte(plaintext), iv)
}

func encryptImpl(key string, pt, iv []byte) string {
	ciphertext := make([]byte, aes.BlockSize+len(pt))
	copy(ciphertext[:aes.BlockSize], iv)

	aesBlockEnc, err := aes.NewCipher(padKey([]byte(key)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to initialize AES encrypter: %v\n", err)
		os.Exit(1)
	}

	cfbEnc := newCFB8Encrypter(aesBlockEnc, iv)
	cfbEnc.XORKeyStream(ciphertext[aes.BlockSize:], pt)

	return base64.StdEncoding.EncodeToString(ciphertext)
}

func decrypt(ciphertext, key string) string {
	bCipher, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		panic(err)
	}
	decoded := make([]byte, len(bCipher)-aes.BlockSize)

	aesBlockDec, err := aes.NewCipher(padKey([]byte(key)))
	if err != nil {
		panic(err)
	}

	aesDecrypt := newCFB8Decrypter(aesBlockDec, bCipher[:aes.BlockSize])
	aesDecrypt.XORKeyStream(decoded, bCipher[aes.BlockSize:])

	return string(decoded)
}

// 8-bit CFB implementation needed to match PyCrypt CFB impl
// Implemented in an idiomatic way to Golang crypto libraries (e.g. CFBEncrypter/Decrypter)
type cfb8 struct {
	blk       cipher.Block
	blockSize int
	in        []byte
	out       []byte
	decrypt   bool
}

// Implemnets cipher.Stream interface
func (x *cfb8) XORKeyStream(dst, src []byte) {
	for i := range src {
		x.blk.Encrypt(x.out, x.in)
		copy(x.in[:x.blockSize-1], x.in[1:])
		if x.decrypt {
			x.in[x.blockSize-1] = src[i]
		}
		dst[i] = src[i] ^ x.out[0]
		if !x.decrypt {
			x.in[x.blockSize-1] = dst[i]
		}
	}
}

// NewCFB8Encrypter returns a Stream which encrypts with cipher feedback mode
// (segment size = 8), using the given Block. The iv must be the same length as
// the Block's block size.
func newCFB8Encrypter(block cipher.Block, iv []byte) cipher.Stream {
	return newCFB8(block, iv, false)
}

// NewCFB8Decrypter returns a Stream which decrypts with cipher feedback mode
// (segment size = 8), using the given Block. The iv must be the same length as
// the Block's block size.
func newCFB8Decrypter(block cipher.Block, iv []byte) cipher.Stream {
	return newCFB8(block, iv, true)
}

func newCFB8(block cipher.Block, iv []byte, decrypt bool) cipher.Stream {
	blockSize := block.BlockSize()
	if len(iv) != blockSize {
		// stack trace will indicate whether it was de or encryption
		panic("cipher.newCFB: IV length must equal block size")
	}
	x := &cfb8{
		blk:       block,
		blockSize: blockSize,
		out:       make([]byte, blockSize),
		in:        make([]byte, blockSize),
		decrypt:   decrypt,
	}
	copy(x.in, iv)

	return x
}
