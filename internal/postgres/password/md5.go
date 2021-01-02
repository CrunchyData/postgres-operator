package password

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
	// #nosec G501
	"crypto/md5"
	"errors"
	"fmt"
)

// ErrMD5PasswordInvalid is returned when the password attributes are invalid
var ErrMD5PasswordInvalid = errors.New(`invalid password attributes. must provide "username" and "password"`)

// MD5Password implements the PostgresPassword interface for hashing passwords
// using the PostgreSQL MD5 method
type MD5Password struct {
	// password is the plaintext password
	password string
	// username is the PostgreSQL username
	username string
}

// Build creates the MD5 password format for PostgreSQL which resembles
// "md5" + md5("password" + "username")
func (m *MD5Password) Build() (string, error) {
	// create the plaintext password/salt that PostgreSQL expects as a byte string
	plaintext := []byte(m.password + m.username)
	// finish the transformation by getting the string value of the MD5 hash and
	// encoding it in hexadecimal for PostgreSQL, appending "md5" to the front
	// #nosec G401
	return fmt.Sprintf("md5%x", md5.Sum(plaintext)), nil
}

// NewMD5Password constructs a new MD5Password struct
func NewMD5Password(username, password string) *MD5Password {
	return &MD5Password{
		password: password,
		username: username,
	}
}
