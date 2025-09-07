// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
	"sigs.k8s.io/yaml"
)

// MarshalsTo converts x to YAML and compares that to y.
func MarshalsTo[T []byte | string](x any, y T) cmp.Comparison {
	b, err := yaml.Marshal(x)
	if err != nil {
		return func() cmp.Result { return cmp.ResultFromError(err) }
	}
	return cmp.DeepEqual(string(b), dedentLines(
		strings.TrimLeft(strings.TrimRight(string(y), "\t\n"), "\n"), 2,
	))
}

var leadingTabs = regexp.MustCompile(`^\t+`)

// dedentLines finds the shortest leading whitespace of every line in data and then removes it from every line.
// When tabWidth is positive, leading tabs are converted to spaces first.
func dedentLines(data string, tabWidth int) string {
	if len(data) < 1 {
		return ""
	}

	var lines = make([]string, 0, 20)
	var lowest, highest string

	for line := range strings.Lines(data) {
		tabs := leadingTabs.FindString(line)

		// Replace any leading tabs with spaces when tabWidth is positive.
		// NOTE: [strings.Repeat] has a fast-path for spaces.
		if need := tabWidth * len(tabs); need > 0 {
			line = strings.Repeat(" ", need) + line[len(tabs):]
		}

		switch {
		case lowest == "", highest == "":
			lowest, highest = line, line

		case len(strings.TrimSpace(line)) > 0:
			lowest = min(lowest, line)
			highest = max(highest, line)
		}

		lines = append(lines, line)
	}

	// This treats one tab the same as one space.
	// That is, it expects all lines to be indented using spaces or using tabs; not both.
	if width := func() int {
		for i := range lowest {
			if (lowest[i] != ' ' && lowest[i] != '\t') || lowest[i] != highest[i] {
				return i
			}
		}
		return len(lowest)
	}(); width > 0 {
		for i := range lines {
			if len(lines[i]) > width {
				lines[i] = lines[i][width:]
			} else {
				lines[i] = "\n"
			}
		}
	}

	return strings.TrimSuffix(strings.Join(lines, ""), "\n") + "\n"
}

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
