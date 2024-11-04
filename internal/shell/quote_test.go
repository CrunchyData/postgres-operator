// Copyright 2024 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package shell

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestQuoteWord(t *testing.T) {
	assert.Equal(t, QuoteWord(""), `''`,
		"expected empty and single-quoted")

	assert.Equal(t, QuoteWord("abc"), `'abc'`,
		"expected single-quoted")

	assert.Equal(t, QuoteWord(`a" b"c`), `'a" b"c'`,
		"expected easy double-quotes")

	assert.Equal(t, QuoteWord(`a' b'c`),
		`'a'`+`"'"`+`' b'`+`"'"`+`'c'`,
		"expected close-quote-open twice")

	assert.Equal(t, QuoteWord(`a''b`),
		`'a'`+`"''"`+`'b'`,
		"expected close-quotes-open once")

	assert.Equal(t, QuoteWord(`x''''y`),
		`'x'`+`"''"`+`''`+`"''"`+`'y'`,
		"expected close-quotes-open twice")
}
