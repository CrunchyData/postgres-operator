// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package kubeapi

import (
	"strings"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// escapeJSONPointer encodes '~' and '/' according to RFC 6901.
var escapeJSONPointer = strings.NewReplacer(
	"~", "~0",
	"/", "~1",
).Replace

// JSON6902 represents a JSON Patch according to RFC 6902; the same as [types.JSONPatchType].
type JSON6902 []any

// NewJSONPatch creates a new JSON Patch according to RFC 6902; the same as [types.JSONPatchType].
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
func (patch *JSON6902) Add(path ...string) func(value any) *JSON6902 {
	i := len(*patch)
	f := func(value any) *JSON6902 {
		(*patch)[i] = map[string]any{
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
func (patch *JSON6902) Remove(path ...string) *JSON6902 {
	*patch = append(*patch, map[string]any{
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
func (patch *JSON6902) Replace(path ...string) func(value any) *JSON6902 {
	i := len(*patch)
	f := func(value any) *JSON6902 {
		(*patch)[i] = map[string]any{
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
func (patch *JSON6902) Bytes() ([]byte, error) { return patch.Data(nil) }

// Data returns the JSON representation of patch.
func (patch *JSON6902) Data(client.Object) ([]byte, error) { return json.Marshal(*patch) }

// IsEmpty returns true when patch has no operations.
func (patch *JSON6902) IsEmpty() bool { return len(*patch) == 0 }

// Type returns [types.JSONPatchType].
func (patch *JSON6902) Type() types.PatchType { return types.JSONPatchType }

// Merge7386 represents a JSON Merge Patch according to RFC 7386; the same as [types.MergePatchType].
type Merge7386 map[string]any

// NewMergePatch creates a new JSON Merge Patch according to RFC 7386; the same as [types.MergePatchType].
func NewMergePatch() *Merge7386 { return &Merge7386{} }

// Add modifies patch to indicate that the member at path should be added or
// replaced with value.
//
// > If the provided merge patch contains members that do not appear
// > within the target, those members are added.  If the target does
// > contain the member, the value is replaced.  Null values in the merge
// > patch are given special meaning to indicate the removal of existing
// > values in the target.
func (patch *Merge7386) Add(path ...string) func(value any) *Merge7386 {
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
		return func(any) *Merge7386 { return patch }
	}

	f := func(value any) *Merge7386 {
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
func (patch *Merge7386) Bytes() ([]byte, error) { return patch.Data(nil) }

// Data returns the JSON representation of patch.
func (patch *Merge7386) Data(client.Object) ([]byte, error) { return json.Marshal(*patch) }

// IsEmpty returns true when patch has no modifications.
func (patch *Merge7386) IsEmpty() bool { return len(*patch) == 0 }

// Type returns [types.MergePatchType].
func (patch *Merge7386) Type() types.PatchType { return types.MergePatchType }
