// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package initialize

// Bool returns a pointer to v.
func Bool(v bool) *bool { return &v }

// ByteMap initializes m when it points to nil.
func ByteMap(m *map[string][]byte) {
	if m != nil && *m == nil {
		*m = make(map[string][]byte)
	}
}

// FromPointer returns the value that p points to.
// When p is nil, it returns the zero value of T.
func FromPointer[T any](p *T) T {
	var v T
	if p != nil {
		v = *p
	}
	return v
}

// Int32 returns a pointer to v.
func Int32(v int32) *int32 { return &v }

// Int64 returns a pointer to v.
func Int64(v int64) *int64 { return &v }

// Pointer returns a pointer to v.
func Pointer[T any](v T) *T { return &v }

// String returns a pointer to v.
func String(v string) *string { return &v }

// StringMap initializes m when it points to nil.
func StringMap(m *map[string]string) {
	if m != nil && *m == nil {
		*m = make(map[string]string)
	}
}
