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

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
)

// ClusterConfigMap populates the shared ConfigMap with fields needed to run Patroni.
func ClusterConfigMap(ctx context.Context,
	inCluster *v1alpha1.PostgresCluster,
	outClusterConfigMap *v1.ConfigMap,
) error {
	var err error

	if outClusterConfigMap.Data == nil {
		outClusterConfigMap.Data = make(map[string]string)
	}

	outClusterConfigMap.Data[configMapFileKey], err = clusterYAML(inCluster)

	return err
}

// InstanceConfigMap populates the shared ConfigMap with fields needed to run Patroni.
func InstanceConfigMap(ctx context.Context,
	inCluster *v1alpha1.PostgresCluster,
	inInstance metav1.Object,
	outInstanceConfigMap *v1.ConfigMap,
) error {
	var err error

	if outInstanceConfigMap.Data == nil {
		outInstanceConfigMap.Data = make(map[string]string)
	}

	outInstanceConfigMap.Data[configMapFileKey], err = instanceYAML(inCluster, inInstance)

	return err
}

// InstancePod populates a PodSpec with the fields needed to run Patroni.
func InstancePod(ctx context.Context,
	inCluster *v1alpha1.PostgresCluster,
	inClusterConfigMap *v1.ConfigMap,
	inClusterPodService *v1.Service,
	inPatroniLeaderService *v1.Service,
	inInstanceConfigMap *v1.ConfigMap,
	outInstancePod *v1.PodTemplateSpec,
) error {
	if outInstancePod.Labels == nil {
		outInstancePod.Labels = make(map[string]string)
	}

	// When using Kubernetes for DCS, Patroni discovers members by listing Pods
	// that have the "scope" label. See the "kubernetes.scope_label" and
	// "kubernetes.labels" settings.
	outInstancePod.Labels[naming.LabelPatroni] = naming.PatroniScope(inCluster)

	container := findOrAppendContainer(&outInstancePod.Spec.Containers,
		naming.ContainerDatabase)

	container.Env = mergeEnvVars(container.Env,
		instanceEnvironment(inCluster, inClusterPodService, inPatroniLeaderService,
			outInstancePod.Spec.Containers)...)

	volume := v1.Volume{Name: "patroni-config"}
	volume.Projected = new(v1.ProjectedVolumeSource)

	// Add our projections after those specified in the CR. Items later in the
	// list take precedence over earlier items (that is, last write wins).
	// - https://kubernetes.io/docs/concepts/storage/volumes/#projected
	volume.Projected.Sources = append(append(
		// TODO(cbandy): User config will come from the spec.
		volume.Projected.Sources, []v1.VolumeProjection(nil)...),
		instanceConfigFiles(inClusterConfigMap, inInstanceConfigMap)...)

	outInstancePod.Spec.Volumes = mergeVolumes(outInstancePod.Spec.Volumes, volume)

	container.VolumeMounts = mergeVolumeMounts(container.VolumeMounts, v1.VolumeMount{
		Name:      volume.Name,
		MountPath: configDirectory,
		ReadOnly:  true,
	})

	return nil
}
