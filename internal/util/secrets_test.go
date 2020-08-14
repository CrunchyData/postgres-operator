package util

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
	"strings"
	"testing"
	"unicode"
)

func TestGeneratePassword(t *testing.T) {
	previous := []string{}

	for i := 0; i < 10; i++ {
		password, err := GeneratePassword(i + 10)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if expected, actual := i+10, len(password); expected != actual {
			t.Fatalf("expected length %v, got %v", expected, actual)
		}
		if i := strings.IndexFunc(password, unicode.IsPrint); i > 0 {
			t.Fatalf("expected only printable characters, got %q in %q", password[i], password)
		}

		for i := range previous {
			if password == previous[i] {
				t.Fatalf("expected passwords to not repeat, got %q after %q", password, previous)
			}
		}
		previous = append(previous, password)
	}
}
