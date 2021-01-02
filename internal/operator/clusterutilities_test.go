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
	"fmt"
	"testing"

	"github.com/crunchydata/postgres-operator/internal/config"

	v1 "k8s.io/api/core/v1"
)

func mockSetupContainers(values map[string]struct {
	name  string
	image string
}) []v1.Container {
	containers := []v1.Container{}

	for _, value := range values {
		container := v1.Container{
			Name:  value.name,
			Image: value.image,
		}

		containers = append(containers, container)
	}

	return containers
}

func TestOverrideClusterContainerImages(t *testing.T) {

	containerDefaults := map[string]struct {
		name  string
		image string
	}{
		"database":   {name: "database", image: config.CONTAINER_IMAGE_CRUNCHY_POSTGRES_HA},
		"crunchyadm": {name: "crunchyadm", image: config.CONTAINER_IMAGE_CRUNCHY_ADMIN},
		"collect":    {name: "collect", image: config.CONTAINER_IMAGE_CRUNCHY_COLLECT},
		"pgbadger":   {name: "pgbadger", image: config.CONTAINER_IMAGE_CRUNCHY_PGBADGER},
		"future":     {name: "future", image: "crunchy-future"},
	}

	t.Run("no override", func(t *testing.T) {
		containers := mockSetupContainers(containerDefaults)

		OverrideClusterContainerImages(containers)

		for _, container := range containers {
			containerDefault, ok := containerDefaults[container.Name]

			if !ok {
				t.Errorf("could not find container %q", container.Name)
				return
			}

			if containerDefault.image != container.Image {
				t.Errorf("image overwritten when it should not have been. expected %q actual %q",
					containerDefault.image, container.Image)
			}
		}
	})

	// test overriding each container and ensure that it takes in the container
	// slice. Skip the "future" container, that will be in an upcoming test
	for name, defaults := range containerDefaults {
		if name == "future" {
			continue
		}

		t.Run(fmt.Sprintf("override %s", name), func(t *testing.T) {
			// override the struct that contains the value
			ContainerImageOverrides[defaults.image] = "overridden"
			containers := mockSetupContainers(containerDefaults)

			OverrideClusterContainerImages(containers)

			// determine if this container is overridden
			for _, container := range containers {
				containerDefault, ok := containerDefaults[container.Name]

				if !ok {
					t.Errorf("could not find container %q", container.Name)
					return
				}

				if containerDefault.name == name && containerDefault.image == container.Image {
					t.Errorf("container %q not overwritten. image name is %q",
						containerDefault.name, container.Image)
				}
			}
			// unoverride at the end of the test
			delete(ContainerImageOverrides, defaults.image)
		})
	}

	// test that future does not get overridden
	t.Run("do not override unmanaged container", func(t *testing.T) {
		ContainerImageOverrides["crunchy-future"] = "overridden"
		containers := mockSetupContainers(containerDefaults)

		OverrideClusterContainerImages(containers)

		// determine if this container is overridden
		for _, container := range containers {
			containerDefault, ok := containerDefaults[container.Name]

			if !ok {
				t.Errorf("could not find container %q", container.Name)
				return
			}

			if containerDefault.name == "future" && containerDefault.image != container.Image {
				t.Errorf("image overwritten when it should not have been. expected %q actual %q",
					containerDefault.image, container.Image)
			}
		}

		delete(ContainerImageOverrides, "crunchy-future")
	})

	// test that gis can be overridden
	t.Run("override postgis", func(t *testing.T) {
		defaults := containerDefaults

		defaults["database"] = struct {
			name  string
			image string
		}{
			name:  "database",
			image: config.CONTAINER_IMAGE_CRUNCHY_POSTGRES_GIS_HA,
		}
		containers := mockSetupContainers(defaults)

		ContainerImageOverrides[config.CONTAINER_IMAGE_CRUNCHY_POSTGRES_GIS_HA] = "overridden"

		OverrideClusterContainerImages(containers)

		// determine if this container is overridden
		for _, container := range containers {
			containerDefault, ok := containerDefaults[container.Name]

			if !ok {
				t.Errorf("could not find container %q", container.Name)
				return
			}

			if containerDefault.name == "database" && containerDefault.image == container.Image {
				t.Errorf("container %q not overwritten. image name is %q",
					containerDefault.name, container.Image)
			}
		}

		delete(ContainerImageOverrides, config.CONTAINER_IMAGE_CRUNCHY_POSTGRES_GIS_HA)
	})
}
