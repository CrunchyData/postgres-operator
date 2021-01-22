package kubeapi

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
	"encoding/json"
	"strings"
)

// escapeJSONPointer encodes '~' and '/' according to RFC 6901.
var escapeJSONPointer = strings.NewReplacer(
	"~", "~0",
	"/", "~1",
).Replace

// JSON6902 represents a JSON Patch according to RFC 6902; the same as
// k8s.io/apimachinery/pkg/types.JSONPatchType.
type JSON6902 []interface{}

// NewJSONPatch creates a new JSON Patch according to RFC 6902; the same as
// k8s.io/apimachinery/pkg/types.JSONPatchType.
func NewJSONPatch() *JSON6902 { return &JSON6902{} }

func (*JSON6902) pointer(tokens ...string) string {
	var b strings.Builder

	for _, t := range tokens {
		_ = b.WriteByte('/')
		_, _ = b.WriteString(escapeJSONPointer(t))
	}

	return b.String()
}

// Add appends an "add" operation to patch.
//
// > The "add" operation performs one of the following functions,
// > depending upon what the target location references:
// >
// > o  If the target location specifies an array index, a new value is
// >    inserted into the array at the specified index.
// >
// > o  If the target location specifies an object member that does not
// >    already exist, a new member is added to the object.
// >
// > o  If the target location specifies an object member that does exist,
// >    that member's value is replaced.
//
func (patch *JSON6902) Add(path ...string) func(value interface{}) *JSON6902 {
	i := len(*patch)
	f := func(value interface{}) *JSON6902 {
		(*patch)[i] = map[string]interface{}{
			"op":    "add",
			"path":  patch.pointer(path...),
			"value": value,
		}
		return patch
	}

	*patch = append(*patch, f)

	return f
}

// Remove appends a "remove" operation to patch.
//
// > The "remove" operation removes the value at the target location.
// >
// > The target location MUST exist for the operation to be successful.
//
func (patch *JSON6902) Remove(path ...string) *JSON6902 {
	*patch = append(*patch, map[string]interface{}{
		"op":   "remove",
		"path": patch.pointer(path...),
	})

	return patch
}

// Replace appends a "replace" operation to patch.
//
// > The "replace" operation replaces the value at the target location
// > with a new value.
// >
// > The target location MUST exist for the operation to be successful.
//
func (patch *JSON6902) Replace(path ...string) func(value interface{}) *JSON6902 {
	i := len(*patch)
	f := func(value interface{}) *JSON6902 {
		(*patch)[i] = map[string]interface{}{
			"op":    "replace",
			"path":  patch.pointer(path...),
			"value": value,
		}
		return patch
	}

	*patch = append(*patch, f)

	return f
}

// Bytes returns the JSON representation of patch.
func (patch JSON6902) Bytes() ([]byte, error) { return json.Marshal(patch) }

// Merge7386 represents a JSON Merge Patch according to RFC 7386; the same as
// k8s.io/apimachinery/pkg/types.MergePatchType.
type Merge7386 map[string]interface{}

// NewMergePatch creates a new JSON Merge Patch according to RFC 7386; the same
// as k8s.io/apimachinery/pkg/types.MergePatchType.
func NewMergePatch() *Merge7386 { return &Merge7386{} }

// Add modifies patch to indicate that the member at path should be added or
// replaced with value.
//
// > If the provided merge patch contains members that do not appear
// > within the target, those members are added.  If the target does
// > contain the member, the value is replaced.  Null values in the merge
// > patch are given special meaning to indicate the removal of existing
// > values in the target.
//
func (patch *Merge7386) Add(path ...string) func(value interface{}) *Merge7386 {
	position := *patch

	for len(path) > 1 {
		p, ok := position[path[0]].(Merge7386)
		if !ok {
			p = Merge7386{}
			position[path[0]] = p
		}

		position = p
		path = path[1:]
	}

	if len(path) < 1 {
		return func(interface{}) *Merge7386 { return patch }
	}

	f := func(value interface{}) *Merge7386 {
		position[path[0]] = value
		return patch
	}

	position[path[0]] = f

	return f
}

// Remove modifies patch to indicate that the member at path should be removed
// if it exists.
func (patch *Merge7386) Remove(path ...string) *Merge7386 {
	return patch.Add(path...)(nil)
}

// Bytes returns the JSON representation of patch.
func (patch Merge7386) Bytes() ([]byte, error) { return json.Marshal(patch) }
