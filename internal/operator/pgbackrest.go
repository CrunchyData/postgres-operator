package operator

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

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/crunchydata/postgres-operator/internal/kubeapi"
)

func addBackRestConfigDirectoryVolume(podSpec *v1.PodSpec, volumeName string, projections []v1.VolumeProjection) {
	// v1.PodSpec.Volumes is keyed on Name.
	volume := kubeapi.FindOrAppendVolume(&podSpec.Volumes, volumeName)
	if volume.Projected == nil {
		volume.Projected = &v1.ProjectedVolumeSource{}
	}
	volume.Projected.Sources = append(volume.Projected.Sources, projections...)
}

func addBackRestConfigDirectoryVolumeMount(container *v1.Container, volumeName string) {
	// v1.Container.VolumeMounts is keyed on MountPoint, *not* Name.
	mount := kubeapi.FindOrAppendVolumeMount(&container.VolumeMounts, volumeName)
	mount.MountPath = "/etc/pgbackrest/conf.d"
}

func addBackRestConfigDirectoryVolumeAndMounts(podSpec *v1.PodSpec, volumeName string, projections []v1.VolumeProjection, containerNames ...string) {
	names := sets.NewString(containerNames...)

	for i := range podSpec.InitContainers {
		if names.Has(podSpec.InitContainers[i].Name) {
			addBackRestConfigDirectoryVolumeMount(&podSpec.InitContainers[i], volumeName)
		}
	}

	for i := range podSpec.Containers {
		if names.Has(podSpec.Containers[i].Name) {
			addBackRestConfigDirectoryVolumeMount(&podSpec.Containers[i], volumeName)
		}
	}

	addBackRestConfigDirectoryVolume(podSpec, volumeName, projections)
}

// AddBackRestConfigVolumeAndMounts modifies podSpec to include pgBackRest configuration.
// Any projections are included as custom pgBackRest configuration.
func AddBackRestConfigVolumeAndMounts(podSpec *v1.PodSpec, clusterName string, projections []v1.VolumeProjection) {
	var combined []v1.VolumeProjection
	defaultConfigNames := clusterName + "-config-backrest"
	varTrue := true

	// Start with custom configurations from the CRD.
	combined = append(combined, projections...)

	// Followed by built-in configurations. Items later in the list take precedence
	// over earlier items (that is, last write wins).
	//
	// - https://docs.openshift.com/container-platform/4.5/nodes/containers/nodes-containers-projected-volumes.html
	// - https://kubernetes.io/docs/concepts/storage/volumes/#projected
	//
	configmap := v1.ConfigMapProjection{}
	configmap.Name = defaultConfigNames
	configmap.Optional = &varTrue
	combined = append(combined, v1.VolumeProjection{ConfigMap: &configmap})

	secret := v1.SecretProjection{}
	secret.Name = defaultConfigNames
	secret.Optional = &varTrue
	combined = append(combined, v1.VolumeProjection{Secret: &secret})

	// The built-in configurations above also happen to bypass a bug in Kubernetes.
	// Kubernetes 1.15 through 1.19 store an empty list of sources as `null` which
	// breaks some clients, notably the Python client used by Patroni 1.6.5.
	// - https://issue.k8s.io/93903

	addBackRestConfigDirectoryVolumeAndMounts(podSpec, "pgbackrest-config", combined, "backrest", "database")
}
