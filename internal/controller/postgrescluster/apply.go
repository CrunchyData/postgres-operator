/*
 Copyright 2021 Crunchy Data Solutions, Inc.
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

package postgrescluster

import (
	"context"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
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
	case *appsv1.StatefulSet:
		applyPodTemplateSpec(patch,
			actual.Spec.Template, intent.(*appsv1.StatefulSet).Spec.Template,
			"spec", "template")

	case *batchv1.Job:
		applyPodTemplateSpec(patch,
			actual.Spec.Template, intent.(*batchv1.Job).Spec.Template,
			"spec", "template")

	case *corev1.Service:
		// Service.Spec.Selector is not +mapType=atomic until Kubernetes 1.22.
		// - https://issue.k8s.io/97970
		selector := intent.(*corev1.Service).Spec.Selector
		if !equality.Semantic.DeepEqual(actual.Spec.Selector, selector) {
			patch.Replace("spec", "selector")(selector)
		}
	}

	// Send the json-patch when necessary.
	if err == nil && !patch.IsEmpty() {
		err = r.patch(ctx, object, patch)
	}
	return err
}

// applyPodSecurityContext is called by apply to work around issues with server-side apply.
func applyPodSecurityContext(
	patch *kubeapi.JSON6902, actual, intent *corev1.PodSecurityContext, path ...string,
) {
	if intent == nil {
		// This won't happen because we populate all PodSecurityContext.
		return
	}
	if actual == nil {
		patch.Replace(path...)(intent)
		return
	}
	if !equality.Semantic.DeepEqual(actual.SupplementalGroups, intent.SupplementalGroups) {
		patch.Replace(append(path, "supplementalGroups")...)(intent.SupplementalGroups)
	}
}

// applyPodTemplateSpec is called by apply to work around issues with server-side apply.
func applyPodTemplateSpec(
	patch *kubeapi.JSON6902, actual, intent corev1.PodTemplateSpec, path ...string,
) {
	applyPodSecurityContext(patch,
		actual.Spec.SecurityContext,
		intent.Spec.SecurityContext,
		append(path, "spec", "securityContext")...)
}
