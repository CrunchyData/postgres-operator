// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import "time"

// https://www.postgresql.org/support/versioning
var finalReleaseDates = map[int]time.Time{
	10: time.Date(2022, time.November+1, 10, 0, 0, 0, 0, time.UTC),
	11: time.Date(2023, time.November+1, +9, 0, 0, 0, 0, time.UTC),
	12: time.Date(2024, time.November+1, 14, 0, 0, 0, 0, time.UTC),
	13: time.Date(2025, time.November+1, 13, 0, 0, 0, 0, time.UTC),
	14: time.Date(2026, time.November+1, 12, 0, 0, 0, 0, time.UTC),
	15: time.Date(2027, time.November+1, 11, 0, 0, 0, 0, time.UTC),
	16: time.Date(2028, time.November+1, +9, 0, 0, 0, 0, time.UTC),
	17: time.Date(2029, time.November+1, +8, 0, 0, 0, 0, time.UTC),
}

// ReleaseIsFinal returns whether or not t is definitively past the final
// scheduled release of a Postgres version.
func ReleaseIsFinal(majorVersion int, t time.Time) bool {
	known, ok := finalReleaseDates[majorVersion]
	return ok && t.After(known)
}
