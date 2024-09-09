// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Types that implement single methods of the [client.Reader] interface.
type (
	ClientGet  func(context.Context, client.ObjectKey, client.Object, ...client.GetOption) error
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

func (fn ClientGet) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	return fn(ctx, key, obj, opts...)
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
