// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestReleaseIsFinal(t *testing.T) {
	// On November 4th, 2024, PG 10 and 11 were EOL and 12-17 were supported.
	testDate, err := time.Parse("2006-Jan-02", "2024-Nov-04")
	assert.NilError(t, err)
	assert.Check(t, ReleaseIsFinal(10, testDate))
	assert.Check(t, ReleaseIsFinal(11, testDate))
	assert.Check(t, !ReleaseIsFinal(12, testDate))
	assert.Check(t, !ReleaseIsFinal(13, testDate))
	assert.Check(t, !ReleaseIsFinal(14, testDate))
	assert.Check(t, !ReleaseIsFinal(15, testDate))
	assert.Check(t, !ReleaseIsFinal(16, testDate))
	assert.Check(t, !ReleaseIsFinal(17, testDate))

	// On December 15th, 2024 we alert that PG 12 is EOL
	testDate = testDate.AddDate(0, 1, 11)
	assert.Check(t, ReleaseIsFinal(12, testDate))

	// ReleaseIsFinal covers PG versions 10 and greater. Any version not covered
	// by the case statement in ReleaseIsFinal returns false
	assert.Check(t, !ReleaseIsFinal(1, testDate))
}
