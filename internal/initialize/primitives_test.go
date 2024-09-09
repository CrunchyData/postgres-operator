// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

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

func TestFromPointer(t *testing.T) {
	t.Run("bool", func(t *testing.T) {
		assert.Equal(t, initialize.FromPointer((*bool)(nil)), false)
		assert.Equal(t, initialize.FromPointer(initialize.Pointer(false)), false)
		assert.Equal(t, initialize.FromPointer(initialize.Pointer(true)), true)
	})

	t.Run("int32", func(t *testing.T) {
		assert.Equal(t, initialize.FromPointer((*int32)(nil)), int32(0))
		assert.Equal(t, initialize.FromPointer(initialize.Pointer(int32(0))), int32(0))
		assert.Equal(t, initialize.FromPointer(initialize.Pointer(int32(-99))), int32(-99))
		assert.Equal(t, initialize.FromPointer(initialize.Pointer(int32(42))), int32(42))
	})

	t.Run("int64", func(t *testing.T) {
		assert.Equal(t, initialize.FromPointer((*int64)(nil)), int64(0))
		assert.Equal(t, initialize.FromPointer(initialize.Pointer(int64(0))), int64(0))
		assert.Equal(t, initialize.FromPointer(initialize.Pointer(int64(-99))), int64(-99))
		assert.Equal(t, initialize.FromPointer(initialize.Pointer(int64(42))), int64(42))
	})

	t.Run("string", func(t *testing.T) {
		assert.Equal(t, initialize.FromPointer((*string)(nil)), "")
		assert.Equal(t, initialize.FromPointer(initialize.Pointer("")), "")
		assert.Equal(t, initialize.FromPointer(initialize.Pointer("sup")), "sup")
	})
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

func TestPointer(t *testing.T) {
	t.Run("bool", func(t *testing.T) {
		n := initialize.Pointer(false)
		if assert.Check(t, n != nil) {
			assert.Equal(t, *n, false)
		}

		y := initialize.Pointer(true)
		if assert.Check(t, y != nil) {
			assert.Equal(t, *y, true)
		}
	})

	t.Run("int32", func(t *testing.T) {
		z := initialize.Pointer(int32(0))
		if assert.Check(t, z != nil) {
			assert.Equal(t, *z, int32(0))
		}

		n := initialize.Pointer(int32(-99))
		if assert.Check(t, n != nil) {
			assert.Equal(t, *n, int32(-99))
		}

		p := initialize.Pointer(int32(42))
		if assert.Check(t, p != nil) {
			assert.Equal(t, *p, int32(42))
		}
	})

	t.Run("int64", func(t *testing.T) {
		z := initialize.Pointer(int64(0))
		if assert.Check(t, z != nil) {
			assert.Equal(t, *z, int64(0))
		}

		n := initialize.Pointer(int64(-99))
		if assert.Check(t, n != nil) {
			assert.Equal(t, *n, int64(-99))
		}

		p := initialize.Pointer(int64(42))
		if assert.Check(t, p != nil) {
			assert.Equal(t, *p, int64(42))
		}
	})

	t.Run("string", func(t *testing.T) {
		z := initialize.Pointer("")
		if assert.Check(t, z != nil) {
			assert.Equal(t, *z, "")
		}

		n := initialize.Pointer("sup")
		if assert.Check(t, n != nil) {
			assert.Equal(t, *n, "sup")
		}
	})
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
