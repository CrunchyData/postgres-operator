// Copyright 2017 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// AddVolumeAndMountsToPod takes a Pod spec and a PVC and adds a Volume to the Pod spec with
// the PVC as the VolumeSource and mounts the volume to all containers and init containers
// in the Pod spec.
func AddVolumeAndMountsToPod(podSpec *corev1.PodSpec, volume *corev1.PersistentVolumeClaim) {

	podSpec.Volumes = append(podSpec.Volumes, corev1.Volume{
		Name: volume.Name,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: volume.Name,
			},
		},
	})

	for i := range podSpec.Containers {
		podSpec.Containers[i].VolumeMounts = append(podSpec.Containers[i].VolumeMounts,
			corev1.VolumeMount{
				Name:      volume.Name,
				MountPath: fmt.Sprintf("/volumes/%s", volume.Name),
			})
	}

	for i := range podSpec.InitContainers {
		podSpec.InitContainers[i].VolumeMounts = append(podSpec.InitContainers[i].VolumeMounts,
			corev1.VolumeMount{
				Name:      volume.Name,
				MountPath: fmt.Sprintf("/volumes/%s", volume.Name),
			})
	}
}
