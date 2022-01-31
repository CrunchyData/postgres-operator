/*
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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
	"encoding/json"
	"fmt"
	"reflect"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
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
		// Changing Service.Spec.Type requires a special apply-patch sometimes.
		if err != nil {
			err = r.handleServiceError(ctx, object.(*corev1.Service), data, err)
		}

		applyServiceSpec(patch, actual.Spec, intent.(*corev1.Service).Spec, "spec")
	}

	// Send the json-patch when necessary.
	if err == nil && !patch.IsEmpty() {
		err = r.patch(ctx, object, patch)
	}
	return err
}

// handleServiceError inspects err for expected Kubernetes API responses to
// writing a Service. It returns err when it cannot resolve the issue, otherwise
// it returns nil.
func (r *Reconciler) handleServiceError(
	ctx context.Context, service *corev1.Service, apply []byte, err error,
) error {
	var status metav1.Status
	if api := apierrors.APIStatus(nil); errors.As(err, &api) {
		status = api.Status()
	}

	// Service.Spec.Ports.NodePort must be cleared for ClusterIP prior to
	// Kubernetes 1.20. When all the errors are about disallowed "nodePort",
	// run a json-patch on the apply-patch to set them all to null.
	// - https://issue.k8s.io/33766
	if service.Spec.Type == corev1.ServiceTypeClusterIP {
		add := json.RawMessage(`"add"`)
		null := json.RawMessage(`null`)
		patch := make(jsonpatch.Patch, 0, len(service.Spec.Ports))

		if apierrors.IsInvalid(err) && status.Details != nil {
			for i, cause := range status.Details.Causes {
				path := json.RawMessage(fmt.Sprintf(`"/spec/ports/%d/nodePort"`, i))

				if cause.Type == metav1.CauseType(field.ErrorTypeForbidden) &&
					cause.Field == fmt.Sprintf("spec.ports[%d].nodePort", i) {
					patch = append(patch,
						jsonpatch.Operation{"op": &add, "value": &null, "path": &path})
				}
			}
		}

		// Amend the apply-patch when all the errors can be fixed.
		if len(patch) == len(service.Spec.Ports) {
			apply, err = patch.Apply(apply)
		}

		// Send the apply-patch with force=true.
		if err == nil {
			patch := client.RawPatch(client.Apply.Type(), apply)
			err = r.patch(ctx, service, patch, client.ForceOwnership)
		}
	}

	return err
}

// applyPodSecurityContext is called by Reconciler.apply to work around issues
// with server-side apply.
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
	// Empty "omitempty" slices are ignored until Kubernetes 1.19.
	// - https://issue.k8s.io/89273
	if !equality.Semantic.DeepEqual(actual.SupplementalGroups, intent.SupplementalGroups) {
		patch.Replace(append(path, "supplementalGroups")...)(intent.SupplementalGroups)
	}
}

// applyPodTemplateSpec is called by Reconciler.apply to work around issues
// with server-side apply.
func applyPodTemplateSpec(
	patch *kubeapi.JSON6902, actual, intent corev1.PodTemplateSpec, path ...string,
) {
	applyPodSecurityContext(patch,
		actual.Spec.SecurityContext,
		intent.Spec.SecurityContext,
		append(path, "spec", "securityContext")...)
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
