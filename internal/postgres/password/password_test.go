package password

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
	"errors"
	"testing"
)

func TestNewPostgresPassword(t *testing.T) {
	username := "hippo"
	password := "datalake"

	t.Run("md5", func(t *testing.T) {
		passwordType := MD5

		postgresPassword, err := NewPostgresPassword(passwordType, username, password)
		if err != nil {
			t.Error(err)
		}

		if _, ok := postgresPassword.(*MD5Password); !ok {
			t.Errorf("postgres password is not md5")
		}

		if postgresPassword.(*MD5Password).username != username {
			t.Errorf("username expected %q actual %q", username, postgresPassword.(*MD5Password).username)
		}

		if postgresPassword.(*MD5Password).password != password {
			t.Errorf("username expected %q actual %q", password, postgresPassword.(*MD5Password).password)
		}
	})

	t.Run("scram", func(t *testing.T) {
		passwordType := SCRAM

		postgresPassword, err := NewPostgresPassword(passwordType, username, password)
		if err != nil {
			t.Error(err)
		}

		if _, ok := postgresPassword.(*SCRAMPassword); !ok {
			t.Errorf("postgres password is not scram")
		}

		if postgresPassword.(*SCRAMPassword).password != password {
			t.Errorf("username expected %q actual %q", password, postgresPassword.(*SCRAMPassword).password)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		passwordType := PasswordType(-1)

		if _, err := NewPostgresPassword(passwordType, username, password); !errors.Is(err, ErrPasswordType) {
			t.Errorf("expected error: %q", err.Error())
		}
	})
}
