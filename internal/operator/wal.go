package operator

/*
 Copyright 2020 - 2022 Crunchy Data Solutions, Inc.
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
	"github.com/crunchydata/postgres-operator/internal/config"
	core_v1 "k8s.io/api/core/v1"
)

// addWALVolumeAndMounts modifies podSpec to include walVolume on each containerNames.
func addWALVolumeAndMounts(podSpec *core_v1.PodSpec, walVolume StorageResult, containerNames ...string) {
	walVolumeMount := config.PostgreSQLWALVolumeMount()

	if podSpec.SecurityContext == nil {
		podSpec.SecurityContext = &core_v1.PodSecurityContext{}
	}

	podSpec.SecurityContext.SupplementalGroups = append(
		podSpec.SecurityContext.SupplementalGroups, walVolume.SupplementalGroups...)

	podSpec.Volumes = append(podSpec.Volumes, core_v1.Volume{
		Name:         walVolumeMount.Name,
		VolumeSource: walVolume.VolumeSource(),
	})

	for i := range podSpec.Containers {
		container := &podSpec.Containers[i]
		for _, name := range containerNames {
			if container.Name == name {
				container.VolumeMounts = append(container.VolumeMounts, walVolumeMount)
			}
		}
	}
}

// AddWALVolumeAndMountsToBackRest modifies a pgBackRest podSpec to include walVolume.
func AddWALVolumeAndMountsToBackRest(podSpec *core_v1.PodSpec, walVolume StorageResult) {
	addWALVolumeAndMounts(podSpec, walVolume, "backrest")
}

// AddWALVolumeAndMountsToPostgreSQL modifies a PostgreSQL podSpec to include walVolume.
func AddWALVolumeAndMountsToPostgreSQL(podSpec *core_v1.PodSpec, walVolume StorageResult, instanceName string) {
	addWALVolumeAndMounts(podSpec, walVolume, "database")

	for i := range podSpec.Containers {
		container := &podSpec.Containers[i]
		if container.Name == "database" {
			container.Env = append(container.Env, core_v1.EnvVar{
				Name:  "PGHA_WALDIR",
				Value: config.PostgreSQLWALPath(instanceName),
			})
		}
	}
}
