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
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/crunchydata/postgres-operator/internal/config"
	fakekubeapi "github.com/crunchydata/postgres-operator/internal/kubeapi/fake"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestGetAnnotations(t *testing.T) {
	cluster := &crv1.Pgcluster{}
	cluster.Spec.Annotations.Global = map[string]string{"global": "yes", "hey": "there"}
	cluster.Spec.Annotations.Postgres = map[string]string{"postgres": "yup", "elephant": "yay"}
	cluster.Spec.Annotations.Backrest = map[string]string{"backrest": "woo"}
	cluster.Spec.Annotations.PgBouncer = map[string]string{"pgbouncer": "yas", "hippo": "awesome"}

	t.Run("annotations empty", func(t *testing.T) {
		cluster := &crv1.Pgcluster{}
		ats := []crv1.ClusterAnnotationType{
			crv1.ClusterAnnotationGlobal,
			crv1.ClusterAnnotationPostgres,
			crv1.ClusterAnnotationBackrest,
			crv1.ClusterAnnotationPgBouncer,
		}

		for _, at := range ats {
			result := GetAnnotations(cluster, at)

			if result != "" {
				t.Errorf("expected empty string, got %q", result)
			}
		}
	})

	tests := []struct {
		testName string
		expected string
		arg      crv1.ClusterAnnotationType
	}{
		{
			testName: "global",
			expected: `{"global":"yes","hey":"there"}`,
			arg:      crv1.ClusterAnnotationGlobal,
		},
		{
			testName: "postgres",
			expected: `{"global":"yes", "hey":"there", "postgres": "yup", "elephant": "yay"}`,
			arg:      crv1.ClusterAnnotationPostgres,
		},
		{
			testName: "pgbackrest",
			expected: `{"global":"yes", "hey":"there", "backrest": "woo"}`,
			arg:      crv1.ClusterAnnotationBackrest,
		},
		{
			testName: "pgbouncer",
			expected: `{"global":"yes", "hey":"there", "pgbouncer": "yas", "hippo": "awesome"}`,
			arg:      crv1.ClusterAnnotationPgBouncer,
		},
	}

	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			var expected, actual interface{}

			if err := json.Unmarshal([]byte(test.expected), &expected); err != nil {
				t.Fatalf("could not unmarshal expected json: %q", err.Error())
			}

			result := GetAnnotations(cluster, test.arg)

			if err := json.Unmarshal([]byte(result), &actual); err != nil {
				t.Fatalf("could not unmarshal actual json: %q", err.Error())
			}

			if !reflect.DeepEqual(expected, actual) {
				t.Errorf("expected %v, got %v", expected, actual)
			}
		})
	}
}

