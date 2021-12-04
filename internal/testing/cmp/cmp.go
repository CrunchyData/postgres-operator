/*
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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

	gotest "gotest.tools/v3/assert/cmp"
	"sigs.k8s.io/yaml"
)

type Comparison = gotest.Comparison

// MarshalMatches converts actual to YAML and compares that to expected.
func MarshalMatches(actual interface{}, expected string) Comparison {
	b, err := yaml.Marshal(actual)
	if err != nil {
		return func() gotest.Result { return gotest.ResultFromError(err) }
	}
	return gotest.DeepEqual(string(b), strings.Trim(expected, "\t\n")+"\n")
}
