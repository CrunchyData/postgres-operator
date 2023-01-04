/*
 Copyright 2021 - 2023 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

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
func Contains(collection, item interface{}) Comparison {
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
					map[string]interface{}{
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
func DeepEqual(x, y interface{}, opts ...gocmp.Option) Comparison {
	return gotest.DeepEqual(x, y, opts...)
}

// MarshalMatches converts actual to YAML and compares that to expected.
func MarshalMatches(actual interface{}, expected string) Comparison {
	b, err := yaml.Marshal(actual)
	if err != nil {
		return func() gotest.Result { return gotest.ResultFromError(err) }
	}
	return gotest.DeepEqual(string(b), strings.Trim(expected, "\t\n")+"\n")
}

// Regexp succeeds if value contains any match of the regular expression re.
// The regular expression may be a *regexp.Regexp or a string that is a valid
// regexp pattern.
func Regexp(re interface{}, value string) Comparison {
	return gotest.Regexp(re, value)
}
