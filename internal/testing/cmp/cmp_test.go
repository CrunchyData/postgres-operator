// Copyright 2021 - 2026 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package cmp

import (
	"fmt"
	"testing"

	"gotest.tools/v3/assert"
)

func TestDedentLines(t *testing.T) {
	for _, tc := range []struct {
		width    int
		input    string
		expected string
	}{
		// empty stays that way
		{width: 0, input: "", expected: ""},
		{width: 1, input: "", expected: ""},
		{width: 2, input: "", expected: ""},

		// adds a missing newline
		{input: "\n", expected: "\n"},
		{input: "x", expected: "x\n"},
		{input: "x\n", expected: "x\n"},
		{input: "x\n\n", expected: "x\n\n"},

		// width does not affect whats removed
		{width: 2, input: "x", expected: "x\n"},
		{width: 2, input: "\tx", expected: "x\n"},
		{width: 2, input: " x", expected: "x\n"},
		{width: 2, input: "  x", expected: "x\n"},
		{width: 2, input: "   x", expected: "x\n"},

		// positive width changes tabs to spaces
		{width: 0, input: "\t\t~\n\t~\n", expected: "\t~\n~\n"},
		{width: 1, input: "\t\t~\n\t~\n", expected: " ~\n~\n"},
		{width: 2, input: "\t\t~\n\t~\n", expected: "  ~\n~\n"},

		// width does not affect spaces
		{width: 0, input: "  ~\n ~\n", expected: " ~\n~\n"},
		{width: 1, input: "  ~\n ~\n", expected: " ~\n~\n"},
		{width: 2, input: "  ~\n ~\n", expected: " ~\n~\n"},

		// smallest indent can be anywhere
		{input: " ~\n  ~\n  ~\n", expected: "~\n ~\n ~\n"},
		{input: "  ~\n ~\n  ~\n", expected: " ~\n~\n ~\n"},
		{input: "  ~\n  ~\n ~\n", expected: " ~\n ~\n~\n"},

		// entirely whitespace becomes newline
		{input: " ", expected: "\n"},
		{input: "  ", expected: "\n"},
		{input: "\t", expected: "\n"},
		{input: "\t\t", expected: "\n"},

		// blank lines preserved
		{input: " ~\n\n ~\n", expected: "~\n\n~\n"},
	} {
		t.Run(fmt.Sprintf("%v:%#v", tc.width, tc.input), func(t *testing.T) {
			assert.DeepEqual(t, dedentLines(tc.input, tc.width), tc.expected)
		})
	}
}
