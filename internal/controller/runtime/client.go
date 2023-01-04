/*
 Copyright 2021 - 2023 Crunchy Data Solutions, Inc.
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
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Types that implement single methods of the [client.Reader] interface.
type (
	// NOTE: The signature of [client.Client.Get] changes in [sigs.k8s.io/controller-runtime@v0.13.0].
	// - https://github.com/kubernetes-sigs/controller-runtime/releases/tag/v0.13.0

	ClientGet  func(context.Context, client.ObjectKey, client.Object) error
	ClientList func(context.Context, client.ObjectList, ...client.ListOption) error
)

// ClientReader implements [client.Reader] by composing assignable functions.
type ClientReader struct {
	ClientGet
	ClientList
}

var _ client.Reader = ClientReader{}

// Types that implement single methods of the [client.Writer] interface.
type (
	ClientCreate    func(context.Context, client.Object, ...client.CreateOption) error
	ClientDelete    func(context.Context, client.Object, ...client.DeleteOption) error
	ClientPatch     func(context.Context, client.Object, client.Patch, ...client.PatchOption) error
	ClientDeleteAll func(context.Context, client.Object, ...client.DeleteAllOfOption) error
	ClientUpdate    func(context.Context, client.Object, ...client.UpdateOption) error
)

// ClientWriter implements [client.Writer] by composing assignable functions.
type ClientWriter struct {
	ClientCreate
	ClientDelete
	ClientDeleteAll
	ClientPatch
	ClientUpdate
}

var _ client.Writer = ClientWriter{}

// NOTE: The following implementations can go away following https://go.dev/issue/47487.
// The function types above would become single-method interfaces.

func (fn ClientCreate) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	return fn(ctx, obj, opts...)
}

func (fn ClientDelete) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	return fn(ctx, obj, opts...)
}

func (fn ClientDeleteAll) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return fn(ctx, obj, opts...)
}

func (fn ClientGet) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	return fn(ctx, key, obj)
}

func (fn ClientList) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return fn(ctx, list, opts...)
}

func (fn ClientPatch) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return fn(ctx, obj, patch, opts...)
}

func (fn ClientUpdate) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return fn(ctx, obj, opts...)
}
