// Copyright 2022 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"reflect"
	"testing"

	"gotest.tools/v3/assert"
	"sigs.k8s.io/yaml"
)

func TestSchemalessObjectDeepCopy(t *testing.T) {
	t.Parallel()

	var n *SchemalessObject
	assert.DeepEqual(t, n, n.DeepCopy())

	var z SchemalessObject
	assert.DeepEqual(t, z, *z.DeepCopy())

	var one SchemalessObject
	assert.NilError(t, yaml.Unmarshal(
		[]byte(`{ str: value, num: 1, arr: [a, 2, true] }`), &one,
	))

	// reflect and go-cmp agree the original and copy are equivalent.
	same := *one.DeepCopy()
	assert.DeepEqual(t, one, same)
	assert.Assert(t, reflect.DeepEqual(one, same))

	// Changes to the copy do not affect the original.
	{
		change := *one.DeepCopy()
		change["str"] = "banana"
		assert.Assert(t, reflect.DeepEqual(one, same))
		assert.Assert(t, !reflect.DeepEqual(one, change))
	}
	{
		change := *one.DeepCopy()
		change["num"] = 99
		assert.Assert(t, reflect.DeepEqual(one, same))
		assert.Assert(t, !reflect.DeepEqual(one, change))
	}
	{
		change := *one.DeepCopy()
		change["arr"].([]any)[0] = "rock"
		assert.Assert(t, reflect.DeepEqual(one, same))
		assert.Assert(t, !reflect.DeepEqual(one, change))
	}
	{
		change := *one.DeepCopy()
		change["arr"] = append(change["arr"].([]any), "more")
		assert.Assert(t, reflect.DeepEqual(one, same))
		assert.Assert(t, !reflect.DeepEqual(one, change))
	}
}
