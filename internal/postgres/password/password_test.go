// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package password

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
