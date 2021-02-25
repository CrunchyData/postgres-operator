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

package naming

import (
	"testing"

	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/util/validation"
)

func TestLabelsValid(t *testing.T) {
	assert.Assert(t, nil == validation.IsQualifiedName(LabelCluster))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelInstance))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelInstanceSet))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelPatroni))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelRole))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelPGBackRest))
	assert.Assert(t, nil == validation.IsQualifiedName(LabelPGBackRestRepo))
}

func TestLabelValuesValid(t *testing.T) {
	assert.Assert(t, nil == validation.IsDNS1123Label(RolePatroniReplica))
	assert.Assert(t, nil == validation.IsDNS1123Label(RolePrimary))
	assert.Assert(t, nil == validation.IsDNS1123Label(RoleReplica))
}
