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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
	if err := r.Client.List(ctx, &services, client.MatchingLabels{
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

	// TODO (jmckulk): check if the requested services exists without our pgAdmin
	// as the owner. If this happens, don't take over ownership of the existing svc.

	// At this point only a service defined by spec.ServiceName should exist.
	// Update the service or create it if it does not exist
	if pgadmin.Spec.ServiceName != "" {
		service := service(pgadmin)
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
