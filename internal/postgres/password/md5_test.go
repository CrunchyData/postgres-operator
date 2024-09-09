// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package password

import (
	"fmt"
	"testing"
)

func TestMD5Build(t *testing.T) {
	// check a few different password combinations
	credentialList := []([]string){
		[]string{`hippo`, `datalake`, `md50587128989adb8f28a0a132c39af1b64`},
		[]string{`zebra`, `datalake`, `md5759511672f269ef472421d84b60a68bc`},
		[]string{`híppo`, `øásis`, `md5b52b986c3cff88dde7b952a8abd5995b`},
		[]string{`hippo`, `md53a0689aa9e31a50b5621971fc89f0c64`, `md55d83ff8796de1daf7f7c71e5fed3b37b`},
	}

	// a credential is valid if it generates the specified md5 hash
	for _, credentials := range credentialList {
		t.Run(fmt.Sprintf("%s:%s", credentials[0], credentials[1]), func(t *testing.T) {
			md5 := MD5Password{
				username: credentials[0],
				password: credentials[1],
			}

			hash, err := md5.Build()
			if err != nil {
				t.Error(err)
			}

			if hash != credentials[2] {
				t.Errorf("expected: %q actual %q", credentials[2], hash)
			}
		})
	}
}

func TestNewMD5Password(t *testing.T) {
	username := "hippo"
	password := "datalake"

	md5 := NewMD5Password(username, password)

	if md5.username != username {
		t.Errorf("username not set properly. expected %q actual %q", username, md5.username)
		return
	}

	if md5.password != password {
		t.Errorf("plaintext password not set properly. expected %q actual %q", password, md5.password)
		return
	}
}
