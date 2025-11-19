// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
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
		"log_file_mode": "0660",

		"ssl":           "on",
		"ssl_ca_file":   "/pgconf/tls/ca.crt",
		"ssl_cert_file": "/pgconf/tls/tls.crt",
		"ssl_key_file":  "/pgconf/tls/tls.key",

		"unix_socket_directories": "/tmp/postgres",

		"wal_level": "logical",
	})
	assert.DeepEqual(t, parameters.Default.AsMap(), map[string]string{
		"jit": "off",

		"log_directory":       "/pgdata/logs/postgres",
		"password_encryption": "scram-sha-256",
	})
}

func TestParameterSet(t *testing.T) {
	t.Run("NilAsMap", func(t *testing.T) {
		m := (*ParameterSet)(nil).AsMap()
		assert.Assert(t, m == nil)
	})

	t.Run("NilDeepCopy", func(t *testing.T) {
		ps := (*ParameterSet)(nil).DeepCopy()
		assert.Assert(t, ps == nil)
	})

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

	assert.DeepEqual(t, ps.String(), ``+
		`abc = 'j''l'`+"\n"+
		`x = 'z'`+"\n")
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

func TestParameterSetEqual(t *testing.T) {
	var Nil *ParameterSet
	ps1 := NewParameterSet()
	ps2 := NewParameterSet()

	// nil equals nil, and empty does not equal nil
	assert.Assert(t, Nil.Equal(nil))
	assert.Assert(t, !Nil.Equal(ps1))
	assert.Assert(t, !ps1.Equal(nil))

	// empty equals empty
	assert.Assert(t, ps1.Equal(ps2))
	assert.Assert(t, ps2.Equal(ps1))

	// different keys are not equal
	ps1.Add("a", "b")
	assert.Assert(t, !ps1.Equal(nil))
	assert.Assert(t, !Nil.Equal(ps1))
	assert.Assert(t, !ps1.Equal(ps2))
	assert.Assert(t, !ps2.Equal(ps1))

	// different values are not equal
	ps2.Add("a", "c")
	assert.Assert(t, !ps1.Equal(ps2))
	assert.Assert(t, !ps2.Equal(ps1))

	// normalized keys+values are equal
	ps1.Add("A", "c")
	assert.Assert(t, ps1.Equal(ps2))
	assert.Assert(t, ps2.Equal(ps1))

	// [assert.DeepEqual] can only compare exported fields.
	// When present, the `(T) Equal(T) bool` method is used instead.
	//
	// https://pkg.go.dev/github.com/google/go-cmp/cmp#Equal
	t.Run("DeepEqual", func(t *testing.T) {
		assert.DeepEqual(t, NewParameterSet(), NewParameterSet())
	})
}
