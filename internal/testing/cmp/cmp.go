// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package cmp

import (
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
func DeepEqual(x, y any, opts ...gocmp.Option) Comparison {
	return gotest.DeepEqual(x, y, opts...)
}

// MarshalMatches converts actual to YAML and compares that to expected.
func MarshalMatches(actual any, expected string) Comparison {
	b, err := yaml.Marshal(actual)
	if err != nil {
		return func() gotest.Result { return gotest.ResultFromError(err) }
	}
	return gotest.DeepEqual(string(b), strings.Trim(expected, "\t\n")+"\n")
}

// Regexp succeeds if value contains any match of the regular expression re.
// The regular expression may be a *regexp.Regexp or a string that is a valid
// regexp pattern.
func Regexp(re any, value string) Comparison {
	return gotest.Regexp(re, value)
}
