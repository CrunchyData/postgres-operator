// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package password

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
