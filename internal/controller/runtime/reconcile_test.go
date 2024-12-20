// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"errors"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestRequests(t *testing.T) {
	none := Requests[client.Object]()
	assert.Assert(t, none != nil, "does not return nil slice")
	assert.DeepEqual(t, none, []reconcile.Request{})

	assert.Assert(t, cmp.Panics(func() {
		Requests[client.Object](nil)
	}), "expected nil pointer dereference")

	// Empty request when no metadata.
	assert.DeepEqual(t, Requests(new(corev1.Secret)), []reconcile.Request{{}})

	secret := new(corev1.Secret)
	secret.Namespace = "asdf"

	expected := reconcile.Request{}
	expected.Namespace = "asdf"
	assert.DeepEqual(t, Requests(secret), []reconcile.Request{expected})

	secret.Name = "123"
	expected.Name = "123"
	assert.DeepEqual(t, Requests(secret), []reconcile.Request{expected})
}

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
