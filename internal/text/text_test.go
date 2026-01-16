// Copyright 2025 - 2026 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package text_test

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/internal/text"
)

func TestTruncateAt(t *testing.T) {
	assert.Equal(t, text.TruncateAt("", -5), "")
	assert.Equal(t, text.TruncateAt("", -0), "")
	assert.Equal(t, text.TruncateAt("", +0), "")
	assert.Equal(t, text.TruncateAt("", +5), "")

	assert.Equal(t, text.TruncateAt("lorem ipsum dolor sit amet", -5), "")
	assert.Equal(t, text.TruncateAt("lorem ipsum dolor sit amet", -0), "")
	assert.Equal(t, text.TruncateAt("lorem ipsum dolor sit amet", +0), "")
	assert.Equal(t, text.TruncateAt("lorem ipsum dolor sit amet", +5), "lorem")
	assert.Equal(t, text.TruncateAt("lorem ipsum dolor sit amet", 15), "lorem ipsum dol")
	assert.Equal(t, text.TruncateAt("lorem ipsum dolor sit amet", 50), "lorem ipsum dolor sit amet")
}
