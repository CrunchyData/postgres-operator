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

package postgres

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestAddPGDATAVolumeToPod(t *testing.T) {

	postgresClusterBase := &v1beta1.PostgresCluster{ObjectMeta: metav1.ObjectMeta{Name: "hippo"}}
	testClaimName := "test-claim-name"

	testsCases := []struct {
		claimName      string
		containers     []v1.Container
		initContainers []v1.Container
	}{{
		claimName:      testClaimName,
		containers:     []v1.Container{{Name: "database"}, {Name: "pgbackrest"}},
		initContainers: []v1.Container{{Name: "database-pgdata-init"}},
	}, {
		claimName:      testClaimName,
		containers:     []v1.Container{{Name: "database"}},
		initContainers: []v1.Container{{Name: "database-pgdata-init"}},
	}, {
		claimName:  testClaimName,
		containers: []v1.Container{},
	}, {
		claimName:  "", // should cause error
		containers: []v1.Container{},
	}}

	for _, tc := range testsCases {
		t.Run(fmt.Sprintf("claimName=%s, containers=%d", tc.claimName, len(tc.containers)), func(t *testing.T) {

			postgresCluster := postgresClusterBase.DeepCopy()
			template := &v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: tc.containers,
				},
			}

			err := AddPGDATAVolumeToPod(postgresCluster, template, tc.claimName,
				getContainerNames(tc.containers), []string{})
			if tc.claimName == "" {
				assert.ErrorContains(t, err, "must not be empty")
				return
			}
			assert.NilError(t, err)

			var foundPGDATAVol bool
			var pgdataVol *v1.Volume
			for i, v := range template.Spec.Volumes {
				if v.Name == naming.PGDATAVolume {
					foundPGDATAVol = true
					pgdataVol = &template.Spec.Volumes[i]
					break
				}
			}
			assert.Assert(t, foundPGDATAVol)
			assert.Assert(t, pgdataVol.PersistentVolumeClaim != nil)
			assert.Assert(t, (pgdataVol.PersistentVolumeClaim.ClaimName == tc.claimName))

			for _, c := range template.Spec.Containers {
				var foundVolumeMount bool
				for _, vm := range c.VolumeMounts {
					if vm.Name == naming.PGDATAVolume && vm.MountPath == naming.PGDATAVMountPath {
						foundVolumeMount = true
					}
				}
				assert.Assert(t, foundVolumeMount)
			}

			for _, c := range template.Spec.InitContainers {
				var foundVolumeMount bool
				for _, vm := range c.VolumeMounts {
					if vm.Name == naming.PGDATAVolume && vm.MountPath == naming.PGDATAVMountPath {
						foundVolumeMount = true
					}
				}
				assert.Assert(t, foundVolumeMount)
			}
		})
	}
}

func TestAddPGDATAInitToPod(t *testing.T) {

	postgresCluster := &v1beta1.PostgresCluster{ObjectMeta: metav1.ObjectMeta{Name: "hippo"}}
	template := &v1.PodTemplateSpec{}

	AddPGDATAInitToPod(postgresCluster, template)

	var container *v1.Container
	for i, c := range template.Spec.InitContainers {
		if c.Name == naming.ContainerDatabasePGDATAInit {
			container = &template.Spec.InitContainers[i]
		}
	}

	assert.Assert(t, container != nil)

	t.Run("ShellCheck", func(t *testing.T) {
		shellcheck, err := exec.LookPath("shellcheck")
		if err != nil {
			t.Skip(`requires "shellcheck" executable`)
		} else {
			output, err := exec.Command(shellcheck, "--version").CombinedOutput()
			assert.NilError(t, err)
			t.Logf("using %q:\n%s", shellcheck, output)
		}

		// Expect a bash command with an inline script.
		assert.DeepEqual(t, container.Command[:2], []string{"bash", "-c"})
		assert.Assert(t, len(container.Command) > 2)
		script := container.Command[2]

		// Write out that inline script.
		dir := t.TempDir()
		file := filepath.Join(dir, "script.bash")
		assert.NilError(t, ioutil.WriteFile(file, []byte(script), 0o600))

		// Expect shellcheck to be happy.
		cmd := exec.Command(shellcheck, "--enable=all", file)
		output, err := cmd.CombinedOutput()
		assert.NilError(t, err, "%q\n%s", cmd.Args, output)
	})
}

