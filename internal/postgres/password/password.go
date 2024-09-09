// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package password

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

// ErrPasswordType is returned when a password type does not exist
var ErrPasswordType = errors.New("password type does not exist")

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
