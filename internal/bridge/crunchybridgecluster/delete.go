/*
 Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
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

package crunchybridgecluster

import (
	"context"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

const finalizer = "crunchybridgecluster.postgres-operator.crunchydata.com/finalizer"

// handleDelete sets a finalizer on cluster and performs the finalization of
// cluster when it is being deleted. It returns (nil, nil) when cluster is
// not being deleted and there are no errors patching the CrunchyBridgeCluster.
// The caller is responsible for returning other values to controller-runtime.
func (r *CrunchyBridgeClusterReconciler) handleDelete(
	ctx context.Context, crunchybridgecluster *v1beta1.CrunchyBridgeCluster, key string,
) (*ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// If the CrunchyBridgeCluster isn't being deleted, add the finalizer
	if crunchybridgecluster.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(crunchybridgecluster, finalizer) {
			controllerutil.AddFinalizer(crunchybridgecluster, finalizer)
			if err := r.Update(ctx, crunchybridgecluster); err != nil {
				return nil, err
			}
		}
		// If the CrunchyBridgeCluster is being deleted,
		// handle the deletion, and remove the finalizer
	} else {
		if controllerutil.ContainsFinalizer(crunchybridgecluster, finalizer) {
			log.Info("deleting cluster", "clusterName", crunchybridgecluster.Spec.ClusterName)

			// TODO(crunchybridgecluster): If is_protected is true, maybe skip this call, but allow the deletion of the K8s object?
			_, deletedAlready, err := r.NewClient().DeleteCluster(ctx, key, crunchybridgecluster.Status.ID)
			// Requeue if error
			if err != nil {
				return &ctrl.Result{}, err
			}

			if !deletedAlready {
				return &ctrl.Result{RequeueAfter: 1 * time.Second}, err
			}

			// Remove finalizer if deleted already
			if deletedAlready {
				log.Info("cluster deleted", "clusterName", crunchybridgecluster.Spec.ClusterName)

				controllerutil.RemoveFinalizer(crunchybridgecluster, finalizer)
				if err := r.Update(ctx, crunchybridgecluster); err != nil {
					return &ctrl.Result{}, err
				}
			}
		}
		// Stop reconciliation as the item is being deleted
		return &ctrl.Result{}, nil
	}

	return nil, nil
}
