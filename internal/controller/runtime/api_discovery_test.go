// Copyright 2023 - 2024 Crunchy Data Solutions, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runtime

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
	assert.Assert(t, !zero.HasOne(API{Group: "security.openshift.io"}, API{Group: "snapshot.storage.k8s.io"}))

	empty := NewAPISet()
	assert.Assert(t, !empty.Has(API{Group: "security.openshift.io"}))
	assert.Assert(t, !empty.Has(API{Group: "security.openshift.io", Kind: "SecurityContextConstraints"}))

	one := NewAPISet(
		API{Group: "security.openshift.io", Kind: "SecurityContextConstraints"},
	)
	assert.Assert(t, one.Has(API{Group: "security.openshift.io"}))
	assert.Assert(t, one.Has(API{Group: "security.openshift.io", Kind: "SecurityContextConstraints"}))
	assert.Assert(t, !one.HasAll(API{Group: "snapshot.storage.k8s.io"}, API{Group: "security.openshift.io"}))
	assert.Assert(t, !one.HasOne(API{Group: "snapshot.storage.k8s.io"}))
	assert.Assert(t, one.HasOne(API{Group: "snapshot.storage.k8s.io"}, API{Group: "security.openshift.io"}))

	two := NewAPISet(
		API{Group: "security.openshift.io", Kind: "SecurityContextConstraints"},
		API{Group: "snapshot.storage.k8s.io", Kind: "VolumeSnapshot"},
	)
	assert.Assert(t, two.Has(API{Group: "security.openshift.io"}))
	assert.Assert(t, two.Has(API{Group: "snapshot.storage.k8s.io"}))
	assert.Assert(t, two.HasAll(API{Group: "snapshot.storage.k8s.io"}, API{Group: "security.openshift.io"}))
	assert.Assert(t, two.HasOne(API{Group: "snapshot.storage.k8s.io"}))
	assert.Assert(t, two.HasOne(API{Group: "snapshot.storage.k8s.io"}, API{Group: "security.openshift.io"}))
}

func TestAPIContext(t *testing.T) {
	t.Parallel()

	// The background context always return false.
	ctx := context.Background()

	assert.Assert(t, !Kubernetes(ctx).Has(API{Group: "security.openshift.io"}))
	assert.Assert(t, !Kubernetes(ctx).Has(API{Group: "snapshot.storage.k8s.io"}))

	// An initialized context returns what is stored.
	set := NewAPISet(API{Group: "security.openshift.io", Kind: "SecurityContextConstraints"})
	ctx = NewAPIContext(ctx, set)

	assert.Assert(t, Kubernetes(ctx).Has(API{Group: "security.openshift.io"}))
	assert.Assert(t, !Kubernetes(ctx).Has(API{Group: "snapshot.storage.k8s.io"}))

	// The stored value is mutable within the context.
	set[API{Group: "snapshot.storage.k8s.io"}] = struct{}{}
	assert.Assert(t, Kubernetes(ctx).Has(API{Group: "snapshot.storage.k8s.io"}))
}
