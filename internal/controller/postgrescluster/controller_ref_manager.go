package postgrescluster

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

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
)

// adoptStatefulSet adopts the provided StatefulSet by adding controller owner refs for the
// provided PostgresCluster.
func (r *Reconciler) adoptStatefulSet(ctx context.Context, postgresCluster *v1alpha1.PostgresCluster,
	statefulset *appsv1.StatefulSet) error {

	sts := statefulset.DeepCopy()
	if err := controllerutil.SetControllerReference(postgresCluster, sts,
		r.Client.Scheme()); err != nil {
		return err
	}

	patchBytes, err := kubeapi.NewMergePatch().
		Add("metadata", "ownerReferences")(sts.ObjectMeta.OwnerReferences).Bytes()
	if err != nil {
		return err
	}

	return r.Client.Patch(ctx, statefulset, client.RawPatch(types.StrategicMergePatchType,
		patchBytes), &client.PatchOptions{
		FieldManager: ControllerName,
	})
}

// claimStatefulSet is responsible for adopting or release StatefulSets based on their current
// controller ownership and whether or not they meet the provided labeling requirements.
// This solution is modeled after the ControllerRefManager logic as found within the controller
// package in the Kubernetes source:
// https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/controller_ref_manager.go
//
// TODO do a non-cache based get of the PostgresCluster prior to adopting anything to prevent
// race conditions with the garbage collector (see
// https://github.com/kubernetes/kubernetes/issues/42639)
func (r *Reconciler) claimStatefulSet(ctx context.Context, postgresCluster *v1alpha1.PostgresCluster,
	statefulset *appsv1.StatefulSet) error {

	controllerRef := metav1.GetControllerOfNoCopy(statefulset)
	if controllerRef != nil {
		// if not owned by this postgrescluster then ignore
		if controllerRef.UID != postgresCluster.GetUID() {
			return nil
		}

		// If owned by this PostgresCluster and the proper PostgresCluster label is present then
		// return true.  Additional labels checks can be added here as needed to determine whether
		// or not a StatefulSet is part of a PostgreSQL cluster and should be adopted or released.
		if _, ok := statefulset.GetLabels()[naming.LabelCluster]; ok {
			return nil
		}

		// If owned but selector doesn't match, then attempt to release.  However, if the
		// PostgresCluster is being deleted then simply return.
		if postgresCluster.GetDeletionTimestamp() != nil {
			return nil
		}

		if err := r.releaseStatefulSet(ctx, postgresCluster,
			statefulset); client.IgnoreNotFound(err) != nil {
			return err
		}

		// successfully released resource or resource no longer exists
		return nil
	}

	// At this point the resource has no controller ref and is therefore an orphan.  Ignore if
	// either the PostgresCluster resource or the ophaned resource is being deleted, or if the selector
	// for the orphaned resource doesn't doesn't include the proper PostgresCluster label
	_, hasPGClusterLabel := statefulset.GetLabels()[naming.LabelCluster]
	if postgresCluster.GetDeletionTimestamp() != nil || !hasPGClusterLabel {
		return nil
	}
	if statefulset.GetDeletionTimestamp() != nil {
		return nil
	}
	if err := r.adoptStatefulSet(ctx, postgresCluster,
		statefulset); err != nil {
		// If adopt attempt failed because the resource no longer exists, then simply
		// ignore.  Otherwise return the error.
		if kerr.IsNotFound(err) {
			return nil
		}
		return err
	}

	// successfully adopted resource
	return nil
}

// getPostgresClusterForStatefulSet is responsible for obtaining the PostgresCluster associated
// with a StatefulSet.
func (r *Reconciler) getPostgresClusterForStatefulSet(ctx context.Context,
	statefulSet *appsv1.StatefulSet) (bool, *v1alpha1.PostgresCluster, error) {

	clusterName := ""

	// first see if it has a PostgresCluster ownership ref or a PostgresCluster label
	controllerRef := metav1.GetControllerOfNoCopy(statefulSet)
	if controllerRef != nil && controllerRef.Kind == "PostgresCluster" {
		clusterName = controllerRef.Name
	} else if _, ok := statefulSet.GetLabels()[naming.LabelCluster]; ok {
		clusterName = statefulSet.GetLabels()[naming.LabelCluster]
	}

	if clusterName == "" {
		return false, nil, nil
	}

	postgresCluster := &v1alpha1.PostgresCluster{}
	if err := r.Client.Get(ctx, types.NamespacedName{
		Name:      clusterName,
		Namespace: statefulSet.GetNamespace(),
	}, postgresCluster); err != nil {
		if kerr.IsNotFound(err) {
			return false, nil, nil
		}
		return false, nil, err
	}

	return true, postgresCluster, nil
}

// manageSTSControllerRefs is responsible for determining whether or not an attempt should be made
// to adopt or release/orphan a StatefulSet.  This includes obtaining the PostgresCluster for
// the StatefulSet and then calling the logic needed to either adopt or release it.
func (r *Reconciler) manageSTSControllerRefs(ctx context.Context, sts *appsv1.StatefulSet) error {

	found, postgresCluster, err := r.getPostgresClusterForStatefulSet(ctx, sts)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	return r.claimStatefulSet(ctx, postgresCluster, sts)
}

// releaseStatefulSet releases the provided stateful set from ownership by the provided
// PostgresCluster.  This is done by removing the PostgresCluster's controller owner
// refs from the StatefulSet.
func (r *Reconciler) releaseStatefulSet(ctx context.Context,
	postgresCluster *v1alpha1.PostgresCluster, statefulset *appsv1.StatefulSet) error {

	// TODO create a strategic merge type in kubeapi instead of using Merge7386
	patch, err := kubeapi.NewMergePatch().
		Add("metadata", "ownerReferences")([]map[string]string{{
		"$patch": "delete",
		"uid":    string(postgresCluster.GetUID()),
	}}).Bytes()
	if err != nil {
		return err
	}

	return r.Client.Patch(ctx, statefulset, client.RawPatch(types.StrategicMergePatchType, patch))
}

// statefulSetControllerRefHandlerFuncs returns the handler funcs that should be utilized to watch
// StatefulSets within the cluster as needed to manage controller ownership refs.
func (r *Reconciler) statefulSetControllerRefHandlerFuncs() *handler.Funcs {

	ctx := context.Background()
	log := logging.FromContext(ctx)
	errMsg := "managing StatefulSet controller refs"

	return &handler.Funcs{
		CreateFunc: func(updateEvent event.CreateEvent, workQueue workqueue.RateLimitingInterface) {
			if err := r.manageSTSControllerRefs(ctx,
				updateEvent.Object.(*appsv1.StatefulSet)); err != nil {
				log.Error(err, errMsg)
			}
		},
		UpdateFunc: func(updateEvent event.UpdateEvent, workQueue workqueue.RateLimitingInterface) {
			if err := r.manageSTSControllerRefs(ctx,
				updateEvent.ObjectNew.(*appsv1.StatefulSet)); err != nil {
				log.Error(err, errMsg)
			}
		},
		DeleteFunc: func(updateEvent event.DeleteEvent, workQueue workqueue.RateLimitingInterface) {
			if err := r.manageSTSControllerRefs(ctx,
				updateEvent.Object.(*appsv1.StatefulSet)); err != nil {
				log.Error(err, errMsg)
			}
		},
	}
}
