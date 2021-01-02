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

import (
	"testing"

	v1 "k8s.io/api/core/v1"
)

func TestFindOrAppendVolume(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		var volumes []v1.Volume
		volume := FindOrAppendVolume(&volumes, "v1")
		if expected, actual := 1, len(volumes); expected != actual {
			t.Fatalf("expected appended volume, got %v", actual)
		}
		if volume != &volumes[0] {
			t.Fatal("expected appended volume")
		}
		if expected, actual := "v1", volume.Name; expected != actual {
			t.Fatalf("expected name to be appended, got %q", actual)
		}
	})

	t.Run("missing", func(t *testing.T) {
		volumes := []v1.Volume{{Name: "v1"}, {Name: "v2"}}
		volume := FindOrAppendVolume(&volumes, "v3")
		if expected, actual := 3, len(volumes); expected != actual {
			t.Fatalf("expected appended volume, got %v", actual)
		}
		if volume != &volumes[2] {
			t.Fatal("expected appended volume")
		}
		if expected, actual := "v3", volume.Name; expected != actual {
			t.Fatalf("expected name to be appended, got %q", actual)
		}
	})

	t.Run("present", func(t *testing.T) {
		volumes := []v1.Volume{{Name: "v1"}, {Name: "v2"}}
		volume := FindOrAppendVolume(&volumes, "v2")
		if expected, actual := 2, len(volumes); expected != actual {
			t.Fatalf("expected nothing to be appended, got %v", actual)
		}
		if volume != &volumes[1] {
			t.Fatal("expected existing volume")
		}
	})
}

func TestFindOrAppendVolumeMount(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		var mounts []v1.VolumeMount
		mount := FindOrAppendVolumeMount(&mounts, "v1")
		if expected, actual := 1, len(mounts); expected != actual {
			t.Fatalf("expected appended mount, got %v", actual)
		}
		if mount != &mounts[0] {
			t.Fatal("expected appended mount")
		}
		if expected, actual := "v1", mount.Name; expected != actual {
			t.Fatalf("expected name to be appended, got %q", actual)
		}
	})

	t.Run("missing", func(t *testing.T) {
		mounts := []v1.VolumeMount{{Name: "v1"}, {Name: "v2"}}
		mount := FindOrAppendVolumeMount(&mounts, "v3")
		if expected, actual := 3, len(mounts); expected != actual {
			t.Fatalf("expected appended mount, got %v", actual)
		}
		if mount != &mounts[2] {
			t.Fatal("expected appended mount")
		}
		if expected, actual := "v3", mount.Name; expected != actual {
			t.Fatalf("expected name to be appended, got %q", actual)
		}
	})

	t.Run("present", func(t *testing.T) {
		mounts := []v1.VolumeMount{{Name: "v1"}, {Name: "v2"}}
		mount := FindOrAppendVolumeMount(&mounts, "v2")
		if expected, actual := 2, len(mounts); expected != actual {
			t.Fatalf("expected nothing to be appended, got %v", actual)
		}
		if mount != &mounts[1] {
			t.Fatal("expected existing mount")
		}
	})
}
