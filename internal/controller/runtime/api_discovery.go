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

	"k8s.io/apimachinery/pkg/runtime/schema"
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
	HasOne(...API) bool
}

// APISet implements [APIs] using empty struct for minimal memory consumption.
type APISet map[API]struct{}

func NewAPISet(api ...API) APISet {
	s := make(APISet)

	for i := range api {
		s[api[i]] = struct{}{}
		s[API{Group: api[i].Group}] = struct{}{}
		s[API{Group: api[i].Group, Version: api[i].Version}] = struct{}{}
		s[API{Group: api[i].Group, Kind: api[i].Kind}] = struct{}{}
	}

	return s
}

// Has returns true when api is available in s.
func (s APISet) Has(api API) bool { return s.HasOne(api) }

// HasAll returns true when every api is available in s.
func (s APISet) HasAll(api ...API) bool {
	for i := range api {
		if _, present := s[api[i]]; !present {
			return false
		}
	}
	return true
}

// HasOne returns true when at least one api is available in s.
func (s APISet) HasOne(api ...API) bool {
	for i := range api {
		if _, present := s[api[i]]; present {
			return true
		}
	}
	return false
}

type apiContextKey struct{}

// Kubernetes returns the APIs previously stored by [NewAPIContext].
// When nothing was stored, it returns an empty [APISet].
func Kubernetes(ctx context.Context) APIs {
	if apis, ok := ctx.Value(apiContextKey{}).(APIs); ok {
		return apis
	}
	return APISet{}
}

// NewAPIContext returns a copy of ctx containing apis. Retrieve it using [Kubernetes].
func NewAPIContext(ctx context.Context, apis APIs) context.Context {
	return context.WithValue(ctx, apiContextKey{}, apis)
}