func TestOverrideClusterContainerImages(t *testing.T) {
	containerDefaults := map[string]struct {
		name  string
		image string
	}{
		"database": {name: "database", image: config.CONTAINER_IMAGE_CRUNCHY_POSTGRES_HA},
		"exporter": {name: "exporter", image: config.CONTAINER_IMAGE_CRUNCHY_POSTGRES_EXPORTER},
		"pgbadger": {name: "pgbadger", image: config.CONTAINER_IMAGE_CRUNCHY_PGBADGER},
		"future":   {name: "future", image: "crunchy-future"},
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

func TestGetPgbackrestBootstrapS3EnvVars(t *testing.T) {
	// create a fake client that will be used to "fake" the initialization of the operator for
	// this test
	fakePGOClient, err := fakekubeapi.NewFakePGOClient()
	if err != nil {
		t.Fatal(err)
	}
	// now initialize the operator using the fake client.  This loads various configs, templates,
	// global vars, etc. as needed to run the tests below
	Initialize(fakePGOClient)

	// create a mock backrest repo secret with default values populated for the various S3
	// annotations
	mockBackRestRepoSecret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				config.ANNOTATION_S3_BUCKET:     "bucket",
				config.ANNOTATION_S3_ENDPOINT:   "endpoint",
				config.ANNOTATION_S3_REGION:     "region",
				config.ANNOTATION_S3_URI_STYLE:  "path",
				config.ANNOTATION_S3_VERIFY_TLS: "false",
			},
		},
	}
	defaultRestoreFromCluster := "restoreFromCluster"

	type Env struct {
		EnvVars []v1.EnvVar
	}

	// test all env vars are properly set according the contents of an existing pgBackRest
	// repo secret
	t.Run("populate from secret", func(t *testing.T) {
		backRestRepoSecret := mockBackRestRepoSecret.DeepCopy()
		s3EnvVars := GetPgbackrestBootstrapS3EnvVars(defaultRestoreFromCluster, backRestRepoSecret)
		// massage the results a bit so that we can parse as proper JSON to validate contents
		s3EnvVarsJSON := strings.TrimSuffix(`{"EnvVars": [`+s3EnvVars, ",\n") + "]}"

		s3Env := &Env{}
		if err := json.Unmarshal([]byte(s3EnvVarsJSON), s3Env); err != nil {
			t.Fatal(err)
		}

		for _, envVar := range s3Env.EnvVars {
			validValue := true
			switch envVar.Name {
			case "PGBACKREST_REPO1_S3_BUCKET":
				validValue = (envVar.Value == mockBackRestRepoSecret.
					GetAnnotations()[config.ANNOTATION_S3_BUCKET])
			case "PGBACKREST_REPO1_S3_ENDPOINT":
				validValue = (envVar.Value == mockBackRestRepoSecret.
					GetAnnotations()[config.ANNOTATION_S3_ENDPOINT])
			case "PGBACKREST_REPO1_S3_REGION":
				validValue = (envVar.Value == mockBackRestRepoSecret.
					GetAnnotations()[config.ANNOTATION_S3_REGION])
			case "PGBACKREST_REPO1_S3_URI_STYLE":
				validValue = (envVar.Value == mockBackRestRepoSecret.
					GetAnnotations()[config.ANNOTATION_S3_URI_STYLE])
			case "PGHA_PGBACKREST_S3_VERIFY_TLS":
				validValue = (envVar.Value == mockBackRestRepoSecret.
					GetAnnotations()[config.ANNOTATION_S3_VERIFY_TLS])
			case "PGBACKREST_REPO1_S3_KEY":
				validValue = (envVar.ValueFrom.SecretKeyRef.Name ==
					fmt.Sprintf(util.BackrestRepoSecretName, defaultRestoreFromCluster)) &&
					(envVar.ValueFrom.SecretKeyRef.Key ==
						util.BackRestRepoSecretKeyAWSS3KeyAWSS3Key)
			case "PGBACKREST_REPO1_S3_KEY_SECRET":
				validValue = (envVar.ValueFrom.SecretKeyRef.Name ==
					fmt.Sprintf(util.BackrestRepoSecretName, defaultRestoreFromCluster)) &&
					(envVar.ValueFrom.SecretKeyRef.Key ==
						util.BackRestRepoSecretKeyAWSS3KeyAWSS3KeySecret)
			}
			if !validValue {
				t.Errorf("Invalid value for env var %s", envVar.Name)
			}
		}
	})

	// test that the proper default S3 URI style is set for the bootstrap S3 env vars when the
	// S3 URI style annotation is an empty string in a pgBackRest repo secret
	t.Run("default URI style", func(t *testing.T) {
		// the expected default for the pgBackRest URI style
		defaultURIStyle := "host"

		backRestRepoSecret := mockBackRestRepoSecret.DeepCopy()
		// set the URI style annotation to an empty string so that we can ensure the proper
		// default is set when no URI style annotation value is present
		backRestRepoSecret.GetAnnotations()[config.ANNOTATION_S3_URI_STYLE] = ""

		s3EnvVars := GetPgbackrestBootstrapS3EnvVars("restoreFromCluster", backRestRepoSecret)
		// massage the results a bit so that we can parse as proper JSON to validate contents
		s3EnvVarsJSON := strings.TrimSuffix(`{"EnvVars": [`+s3EnvVars, ",\n") + "]}"

		s3Env := &Env{}
		if err := json.Unmarshal([]byte(s3EnvVarsJSON), s3Env); err != nil {
			t.Error(err)
		}

		validValue := false
		for _, envVar := range s3Env.EnvVars {
			if envVar.Name == "PGBACKREST_REPO1_S3_URI_STYLE" &&
				envVar.Value == defaultURIStyle {
				validValue = true
			}
		}
		if !validValue {
			t.Errorf("Invalid default URI style, it should be '%s'", defaultURIStyle)
		}
	})
}
