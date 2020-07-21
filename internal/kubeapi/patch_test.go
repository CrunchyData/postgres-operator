package kubeapi

/*
 Copyright 2020 Crunchy Data Solutions, Inc.
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
	"encoding/json"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/labels"
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

	{
		b, err := NewJSONPatch().Bytes()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		assertJSON(t, `[]`, b)
	}
	{
		b, err := NewJSONPatch().Add(9, "a", "x/y", "0").Bytes()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		assertJSON(t, `[{"op":"add","path":"/a/x~1y/0","value":9}]`, b)
	}
	{
		b, err := NewJSONPatch().Remove("b", "m/n/o").Bytes()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		assertJSON(t, `[{"op":"remove","path":"/b/m~1n~1o"}]`, b)
	}
	{
		b, err := NewJSONPatch().Replace("5", "metadata", "labels", "some/thing").Bytes()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		assertJSON(t, `[{"op":"replace","path":"/metadata/labels/some~1thing","value":"5"}]`, b)
	}
	{
		b, err := NewJSONPatch().
			Add(1, "a", "b", "c").
			Remove("x", "y", "z").
			Replace(nil, "1", "2", "3").
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

	{
		b, err := NewMergePatch().Bytes()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		assertJSON(t, `{}`, b)
	}
	{
		b, err := NewMergePatch().Add(9, "a", "x/y", "0").Bytes()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		assertJSON(t, `{"a":{"x/y":{"0":9}}}`, b)
	}
	{
		b, err := NewMergePatch().Remove("b", "m/n/o").Bytes()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		assertJSON(t, `{"b":{"m/n/o":null}}`, b)
	}
	{
		b, err := NewMergePatch().
			Add(1, "a", "b", "c").
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
			Add("lv", "metadata", "labels", "lk").
			Add("av1", "metadata", "annotations", "ak1").
			Add("av2", "metadata", "annotations", "ak2"),

		// fewer calls using the patch type
		NewMergePatch().
			Add(Merge7386{"lk": "lv"}, "metadata", "labels").
			Add(Merge7386{"ak1": "av1", "ak2": "av2"}, "metadata", "annotations"),

		// fewer calls using other types
		NewMergePatch().
			Add(labels.Set{"lk": "lv"}, "metadata", "labels").
			Add(map[string]string{"ak1": "av1", "ak2": "av2"}, "metadata", "annotations"),

		// one call using the patch type
		NewMergePatch().
			Add(Merge7386{
				"labels":      Merge7386{"lk": "lv"},
				"annotations": Merge7386{"ak1": "av1", "ak2": "av2"},
			}, "metadata"),

		// one call using other types
		NewMergePatch().
			Add(map[string]interface{}{
				"labels":      labels.Set{"lk": "lv"},
				"annotations": map[string]string{"ak1": "av1", "ak2": "av2"},
			}, "metadata"),
	}

	for i, patch := range patches {
		b, err := patch.Bytes()
		if err != nil {
			t.Fatalf("expected no error for %v, got %v", i, err)
		}

		assertJSON(t, expected, b)
	}
}
