/*
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/sets"
)

func findOrAppendContainer(containers *[]corev1.Container, name string) *corev1.Container {
	for i := range *containers {
		if (*containers)[i].Name == name {
			return &(*containers)[i]
		}
	}

	*containers = append(*containers, corev1.Container{Name: name})
	return &(*containers)[len(*containers)-1]
}

func mergeEnvVars(from []corev1.EnvVar, vars ...corev1.EnvVar) []corev1.EnvVar {
	names := sets.NewString()
	for i := range vars {
		names.Insert(vars[i].Name)
	}

	// Partition original slice by whether or not the name was passed in.
	var existing, others []corev1.EnvVar
	for i := range from {
		if names.Has(from[i].Name) {
			existing = append(existing, from[i])
		} else {
			others = append(others, from[i])
		}
	}

	// When the new vars don't match, replace them.
	if !equality.Semantic.DeepEqual(existing, vars) {
		return append(others, vars...)
	}

	return from
}

func mergeVolumes(from []corev1.Volume, vols ...corev1.Volume) []corev1.Volume {
	names := sets.NewString()
	for i := range vols {
		names.Insert(vols[i].Name)
	}

	// Partition original slice by whether or not the name was passed in.
	var existing, others []corev1.Volume
	for i := range from {
		if names.Has(from[i].Name) {
			existing = append(existing, from[i])
		} else {
			others = append(others, from[i])
		}
	}

	// When the new vols don't match, replace them.
	if !equality.Semantic.DeepEqual(existing, vols) {
		return append(others, vols...)
	}

	return from
}

func mergeVolumeMounts(from []corev1.VolumeMount, mounts ...corev1.VolumeMount) []corev1.VolumeMount {
	names := sets.NewString()
	for i := range mounts {
		names.Insert(mounts[i].Name)
	}

	// Partition original slice by whether or not the name was passed in.
	var existing, others []corev1.VolumeMount
	for i := range from {
		if names.Has(from[i].Name) {
			existing = append(existing, from[i])
		} else {
			others = append(others, from[i])
		}
	}

	// When the new mounts don't match, replace them.
	if !equality.Semantic.DeepEqual(existing, mounts) {
		return append(others, mounts...)
	}

	return from
}
