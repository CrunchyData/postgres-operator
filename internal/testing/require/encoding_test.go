// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package require_test

import (
	"reflect"
	"testing"

	"github.com/crunchydata/postgres-operator/internal/testing/require"
)

func TestUnmarshalInto(t *testing.T) {
	for _, tt := range []struct {
		input    string
		expected any
	}{
		// Any fraction that amounts to an integral number is converted to an integer.
		// See: https://go.dev/play/p/dvXRVhYO8UH
		{input: `3`, expected: int64(3)},
		{input: `3.000`, expected: int64(3)},
		{input: `0.03e2`, expected: int64(3)},
		{input: `{a: 5}`, expected: map[string]any{"a": int64(5)}},
		{input: `{a: 5.000}`, expected: map[string]any{"a": int64(5)}},
		{input: `{a: 0.05e2}`, expected: map[string]any{"a": int64(5)}},

		// YAML or JSON
		{input: `asdf`, expected: "asdf"},
		{input: `"asdf"`, expected: "asdf"},
		{input: `[1, 2.3, true]`, expected: []any{int64(1), float64(2.3), true}},
	} {
		sink := reflect.Zero(reflect.TypeOf(tt.expected)).Interface()
		require.UnmarshalInto(t, &sink, tt.input)

		if !reflect.DeepEqual(tt.expected, sink) {
			t.Fatalf("expected %[1]T(%#[1]v), got %[2]T(%#[2]v)", tt.expected, sink)
		}
	}
}
