// Copyright 2023 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package standalone_pgadmin

import (
	"context"
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// apply sends an apply patch to object's endpoint in the Kubernetes API and
// updates object with any returned content. The fieldManager is set by
// r.Writer and the force parameter is true.
// - https://docs.k8s.io/reference/using-api/server-side-apply/#managers
// - https://docs.k8s.io/reference/using-api/server-side-apply/#conflicts
//
// TODO(tjmoore4): This function is duplicated from a version that takes a PostgresCluster object.
func (r *PGAdminReconciler) apply(ctx context.Context, object client.Object) error {
	// Generate an apply-patch by comparing the object to its zero value.
	zero := reflect.New(reflect.TypeOf(object).Elem()).Interface()
	data, err := client.MergeFrom(zero.(client.Object)).Data(object)
	apply := client.RawPatch(client.Apply.Type(), data)

	// Send the apply-patch with force=true.
	if err == nil {
		err = r.Writer.Patch(ctx, object, apply, client.ForceOwnership)
	}

	return err
}
