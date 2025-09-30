// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package require

import (
	"errors"
	"testing"

	"gotest.tools/v3/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StatusError returns the [metav1.Status] within err's tree.
// It calls t.Fatal when err is nil or there is no status.
func StatusError(t testing.TB, err error) metav1.Status {
	t.Helper()

	status, ok := err.(apierrors.APIStatus)
	assert.Assert(t, ok || errors.As(err, &status),
		"%T does not implement %T", err, status)

	return status.Status()
}

// StatusErrorDetails returns the details of [metav1.Status] within err's tree.
// It calls t.Fatal when err is nil, there is no status, or its Details field is nil.
func StatusErrorDetails(t testing.TB, err error) metav1.StatusDetails {
	t.Helper()

	status := StatusError(t, err)
	assert.Assert(t, status.Details != nil)
	return *status.Details
}

// Value returns v or panics when err is not nil.
func Value[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
