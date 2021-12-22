/*
 Copyright 2021 Crunchy Data Solutions, Inc.
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

func TestGeneratePassword(t *testing.T) {
	// different lengths
	for _, length := range []int{1, 2, 3, 5, 20, 200} {
		password, err := GeneratePassword(length)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if expected, actual := length, len(password); expected != actual {
			t.Fatalf("expected length %v, got %v", expected, actual)
		}
		if i := strings.IndexFunc(password, func(r rune) bool { return !unicode.IsPrint(r) }); i > -1 {
			t.Fatalf("expected only printable characters, got %q in %q", password[i], password)
		}
		if i := strings.IndexAny(password, "`\\"); i > -1 {
			t.Fatalf("expected no exclude characters, got %q in %q", password[i], password)
		}
	}

	// random contents
	previous := []string{}

	for i := 0; i < 10; i++ {
		password, err := GeneratePassword(5)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if i := strings.IndexFunc(password, func(r rune) bool { return !unicode.IsPrint(r) }); i > -1 {
			t.Fatalf("expected only printable characters, got %q in %q", password[i], password)
		}
		if i := strings.IndexAny(password, "`\\"); i > -1 {
			t.Fatalf("expected no exclude characters, got %q in %q", password[i], password)
		}

		for i := range previous {
			if password == previous[i] {
				t.Fatalf("expected passwords to not repeat, got %q after %q", password, previous)
			}
		}
		previous = append(previous, password)
	}
}

func TestPolicyASCII(t *testing.T) {
	// [GeneratePassword] used to pick random characters by doing
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
