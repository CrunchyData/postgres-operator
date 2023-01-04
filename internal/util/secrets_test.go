/*
 Copyright 2021 - 2023 Crunchy Data Solutions, Inc.
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
	"errors"
	"strings"
	"testing"
	"testing/iotest"
	"unicode"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestAccumulate(t *testing.T) {
	called := 0
	result, err := accumulate(10, func() (byte, error) {
		called++
		return byte('A' + called), nil
	})

	assert.NilError(t, err)
	assert.Equal(t, called, 10)
	assert.Equal(t, result, "BCDEFGHIJK")

	t.Run("Error", func(t *testing.T) {
		called := 0
		expected := errors.New("zap")
		result, err := accumulate(10, func() (byte, error) {
			called++
			if called < 5 {
				return byte('A' + called), nil
			} else {
				return 'Z', expected
			}
		})

		assert.Equal(t, err, expected)
		assert.Equal(t, called, 5, "expected an early return")
		assert.Equal(t, result, "")
	})
}

func TestGenerateAlphaNumericPassword(t *testing.T) {
	for _, length := range []int{0, 1, 2, 3, 5, 20, 200} {
		password, err := GenerateAlphaNumericPassword(length)

		assert.NilError(t, err)
		assert.Equal(t, length, len(password))
		assert.Assert(t, cmp.Regexp(`^[A-Za-z0-9]*$`, password))
	}

	previous := sets.String{}
	for i := 0; i < 10; i++ {
		password, err := GenerateAlphaNumericPassword(5)

		assert.NilError(t, err)
		assert.Assert(t, cmp.Regexp(`^[A-Za-z0-9]{5}$`, password))

		assert.Assert(t, !previous.Has(password), "%q generated twice", password)
		previous.Insert(password)
	}
}

func TestGenerateASCIIPassword(t *testing.T) {
	for _, length := range []int{0, 1, 2, 3, 5, 20, 200} {
		password, err := GenerateASCIIPassword(length)

		assert.NilError(t, err)
		assert.Equal(t, length, len(password))

		// Check every rune in the string. See [TestPolicyASCII].
		for _, c := range password {
			assert.Assert(t, strings.ContainsRune(policyASCII, c), "%q is not acceptable", c)
		}
	}

	previous := sets.String{}
	for i := 0; i < 10; i++ {
		password, err := GenerateASCIIPassword(5)

		assert.NilError(t, err)
		assert.Equal(t, 5, len(password))

		// Check every rune in the string. See [TestPolicyASCII].
		for _, c := range password {
			assert.Assert(t, strings.ContainsRune(policyASCII, c), "%q is not acceptable", c)
		}

		assert.Assert(t, !previous.Has(password), "%q generated twice", password)
		previous.Insert(password)
	}
}

func TestPolicyASCII(t *testing.T) {
	// [GenerateASCIIPassword] used to pick random characters by doing
	// arithmetic on ASCII codepoints. It now uses a constant set of characters
	// that satisfy the following properties. For more information on these
	// selections, consult the ASCII man page, `man ascii`.

	// lower and upper are the lowest and highest ASCII characters to use.
	const lower = 40
	const upper = 126

	// exclude is a map of characters that we choose to exclude from
	// the password to simplify usage in the shell.
	const exclude = "`\\"

	count := map[rune]int{}

	// Check every rune in the string.
	for _, c := range policyASCII {
		assert.Assert(t, unicode.IsPrint(c), "%q is not printable", c)
		assert.Assert(t, c <= unicode.MaxASCII, "%q is not ASCII", c)
		assert.Assert(t, lower <= c && c < upper, "%q is outside the range", c)
		assert.Assert(t, !strings.ContainsRune(exclude, c), "%q should be excluded", c)

		count[c]++
		assert.Assert(t, count[c] == 1, "%q occurs more than once", c)
	}

	// Every acceptable byte is in the policy.
	assert.Equal(t, len(policyASCII), upper-lower-len(exclude))
}

func TestRandomCharacter(t *testing.T) {
	// The random source cannot be nil and the character class cannot be empty.
	assert.Assert(t, cmp.Panics(func() { randomCharacter(nil, "") }))
	assert.Assert(t, cmp.Panics(func() { randomCharacter(nil, "asdf") }))
	assert.Assert(t, cmp.Panics(func() { randomCharacter(iotest.ErrReader(nil), "") }))

	// The function returns any error from the random source.
	expected := errors.New("doot")
	_, err := randomCharacter(iotest.ErrReader(expected), "asdf")()
	assert.Equal(t, err, expected)
}
