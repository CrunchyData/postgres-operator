package pgadmin

/*
 Copyright 2020 - 2021 Crunchy Data Solutions, Inc.
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
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

// HashPassword emulates the PBKDF2 password hashing mechanism using a salt
// randomly generated and stored in the pgadmin database
//
// It returns a string of the Modular Crypt Format result of the hash,
// suitable for insertion/replacement of pgadmin login password fields
func HashPassword(qr *queryRunner, pass string) (string, error) {
	// Hashing parameters
	const saltLenBytes = 16
	const iterations = 25000
	const hashLenBytes = 64

	if qr.secSalt == "" {
		// Retrieve the database-specific random salt
		securitySalt, err := qr.Query("SELECT value FROM keys WHERE name='SECURITY_PASSWORD_SALT';")
		if err != nil {
			return "", err
		}
		qr.secSalt = securitySalt
	}

	// This looks strange, but the algorithm really does use the byte
	// representation of the string (i.e. string isn't base64 or other
	// encoding format)
	saltBytes := []byte(qr.secSalt)

	// Generate a "new" password derived from the provided password
	// Satisfies OWASP sec. 2.4.5: 'provide additional iteration of a key derivation'
	mac := hmac.New(sha512.New, saltBytes)
	_, _ = mac.Write([]byte(pass))
	macBytes := mac.Sum(nil)
	macBase64 := base64.StdEncoding.EncodeToString(macBytes)

	// Generate random salt for the pbkdf2 run, this is the salt that ends
	// up in the salt field of the returned hash
	hashSalt := make([]byte, saltLenBytes)
	if _, err := rand.Read(hashSalt); err != nil {
		return "", err
	}

	hashed := pbkdf2.Key([]byte(macBase64), hashSalt, iterations, hashLenBytes, sha512.New)

	// Base64 encode and convert to storage format expected by Flask-Security
	saltEncoded := strings.ReplaceAll(base64.RawStdEncoding.EncodeToString(hashSalt), "+", ".")
	keyEncoded := strings.ReplaceAll(base64.RawStdEncoding.EncodeToString(hashed), "+", ".")

	return fmt.Sprintf("$pbkdf2-sha512$%d$%s$%s", iterations, saltEncoded, keyEncoded), nil
}
