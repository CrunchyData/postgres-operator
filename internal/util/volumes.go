// Copyright 2017 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// AdditionalVolumeMount creates a [corev1.VolumeMount] at `/volumes/{name}` of volume `volumes-{name}`.
func AdditionalVolumeMount(name string, readOnly bool) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      fmt.Sprintf("volumes-%s", name),
		MountPath: "/volumes/" + name,
		ReadOnly:  readOnly,
	}
}

// AddAdditionalVolumesAndMounts adds volumes as [corev1.Volume]s and [corev1.VolumeMount]s in template.
// Volume names are chosen in [AdditionalVolumeMount].
func AddAdditionalVolumesAndMounts(template *corev1.PodTemplateSpec, volumes []v1beta1.AdditionalVolume) []string {
	missingContainers := []string{}

	for _, spec := range volumes {
		mount := AdditionalVolumeMount(spec.Name, spec.ReadOnly)
		template.Spec.Volumes = append(template.Spec.Volumes, spec.AsVolume(mount.Name))

		// Create a set of all the requested containers,
		// then in the loops below when we attach the volume to a container,
		// we can safely remove that container name from the set.
		// This gives us a way to track the containers that are requested but not found.
		// This relies on `containers` and `initContainers` together being unique.
		// - https://github.com/kubernetes/api/blob/b40c1cacbb902b21a7e0c7bf0967321860c1a632/core/v1/types.go#L3895C27-L3896C33
		names := sets.New(spec.Containers...)

		for i, c := range template.Spec.Containers {
			if spec.Containers == nil || names.Has(c.Name) {
				c.VolumeMounts = append(c.VolumeMounts, mount)
				template.Spec.Containers[i] = c
			}
			names.Delete(c.Name)
		}

		for i, c := range template.Spec.InitContainers {
			if spec.Containers == nil || names.Has(c.Name) {
				c.VolumeMounts = append(c.VolumeMounts, mount)
				template.Spec.InitContainers[i] = c
			}
			names.Delete(c.Name)
		}

		missingContainers = append(missingContainers, names.UnsortedList()...)
	}

	return missingContainers
}

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
