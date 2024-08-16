// Copyright 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
)

func TestAPISet(t *testing.T) {
	t.Parallel()

	var zero APISet
	assert.Assert(t, !zero.Has(API{Group: "security.openshift.io"}))
	assert.Assert(t, !zero.Has(API{Group: "security.openshift.io", Kind: "SecurityContextConstraints"}))
	assert.Assert(t, !zero.HasAll(API{Group: "security.openshift.io"}, API{Group: "snapshot.storage.k8s.io"}))
	assert.Assert(t, !zero.HasAny(API{Group: "security.openshift.io"}, API{Group: "snapshot.storage.k8s.io"}))

	empty := NewAPISet()
	assert.Assert(t, !empty.Has(API{Group: "security.openshift.io"}))
	assert.Assert(t, !empty.Has(API{Group: "security.openshift.io", Kind: "SecurityContextConstraints"}))

	one := NewAPISet(
		API{Group: "security.openshift.io", Kind: "SecurityContextConstraints"},
	)
	assert.Assert(t, one.Has(API{Group: "security.openshift.io"}))
	assert.Assert(t, one.Has(API{Group: "security.openshift.io", Kind: "SecurityContextConstraints"}))
	assert.Assert(t, !one.HasAll(API{Group: "snapshot.storage.k8s.io"}, API{Group: "security.openshift.io"}))
	assert.Assert(t, !one.HasAny(API{Group: "snapshot.storage.k8s.io"}))
	assert.Assert(t, one.HasAny(API{Group: "snapshot.storage.k8s.io"}, API{Group: "security.openshift.io"}))

	two := NewAPISet(
		API{Group: "security.openshift.io", Kind: "SecurityContextConstraints"},
		API{Group: "snapshot.storage.k8s.io", Kind: "VolumeSnapshot"},
	)
	assert.Assert(t, two.Has(API{Group: "security.openshift.io"}))
	assert.Assert(t, two.Has(API{Group: "snapshot.storage.k8s.io"}))
	assert.Assert(t, two.HasAll(API{Group: "snapshot.storage.k8s.io"}, API{Group: "security.openshift.io"}))
	assert.Assert(t, two.HasAny(API{Group: "snapshot.storage.k8s.io"}))
	assert.Assert(t, two.HasAny(API{Group: "snapshot.storage.k8s.io"}, API{Group: "security.openshift.io"}))
}

func TestAPIContext(t *testing.T) {
	t.Parallel()

	// The background context always return false.
	ctx := context.Background()

	assert.Assert(t, !Has(ctx, API{Group: "security.openshift.io"}))
	assert.Assert(t, !Has(ctx, API{Group: "snapshot.storage.k8s.io"}))

	// An initialized context returns what is stored.
	set := NewAPISet(API{Group: "security.openshift.io", Kind: "SecurityContextConstraints"})
	ctx = NewAPIContext(ctx, set)

	assert.Assert(t, Has(ctx, API{Group: "security.openshift.io"}))
	assert.Assert(t, !Has(ctx, API{Group: "snapshot.storage.k8s.io"}))

	// The stored value is mutable within the context.
	set[API{Group: "snapshot.storage.k8s.io"}] = struct{}{}
	assert.Assert(t, Has(ctx, API{Group: "snapshot.storage.k8s.io"}))
}
