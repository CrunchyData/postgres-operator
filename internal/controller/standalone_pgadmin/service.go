// Copyright 2023 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package standalone_pgadmin

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/pkg/errors"

	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// +kubebuilder:rbac:groups="",resources="services",verbs={get}
// +kubebuilder:rbac:groups="",resources="services",verbs={create,delete,patch}

// reconcilePGAdminService will reconcile a ClusterIP service that points to
// pgAdmin.
func (r *PGAdminReconciler) reconcilePGAdminService(
	ctx context.Context,
	pgadmin *v1beta1.PGAdmin,
) error {
	log := logging.FromContext(ctx)

	// Since spec.Service only accepts a single service name, we shouldn't ever
	// have more than one service. However, if the user changes ServiceName, we
	// need to delete any existing service(s). At the start of every reconcile
	// get all services that match the current pgAdmin labels.
	services := corev1.ServiceList{}
	if err := r.Client.List(ctx, &services,
		client.InNamespace(pgadmin.Namespace),
		client.MatchingLabels{
			naming.LabelStandalonePGAdmin: pgadmin.Name,
			naming.LabelRole:              naming.RolePGAdmin,
		}); err != nil {
		return err
	}

	// Delete any controlled and labeled service that is not defined in the spec.
	for i := range services.Items {
		if services.Items[i].Name != pgadmin.Spec.ServiceName {
			log.V(1).Info(
				"Deleting service(s) not defined in spec.ServiceName that are owned by pgAdmin",
				"serviceName", services.Items[i].Name)
			if err := r.deleteControlled(ctx, pgadmin, &services.Items[i]); err != nil {
				return err
			}
		}
	}

	// At this point only a service defined by spec.ServiceName should exist.
	// Check if the user has requested a service through ServiceName
	if pgadmin.Spec.ServiceName != "" {
		// Look for an existing service with name ServiceName in the namespace
		existingService := &corev1.Service{}
		err := r.Client.Get(ctx, types.NamespacedName{
			Name:      pgadmin.Spec.ServiceName,
			Namespace: pgadmin.GetNamespace(),
		}, existingService)
		if client.IgnoreNotFound(err) != nil {
			return err
		}

		// If we found an existing service in our namespace with ServiceName
		if !apierrors.IsNotFound(err) {

			// Check if the existing service has ownerReferences.
			// If it doesn't we can go ahead and reconcile the service.
			// If it does then we need to check if we are the controller.
			if len(existingService.OwnerReferences) != 0 {

				// If the service is not controlled by this pgAdmin then we shouldn't reconcile
				if !metav1.IsControlledBy(existingService, pgadmin) {
					err := errors.New("Service is controlled by another object")
					log.V(1).Error(err, "PGO does not force ownership on existing services",
						"ServiceName", pgadmin.Spec.ServiceName)
					r.Recorder.Event(pgadmin,
						corev1.EventTypeWarning, "InvalidServiceWarning",
						"Failed to reconcile Service ServiceName: "+pgadmin.Spec.ServiceName)

					return err
				}
			}
		}

		// A service has been requested and we are allowed to create or reconcile
		service := service(pgadmin)

		// Set the controller reference on the service
		if err := errors.WithStack(r.setControllerReference(pgadmin, service)); err != nil {
			return err
		}

		return errors.WithStack(r.apply(ctx, service))
	}

	// If we get here then ServiceName was not provided through the spec
	return nil
}

// Generate a corev1.Service for pgAdmin
func service(pgadmin *v1beta1.PGAdmin) *corev1.Service {

	service := &corev1.Service{}
	service.ObjectMeta = metav1.ObjectMeta{
		Name:      pgadmin.Spec.ServiceName,
		Namespace: pgadmin.Namespace,
	}
	service.SetGroupVersionKind(
		corev1.SchemeGroupVersion.WithKind("Service"))

	service.Annotations = pgadmin.Spec.Metadata.GetAnnotationsOrNil()
	service.Labels = naming.Merge(
		pgadmin.Spec.Metadata.GetLabelsOrNil(),
		naming.StandalonePGAdminLabels(pgadmin.Name))

	service.Spec.Type = corev1.ServiceTypeClusterIP
	service.Spec.Selector = map[string]string{
		naming.LabelStandalonePGAdmin: pgadmin.Name,
	}
	service.Spec.Ports = []corev1.ServicePort{{
		Name:       "pgadmin-port",
		Port:       pgAdminPort,
		Protocol:   corev1.ProtocolTCP,
		TargetPort: intstr.FromInt(pgAdminPort),
	}}

	return service
}
