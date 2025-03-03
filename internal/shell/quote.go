// Copyright 2024 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package shell

import "strings"

// escapeSingleQuoted is used by [QuoteWord].
var escapeSingleQuoted = strings.NewReplacer(
	// slightly shorter results for the unlikely pair of quotes.
	`''`, `'"''"'`,

	// first, close the single-quote U+0027,
	// add one between double-quotes U+0022,
	// then reopen the single-quote U+0027.
	`'`, `'"'"'`,
).Replace

// QuoteWord ensures that v is interpreted by a shell as a single word.
func QuoteWord(v string) string {
	// https://pubs.opengroup.org/onlinepubs/9799919799/utilities/V3_chap02.html
	// https://www.gnu.org/software/bash/manual/html_node/Quoting.html
	return `'` + escapeSingleQuoted(v) + `'`
}

// QuoteWords ensures that s is interpreted by a shell as individual words.
func QuoteWords(s ...string) []string {
	quoted := make([]string, len(s))
	for i := range s {
		quoted[i] = QuoteWord(s[i])
	}
	return quoted
}
