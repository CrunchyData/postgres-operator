// Copyright 2021 - 2023 Crunchy Data Solutions, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pgupgrade

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// JSON6902 represents a JSON Patch according to RFC 6902; the same as
// k8s.io/apimachinery/pkg/types.JSONPatchType.
type JSON6902 []interface{}

// NewJSONPatch creates a new JSON Patch according to RFC 6902; the same as
// k8s.io/apimachinery/pkg/types.JSONPatchType.
func NewJSONPatch() *JSON6902 { return &JSON6902{} }

// escapeJSONPointer encodes '~' and '/' according to RFC 6901.
var escapeJSONPointer = strings.NewReplacer(
	"~", "~0",
	"/", "~1",
).Replace

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
func (patch JSON6902) Bytes() ([]byte, error) { return patch.Data(nil) }

// Data returns the JSON representation of patch.
func (patch JSON6902) Data(client.Object) ([]byte, error) { return json.Marshal(patch) }

// IsEmpty returns true when patch has no operations.
func (patch JSON6902) IsEmpty() bool { return len(patch) == 0 }

// Type returns k8s.io/apimachinery/pkg/types.JSONPatchType.
func (patch JSON6902) Type() types.PatchType { return types.JSONPatchType }

// patch sends patch to object's endpoint in the Kubernetes API and updates
// object with any returned content. The fieldManager is set to r.Owner, but
// can be overridden in options.
// - https://docs.k8s.io/reference/using-api/server-side-apply/#managers
func (r *PGUpgradeReconciler) patch(
	ctx context.Context, object client.Object,
	patch client.Patch, options ...client.PatchOption,
) error {
	options = append([]client.PatchOption{r.Owner}, options...)
	return r.Client.Patch(ctx, object, patch, options...)
}

// apply sends an apply patch to object's endpoint in the Kubernetes API and
// updates object with any returned content. The fieldManager is set to
// r.Owner and the force parameter is true.
// - https://docs.k8s.io/reference/using-api/server-side-apply/#managers
// - https://docs.k8s.io/reference/using-api/server-side-apply/#conflicts
func (r *PGUpgradeReconciler) apply(ctx context.Context, object client.Object) error {
	// Generate an apply-patch by comparing the object to its zero value.
	zero := reflect.New(reflect.TypeOf(object).Elem()).Interface()
	data, err := client.MergeFrom(zero.(client.Object)).Data(object)
	apply := client.RawPatch(client.Apply.Type(), data)

	// Send the apply-patch with force=true.
	if err == nil {
		err = r.patch(ctx, object, apply, client.ForceOwnership)
	}

	return err
}
