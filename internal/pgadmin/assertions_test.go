/*
 Copyright 2021 Crunchy Data Solutions, Inc.
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

package pgadmin

import (
	"gotest.tools/v3/assert/cmp"
	"sigs.k8s.io/yaml"
)

func marshalEquals(actual interface{}, expected string) cmp.Comparison {
	b, err := yaml.Marshal(actual)
	return func() cmp.Result {
		if err != nil {
			return cmp.ResultFromError(err)
		}
		return cmp.DeepEqual(string(b), expected)()
	}
}
