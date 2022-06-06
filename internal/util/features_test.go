/*
 Copyright 2017 - 2022 Crunchy Data Solutions, Inc.
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

package util

import (
	"testing"

	"gotest.tools/v3/assert"
	"k8s.io/component-base/featuregate"
)

func TestAddAndSetFeatureGates(t *testing.T) {

	// set test features
	const TestGate1 featuregate.Feature = "TestGate1"
	const TestGate2 featuregate.Feature = "TestGate2"
	const TestGate3 featuregate.Feature = "TestGate3"

	pgoFeatures = map[featuregate.Feature]featuregate.FeatureSpec{
		TestGate1: {Default: false, PreRelease: featuregate.Beta},
		TestGate2: {Default: false, PreRelease: featuregate.Beta},
		TestGate3: {Default: false, PreRelease: featuregate.Beta},
	}

	t.Run("No feature gates set", func(t *testing.T) {
		err := AddAndSetFeatureGates("")
		assert.NilError(t, err)
	})

	t.Run("One feature gate set", func(t *testing.T) {
		err := AddAndSetFeatureGates("TestGate1=true")
		assert.NilError(t, err)
	})

	t.Run("Two feature gates set", func(t *testing.T) {
		err := AddAndSetFeatureGates("TestGate1=true,TestGate3=true")
		assert.NilError(t, err)
	})

	t.Run("All available feature gates set", func(t *testing.T) {
		err := AddAndSetFeatureGates("TestGate1=true,TestGate2=true,TestGate3=true")
		assert.NilError(t, err)
	})

	t.Run("One unrecognized gate set", func(t *testing.T) {
		err := AddAndSetFeatureGates("NotAGate=true")
		assert.ErrorContains(t, err, "unrecognized feature gate: NotAGate")
	})

	t.Run("One recognized gate, one unrecognized gate", func(t *testing.T) {
		err := AddAndSetFeatureGates("TestGate1=true,NotAGate=true")
		assert.ErrorContains(t, err, "unrecognized feature gate: NotAGate")
	})

	t.Run("Gate value not set", func(t *testing.T) {
		err := AddAndSetFeatureGates("GateNotSet")
		assert.ErrorContains(t, err, "missing bool value for GateNotSet")
	})

	t.Run("Gate value not boolean", func(t *testing.T) {
		err := AddAndSetFeatureGates("GateNotSet=foo")
		assert.ErrorContains(t, err, "invalid value of GateNotSet=foo, err: strconv.ParseBool")
	})
}
