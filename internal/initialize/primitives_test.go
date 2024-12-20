// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
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

func TestMap(t *testing.T) {
	t.Run("map[string][]byte", func(t *testing.T) {
		// Ignores nil pointer.
		initialize.Map((*map[string][]byte)(nil))

		var m map[string][]byte

		// Starts nil.
		assert.Assert(t, m == nil)

		// Gets initialized.
		initialize.Map(&m)
		assert.DeepEqual(t, m, map[string][]byte{})

		// Now writable.
		m["x"] = []byte("y")

		// Doesn't overwrite.
		initialize.Map(&m)
		assert.DeepEqual(t, m, map[string][]byte{"x": []byte("y")})
	})

	t.Run("map[string]string", func(t *testing.T) {
		// Ignores nil pointer.
		initialize.Map((*map[string]string)(nil))

		var m map[string]string

		// Starts nil.
		assert.Assert(t, m == nil)

		// Gets initialized.
		initialize.Map(&m)
		assert.DeepEqual(t, m, map[string]string{})

		// Now writable.
		m["x"] = "y"

		// Doesn't overwrite.
		initialize.Map(&m)
		assert.DeepEqual(t, m, map[string]string{"x": "y"})
	})
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

func TestPointers(t *testing.T) {
	t.Run("arguments", func(t *testing.T) {
		assert.Assert(t, nil != initialize.Pointers[int](), "does not return nil slice")
		assert.DeepEqual(t, []*int{}, initialize.Pointers[int]())

		s1 := initialize.Pointers(0, -99, 42)
		if assert.Check(t, len(s1) == 3, "got %#v", s1) {
			if assert.Check(t, s1[0] != nil) {
				assert.Equal(t, *s1[0], 0)
			}
			if assert.Check(t, s1[1] != nil) {
				assert.Equal(t, *s1[1], -99)
			}
			if assert.Check(t, s1[2] != nil) {
				assert.Equal(t, *s1[2], 42)
			}
		}

		// Values are the same, but pointers differ.
		s2 := initialize.Pointers(0, -99, 42)
		assert.DeepEqual(t, s1, s2)
		assert.Assert(t, s1[0] != s2[0])
		assert.Assert(t, s1[1] != s2[1])
		assert.Assert(t, s1[2] != s2[2])
	})

	t.Run("slice", func(t *testing.T) {
		var z []string
		assert.Assert(t, nil != initialize.Pointers(z...), "does not return nil slice")
		assert.DeepEqual(t, []*string{}, initialize.Pointers(z...))

		v := []string{"doot", "", "baz"}
		s1 := initialize.Pointers(v...)
		if assert.Check(t, len(s1) == 3, "got %#v", s1) {
			if assert.Check(t, s1[0] != nil) {
				assert.Equal(t, *s1[0], "doot")
			}
			if assert.Check(t, s1[1] != nil) {
				assert.Equal(t, *s1[1], "")
			}
			if assert.Check(t, s1[2] != nil) {
				assert.Equal(t, *s1[2], "baz")
			}
		}

		// Values and pointers are the same.
		s2 := initialize.Pointers(v...)
		assert.DeepEqual(t, s1, s2)
		assert.Assert(t, s1[0] == s2[0])
		assert.Assert(t, s1[1] == s2[1])
		assert.Assert(t, s1[2] == s2[2])
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
