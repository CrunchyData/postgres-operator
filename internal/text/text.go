// Copyright 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package text

// TruncateAt returns the first n bytes of s.
// It returns empty string when s is empty or n is less than one.
func TruncateAt(s string, n int) string {
	return s[:max(0, min(n, len(s)))]
}
