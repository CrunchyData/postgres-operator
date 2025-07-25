// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package require

import (
	"errors"
	"testing"

	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/json"
	"sigs.k8s.io/yaml"
)

// UnmarshalInto parses input as YAML (or JSON) the same way as the Kubernetes
// API Server writing into output. It calls t.Fatal when something fails.
func UnmarshalInto[Data ~string | ~[]byte, Destination *T, T any](
	t testing.TB, output Destination, input Data,
) {
	t.Helper()

	// The REST API uses serializers:
	//
	// https://pkg.go.dev/k8s.io/apimachinery/pkg/runtime/serializer/json
	// https://pkg.go.dev/k8s.io/apimachinery/pkg/runtime/serializer/yaml
	//
	// The util package follows similar paths (strict, preserve ints, etc.)
	//
	// https://pkg.go.dev/k8s.io/apimachinery/pkg/util/json
	// https://pkg.go.dev/k8s.io/apimachinery/pkg/util/yaml

	data, err := yaml.YAMLToJSONStrict([]byte(input))
	assert.NilError(t, err)

	strict, err := json.UnmarshalStrict(data, output)
	assert.NilError(t, err)
	assert.NilError(t, errors.Join(strict...))
}

// UnmarshalIntoField parses input as YAML (or JSON) the same way as the Kubernetes API Server.
// The result goes into a (nested) field of output. It calls t.Fatal when something fails.
func UnmarshalIntoField[Data ~string | ~[]byte](
	t testing.TB, output *unstructured.Unstructured, input Data, fields ...string,
) {
	t.Helper()

	if len(fields) == 0 {
		t.Fatal("BUG: called without a destination")
	}

	if output.Object == nil {
		output.Object = map[string]any{}
	}

	var value any
	UnmarshalInto(t, &value, []byte(input))

	assert.NilError(t, unstructured.SetNestedField(output.Object, value, fields...))
}
