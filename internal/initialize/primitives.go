// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package initialize

// Bool returns a pointer to v.
func Bool(v bool) *bool { return &v }

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

// Map initializes m when it points to nil.
func Map[M ~map[K]V, K comparable, V any](m *M) {
	// See https://pkg.go.dev/maps for similar type constraints.

	if m != nil && *m == nil {
		*m = make(M)
	}
}

// Pointer returns a pointer to v.
func Pointer[T any](v T) *T { return &v }

// String returns a pointer to v.
func String(v string) *string { return &v }
