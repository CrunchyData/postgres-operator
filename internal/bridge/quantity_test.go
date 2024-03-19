/*
 Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
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

package bridge

import (
	"testing"

	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestFromCPU(t *testing.T) {
	zero := FromCPU(0)
	assert.Assert(t, zero.IsZero())
	assert.Equal(t, zero.String(), "0")

	one := FromCPU(1)
	assert.Equal(t, one.String(), "1")

	negative := FromCPU(-2)
	assert.Equal(t, negative.String(), "-2")
}

func TestFromGibibytes(t *testing.T) {
	zero := FromGibibytes(0)
	assert.Assert(t, zero.IsZero())
	assert.Equal(t, zero.String(), "0")

	one := FromGibibytes(1)
	assert.Equal(t, one.String(), "1Gi")

	negative := FromGibibytes(-2)
	assert.Equal(t, negative.String(), "-2Gi")
}

func TestToGibibytes(t *testing.T) {
	zero := resource.MustParse("0")
	assert.Equal(t, ToGibibytes(zero), int64(0))

	// Negative quantities become zero.
	negative := resource.MustParse("-4G")
	assert.Equal(t, ToGibibytes(negative), int64(0))

	// Decimal quantities round up.
	decimal := resource.MustParse("9000M")
	assert.Equal(t, ToGibibytes(decimal), int64(9))

	// Binary quantities round up.
	binary := resource.MustParse("8000Mi")
	assert.Equal(t, ToGibibytes(binary), int64(8))

	fourGi := resource.MustParse("4096Mi")
	assert.Equal(t, ToGibibytes(fourGi), int64(4))

	moreThanFourGi := resource.MustParse("4097Mi")
	assert.Equal(t, ToGibibytes(moreThanFourGi), int64(5))
}
