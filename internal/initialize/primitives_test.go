/*
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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

package initialize_test

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/internal/initialize"
)

func TestBool(t *testing.T) {
	n := initialize.Bool(false)
	if assert.Check(t, n != nil) {
		assert.Equal(t, *n, false)
	}

	y := initialize.Bool(true)
	if assert.Check(t, y != nil) {
		assert.Equal(t, *y, true)
	}
}

func TestByteMap(t *testing.T) {
	// Ignores nil pointer.
	initialize.ByteMap(nil)

	var m map[string][]byte

	// Starts nil.
	assert.Assert(t, m == nil)

	// Gets initialized.
	initialize.ByteMap(&m)
	assert.DeepEqual(t, m, map[string][]byte{})

	// Now writable.
	m["x"] = []byte("y")

	// Doesn't overwrite.
	initialize.ByteMap(&m)
	assert.DeepEqual(t, m, map[string][]byte{"x": []byte("y")})
}

func TestInt32(t *testing.T) {
	z := initialize.Int32(0)
	if assert.Check(t, z != nil) {
		assert.Equal(t, *z, int32(0))
	}

	n := initialize.Int32(-99)
	if assert.Check(t, n != nil) {
		assert.Equal(t, *n, int32(-99))
	}

	p := initialize.Int32(42)
	if assert.Check(t, p != nil) {
		assert.Equal(t, *p, int32(42))
	}
}

func TestInt64(t *testing.T) {
	z := initialize.Int64(0)
	if assert.Check(t, z != nil) {
		assert.Equal(t, *z, int64(0))
	}

	n := initialize.Int64(-99)
	if assert.Check(t, n != nil) {
		assert.Equal(t, *n, int64(-99))
	}

	p := initialize.Int64(42)
	if assert.Check(t, p != nil) {
		assert.Equal(t, *p, int64(42))
	}
}

func TestString(t *testing.T) {
	z := initialize.String("")
	if assert.Check(t, z != nil) {
		assert.Equal(t, *z, "")
	}

	n := initialize.String("sup")
	if assert.Check(t, n != nil) {
		assert.Equal(t, *n, "sup")
	}
}

func TestStringMap(t *testing.T) {
	// Ignores nil pointer.
	initialize.StringMap(nil)

	var m map[string]string

	// Starts nil.
	assert.Assert(t, m == nil)

	// Gets initialized.
	initialize.StringMap(&m)
	assert.DeepEqual(t, m, map[string]string{})

	// Now writable.
	m["x"] = "y"

	// Doesn't overwrite.
	initialize.StringMap(&m)
	assert.DeepEqual(t, m, map[string]string{"x": "y"})
}
