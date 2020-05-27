package password

/*
 Copyright 2020 Crunchy Data Solutions, Inc.
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
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"hash"
	"unicode"
	"unicode/utf8"

	"github.com/xdg/stringprep"
	"golang.org/x/crypto/pbkdf2"
)

// scramVerifierFormat is the format of the value that is stored by PostgreSQL
// and follows the format:
//
// <DIGEST>$<ITERATIONS>:<SALT>$<STORED_KEY>:<SERVER_KEY>
//
// where:
// DIGEST = SCRAM-SHA-256 (only value for now in PostgreSQL)
// ITERATIONS = the number of iteratiosn to use for PBKDF2
// SALT = the salt used as part of the PBKDF2, stored in base64
// STORED_KEY = the hash of the client key, stored in base64
// SERVER_KEY = the hash of the server key
const scramVerifierFormat = "SCRAM-SHA-256$%d:%s$%s:%s"

// These constants are defined as part of the PostgreSQL implementation for
// SCRAM, but can be overridden by the user
// https://git.postgresql.org/gitweb/?p=postgresql.git;a=blob;f=src/include/common/scram-common.h
const (
	// scramDefaultIterations is the number of iterations to make as part of the PBKDF2
	// algorithm
	scramDefaultIterations = 4096
	// scramDefaultSaltLength is the length of the generated salt used in creating the
	// hashed password
	scramDefaultSaltLength = 16
)

// scramDefaultHash is the hashing algorithm to use
var scramDefaultHash = sha256.New

// the following are used as part of the SCRAM verifier generation
var (
	scramClientKeyMessage = []byte("Client Key")
	scramServerKeyMessage = []byte("Server Key")
)

var (
	// ErrSCRAMPasswordInvalid is returned when the password attributes are invalid
	ErrSCRAMPasswordInvalid = errors.New(`invalid password attributes. must provide "password"`)
	// ErrSCRAMSaltLengthInvalid is returned when the salt length is less than 1
	ErrSCRAMSaltLengthInvalid = errors.New(`salt length must be at least 1`)
)

// SCRAMPassword contains the building blocks to build a PostgreSQL SCRAM
// verifier. Implements the PostgresPassword interface
type SCRAMPassword struct {
	// Iterations is the number of iterations to run the PBKDF2 algorithm when
	// generating the hashed salted password. This defaults to 4096, which is the
	// PostgreSQL default
	Iterations int
	// SaltLength is the length of the generated salt that is used as part of the
	// PBKDF2 algorithm
	SaltLength int
	// generateSalt is a function that is used to generate a salt. This can be
	// mocked for testing purposes
	generateSalt func(int) ([]byte, error)
	// password is the plaintext password. This is really the most important
	// attribute
	password string
}

// Build creates the SCRAM verifier, which follows the methods defined in the
// PostgreSQL source, i.e.
//
// https://git.postgresql.org/gitweb/?p=postgresql.git;a=blob;f=src/include/common/scram-common.h
func (s *SCRAMPassword) Build() (string, error) {
	// get a generated salt
	salt, err := s.generateSalt(s.SaltLength)

	if err != nil {
		return "", err
	}

	// before generating the salted password, we have to normalize the password
	// using SASLprep
	password := s.saslPrep()

	saltedPassword := pbkdf2.Key([]byte(password), salt, s.Iterations, scramDefaultHash().Size(), scramDefaultHash)

	// time to create the HMAC generated values (client key, server key)
	clientKey := s.hmac(scramDefaultHash, saltedPassword, scramClientKeyMessage)
	serverKey := s.hmac(scramDefaultHash, saltedPassword, scramServerKeyMessage)

	// get the stored key, which is the hash of the client key
	storedKey := s.hash(scramDefaultHash, clientKey)

	// finally, we can build the scram verified!
	verifier := fmt.Sprintf(scramVerifierFormat,
		s.Iterations, s.encode(salt), s.encode(storedKey), s.encode(serverKey))

	return verifier, nil
}

// encode creates a base64 encoding of a value that's returned as a string
func (s *SCRAMPassword) encode(value []byte) string {
	return base64.StdEncoding.EncodeToString(value)
}

// hash creates a SHA hash. Uses SHA-256, but can be swapped
// in the future to use another hashing algorithm
func (s *SCRAMPassword) hash(h func() hash.Hash, message []byte) []byte {
	hf := h()
	// hash.Hash.Write() never returns an error
	_, _ = hf.Write(message)
	return hf.Sum(nil)
}

// hmac performs a HMAC on a particular value. Uses SHA-256, but can be swapped
// in the future to use another hashing algorithm
func (s *SCRAMPassword) hmac(h func() hash.Hash, key, message []byte) []byte {
	hm := hmac.New(h, key)
	// hash.Hash.Write() never returns an error
	_, _ = hm.Write(message)
	return hm.Sum(nil)
}

// isASCII returns true if the string that is passed in is composed entirely of
// ASCII characters
func (s *SCRAMPassword) isASCII() bool {
	// iterate through each character of the plaintext password and determine if
	// it is ASCII. if it is not ASCII, exit early
	// per research, this loop is optimized to be fast for searching
	for i := 0; i < len(s.password); i++ {
		if s.password[i] > unicode.MaxASCII {
			return false
		}
	}

	return true
}

// saslPrep returns the canonical form of a password for PostgreSQL when
// using SCRAM. It differs from RFC 4013 in that it returns the original,
// unmodified password when:
//
//  - the input is not valid UTF-8
//  - the output would be empty
//  - the output would contain prohibited characters
//  - the output would contain ambiguous bidirectional characters
//
// See:
//
// https://git.postgresql.org/gitweb/?p=postgresql.git;a=blob;f=src/common/saslprep.c
func (s *SCRAMPassword) saslPrep() string {
	// if the password is only ASCII or it is not a valid UTF8 password, return
	// the original password here
	if s.isASCII() || !utf8.ValidString(s.password) {
		return s.password
	}

	// perform SASLprep on the password. if the SASLprep fails or returns an
	// empty string, return the original password
	// Otherwise return the clean pasword
	if cleanedPassword, err := stringprep.SASLprep.Prepare(s.password); cleanedPassword == "" || err != nil {
		return s.password
	} else {
		return cleanedPassword
	}
}

// NewSCRAMPassword constructs a new SCRAMPassword struct with sane defaults
func NewSCRAMPassword(password string) *SCRAMPassword {
	return &SCRAMPassword{
		Iterations:   scramDefaultIterations,
		generateSalt: scramGenerateSalt,
		password:     password,
		SaltLength:   scramDefaultSaltLength,
	}
}

// scramGenerateSalt generates aseries of cryptographic bytes of a specified
// length for purposes of SCRAM. must be at least 1
func scramGenerateSalt(length int) ([]byte, error) {
	// length must be at least one
	if length < 1 {
		return []byte{}, ErrSCRAMSaltLengthInvalid
	}

	// create a salt of size length. The slice needs to be allocated as the
	// crypto random number generator copies the byte values into it
	salt := make([]byte, length)

	// generate the salt
	_, err := rand.Read(salt)

	// return the value that is now in the salt and/or the error
	return salt, err
}
