// Copyright 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
)

// API is a combination of Group, Version, and Kind that can be used to check
// what is available in the Kubernetes API. There are four ways to populate it:
//  1. Group without Version nor Kind means any resource in that Group.
//  2. Group with Version but no Kind means any resource in that GV.
//  3. Group with Kind but no Version means that Kind in any Version of the Group.
//  4. Group with Version and Kind means that exact GVK.
type API = schema.GroupVersionKind

type APIs interface {
	Has(API) bool
	HasAll(...API) bool
	HasAny(...API) bool
}

// APISet implements [APIs] using empty struct for minimal memory consumption.
type APISet = sets.Set[API]

func NewAPISet(api ...API) APISet {
	// Start with everything that's passed in; full GVKs are here.
	s := sets.New(api...)

	// Add the other combinations; Group, GV, and GK.
	for i := range api {
		s.Insert(
			API{Group: api[i].Group},
			API{Group: api[i].Group, Version: api[i].Version},
			API{Group: api[i].Group, Kind: api[i].Kind},
		)
	}

	return s
}

type apiContextKey struct{}

// Has returns true when api was previously stored by [NewAPIContext].
func Has(ctx context.Context, api API) bool {
	if i, ok := ctx.Value(apiContextKey{}).(interface{ Has(API) bool }); ok {
		return i.Has(api)
	}
	return false
}

// NewAPIContext returns a copy of ctx containing apis. Interrogate it using [Has].
func NewAPIContext(ctx context.Context, apis APIs) context.Context {
	return context.WithValue(ctx, apiContextKey{}, apis)
}
