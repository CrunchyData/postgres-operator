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

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/patroni"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
)

// +kubebuilder:rbac:resources=configmaps,verbs=patch

// reconcileClusterConfigMap writes the ConfigMap that contains generated
// files (etc) that apply to the entire cluster.
func (r *Reconciler) reconcileClusterConfigMap(
	ctx context.Context, cluster *v1alpha1.PostgresCluster,
	pgHBAs postgres.HBAs, pgParameters postgres.Parameters,
) (*v1.ConfigMap, error) {
	clusterConfigMap := &v1.ConfigMap{ObjectMeta: naming.ClusterConfigMap(cluster)}
	clusterConfigMap.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("ConfigMap"))

	err := errors.WithStack(r.setControllerReference(cluster, clusterConfigMap))

	clusterConfigMap.Labels = map[string]string{
		naming.LabelCluster: cluster.Name,
	}

	if err == nil {
		err = patroni.ClusterConfigMap(ctx, cluster, pgHBAs, pgParameters, clusterConfigMap)
	}
	if err == nil {
		err = errors.WithStack(r.apply(ctx, clusterConfigMap))
	}

	return clusterConfigMap, err
}

// +kubebuilder:rbac:resources=services,verbs=patch

// reconcileClusterPodService writes the Service that can provide stable DNS
// names to Pods related to cluster.
func (r *Reconciler) reconcileClusterPodService(
	ctx context.Context, cluster *v1alpha1.PostgresCluster,
) (*v1.Service, error) {
	clusterPodService := &v1.Service{ObjectMeta: naming.ClusterPodService(cluster)}
	clusterPodService.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Service"))

	err := errors.WithStack(r.setControllerReference(cluster, clusterPodService))

	clusterPodService.Labels = map[string]string{
		naming.LabelCluster: cluster.Name,
	}

	// Allocate no IP address (headless) and match any Pod with the cluster
	// label, regardless of its readiness. Not particularly useful by itself, but
	// this allows a properly configured Pod to get a DNS record based on its name.
	// - https://docs.k8s.io/concepts/services-networking/service/#headless-services
	// - https://docs.k8s.io/concepts/services-networking/dns-pod-service/#pods
	clusterPodService.Spec.ClusterIP = v1.ClusterIPNone
	clusterPodService.Spec.PublishNotReadyAddresses = true
	clusterPodService.Spec.Selector = map[string]string{
		naming.LabelCluster: cluster.Name,
	}

	if err == nil {
		err = errors.WithStack(r.apply(ctx, clusterPodService))
	}

	return clusterPodService, err
}

// +kubebuilder:rbac:resources=endpoints;services,verbs=patch

// reconcileClusterPrimaryService writes the Service and Endpoints that resolve
// to the PostgreSQL primary instance.
func (r *Reconciler) reconcileClusterPrimaryService(
	ctx context.Context, cluster *v1alpha1.PostgresCluster, leader *v1.Service,
) error {
	clusterPrimaryService := &v1.Service{ObjectMeta: naming.ClusterPrimaryService(cluster)}
	clusterPrimaryService.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Service"))

	err := errors.WithStack(r.setControllerReference(cluster, clusterPrimaryService))

	clusterPrimaryService.Labels = map[string]string{
		naming.LabelCluster: cluster.Name,
		naming.LabelRole:    naming.RolePrimary,
	}

	if err == nil && leader == nil {
		// TODO(cbandy): We need to build a different kind of Service here.
		err = errors.New("Patroni DCS other than Kubernetes Endpoints is not implemented")
	}

	// We want to name and label our primary Service consistently. When Patroni is
	// using Endpoints for its DCS, however, they and any Service that uses them
	// must use the same name as the Patroni "scope" which has its own constraints.
	//
	// To stay free from those constraints, our primary Service will resolve to
	// the ClusterIP of the Service created in the reconcilePatroniLeaderLease
	// method when Patroni is using Endpoints.

	// Allocate no IP address (headless) and manage the Endpoints ourselves.
	// - https://docs.k8s.io/concepts/services-networking/service/#headless-services
	// - https://docs.k8s.io/concepts/services-networking/service/#services-without-selectors
	clusterPrimaryService.Spec.ClusterIP = v1.ClusterIPNone
	clusterPrimaryService.Spec.Selector = nil

	clusterPrimaryService.Spec.Ports = []v1.ServicePort{{
		Name:     naming.PortPostgreSQL,
		Port:     *cluster.Spec.Port,
		Protocol: v1.ProtocolTCP,
	}}

	if err == nil {
		err = errors.WithStack(r.apply(ctx, clusterPrimaryService))
	}

	// Endpoints for a Service have the same name as the Service.
	endpoints := &v1.Endpoints{ObjectMeta: naming.ClusterPrimaryService(cluster)}
	endpoints.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Endpoints"))

	if err == nil {
		err = errors.WithStack(r.setControllerReference(cluster, endpoints))
	}

	endpoints.Labels = map[string]string{
		naming.LabelCluster: cluster.Name,
		naming.LabelRole:    naming.RolePrimary,
	}

	// Resolve to the ClusterIP for which Patroni has configured the Endpoints.
	endpoints.Subsets = []v1.EndpointSubset{{
		Addresses: []v1.EndpointAddress{{IP: leader.Spec.ClusterIP}},
	}}

	// Copy the EndpointPorts from the ServicePorts.
	for _, sp := range clusterPrimaryService.Spec.Ports {
		endpoints.Subsets[0].Ports = append(endpoints.Subsets[0].Ports,
			v1.EndpointPort{
				Name:     sp.Name,
				Port:     sp.Port,
				Protocol: sp.Protocol,
			})
	}

	if err == nil {
		err = errors.WithStack(r.apply(ctx, endpoints))
	}

	return err
}
