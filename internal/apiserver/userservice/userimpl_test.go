package userservice

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

	pgpassword "github.com/crunchydata/postgres-operator/internal/postgres/password"
)

func TestGeneratePassword(t *testing.T) {
	username := ""
	password := ""
	passwordType := pgpassword.MD5
	generateNewPassword := false
	generatedPasswordLength := 32

	t.Run("no changes", func(t *testing.T) {

		changed, _, _, err := generatePassword(username, password, passwordType, generateNewPassword, generatedPasswordLength)

		if err != nil {
			t.Error(err)
			return
		}

		if changed {
			t.Errorf("password should not be generated if password is empty and generate password is false")
		}
	})

	t.Run("generate password", func(t *testing.T) {
		generateNewPassword := true

		t.Run("valid", func(t *testing.T) {
			changed, newPassword, _, err := generatePassword(username, password, passwordType, generateNewPassword, generatedPasswordLength)

			if err != nil {
				t.Error(err)
				return
			}

			if !changed {
				t.Errorf("new password should be returned")
			}

			if len(newPassword) != generatedPasswordLength {
				t.Errorf("generated password length expected %d actual %d", generatedPasswordLength, len(newPassword))
			}
		})

		t.Run("does not override custom password", func(t *testing.T) {
			password := "custom"
			changed, newPassword, _, err := generatePassword(username, password, passwordType, generateNewPassword, generatedPasswordLength)

			if err != nil {
				t.Error(err)
				return
			}

			if !changed {
				t.Errorf("new password should be returned")
			}

			if password != newPassword {
				t.Errorf("password should be %q but instead is %q", password, newPassword)
			}
		})

		t.Run("password length can be adjusted", func(t *testing.T) {
			generatedPasswordLength := 16
			changed, newPassword, _, err := generatePassword(username, password, passwordType, generateNewPassword, generatedPasswordLength)

			if err != nil {
				t.Error(err)
				return
			}

			if !changed {
				t.Errorf("new password should be returned")
			}

			if len(newPassword) != generatedPasswordLength {
				t.Errorf("generated password length expected %d actual %d", generatedPasswordLength, len(newPassword))
			}
		})

		t.Run("should be nonzero length", func(t *testing.T) {
			generatedPasswordLength := 0
			changed, newPassword, _, err := generatePassword(username, password, passwordType, generateNewPassword, generatedPasswordLength)

			if err != nil {
				t.Error(err)
				return
			}

			if !changed {
				t.Errorf("new password should be returned")
			}

			if len(newPassword) == 0 {
				t.Error("password length should be greater than 0")
			}
		})
	})

	t.Run("hashing", func(t *testing.T) {
		username := "hippo"
		password := "datalake"

		t.Run("md5", func(t *testing.T) {
			changed, _, hashedPassword, err := generatePassword(username, password,
				passwordType, generateNewPassword, generatedPasswordLength)

			if err != nil {
				t.Error(err)
				return
			}

			if !changed {
				t.Errorf("new password should be returned")
			}

			if !strings.HasPrefix(hashedPassword, "md5") && len(hashedPassword) != 32 {
				t.Errorf("not a valid md5 hash: %q", hashedPassword)
			}
		})

		t.Run("scram-sha-256", func(t *testing.T) {
			passwordType := pgpassword.SCRAM
			changed, _, hashedPassword, err := generatePassword(username, password,
				passwordType, generateNewPassword, generatedPasswordLength)

			if err != nil {
				t.Error(err)
				return
			}

			if !changed {
				t.Errorf("new password should be returned")
			}

			if !strings.HasPrefix(hashedPassword, "SCRAM-SHA-256$") {
				t.Errorf("not a valid scram-sha-256 verifier: %q", hashedPassword)
			}
		})
	})

}
