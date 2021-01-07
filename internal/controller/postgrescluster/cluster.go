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
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/patroni"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
)

// +kubebuilder:rbac:resources=services,verbs=get;create;update

// reconcileClusterService writes the Service that points to the PostgreSQL
// instance that is leader.
// TODO(cbandy): see if it's possible to reduce the influence Patroni has here.
func (r *Reconciler) reconcileClusterService(
	ctx context.Context, cluster *v1alpha1.PostgresCluster,
) (*v1.Service, error) {
	log := logging.FromContext(ctx)

	// TODO(cbandy): Use server-side apply instead.
	cs := &v1.Service{ObjectMeta: naming.ClusterService(cluster)}
	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, cs, func() (err error) {
		if cs.Labels == nil {
			cs.Labels = make(map[string]string)
		}
		cs.Labels[naming.LabelCluster] = cluster.Name
		// TODO(cbandy): Set LabelRole to indicate this points to primary?

		cs.Spec.Ports = []v1.ServicePort{{
			Port:       *cluster.Spec.Port,
			TargetPort: intstr.FromString(naming.PortPostgreSQL),
		}}

		if err == nil {
			err = errors.WithStack(r.setControllerReference(cluster, cs))
		}
		if err == nil {
			err = patroni.ClusterService(ctx, cluster, cs)
		}

		return err
	})
	if err == nil {
		log.V(1).Info("reconciled cluster service", "operation", op)
	}

	return cs, err
}

// +kubebuilder:rbac:resources=configmaps,verbs=patch

// reconcileClusterConfigMap writes the ConfigMap that contains generated
// files (etc) that apply to the entire cluster.
func (r *Reconciler) reconcileClusterConfigMap(
	ctx context.Context, cluster *v1alpha1.PostgresCluster,
) (*v1.ConfigMap, error) {
	clusterConfigMap := &v1.ConfigMap{ObjectMeta: naming.ClusterConfigMap(cluster)}
	clusterConfigMap.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("ConfigMap"))

	err := errors.WithStack(r.setControllerReference(cluster, clusterConfigMap))

	clusterConfigMap.Labels = map[string]string{
		naming.LabelCluster: cluster.Name,
	}

	if err == nil {
		err = patroni.ClusterConfigMap(ctx, cluster, clusterConfigMap)
	}
	if err == nil {
		err = errors.WithStack(
			r.patch(ctx, clusterConfigMap, client.Apply, client.ForceOwnership))
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
		err = errors.WithStack(
			r.patch(ctx, clusterPodService, client.Apply, client.ForceOwnership))
	}

	return clusterPodService, err
}
