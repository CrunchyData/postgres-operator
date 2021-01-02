package kubeapi

/*
 Copyright 2020 - 2021 Crunchy Data Solutions, Inc.
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

import v1 "k8s.io/api/core/v1"

// FindOrAppendVolume returns a pointer to the Volume in volumes named name.
// If no such Volume exists, it creates one with that name and returns it.
func FindOrAppendVolume(volumes *[]v1.Volume, name string) *v1.Volume {
	for i := range *volumes {
		if (*volumes)[i].Name == name {
			return &(*volumes)[i]
		}
	}

	*volumes = append(*volumes, v1.Volume{Name: name})
	return &(*volumes)[len(*volumes)-1]
}

// FindOrAppendVolumeMount returns a pointer to the VolumeMount in mounts named
// name. If no such VolumeMount exists, it creates one with that name and
// returns it.
func FindOrAppendVolumeMount(mounts *[]v1.VolumeMount, name string) *v1.VolumeMount {
	for i := range *mounts {
		if (*mounts)[i].Name == name {
			return &(*mounts)[i]
		}
	}

	*mounts = append(*mounts, v1.VolumeMount{Name: name})
	return &(*mounts)[len(*mounts)-1]
}
