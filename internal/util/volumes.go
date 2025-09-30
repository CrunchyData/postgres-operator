// Copyright 2017 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

const (
	// Default auto-grow trigger percentage
	AutoGrowTriggerDefault = "75"
)

// AdditionalVolumeMount creates a [corev1.VolumeMount] at `/volumes/{name}` of volume `volumes-{name}`.
func AdditionalVolumeMount(name string, readOnly bool) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      fmt.Sprintf("volumes-%s", name),
		MountPath: "/volumes/" + name,
		ReadOnly:  readOnly,
	}
}

// AddAdditionalVolumesAndMounts adds volumes as [corev1.Volume]s and [corev1.VolumeMount]s in pod.
// Volume names are chosen in [AdditionalVolumeMount].
func AddAdditionalVolumesAndMounts(pod *corev1.PodSpec, volumes []v1beta1.AdditionalVolume) []string {
	return addVolumesAndMounts(pod, volumes, AdditionalVolumeMount)
}

// AddCloudLogVolumeToPod takes a Pod spec and a PVC and adds a Volume to the Pod spec with
// the PVC as the VolumeSource and mounts the volume to all containers and init containers
// in the Pod spec.
func AddCloudLogVolumeToPod(podSpec *corev1.PodSpec, pvcName string) {
	additional := []v1beta1.AdditionalVolume{{
		ClaimName: pvcName,
		Name:      pvcName,
		ReadOnly:  false,
	}}

	addVolumesAndMounts(podSpec, additional, func(string, bool) corev1.VolumeMount {
		return corev1.VolumeMount{
			// This name has no prefix and differs from [AdditionalVolumeMount].
			Name:      pvcName,
			MountPath: fmt.Sprintf("/volumes/%s", pvcName),
			ReadOnly:  false,
		}
	})
}

func addVolumesAndMounts(pod *corev1.PodSpec, volumes []v1beta1.AdditionalVolume, namer func(string, bool) corev1.VolumeMount) []string {
	missingContainers := []string{}

	for _, spec := range volumes {
		// If it is an image volume, override readOnly to true
		readOnly := spec.ReadOnly
		if spec.Image != nil {
			readOnly = true
		}

		mount := namer(spec.Name, readOnly)
		pod.Volumes = append(pod.Volumes, spec.AsVolume(mount.Name))

		// Create a set of all the requested containers,
		// then in the loops below when we attach the volume to a container,
		// we can safely remove that container name from the set.
		// This gives us a way to track the containers that are requested but not found.
		// This relies on `containers` and `initContainers` together being unique.
		// - https://github.com/kubernetes/api/blob/b40c1cacbb902b21a7e0c7bf0967321860c1a632/core/v1/types.go#L3895C27-L3896C33
		names := sets.New(spec.Containers...)

		for i, c := range pod.Containers {
			if spec.Containers == nil || names.Has(c.Name) {
				c.VolumeMounts = append(c.VolumeMounts, mount)
				pod.Containers[i] = c
			}
			names.Delete(c.Name)
		}

		for i, c := range pod.InitContainers {
			if spec.Containers == nil || names.Has(c.Name) {
				c.VolumeMounts = append(c.VolumeMounts, mount)
				pod.InitContainers[i] = c
			}
			names.Delete(c.Name)
		}

		missingContainers = append(missingContainers, names.UnsortedList()...)
	}

	return missingContainers
}

// GetAutoGrowFromSpec extracts AutoGrow settings from the provided AutoGrowSpec.
// It returns two strings: (trigger, maxGrow).
//
//   - trigger: the auto-grow threshold as a decimal percentage string (e.g., "75").
//     Defaults to "75" when no trigger is configured.
//   - maxGrow: the maximum growth size as a whole number of mebibytes (MiB).
//     When no MaxGrow is configured, this is an empty string.
//
// If AutoGrow is nil, the function uses the default trigger ("75") and leaves
// maxGrow empty. When set, Trigger is formatted as a base-10 string and MaxGrow is
// converted from a resource quantity (bytes) to mebibytes by dividing bytes by 1024*1024.
func GetAutoGrowFromSpec(spec *v1beta1.VolumeClaimSpecWithAutoGrow) (string, string) {
	// MaxGrow is optional; an empty string means "no limit" on growth and the volume
	// will grow by 50% each time it is triggered.
	maxGrow := ""

	// We always want to set default trigger; We will override it if
	// the user has set a different value.
	trigger := AutoGrowTriggerDefault

	// If AutoGrow is configured on the VolumeClaimSpecWithAutoGrow, extract
	// the Trigger and MaxGrow values
	if spec != nil && spec.AutoGrow != nil {
		if t := spec.AutoGrow.Trigger; t != nil {
			trigger = fmt.Sprintf("%d", initialize.FromPointer(t))
		}
		if mg := spec.AutoGrow.MaxGrow; mg != nil {
			// Value() returns bytes, convert to mebibytes
			mebibytes := mg.Value() / (1024 * 1024)
			maxGrow = fmt.Sprintf("%d", mebibytes)
		}
	}
	return trigger, maxGrow
}
