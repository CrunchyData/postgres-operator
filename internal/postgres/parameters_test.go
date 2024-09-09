// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

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

func TestParameterSetAppendToList(t *testing.T) {
	ps := NewParameterSet()

	ps.AppendToList("empty")
	assert.Assert(t, ps.Has("empty"))
	assert.Equal(t, ps.Value("empty"), "")

	ps.AppendToList("empty")
	assert.Equal(t, ps.Value("empty"), "", "expected no change")

	ps.AppendToList("full", "a")
	assert.Equal(t, ps.Value("full"), "a")

	ps.AppendToList("full", "b")
	assert.Equal(t, ps.Value("full"), "a,b")

	ps.AppendToList("full")
	assert.Equal(t, ps.Value("full"), "a,b", "expected no change")

	ps.AppendToList("full", "a", "cd", `"e"`)
	assert.Equal(t, ps.Value("full"), `a,b,a,cd,"e"`)
}
