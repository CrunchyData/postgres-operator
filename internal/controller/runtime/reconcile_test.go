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

package runtime

import (
	"errors"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestErrorWithBackoff(t *testing.T) {
	result, err := ErrorWithBackoff(nil)
	assert.Assert(t, result.IsZero())
	assert.NilError(t, err)

	expected := errors.New("doot")
	result, err = ErrorWithBackoff(expected)
	assert.Assert(t, result.IsZero())
	assert.Equal(t, err, expected)
}

func TestErrorWithoutBackoff(t *testing.T) {
	result, err := ErrorWithoutBackoff(nil)
	assert.Assert(t, result.IsZero())
	assert.NilError(t, err)

	expected := errors.New("doot")
	result, err = ErrorWithoutBackoff(expected)
	assert.Assert(t, result.IsZero())
	assert.Assert(t, errors.Is(err, reconcile.TerminalError(nil)))
	assert.Equal(t, errors.Unwrap(err), expected)
}

func TestRequeueWithBackoff(t *testing.T) {
	result := RequeueWithBackoff()
	assert.Assert(t, result.Requeue)
	assert.Assert(t, result.RequeueAfter == 0)
}

func TestRequeueWithoutBackoff(t *testing.T) {
	result := RequeueWithoutBackoff(0)
	assert.Assert(t, result.Requeue)
	assert.Assert(t, result.RequeueAfter > 0)

	result = RequeueWithoutBackoff(-1)
	assert.Assert(t, result.Requeue)
	assert.Assert(t, result.RequeueAfter > 0)

	result = RequeueWithoutBackoff(time.Minute)
	assert.Assert(t, result.Requeue)
	assert.Equal(t, result.RequeueAfter, time.Minute)
}
