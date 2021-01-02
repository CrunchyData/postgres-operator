package apiservermsgs

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
	"testing"

	pgpassword "github.com/crunchydata/postgres-operator/internal/postgres/password"
)

func TestGetPasswordType(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		tests := map[string]pgpassword.PasswordType{
			"":              pgpassword.MD5,
			"md5":           pgpassword.MD5,
			"scram":         pgpassword.SCRAM,
			"scram-sha-256": pgpassword.SCRAM,
		}

		for passwordTypeStr, expected := range tests {
			t.Run(passwordTypeStr, func(t *testing.T) {
				passwordType, err := GetPasswordType(passwordTypeStr)

				if err != nil {
					t.Error(err)
					return
				}

				if passwordType != expected {
					t.Errorf("password type %q should yield %d", passwordTypeStr, expected)
				}
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		tests := map[string]error{
			"magic":         ErrPasswordTypeInvalid,
			"scram-sha-512": ErrPasswordTypeInvalid,
		}

		for passwordTypeStr, expected := range tests {
			t.Run(passwordTypeStr, func(t *testing.T) {
				if _, err := GetPasswordType(passwordTypeStr); err != expected {
					t.Errorf("password type %q should yield error %q", passwordTypeStr, expected.Error())
				}
			})
		}
	})
}
