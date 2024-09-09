// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

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
