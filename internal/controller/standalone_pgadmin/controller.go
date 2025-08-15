// Copyright 2023 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package standalone_pgadmin

import (
	"context"
	"errors"
	"io"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/tracing"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// PGAdminReconciler reconciles a PGAdmin object
type PGAdminReconciler struct {
	PodExec func(
		ctx context.Context, namespace, pod, container string,
		stdin io.Reader, stdout, stderr io.Writer, command ...string,
	) error

	Reader interface {
		Get(context.Context, client.ObjectKey, client.Object, ...client.GetOption) error
		List(context.Context, client.ObjectList, ...client.ListOption) error
	}
	Writer interface {
		Delete(context.Context, client.Object, ...client.DeleteOption) error
		Patch(context.Context, client.Object, client.Patch, ...client.PatchOption) error
	}
	StatusWriter interface {
		Patch(context.Context, client.Object, client.Patch, ...client.SubResourcePatchOption) error
	}

	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="pgadmins",verbs={get,list,watch}
//+kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="postgresclusters",verbs={list,watch}
//+kubebuilder:rbac:groups="",resources="persistentvolumeclaims",verbs={list,watch}
//+kubebuilder:rbac:groups="",resources="secrets",verbs={list,watch}
//+kubebuilder:rbac:groups="",resources="configmaps",verbs={list,watch}
//+kubebuilder:rbac:groups="apps",resources="statefulsets",verbs={list,watch}

// ManagedReconciler creates a [PGAdminReconciler] and adds it to m.
func ManagedReconciler(m manager.Manager) error {
	exec, err := runtime.NewPodExecutor(m.GetConfig())
	kubernetes := client.WithFieldOwner(m.GetClient(), naming.ControllerPGAdmin)
	recorder := m.GetEventRecorderFor(naming.ControllerPGAdmin)

	reconciler := &PGAdminReconciler{
		PodExec:      exec,
		Reader:       kubernetes,
		Recorder:     recorder,
		StatusWriter: kubernetes.Status(),
		Writer:       kubernetes,
	}

	return errors.Join(err, builder.ControllerManagedBy(m).
		For(&v1beta1.PGAdmin{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.Secret{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Watches(
			v1beta1.NewPostgresCluster(),
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, cluster client.Object) []reconcile.Request {
				return runtime.Requests(reconciler.findPGAdminsForPostgresCluster(ctx, cluster)...)
			}),
		).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, secret client.Object) []reconcile.Request {
				return runtime.Requests(reconciler.findPGAdminsForSecret(ctx, client.ObjectKeyFromObject(secret))...)
			}),
		).
		Complete(reconcile.AsReconciler(kubernetes, reconciler)))
}

//+kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="pgadmins/status",verbs={patch}

// Reconcile moves the current state of pgAdmin closer to the state described in its specification.
func (r *PGAdminReconciler) Reconcile(ctx context.Context, pgAdmin *v1beta1.PGAdmin) (reconcile.Result, error) {

	var err error
	ctx, span := tracing.Start(ctx, "reconcile-pgadmin")
	log := logging.FromContext(ctx)
	defer span.End()

	// Write any changes to the pgadmin status on the way out.
	before := pgAdmin.DeepCopy()
	defer func() {
		if !equality.Semantic.DeepEqual(before.Status, pgAdmin.Status) {
			statusErr := r.StatusWriter.Patch(ctx, pgAdmin, client.MergeFrom(before))
			if statusErr != nil {
				log.Error(statusErr, "Patching PGAdmin status")
			}
			if err == nil {
				err = statusErr
			}
		}
	}()

	log.V(1).Info("Reconciling pgAdmin")

	// Set defaults if unset
	pgAdmin.Default()

	var (
		configmap  *corev1.ConfigMap
		dataVolume *corev1.PersistentVolumeClaim
		clusters   map[string][]*v1beta1.PostgresCluster
		_          *corev1.Service
	)

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
		err = r.reconcilePGAdminService(ctx, pgAdmin)
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
		log.V(1).Info("Reconciled pgAdmin")
	}

	return reconcile.Result{}, tracing.Escape(span, err)
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
	return controllerutil.SetControllerReference(owner, controlled, runtime.Scheme)
}

// deleteControlled safely deletes object when it is controlled by pgAdmin.
func (r *PGAdminReconciler) deleteControlled(
	ctx context.Context, pgadmin *v1beta1.PGAdmin, object client.Object,
) error {
	if metav1.IsControlledBy(object, pgadmin) {
		uid := object.GetUID()
		version := object.GetResourceVersion()
		exactly := client.Preconditions{UID: &uid, ResourceVersion: &version}

		return r.Writer.Delete(ctx, object, exactly)
	}

	return nil
}
