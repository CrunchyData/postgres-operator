package operator

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

import (
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
)

func TestAddBackRestConfigVolumeAndMounts(t *testing.T) {
	t.Parallel()

	{
		// Don't rely on anything in particular from operator templates.
		var spec v1.PodSpec
		AddBackRestConfigVolumeAndMounts(&spec, "cname", nil)

		if expected, actual := 1, len(spec.Volumes); expected != actual {
			t.Fatalf("expected a new volume, got %v", actual)
		}
		if spec.Volumes[0].Projected == nil || len(spec.Volumes[0].Projected.Sources) == 0 {
			t.Fatalf("expected a non-empty projected volume, got %#v", spec.Volumes[0])
		}

		spec = v1.PodSpec{}
		AddBackRestConfigVolumeAndMounts(&spec, "cname", []v1.VolumeProjection{
			{ConfigMap: &v1.ConfigMapProjection{
				LocalObjectReference: v1.LocalObjectReference{Name: "somesuch"},
			}},
		})

		if expected, actual := 1, len(spec.Volumes); expected != actual {
			t.Fatalf("expected a new volume, got %v", actual)
		}
		if spec.Volumes[0].Projected == nil || len(spec.Volumes[0].Projected.Sources) == 0 {
			t.Fatalf("expected a non-empty projected volume, got %#v", spec.Volumes[0])
		}
		if spec.Volumes[0].Projected.Sources[0].ConfigMap == nil ||
			spec.Volumes[0].Projected.Sources[0].ConfigMap.Name != "somesuch" {
			t.Fatalf("expected custom config first, got %v", spec.Volumes[0].Projected.Sources)
		}
	}

	{
		// Mount into existing containers, with or without existing mounts.
		spec := v1.PodSpec{
			Containers: []v1.Container{
				{Name: "database"},
				{Name: "database", VolumeMounts: []v1.VolumeMount{
					{Name: "already"},
				}},
				{Name: "database", VolumeMounts: []v1.VolumeMount{
					{Name: "pgbackrest-config"},
				}},
			},
		}

		AddBackRestConfigVolumeAndMounts(&spec, "cname", nil)

		if expected, actual := 3, len(spec.Containers); expected != actual {
			t.Fatalf("expected no new containers, got %v", actual)
		}
		if expected, actual := 1, len(spec.Volumes); expected != actual {
			t.Fatalf("expected a new volume, got %v", actual)
		}

		if expected, actual := 1, len(spec.Containers[0].VolumeMounts); expected != actual {
			t.Fatalf("expected a new mount, got %v", actual)
		}
		if !strings.Contains(spec.Containers[0].VolumeMounts[0].MountPath, "pgbackrest") {
			t.Fatalf("expected new mount to be for pgbackrest, got %#v", spec.Containers[0].VolumeMounts[0])
		}

		if expected, actual := 2, len(spec.Containers[1].VolumeMounts); expected != actual {
			t.Fatalf("expected a new mount, got %v", actual)
		}
		if !strings.Contains(spec.Containers[1].VolumeMounts[1].MountPath, "pgbackrest") {
			t.Fatalf("expected new mount to be for pgbackrest, got %#v", spec.Containers[1].VolumeMounts[0])
		}

		if expected, actual := 1, len(spec.Containers[2].VolumeMounts); expected != actual {
			t.Fatalf("expected no new mounts, got %v", actual)
		}
		if !strings.Contains(spec.Containers[2].VolumeMounts[0].MountPath, "pgbackrest") {
			t.Fatalf("expected existing mount to be updated, got %#v", spec.Containers[2].VolumeMounts[0])
		}
	}
}
