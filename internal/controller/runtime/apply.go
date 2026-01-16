// Copyright 2021 - 2026 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Apply sends an apply patch with force=true using cc and updates object with any returned content.
// The client is responsible for setting fieldManager; see [client.WithFieldOwner].
//
// - https://docs.k8s.io/reference/using-api/server-side-apply#managers
// - https://docs.k8s.io/reference/using-api/server-side-apply#conflicts
func Apply[
	// NOTE: This interface can go away following https://go.dev/issue/47487.
	ClientPatch interface {
		Patch(context.Context, client.Object, client.Patch, ...client.PatchOption) error
	},
	T interface{ client.Object },
](ctx context.Context, cc ClientPatch, object T) error {
	// Generate an apply-patch by comparing the object to its zero value.
	data, err := client.MergeFrom(*new(T)).Data(object)
	apply := client.RawPatch(client.Apply.Type(), data)

	// Keep a copy of the object before any API calls.
	intent := object.DeepCopyObject()

	// Send the apply-patch with force=true.
	if err == nil {
		err = cc.Patch(ctx, object, apply, client.ForceOwnership)
	}

	// Some fields cannot be server-side applied correctly.
	// When their outcome does not match the intent, send a json-patch to get really specific.
	patch := NewJSONPatch()

	switch actual := any(object).(type) {
	case *corev1.Service:
		intent := intent.(*corev1.Service)

		// Service.Spec.Selector cannot be unset; perhaps https://issue.k8s.io/117447
		if !equality.Semantic.DeepEqual(actual.Spec.Selector, intent.Spec.Selector) {
			patch.Replace("spec", "selector")(intent.Spec.Selector)
		}
	}

	// Send the json-patch when necessary.
	if err == nil && !patch.IsEmpty() {
		err = cc.Patch(ctx, object, patch)
	}
	return err
}