func TestCopyClientTLS(t *testing.T) {

	postgresCluster := &v1beta1.PostgresCluster{ObjectMeta: metav1.ObjectMeta{Name: "hippo"}}
	template := &v1.PodTemplateSpec{}

	CopyReplicationTLS(postgresCluster, template)

	var foundPGDATAInitContainer bool
	for _, c := range template.Spec.InitContainers {
		if c.Name == naming.ContainerClientCertInit {
			foundPGDATAInitContainer = true
			break
		}
	}

	assert.Assert(t, foundPGDATAInitContainer)
}

func TestAddCertVolumeToPod(t *testing.T) {

	postgresCluster := &v1beta1.PostgresCluster{ObjectMeta: metav1.ObjectMeta{Name: "hippo"}}
	template := &v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			Containers: []v1.Container{{
				Name: "database",
			},
			},
			InitContainers: []v1.Container{{
				Name: "database-client-cert-init",
			},
			},
		},
	}
	mode := int32(0o600)
	// example auto-generated secret projection
	testServerSecretProjection := &v1.SecretProjection{
		LocalObjectReference: v1.LocalObjectReference{
			Name: naming.PostgresTLSSecret(postgresCluster).Name,
		},
		Items: []v1.KeyToPath{
			{
				Key:  clusterCertFile,
				Path: clusterCertFile,
				Mode: &mode,
			},
			{
				Key:  clusterKeyFile,
				Path: clusterKeyFile,
				Mode: &mode,
			},
			{
				Key:  rootCertFile,
				Path: rootCertFile,
				Mode: &mode,
			},
		},
	}

	testClientSecretProjection := &v1.SecretProjection{
		LocalObjectReference: v1.LocalObjectReference{
			Name: naming.ReplicationClientCertSecret(postgresCluster).Name,
		},
		Items: []v1.KeyToPath{
			{
				Key:  naming.ReplicationCert,
				Path: naming.ReplicationCertPath,
				Mode: &mode,
			},
			{
				Key:  naming.ReplicationPrivateKey,
				Path: naming.ReplicationPrivateKeyPath,
				Mode: &mode,
			},
		},
	}

	err := AddCertVolumeToPod(postgresCluster, template, naming.ContainerClientCertInit,
		naming.ContainerDatabase, testServerSecretProjection, testClientSecretProjection)
	assert.NilError(t, err)

	var foundCertVol bool
	var certVol *v1.Volume
	for i, v := range template.Spec.Volumes {
		if v.Name == naming.CertVolume {
			foundCertVol = true
			certVol = &template.Spec.Volumes[i]
			break
		}
	}

	assert.Assert(t, foundCertVol)
	assert.Assert(t, len(certVol.Projected.Sources) > 1)

	var serverSecret *v1.SecretProjection
	var clientSecret *v1.SecretProjection

	for _, source := range certVol.Projected.Sources {

		if source.Secret.Name == naming.PostgresTLSSecret(postgresCluster).Name {
			serverSecret = source.Secret
		}
		if source.Secret.Name == naming.ReplicationClientCertSecret(postgresCluster).Name {
			clientSecret = source.Secret
		}
	}

	if assert.Check(t, serverSecret != nil) {
		assert.Assert(t, len(serverSecret.Items) == 3)

		assert.Equal(t, serverSecret.Items[0].Key, clusterCertFile)
		assert.Equal(t, serverSecret.Items[0].Path, clusterCertFile)
		assert.Equal(t, serverSecret.Items[0].Mode, &mode)

		assert.Equal(t, serverSecret.Items[1].Key, clusterKeyFile)
		assert.Equal(t, serverSecret.Items[1].Path, clusterKeyFile)
		assert.Equal(t, serverSecret.Items[1].Mode, &mode)

		assert.Equal(t, serverSecret.Items[2].Key, rootCertFile)
		assert.Equal(t, serverSecret.Items[2].Path, rootCertFile)
		assert.Equal(t, serverSecret.Items[2].Mode, &mode)
	}

	if assert.Check(t, clientSecret != nil) {
		assert.Assert(t, len(clientSecret.Items) == 2)

		assert.Equal(t, clientSecret.Items[0].Key, naming.ReplicationCert)
		assert.Equal(t, clientSecret.Items[0].Path, naming.ReplicationCertPath)
		assert.Equal(t, clientSecret.Items[0].Mode, &mode)

		assert.Equal(t, clientSecret.Items[1].Key, naming.ReplicationPrivateKey)
		assert.Equal(t, clientSecret.Items[1].Path, naming.ReplicationPrivateKeyPath)
		assert.Equal(t, clientSecret.Items[1].Mode, &mode)
	}
}

func getContainerNames(containers []v1.Container) []string {
	names := make([]string, len(containers))
	for i, c := range containers {
		names[i] = c.Name
	}
	return names
}
