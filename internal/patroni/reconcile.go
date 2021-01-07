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

package patroni

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
)

// ClusterConfigMap populates the shared ConfigMap with fields needed to run Patroni.
func ClusterConfigMap(ctx context.Context,
	inCluster *v1alpha1.PostgresCluster,
	outClusterConfigMap *v1.ConfigMap,
) error {
	return clusterConfigMap(inCluster, outClusterConfigMap)
}

// ClusterService populates the primary Service so that it refers to the Patroni
// leader.
func ClusterService(ctx context.Context,
	inCluster *v1alpha1.PostgresCluster,
	outClusterService *v1.Service,
) error {
	// When using Endpoints for DCS, Patroni manages the destination addresses of
	// the Service, and the Selector must be empty.
	// - https://kubernetes.io/docs/concepts/services-networking/service/#services-without-selectors
	outClusterService.Spec.Selector = nil

	// TODO(cbandy): When using Endpoints for DCS its possible for Patroni to
	// write the Endpoints *after* the Service has been GC'd. These Endpoints
	// are orphaned and hang around even after the PostgresCluster is gone.
	// Consider using a finalizer to ensure the Service is deleted after Patroni
	// stops running.

	return nil
}

// InstanceConfigMap populates the shared ConfigMap with fields needed to run Patroni.
func InstanceConfigMap(ctx context.Context,
	inCluster *v1alpha1.PostgresCluster,
	inInstance metav1.Object,
	outInstanceConfigMap *v1.ConfigMap,
) error {
	return instanceConfigMap(inCluster, inInstance, outInstanceConfigMap)
}

// InstancePod populates a PodSpec with the fields needed to run Patroni.
func InstancePod(ctx context.Context,
	inCluster *v1alpha1.PostgresCluster,
	inClusterConfigMap *v1.ConfigMap,
	inClusterService *v1.Service,
	inInstanceConfigMap *v1.ConfigMap,
	outInstancePod *v1.PodSpec,
) error {
	instanceEnvVars(inCluster, outInstancePod)
	instanceConfigVolumeAndMount(inCluster, inClusterConfigMap, inInstanceConfigMap, outInstancePod)
	return nil
}
