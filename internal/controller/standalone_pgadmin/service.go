/*
 Copyright 2023 Crunchy Data Solutions, Inc.
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

package standalone_pgadmin

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// +kubebuilder:rbac:groups="",resources="services",verbs={get}
// +kubebuilder:rbac:groups="",resources="services",verbs={create,delete,patch}

// reconcilePGAdminService writes the Service that resolves to pgAdmin.
func (r *PGAdminReconciler) reconcilePGAdminService(
	ctx context.Context, pgadmin *v1beta1.PGAdmin,
) (*corev1.Service, error) {
	// The NodePort can only be set when the Service type is NodePort or
	// LoadBalancer. However, due to a known issue prior to Kubernetes
	// 1.20, we clear these errors during our apply. To preserve the
	// appropriate behavior, we log an Event and return an error.
	// TODO(tjmoore4): Once Validation Rules are available, this check
	// and event could potentially be removed in favor of that validation
	if pgadmin.Spec.Service != nil &&
		pgadmin.Spec.Service.NodePort != nil &&
		corev1.ServiceType(pgadmin.Spec.Service.Type) == corev1.ServiceTypeClusterIP {

		r.Recorder.Eventf(pgadmin, corev1.EventTypeWarning, "MisconfiguredClusterIP",
			"NodePort cannot be set with type ClusterIP on Service %q", naming.StandalonePGAdmin(pgadmin).Name)
		return nil, fmt.Errorf("NodePort cannot be set with type ClusterIP on Service %q",
			naming.StandalonePGAdmin(pgadmin).Name)
	}

	service, err := service(pgadmin)

	if err == nil {
		err = errors.WithStack(r.setControllerReference(pgadmin, service))
	}

	if err == nil {
		err = errors.WithStack(r.apply(ctx, service))
	}
	return service, err
}

// service returns a v1.Service that exposes pgAdmin pods.
func service(
	pgadmin *v1beta1.PGAdmin) (*corev1.Service, error,
) {
	service := &corev1.Service{ObjectMeta: naming.StandalonePGAdminService(pgadmin)}
	service.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Service"))

	service.Annotations = pgadmin.Spec.Metadata.GetAnnotationsOrNil()
	service.Labels = naming.Merge(
		pgadmin.Spec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelStandalonePGAdmin: pgadmin.Name,
			naming.LabelRole:              naming.RolePGAdmin,
		})

	if spec := pgadmin.Spec.Service; spec != nil {
		service.Annotations = naming.Merge(service.Annotations,
			spec.Metadata.GetAnnotationsOrNil())
		service.Labels = naming.Merge(service.Labels,
			spec.Metadata.GetLabelsOrNil())
	}

	// Allocate an IP address and/or node port and let Kubernetes manage the
	// Endpoints by selecting Pods with the pgAdmin role.
	// - https://docs.k8s.io/concepts/services-networking/service/#defining-a-service
	service.Spec.Selector = map[string]string{
		naming.LabelStandalonePGAdmin: pgadmin.Name,
		naming.LabelRole:              naming.RoleStandalonePGAdmin,
	}

	// The TargetPort must be the name (not the number) of the pgAdmin
	// ContainerPort. This name allows the port number to differ between Pods,
	// which can happen during a rolling update.
	//
	// TODO(tjmoore4): A custom service port is not currently supported as this
	// requires updates to the pgAdmin service configuration.
	servicePort := corev1.ServicePort{
		Name:       naming.PortPGAdmin,
		Port:       *initialize.Int32(pgAdminPort),
		Protocol:   corev1.ProtocolTCP,
		TargetPort: intstr.FromString(naming.PortPGAdmin),
	}

	if spec := pgadmin.Spec.Service; spec == nil {
		service.Spec.Type = corev1.ServiceTypeClusterIP
	} else {
		service.Spec.Type = corev1.ServiceType(spec.Type)
		if spec.NodePort != nil {
			if service.Spec.Type == corev1.ServiceTypeClusterIP {
				// We should record an event in the parent `reconcilePGAdminService`
				// and never get here in this situation.
				// But just to be safe, let's check and return an error here.
				return nil, fmt.Errorf("NodePort cannot be set with type ClusterIP on Service %q", service.Name)
			}
			servicePort.NodePort = *spec.NodePort
		}
	}
	service.Spec.Ports = []corev1.ServicePort{servicePort}

	return service, nil
}
