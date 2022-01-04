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

package initialize_test

import (
	"testing"

	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/crunchydata/postgres-operator/internal/initialize"
)

func TestIntOrStringInt32(t *testing.T) {
	// Same content as the upstream constructor.
	upstream := intstr.FromInt(42)
	n := initialize.IntOrStringInt32(42)

	assert.DeepEqual(t, &upstream, n)
}

func TestIntOrStringString(t *testing.T) {
	upstream := intstr.FromString("50%")
	s := initialize.IntOrStringString("50%")

	assert.DeepEqual(t, &upstream, s)
}
func TestIntOrString(t *testing.T) {
	upstream := intstr.FromInt(0)

	ios := initialize.IntOrString(intstr.FromInt(0))
	assert.DeepEqual(t, *ios, upstream)
}
