// Copyright 2017 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"strings"
)

// SQLQuoteIdentifier quotes an "identifier" (e.g. a table or a column name) to
// be used as part of an SQL statement.
//
// Any double quotes in name will be escaped.  The quoted identifier will be
// case-sensitive when used in a query.  If the input string contains a zero
// byte, the result will be truncated immediately before it.
//
// Implementation borrowed from lib/pq: https://github.com/lib/pq which is
// licensed under the MIT License
func SQLQuoteIdentifier(identifier string) string {
	end := strings.IndexRune(identifier, 0)

	if end > -1 {
		identifier = identifier[:end]
	}

	return `"` + strings.Replace(identifier, `"`, `""`, -1) + `"`
}

// SQLQuoteLiteral quotes a 'literal' (e.g. a parameter, often used to pass literal
// to DDL and other statements that do not accept parameters) to be used as part
// of an SQL statement.
//
// Any single quotes in name will be escaped. Any backslashes (i.e. "\") will be
// replaced by two backslashes (i.e. "\\") and the C-style escape identifier
// that PostgreSQL provides ('E') will be prepended to the string.
//
// Implementation borrowed from lib/pq: https://github.com/lib/pq which is
// licensed under the MIT License. Curiously, @jkatz and @cbandy were the ones
// who worked on the patch to add this, prior to being at Crunchy Data
func SQLQuoteLiteral(literal string) string {
	// This follows the PostgreSQL internal algorithm for handling quoted literals
	// from libpq, which can be found in the "PQEscapeStringInternal" function,
	// which is found in the libpq/fe-exec.c source file:
	// https://git.postgresql.org/gitweb/?p=postgresql.git;a=blob;f=src/interfaces/libpq/fe-exec.c
	//
	// substitute any single-quotes (') with two single-quotes ('')
	literal = strings.Replace(literal, `'`, `''`, -1)
	// determine if the string has any backslashes (\) in it.
	// if it does, replace any backslashes (\) with two backslashes (\\)
	// then, we need to wrap the entire string with a PostgreSQL
	// C-style escape. Per how "PQEscapeStringInternal" handles this case, we
	// also add a space before the "E"
	if strings.Contains(literal, `\`) {
		literal = strings.Replace(literal, `\`, `\\`, -1)
		literal = ` E'` + literal + `'`
	} else {
		// otherwise, we can just wrap the literal with a pair of single quotes
		literal = `'` + literal + `'`
	}
	return literal
}
