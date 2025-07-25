// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package require_test

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/crunchydata/postgres-operator/internal/testing/require"
)

func TestUnmarshalInto(t *testing.T) {
	t.Parallel()

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
		{input: `{a: b, c, d}`, expected: map[string]any{"a": "b", "c": nil, "d": nil}},
	} {
		sink := reflect.Zero(reflect.TypeOf(tt.expected)).Interface()
		require.UnmarshalInto(t, &sink, tt.input)

		if !reflect.DeepEqual(tt.expected, sink) {
			t.Fatalf("expected %[1]T(%#[1]v), got %[2]T(%#[2]v)", tt.expected, sink)
		}
	}
}

func TestUnmarshalIntoField(t *testing.T) {
	t.Parallel()

	var u unstructured.Unstructured

	t.Run("NestedString", func(t *testing.T) {
		u.Object = nil
		require.UnmarshalIntoField(t, &u, `asdf`, "spec", "nested", "field")

		if !reflect.DeepEqual(u.Object, map[string]any{
			"spec": map[string]any{
				"nested": map[string]any{
					"field": "asdf",
				},
			},
		}) {
			t.Fatalf("got %[1]T(%#[1]v)", u.Object)
		}
	})

	t.Run("Numeric", func(t *testing.T) {
		u.Object = nil
		require.UnmarshalIntoField(t, &u, `99`, "one")
		require.UnmarshalIntoField(t, &u, `5.7`, "two")

		// Kubernetes distinguishes between integral and fractional numbers.
		if !reflect.DeepEqual(u.Object, map[string]any{
			"one": int64(99),
			"two": float64(5.7),
		}) {
			t.Fatalf("got %[1]T(%#[1]v)", u.Object)
		}
	})

	// Correctly fails with: BUG: called without a destination
	// require.UnmarshalIntoField(t, &u, `true`)
}
