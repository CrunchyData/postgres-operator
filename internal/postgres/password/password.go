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
	"errors"
)

// PasswordType helps to specify the type of password method (e.g. md5)
// to use
type PasswordType int

// The following constants are used to aid in the select of the types of
// PostgreSQL password generators
const (
	// MD5 refers to the MD5Password method
	MD5 PasswordType = iota
	SCRAM
)

var (
	// ErrPasswordType is returned when a password type does not exist
	ErrPasswordType = errors.New("password type does not exist")
)

// PostgresPassword is the interface that defines the methods required to build
// a password for PostgreSQL in a desired format (e.g. MD5)
type PostgresPassword interface {
	// Build transforms any plaintext or other attributes that are provided by
	// the interface implementor and returns a password. If an error occurs in the
	// building process, it is returned.
	//
	// If the build does error, return an empty string
	Build() (string, error)
}

// NewPostgresPassword accepts a type of password (e.g. md5) which is used to
// select the correct generation interface, as well as a username and password
// and returns the appropriate interface
//
// An error is returned if the proper password generator cannot be found
func NewPostgresPassword(passwordType PasswordType, username, password string) (PostgresPassword, error) {
	switch passwordType {
	default:
		// if no password type is selected, return an error
		return nil, ErrPasswordType
	case MD5:
		return NewMD5Password(username, password), nil
	case SCRAM:
		return NewSCRAMPassword(password), nil
	}
}
