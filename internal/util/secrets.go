/*
 Copyright 2017 - 2021 Crunchy Data Solutions, Inc.
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

package util

import (
	"crypto/rand"
	"io"
	"math/big"
)

// The following constants are used as a part of password generation.
const (
	// DefaultGeneratedPasswordLength is the default length of what a generated
	// password should be if it's not set elsewhere
	DefaultGeneratedPasswordLength = 24
)

// GeneratePassword generates a password of a given length out of the acceptable
// ASCII characters suitable for a password
func GeneratePassword(length int) (string, error) {
	return accumulate(length, randomASCII)
}

// accumulate gathers n bytes from f and returns them as a string. It returns
// an empty string when f returns an error.
func accumulate(n int, f func() (byte, error)) (string, error) {
	result := make([]byte, n)

	for i := range result {
		if b, err := f(); err == nil {
			result[i] = b
		} else {
			return "", err
		}
	}

	return string(result), nil
}

// randomCharacter builds a function that returns random bytes from class.
func randomCharacter(random io.Reader, class string) func() (byte, error) {
	if random == nil {
		panic("requires a random source")
	}
	if len(class) == 0 {
		panic("class cannot be empty")
	}

	size := big.NewInt(int64(len(class)))

	return func() (byte, error) {
		if i, err := rand.Int(random, size); err == nil {
			return class[int(i.Int64())], nil
		} else {
			return 0, err
		}
	}
}

// policyASCII is the list of acceptable characters from which to generate an
// ASCII password.
const policyASCII = `` +
	`()*+,-./` + `:;<=>?@` + `[]^_` + `{|}` +
	`ABCDEFGHIJKLMNOPQRSTUVWXYZ` +
	`abcdefghijklmnopqrstuvwxyz` +
	`0123456789`

var randomASCII = randomCharacter(rand.Reader, policyASCII)
