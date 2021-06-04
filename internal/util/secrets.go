package util

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

import (
	"crypto/rand"
	"math/big"
	"strconv"
	"strings"
)

// The following constants are used as a part of password generation. For more
// information on these selections, please consulting the ASCII man page
// (`man ascii`)
const (
	// DefaultGeneratedPasswordLength is the length of what a generated password
	// is if it's not set in the pgo.yaml file, and to create some semblance of
	// consistency
	DefaultGeneratedPasswordLength = 24

	// passwordCharLower is the lowest ASCII character to use for generating a
	// password, which is 40
	passwordCharLower = 40
	// passwordCharUpper is the highest ASCII character to use for generating a
	// password, which is 126
	passwordCharUpper = 126
	// passwordCharExclude is a map of characters that we choose to exclude from
	// the password to simplify usage in the shell. There is still enough entropy
	// that exclusion of these characters is OK.
	passwordCharExclude = "`\\"
)

// passwordCharSelector is a "big int" that we need to select the random ASCII
// character for the password. Since the random integer generator looks for
// values from [0,X), we need to force this to be [40,126]
var passwordCharSelector = big.NewInt(passwordCharUpper - passwordCharLower)

// GeneratePassword generates a password of a given length out of the acceptable
// ASCII characters suitable for a password
func GeneratePassword(length int) (string, error) {
	password := make([]byte, length)
	i := 0

	for i < length {
		val, err := rand.Int(rand.Reader, passwordCharSelector)
		// if there is an error generating the random integer, return
		if err != nil {
			return "", err
		}

		char := byte(passwordCharLower + val.Int64())

		// if the character is in the exclusion list, continue
		if idx := strings.IndexAny(string(char), passwordCharExclude); idx > -1 {
			continue
		}

		password[i] = char
		i++
	}

	return string(password), nil
}

// GeneratedPasswordLength returns the value for what the length of a
// randomly generated password should be. It first determines if the user
// provided this value via a configuration file, and if not and/or the value is
// invalid, uses the default value
func GeneratedPasswordLength(configuredPasswordLength string) int {
	// set the generated password length for random password generation
	// note that "configuredPasswordLength" may be an empty string, and as such
	// the below line could fail. That's ok though! as we have a default set up
	generatedPasswordLength, err := strconv.Atoi(configuredPasswordLength)
	// if there is an error...set it to a default
	if err != nil {
		generatedPasswordLength = DefaultGeneratedPasswordLength
	}

	return generatedPasswordLength
}
