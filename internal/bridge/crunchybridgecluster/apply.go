// Copyright 2021 - 2023 Crunchy Data Solutions, Inc.
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

package crunchybridgecluster

import (
	"context"
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// patch sends patch to object's endpoint in the Kubernetes API and updates
// object with any returned content. The fieldManager is set to r.Owner, but
// can be overridden in options.
// - https://docs.k8s.io/reference/using-api/server-side-apply/#managers
//
// NOTE: This function is duplicated from a version in the postgrescluster package
func (r *CrunchyBridgeClusterReconciler) patch(
	ctx context.Context, object client.Object,
	patch client.Patch, options ...client.PatchOption,
) error {
	options = append([]client.PatchOption{r.Owner}, options...)
	return r.Client.Patch(ctx, object, patch, options...)
}

// apply sends an apply patch to object's endpoint in the Kubernetes API and
// updates object with any returned content. The fieldManager is set to
// r.Owner and the force parameter is true.
// - https://docs.k8s.io/reference/using-api/server-side-apply/#managers
// - https://docs.k8s.io/reference/using-api/server-side-apply/#conflicts
//
// NOTE: This function is duplicated from a version in the postgrescluster package
func (r *CrunchyBridgeClusterReconciler) apply(ctx context.Context, object client.Object) error {
	// Generate an apply-patch by comparing the object to its zero value.
	zero := reflect.New(reflect.TypeOf(object).Elem()).Interface()
	data, err := client.MergeFrom(zero.(client.Object)).Data(object)
	apply := client.RawPatch(client.Apply.Type(), data)

	// Send the apply-patch with force=true.
	if err == nil {
		err = r.patch(ctx, object, apply, client.ForceOwnership)
	}

	return err
}
