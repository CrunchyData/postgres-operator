// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package kubeapi

import (
	"encoding/json"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

func assertJSON(t testing.TB, expected interface{}, actual []byte) {
	t.Helper()

	var e, a interface{}
	var err error

	if b, ok := expected.([]byte); ok {
		err = json.Unmarshal(b, &e)
	} else if s, ok := expected.(string); ok {
		err = json.Unmarshal([]byte(s), &e)
	} else {
		t.Fatalf("bug in test: unexpected type %T", expected)
	}

	if err != nil {
		t.Fatalf("bug in test: %v", err)
	}

	if err = json.Unmarshal(actual, &a); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(e, a) {
		t.Errorf("\n--- Expected\n+++ Actual\n- %#v\n+ %#v", e, a)
	}
}

func TestEscapeJSONPointer(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct{ input, expected string }{
		{"~1", "~01"},
		{"~~", "~0~0"},
		{"/1", "~11"},
		{"//", "~1~1"},
		{"~/", "~0~1"},
		{"some/label", "some~1label"},
	} {
		actual := escapeJSONPointer(tt.input)
		if actual != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, actual)
		}
	}
}

func TestJSON6902(t *testing.T) {
	t.Parallel()

	if actual := NewJSONPatch().Type(); actual != types.JSONPatchType {
		t.Fatalf("expected %q, got %q", types.JSONPatchType, actual)
	}

	// An empty patch is valid.
	{
		patch := NewJSONPatch()
		if !patch.IsEmpty() {
			t.Fatal("expected empty")
		}
		b, err := patch.Bytes()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		assertJSON(t, `[]`, b)
	}

	// Calling Add without its value is an error.
	{
		patch := NewJSONPatch()
		patch.Add("a")
		_, err := patch.Bytes()
		if err == nil {
			t.Fatal("expected an error, got none")
		}
	}
	{
		b, err := NewJSONPatch().Add("a", "x/y", "0")(9).Bytes()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		assertJSON(t, `[{"op":"add","path":"/a/x~1y/0","value":9}]`, b)
	}

	// Remove takes no value.
	{
		b, err := NewJSONPatch().Remove("b", "m/n/o").Bytes()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		assertJSON(t, `[{"op":"remove","path":"/b/m~1n~1o"}]`, b)
	}

	// Calling Replace without its value is an error.
	{
		patch := NewJSONPatch()
		patch.Replace("a")
		_, err := patch.Bytes()
		if err == nil {
			t.Fatal("expected an error, got none")
		}
	}
	{
		b, err := NewJSONPatch().Replace("metadata", "labels", "some/thing")("5").Bytes()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		assertJSON(t, `[{"op":"replace","path":"/metadata/labels/some~1thing","value":"5"}]`, b)
	}

	// Calls are chainable.
	{
		b, err := NewJSONPatch().
			Add("a", "b", "c")(1).
			Remove("x", "y", "z").
			Replace("1", "2", "3")(nil).
			Bytes()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		assertJSON(t, `[
			{"op":"add","path":"/a/b/c","value":1},
			{"op":"remove","path":"/x/y/z"},
			{"op":"replace","path":"/1/2/3","value":null}
		]`, b)
	}
}

func TestMerge7386(t *testing.T) {
	t.Parallel()

	if actual := NewMergePatch().Type(); actual != types.MergePatchType {
		t.Fatalf("expected %q, got %q", types.MergePatchType, actual)
	}

	// An empty patch is valid.
	{
		patch := NewMergePatch()
		if !patch.IsEmpty() {
			t.Fatal("expected empty")
		}
		b, err := patch.Bytes()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		assertJSON(t, `{}`, b)
	}

	// Calling Add without a path does nothing.
	{
		b, err := NewMergePatch().Add()("anything").Bytes()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		assertJSON(t, `{}`, b)
	}

	// Calling Add without its value is an error.
	{
		patch := NewMergePatch()
		patch.Add("a")
		_, err := patch.Bytes()
		if err == nil {
			t.Fatal("expected an error, got none")
		}
	}
	{
		b, err := NewMergePatch().Add("a", "x/y", "0")(9).Bytes()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		assertJSON(t, `{"a":{"x/y":{"0":9}}}`, b)
	}

	// Remove takes no value.
	{
		b, err := NewMergePatch().Remove("b", "m/n/o").Bytes()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		assertJSON(t, `{"b":{"m/n/o":null}}`, b)
	}

	// Calls are chainable.
	{
		b, err := NewMergePatch().
			Add("a", "b", "c")(1).
			Remove("x", "y", "z").
			Bytes()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		assertJSON(t, `{
			"a":{"b":{"c":1}},
			"x":{"y":{"z":null}}
		}`, b)
	}
}

// TestMerge7386Equivalence demonstrates that the same effect can be spelled
// different ways with Merge7386.
func TestMerge7386Equivalence(t *testing.T) {
	t.Parallel()

	expected := `{
		"metadata": {
			"labels": {"lk":"lv"},
			"annotations": {"ak1":"av1", "ak2":"av2"}
		}
	}`

	patches := []*Merge7386{
		// multiple calls to Add
		NewMergePatch().
			Add("metadata", "labels", "lk")("lv").
			Add("metadata", "annotations", "ak1")("av1").
			Add("metadata", "annotations", "ak2")("av2"),

		// fewer calls using the patch type
		NewMergePatch().
			Add("metadata", "labels")(Merge7386{"lk": "lv"}).
			Add("metadata", "annotations")(Merge7386{"ak1": "av1", "ak2": "av2"}),

		// fewer calls using other types
		NewMergePatch().
			Add("metadata", "labels")(labels.Set{"lk": "lv"}).
			Add("metadata", "annotations")(map[string]string{"ak1": "av1", "ak2": "av2"}),

		// one call using the patch type
		NewMergePatch().
			Add("metadata")(Merge7386{
			"labels":      Merge7386{"lk": "lv"},
			"annotations": Merge7386{"ak1": "av1", "ak2": "av2"},
		}),

		// one call using other types
		NewMergePatch().
			Add("metadata")(map[string]interface{}{
			"labels":      labels.Set{"lk": "lv"},
			"annotations": map[string]string{"ak1": "av1", "ak2": "av2"},
		}),
	}

	for i, patch := range patches {
		b, err := patch.Bytes()
		if err != nil {
			t.Fatalf("expected no error for %v, got %v", i, err)
		}

		assertJSON(t, expected, b)
	}
}
