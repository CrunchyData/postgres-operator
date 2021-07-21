/*
 Copyright 2021 Crunchy Data Solutions, Inc.
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

package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDefaultFromEnv(t *testing.T) {

	t.Run("env var set, value given", func(t *testing.T) {

		// TODO(tjmoore4): The uses of os.Setenv/os.Unsetenv should be repaced
		// once we move to Go 1.17.
		// Go 1.17 has a testing.T.Setenv() that makes sure to
		// undo the effect when the test ends.
		// This should allow the use of Unsetenv to be removed below.
		// https://tip.golang.org/doc/go1.17#testing
		// https://github.com/golang/go/blob/3e48c0381fd1/src/testing/testing.go#L986
		os.Setenv("TEST_ENV_VAR", "testEnvValue")
		assert.Equal(t, defaultFromEnv("testValue", "TEST_ENV_VAR"), "testValue")
	})

	t.Run("env var set, value not given", func(t *testing.T) {

		os.Setenv("TEST_ENV_VAR", "testEnvValue")
		assert.Equal(t, defaultFromEnv("", "TEST_ENV_VAR"), "testEnvValue")
	})

	t.Run("env var not set, value given", func(t *testing.T) {

		os.Unsetenv("TEST_ENV_VAR")
		assert.Equal(t, defaultFromEnv("testValue", "TEST_ENV_VAR"), "testValue")
	})

	t.Run("env var not set, value not given", func(t *testing.T) {

		os.Unsetenv("TEST_ENV_VAR")
		assert.Equal(t, defaultFromEnv("", "TEST_ENV_VAR"), "")
	})
}

func TestPGBackRestContainerImage(t *testing.T) {

	// set up test testcluster
	testcluster := &v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testcluster",
			Namespace: "testnamespace",
		},
		Spec: v1beta1.PostgresClusterSpec{
			Metadata:        &v1beta1.Metadata{},
			PostgresVersion: 12,
			PostGISVersion:  "3.0",
			InstanceSets:    []v1beta1.PostgresInstanceSetSpec{},
			Backups: v1beta1.Backups{
				PGBackRest: v1beta1.PGBackRestArchive{},
			},
		},
	}

	t.Run("env var not set and spec empty", func(t *testing.T) {

		os.Unsetenv(CrunchyPGBackRest)
		assert.Equal(t, PGBackRestContainerImage(testcluster), "")
	})

	t.Run("get image from env var when image missing in spec", func(t *testing.T) {

		os.Setenv(CrunchyPGBackRest, "envVarPGBackRestImage")
		assert.Equal(t, PGBackRestContainerImage(testcluster), "envVarPGBackRestImage")
	})

	t.Run("image name from spec", func(t *testing.T) {

		os.Setenv(CrunchyPGBackRest, "envVarPGBackRestImage")
		testcluster.Spec.Backups.PGBackRest.Image = "specPGBackRestImage"
		assert.Equal(t, PGBackRestContainerImage(testcluster), "specPGBackRestImage")
	})

	t.Run("env var and spec empty", func(t *testing.T) {

		os.Setenv(CrunchyPGBackRest, "")
		testcluster.Spec.Backups.PGBackRest.Image = ""
		assert.Equal(t, PGBackRestContainerImage(testcluster), "")
	})
}

func TestPGBouncerContainerImage(t *testing.T) {

	// set up test testcluster
	testcluster := &v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testcluster",
			Namespace: "testnamespace",
		},
		Spec: v1beta1.PostgresClusterSpec{
			Metadata:        &v1beta1.Metadata{},
			PostgresVersion: 12,
			InstanceSets:    []v1beta1.PostgresInstanceSetSpec{},
			Proxy: &v1beta1.PostgresProxySpec{
				PGBouncer: &v1beta1.PGBouncerPodSpec{},
			},
		},
	}

	t.Run("env var not set and spec empty", func(t *testing.T) {

		os.Unsetenv(CrunchyPGBouncer)
		assert.Equal(t, PGBouncerContainerImage(testcluster), "")
	})

	t.Run("get image from env var when image missing in spec", func(t *testing.T) {

		os.Setenv(CrunchyPGBouncer, "envVarPGBouncerImage")
		assert.Equal(t, PGBouncerContainerImage(testcluster), "envVarPGBouncerImage")
	})

	t.Run("image name from spec", func(t *testing.T) {

		os.Setenv(CrunchyPGBouncer, "envVarPGBouncerImage")
		testcluster.Spec.Proxy.PGBouncer.Image = "specPGBouncerImage"
		assert.Equal(t, PGBouncerContainerImage(testcluster), "specPGBouncerImage")
	})

	t.Run("env var and spec empty", func(t *testing.T) {

		os.Setenv(CrunchyPGBouncer, "")
		testcluster.Spec.Proxy.PGBouncer.Image = ""
		assert.Equal(t, PGBouncerContainerImage(testcluster), "")
	})
}

func TestPGExporterContainerImage(t *testing.T) {

	// set up testcluster
	testcluster := &v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testcluster",
			Namespace: "testnamespace",
		},
		Spec: v1beta1.PostgresClusterSpec{
			Metadata:        &v1beta1.Metadata{},
			PostgresVersion: 12,
			InstanceSets:    []v1beta1.PostgresInstanceSetSpec{},
			Monitoring: &v1beta1.MonitoringSpec{
				PGMonitor: &v1beta1.PGMonitorSpec{
					Exporter: &v1beta1.ExporterSpec{},
				},
			},
		},
	}

	t.Run("env var not set and spec empty", func(t *testing.T) {

		os.Unsetenv(CrunchyPGExporter)
		assert.Equal(t, PGExporterContainerImage(testcluster), "")
	})

	t.Run("get image from env var when image missing in spec", func(t *testing.T) {

		os.Setenv(CrunchyPGExporter, "envVarPGExporterImage")
		assert.Equal(t, PGExporterContainerImage(testcluster), "envVarPGExporterImage")
	})

	t.Run("image name from spec", func(t *testing.T) {

		os.Setenv(CrunchyPGExporter, "envVarPGExporterImage")
		testcluster.Spec.Monitoring.PGMonitor.Exporter.Image = "specPGExporterImage"
		assert.Equal(t, PGExporterContainerImage(testcluster), "specPGExporterImage")
	})

	t.Run("env var and spec empty", func(t *testing.T) {

		os.Setenv(CrunchyPGExporter, "")
		testcluster.Spec.Monitoring.PGMonitor.Exporter.Image = ""
		assert.Equal(t, PGExporterContainerImage(testcluster), "")
	})
}

func TestPostgresContainerImage(t *testing.T) {

	// set up testcluster
	testcluster := &v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testcluster",
			Namespace: "testnamespace",
		},
		Spec: v1beta1.PostgresClusterSpec{
			Metadata:        &v1beta1.Metadata{},
			PostgresVersion: 12,
			InstanceSets:    []v1beta1.PostgresInstanceSetSpec{},
		},
	}

	t.Run("env var not set and spec empty", func(t *testing.T) {

		assert.Equal(t, PostgresContainerImage(testcluster), "")
	})

	t.Run("env var and spec empty", func(t *testing.T) {

		os.Setenv("RELATED_IMAGE_POSTGRES_12", "")
		assert.Equal(t, PostgresContainerImage(testcluster), "")
	})

	t.Run("check env var when image missing in spec", func(t *testing.T) {

		os.Setenv("RELATED_IMAGE_POSTGRES_12", "envVarImage")
		fmt.Printf("IMAGE RETURNED: %s\n", PostgresContainerImage(testcluster))
		assert.Equal(t, PostgresContainerImage(testcluster), "envVarImage")
	})

	t.Run("return image name from spec", func(t *testing.T) {

		testcluster.Spec.Image = "specImageName"
		os.Setenv("RELATED_IMAGE_POSTGRES_12", "envVarImage")
		assert.Equal(t, PostgresContainerImage(testcluster), "specImageName")
	})

	t.Run("return GIS image name from env var", func(t *testing.T) {

		testcluster.Spec.Image = ""
		testcluster.Spec.PostGISVersion = "3.0"

		os.Setenv("RELATED_IMAGE_POSTGRES_12_GIS_3.0", "envVarGISImage")
		assert.Equal(t, PostgresContainerImage(testcluster), "envVarGISImage")
	})

	t.Run("return GIS image name from spec", func(t *testing.T) {

		testcluster.Spec.Image = "specGISImageName"
		testcluster.Spec.PostGISVersion = "3.0"
		os.Setenv("RELATED_IMAGE_POSTGRES_12_GIS_3.0", "envVarGISImage")
		assert.Equal(t, PostgresContainerImage(testcluster), "specGISImageName")
	})

}
