// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package crunchybridgecluster

import (
	"context"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/initialize"
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
				return initialize.Pointer(runtime.RequeueWithoutBackoff(time.Second)), err
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
