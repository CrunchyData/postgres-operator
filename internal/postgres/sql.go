// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import "strings"

// escapeLiteral is called by QuoteLiteral to add backslashes before special
// characters of the "escape" string syntax. Double quote marks to escape them
// regardless of the "backslash_quote" parameter.
var escapeLiteral = strings.NewReplacer(`'`, `''`, `\`, `\\`).Replace

// QuoteLiteral escapes v so it can be safely used as a literal (or constant)
// in an SQL statement.
func QuoteLiteral(v string) string {
	// Use the "escape" syntax to ensure that backslashes behave consistently regardless
	// of the "standard_conforming_strings" parameter. Include a space before so
	// the "E" cannot change the meaning of an adjacent SQL keyword or identifier.
	// - https://www.postgresql.org/docs/current/sql-syntax-lexical.html
	return ` E'` + escapeLiteral(v) + `'`
}
