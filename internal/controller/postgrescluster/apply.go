// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/kubeapi"
)

// apply sends an apply patch to object's endpoint in the Kubernetes API and
// updates object with any returned content. The fieldManager is set to
// r.Owner and the force parameter is true.
// - https://docs.k8s.io/reference/using-api/server-side-apply/#managers
// - https://docs.k8s.io/reference/using-api/server-side-apply/#conflicts
func (r *Reconciler) apply(ctx context.Context, object client.Object) error {
	// Generate an apply-patch by comparing the object to its zero value.
	zero := reflect.New(reflect.TypeOf(object).Elem()).Interface()
	data, err := client.MergeFrom(zero.(client.Object)).Data(object)
	apply := client.RawPatch(client.Apply.Type(), data)

	// Keep a copy of the object before any API calls.
	intent := object.DeepCopyObject()
	patch := kubeapi.NewJSONPatch()

	// Send the apply-patch with force=true.
	if err == nil {
		err = r.patch(ctx, object, apply, client.ForceOwnership)
	}

	// Some fields cannot be server-side applied correctly. When their outcome
	// does not match the intent, send a json-patch to get really specific.
	switch actual := object.(type) {
	case *corev1.Service:
		applyServiceSpec(patch, actual.Spec, intent.(*corev1.Service).Spec, "spec")
	}

	// Send the json-patch when necessary.
	if err == nil && !patch.IsEmpty() {
		err = r.patch(ctx, object, patch)
	}
	return err
}

// applyServiceSpec is called by Reconciler.apply to work around issues
// with server-side apply.
func applyServiceSpec(
	patch *kubeapi.JSON6902, actual, intent corev1.ServiceSpec, path ...string,
) {
	// Service.Spec.Selector is not +mapType=atomic until Kubernetes 1.22.
	// - https://issue.k8s.io/97970
	if !equality.Semantic.DeepEqual(actual.Selector, intent.Selector) {
		patch.Replace(append(path, "selector")...)(intent.Selector)
	}
}
