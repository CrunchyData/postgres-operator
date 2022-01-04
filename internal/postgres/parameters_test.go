/*
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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

package postgres

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestNewParameters(t *testing.T) {
	parameters := NewParameters()

	assert.DeepEqual(t, parameters.Mandatory.AsMap(), map[string]string{
		"ssl":           "on",
		"ssl_ca_file":   "/pgconf/tls/ca.crt",
		"ssl_cert_file": "/pgconf/tls/tls.crt",
		"ssl_key_file":  "/pgconf/tls/tls.key",

		"unix_socket_directories": "/tmp/postgres",

		"wal_level": "logical",
	})
	assert.DeepEqual(t, parameters.Default.AsMap(), map[string]string{
		"jit": "off",

		"password_encryption": "scram-sha-256",
	})
}

func TestParameterSet(t *testing.T) {
	ps := NewParameterSet()

	ps.Add("x", "y")
	assert.Assert(t, ps.Has("X"))
	assert.Equal(t, ps.Value("x"), "y")

	v, ok := ps.Get("X")
	assert.Assert(t, ok)
	assert.Equal(t, v, "y")

	ps.Add("X", "z")
	assert.Equal(t, ps.Value("x"), "z")

	ps.Add("abc", "j'l")
	assert.DeepEqual(t, ps.AsMap(), map[string]string{
		"abc": "j'l",
		"x":   "z",
	})

	ps2 := ps.DeepCopy()
	assert.Assert(t, ps2.Has("abc"))
	assert.Equal(t, ps2.Value("x"), ps.Value("x"))

	ps2.Add("x", "n")
	assert.Assert(t, ps2.Value("x") != ps.Value("x"))
}
