// Copyright 2023 - 2024 Crunchy Data Solutions, Inc.
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

package standalone_pgadmin

import (
	"context"
	"io"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/source"

	controllerruntime "github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// PGAdminReconciler reconciles a PGAdmin object
type PGAdminReconciler struct {
	client.Client
	Owner   client.FieldOwner
	PodExec func(
		namespace, pod, container string,
		stdin io.Reader, stdout, stderr io.Writer, command ...string,
	) error
	Recorder    record.EventRecorder
	IsOpenShift bool
}

//+kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="postgresclusters",verbs={list,watch}
//+kubebuilder:rbac:groups="",resources="persistentvolumeclaims",verbs={list,watch}
//+kubebuilder:rbac:groups="",resources="secrets",verbs={list,watch}
//+kubebuilder:rbac:groups="",resources="configmaps",verbs={list,watch}
//+kubebuilder:rbac:groups="apps",resources="statefulsets",verbs={list,watch}

// SetupWithManager sets up the controller with the Manager.
//
// TODO(tjmoore4): This function is duplicated from a version that takes a PostgresCluster object.
func (r *PGAdminReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.PodExec == nil {
		var err error
		r.PodExec, err = controllerruntime.NewPodExecutor(mgr.GetConfig())
		if err != nil {
			return err
		}
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.PGAdmin{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.Secret{}).
		Owns(&appsv1.StatefulSet{}).
		Watches(
			&source.Kind{Type: v1beta1.NewPostgresCluster()},
			r.watchPostgresClusters(),
		).
		Watches(
			&source.Kind{Type: &corev1.Secret{}},
			r.watchForRelatedSecret(),
		).
		Complete(r)
}

//+kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="pgadmins",verbs={get}
//+kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="pgadmins/status",verbs={patch}

// Reconcile which aims to move the current state of the pgAdmin closer to the
// desired state described in a [v1beta1.PGAdmin] identified by request.
func (r *PGAdminReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	var err error
	log := logging.FromContext(ctx)

	pgAdmin := &v1beta1.PGAdmin{}
	if err := r.Get(ctx, req.NamespacedName, pgAdmin); err != nil {
		// NotFound cannot be fixed by requeuing so ignore it. During background
		// deletion, we receive delete events from pgadmin's dependents after
		// pgadmin is deleted.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Write any changes to the pgadmin status on the way out.
	before := pgAdmin.DeepCopy()
	defer func() {
		if !equality.Semantic.DeepEqual(before.Status, pgAdmin.Status) {
			statusErr := r.Status().Patch(ctx, pgAdmin, client.MergeFrom(before), r.Owner)
			if statusErr != nil {
				log.Error(statusErr, "Patching PGAdmin status")
			}
			if err == nil {
				err = statusErr
			}
		}
	}()

	log.Info("Reconciling pgAdmin")

	// Set defaults if unset
	pgAdmin.Default()

	var (
		configmap  *corev1.ConfigMap
		dataVolume *corev1.PersistentVolumeClaim
		clusters   map[string]*v1beta1.PostgresClusterList
	)

	_, err = r.reconcilePGAdminSecret(ctx, pgAdmin)

	if err == nil {
		clusters, err = r.getClustersForPGAdmin(ctx, pgAdmin)
	}
	if err == nil {
		configmap, err = r.reconcilePGAdminConfigMap(ctx, pgAdmin, clusters)
	}
	if err == nil {
		dataVolume, err = r.reconcilePGAdminDataVolume(ctx, pgAdmin)
	}
	if err == nil {
		err = r.reconcilePGAdminStatefulSet(ctx, pgAdmin, configmap, dataVolume)
	}
	if err == nil {
		err = r.reconcilePGAdminUsers(ctx, pgAdmin)
	}

	if err == nil {
		// at this point everything reconciled successfully, and we can update the
		// observedGeneration
		pgAdmin.Status.ObservedGeneration = pgAdmin.GetGeneration()
		log.V(1).Info("reconciled pgadmin")
	}

	return ctrl.Result{}, err
}

// The owner reference created by controllerutil.SetControllerReference blocks
// deletion. The OwnerReferencesPermissionEnforcement plugin requires that the
// creator of such a reference have either "delete" permission on the owner or
// "update" permission on the owner's "finalizers" subresource.
// - https://docs.k8s.io/reference/access-authn-authz/admission-controllers/
// +kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="pgadmins/finalizers",verbs={update}

// setControllerReference sets owner as a Controller OwnerReference on controlled.
// Only one OwnerReference can be a controller, so it returns an error if another
// is already set.
//
// TODO(tjmoore4): This function is duplicated from a version that takes a PostgresCluster object.
func (r *PGAdminReconciler) setControllerReference(
	owner *v1beta1.PGAdmin, controlled client.Object,
) error {
	return controllerutil.SetControllerReference(owner, controlled, r.Client.Scheme())
}
