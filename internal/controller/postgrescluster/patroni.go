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

package postgrescluster

import (
	"context"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
)

// +kubebuilder:rbac:resources=endpoints;services,verbs=patch

// reconcilePatroniDistributedConfiguration sets labels and ownership on
// objects created by Patroni.
func (r *Reconciler) reconcilePatroniDistributedConfiguration(
	ctx context.Context, cluster *v1alpha1.PostgresCluster,
) error {
	// When using Endpoints for DCS, Patroni will create and write an Endpoints
	// object, but it won't set our role label nor cluster ownership.
	dcsEndpoints := &v1.Endpoints{ObjectMeta: naming.PatroniDistributedConfiguration(cluster)}
	dcsEndpoints.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Endpoints"))

	err := errors.WithStack(r.setControllerReference(cluster, dcsEndpoints))

	dcsEndpoints.Labels = map[string]string{
		naming.LabelCluster: cluster.Name,
		naming.LabelRole:    "patroni",
	}

	if err == nil {
		err = errors.WithStack(
			r.patch(ctx, dcsEndpoints, client.Apply, client.ForceOwnership))
	}

	// When using Endpoints for DCS, Patroni needs a Service to ensure that the
	// above Endpoints object is not removed by Kubernetes. Patroni will create
	// this object if it has permission to do so, but it won't set our role label
	// nor cluster ownership.
	// - https://github.com/zalando/patroni/blob/v2.0.1/patroni/dcs/kubernetes.py#L865-L881
	dcsService := &v1.Service{ObjectMeta: naming.PatroniDistributedConfiguration(cluster)}
	dcsService.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Service"))

	if err == nil {
		err = errors.WithStack(r.setControllerReference(cluster, dcsService))
	}

	dcsService.Labels = map[string]string{
		naming.LabelCluster: cluster.Name,
		naming.LabelRole:    "patroni",
	}

	// Allocate no IP address (headless) and create no Endpoints.
	// - https://docs.k8s.io/concepts/services-networking/service/#headless-services
	dcsService.Spec.ClusterIP = v1.ClusterIPNone
	dcsService.Spec.Selector = nil

	if err == nil {
		err = errors.WithStack(
			r.patch(ctx, dcsService, client.Apply, client.ForceOwnership))
	}

	// TODO(cbandy): Investigate "{scope}-failover" endpoints; is it DCS "failover_path"?
	// TODO(cbandy): Investigate DCS "sync_path".

	return err
}
