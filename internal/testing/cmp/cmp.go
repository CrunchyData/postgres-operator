// Copyright 2021 - 2026 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package cmp

import (
	stdcmp "cmp"
	"regexp"
	"strings"

	gocmp "github.com/google/go-cmp/cmp"
	gotest "gotest.tools/v3/assert/cmp"
	"sigs.k8s.io/yaml"
)

type Comparison = gotest.Comparison

// Contains succeeds if item is in collection. The collection may be a string,
// map, slice, or array. See [gotest.tools/v3/assert/cmp.Contains]. When either
// item or collection is a multi-line string, the failure message contains a
// multi-line report of the differences.
func Contains(collection, item any) Comparison {
	cString, cStringOK := collection.(string)
	iString, iStringOK := item.(string)

	if cStringOK && iStringOK {
		if strings.Contains(cString, "\n") || strings.Contains(iString, "\n") {
			return func() gotest.Result {
				if strings.Contains(cString, iString) {
					return gotest.ResultSuccess
				}
				return gotest.ResultFailureTemplate(`
--- {{ with callArg 0 }}{{ formatNode . }}{{else}}←{{end}} string does not contain
+++ {{ with callArg 1 }}{{ formatNode . }}{{else}}→{{end}} substring
{{ .Data.diff }}`,
					map[string]any{
						"diff": gocmp.Diff(collection, item),
					})
			}
		}
	}

	return gotest.Contains(collection, item)
}

// DeepEqual compares two values using [github.com/google/go-cmp/cmp] and
// succeeds if the values are equal. The comparison can be customized using
// comparison Options. See [github.com/google/go-cmp/cmp.Option] constructors
// and [github.com/google/go-cmp/cmp/cmpopts].
func DeepEqual[T any](x, y T, opts ...gocmp.Option) Comparison {
	return gotest.DeepEqual(x, y, opts...)
}

// Equal succeeds if x == y, the same as [gotest.tools/v3/assert.Equal].
// The type constraint makes it easier to compare against numeric literals and typed constants.
func Equal[T any](x, y T) Comparison {
	return gotest.Equal(x, y)
}

// Len succeeds if actual has the expected length.
func Len[Slice ~[]E, E any](actual Slice, expected int) Comparison {
	return gotest.Len(actual, expected)
}

// LenMap succeeds if actual has the expected length.
func LenMap[Map ~map[K]V, K comparable, V any](actual Map, expected int) Comparison {
	// There doesn't seem to be a way to express "map or slice" in type constraints
	// that [Go 1.22] compiler can nicely infer. Ideally, this function goes
	// away when a better constraint can be expressed on [Len].

	return gotest.Len(actual, expected)
}

// MarshalContains converts actual to YAML and succeeds if expected is in the result.
func MarshalContains(actual any, expected string) Comparison {
	b, err := yaml.Marshal(actual)
	if err != nil {
		return func() gotest.Result { return gotest.ResultFromError(err) }
	}
	return Contains(string(b), expected)
}

// MarshalMatches converts actual to YAML and compares that to expected.
func MarshalMatches(actual any, expected string) Comparison {
	b, err := yaml.Marshal(actual)
	if err != nil {
		return func() gotest.Result { return gotest.ResultFromError(err) }
	}
	return gotest.DeepEqual(string(b), dedentLines(
		strings.TrimLeft(strings.TrimRight(expected, "\t\n"), "\n"), 2,
	))
}

// Or is an alias to [stdcmp.Or] in the standard library:
//   - It returns the leftmost argument that is not zero.
//   - It returns zero when all its arguments are zero.
//
// This is here so test authors can import fewer "cmp" packages.
func Or[T comparable](values ...T) T {
	return stdcmp.Or(values...)
}

// Regexp succeeds if value contains any match of the regular expression.
// The regular expression may be a *regexp.Regexp or a string that is a valid
// regexp pattern.
func Regexp[RE *regexp.Regexp | ~string](regex RE, value string) Comparison {
	return gotest.Regexp(regex, value)
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
