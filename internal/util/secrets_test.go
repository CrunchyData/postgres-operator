package util

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
	"strings"
	"testing"
	"unicode"
)

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
		if i := strings.IndexAny(password, passwordCharExclude); i > -1 {
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
		if i := strings.IndexAny(password, passwordCharExclude); i > -1 {
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
