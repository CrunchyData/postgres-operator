// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestQuoteLiteral(t *testing.T) {
	assert.Equal(t, QuoteLiteral(``), ` E''`)
	assert.Equal(t, QuoteLiteral(`ab"cd\ef'gh`), ` E'ab"cd\\ef''gh'`)
}
