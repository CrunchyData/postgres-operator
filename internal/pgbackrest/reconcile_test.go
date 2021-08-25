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

package pgbackrest

import (
	"fmt"
	"testing"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
	"gotest.tools/v3/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAddRepoVolumesToPod(t *testing.T) {

	postgresCluster := &v1beta1.PostgresCluster{ObjectMeta: metav1.ObjectMeta{Name: "hippo"}}

	testsCases := []struct {
		repos      []v1beta1.PGBackRestRepo
		containers []v1.Container
		testMap    map[string]string
	}{{
		repos: []v1beta1.PGBackRestRepo{
			{Name: "repo1", Volume: &v1beta1.RepoPVC{}},
			{Name: "repo2", Volume: &v1beta1.RepoPVC{}},
		},
		containers: []v1.Container{{Name: "database"}, {Name: "pgbackrest"}},
		testMap:    map[string]string{},
	}, {
		repos: []v1beta1.PGBackRestRepo{
			{Name: "repo1", Volume: &v1beta1.RepoPVC{}},
			{Name: "repo2", Volume: &v1beta1.RepoPVC{}},
		},
		containers: []v1.Container{{Name: "database"}},
		testMap:    map[string]string{},
	}, {
		repos:      []v1beta1.PGBackRestRepo{{Name: "repo1", Volume: &v1beta1.RepoPVC{}}},
		containers: []v1.Container{{Name: "database"}, {Name: "pgbackrest"}},
		testMap:    map[string]string{},
	}, {
		repos:      []v1beta1.PGBackRestRepo{{Name: "repo1", Volume: &v1beta1.RepoPVC{}}},
		containers: []v1.Container{{Name: "database"}},
		testMap:    map[string]string{},
	},
		// rerun the same tests, but this time simulate an existing PVC
		{
			repos: []v1beta1.PGBackRestRepo{
				{Name: "repo1", Volume: &v1beta1.RepoPVC{}},
				{Name: "repo2", Volume: &v1beta1.RepoPVC{}},
			},
			containers: []v1.Container{{Name: "database"}, {Name: "pgbackrest"}},
			testMap: map[string]string{
				"repo1": "hippo-repo1",
			},
		}, {
			repos: []v1beta1.PGBackRestRepo{
				{Name: "repo1", Volume: &v1beta1.RepoPVC{}},
				{Name: "repo2", Volume: &v1beta1.RepoPVC{}},
			},
			containers: []v1.Container{{Name: "database"}},
			testMap: map[string]string{
				"repo1": "hippo-repo1",
			},
		}, {
			repos:      []v1beta1.PGBackRestRepo{{Name: "repo1", Volume: &v1beta1.RepoPVC{}}},
			containers: []v1.Container{{Name: "database"}, {Name: "pgbackrest"}},
			testMap: map[string]string{
				"repo1": "hippo-repo1",
			},
		}, {
			repos:      []v1beta1.PGBackRestRepo{{Name: "repo1", Volume: &v1beta1.RepoPVC{}}},
			containers: []v1.Container{{Name: "database"}},
			testMap: map[string]string{
				"repo1": "hippo-repo1",
			},
		}}

	for _, tc := range testsCases {
		t.Run(fmt.Sprintf("repos=%d, containers=%d", len(tc.repos), len(tc.containers)), func(t *testing.T) {
			postgresCluster.Spec.Backups.PGBackRest.Repos = tc.repos
			template := &v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: tc.containers,
				},
			}
			err := AddRepoVolumesToPod(postgresCluster, template, tc.testMap, getContainerNames(tc.containers)...)
			assert.NilError(t, err)

			// verify volumes and volume mounts
			for _, r := range tc.repos {
				var foundVolume bool
				for _, v := range template.Spec.Volumes {
					if v.Name == r.Name && v.VolumeSource.PersistentVolumeClaim.ClaimName ==
						naming.PGBackRestRepoVolume(postgresCluster, r.Name).Name {
						foundVolume = true
						break
					}
				}

				if !foundVolume {
					t.Error(fmt.Errorf("volume %s is missing or invalid", r.Name))
				}

				for _, c := range template.Spec.Containers {
					var foundVolumeMount bool
					for _, vm := range c.VolumeMounts {
						if vm.Name == r.Name && vm.MountPath == "/pgbackrest/"+r.Name {
							foundVolumeMount = true
							break
						}
					}
					if !foundVolumeMount {
						t.Error(fmt.Errorf("volume mount %s is missing or invalid", r.Name))
					}
				}
			}
		})
	}
}

func TestAddConfigsToPod(t *testing.T) {

	postgresCluster := &v1beta1.PostgresCluster{ObjectMeta: metav1.ObjectMeta{Name: "hippo"}}

	testCases := []struct {
		configs    []v1.VolumeProjection
		containers []v1.Container
	}{{
		configs: []v1.VolumeProjection{
			{ConfigMap: &v1.ConfigMapProjection{
				LocalObjectReference: v1.LocalObjectReference{Name: "cust-config.conf"}}},
			{Secret: &v1.SecretProjection{
				LocalObjectReference: v1.LocalObjectReference{Name: "cust-secret.conf"}}}},
		containers: []v1.Container{{Name: "database"}, {Name: "pgbackrest"}},
	}, {
		configs: []v1.VolumeProjection{
			{ConfigMap: &v1.ConfigMapProjection{
				LocalObjectReference: v1.LocalObjectReference{Name: "cust-config.conf"}}},
			{Secret: &v1.SecretProjection{
				LocalObjectReference: v1.LocalObjectReference{Name: "cust-secret.conf"}}}},
		containers: []v1.Container{{Name: "pgbackrest"}},
	}, {
		configs:    []v1.VolumeProjection{},
		containers: []v1.Container{{Name: "database"}, {Name: "pgbackrest"}},
	}, {
		configs:    []v1.VolumeProjection{},
		containers: []v1.Container{{Name: "pgbackrest"}},
	}}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("configs=%d, containers=%d", len(tc.configs), len(tc.containers)), func(t *testing.T) {
			postgresCluster.Spec.Backups.PGBackRest.Configuration = tc.configs
			template := &v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: tc.containers,
				},
			}

			err := AddConfigsToPod(postgresCluster, template, CMInstanceKey,
				getContainerNames(tc.containers)...)
			assert.NilError(t, err)

			// check that the backrest config volume exists
			var configVol *v1.Volume
			var foundConfigVol bool
			for i, v := range template.Spec.Volumes {
				if v.Name == ConfigVol {
					foundConfigVol = true
					configVol = &template.Spec.Volumes[i]
					break
				}
			}
			if !foundConfigVol {
				t.Error(fmt.Errorf("volume %s is missing", ConfigVol))
			}

			// check that the backrest config volume contains default configs
			var foundDefaultConfigMapVol bool
			cmName := naming.PGBackRestConfig(postgresCluster).Name
			for _, s := range configVol.Projected.Sources {
				if s.ConfigMap != nil && s.ConfigMap.Name == cmName {
					foundDefaultConfigMapVol = true
					break
				}
			}
			if !foundDefaultConfigMapVol {
				t.Error(fmt.Errorf("ConfigMap %s is missing", cmName))
			}

			// verify custom configs are present in the backrest config volume
			for _, c := range tc.configs {
				var foundCustomConfig bool
				for _, s := range configVol.Projected.Sources {
					if equality.Semantic.DeepEqual(c, s) {
						foundCustomConfig = true
						break
					}
				}
				assert.Assert(t, foundCustomConfig)
			}

			// verify the containers specified have the proper volume mounts
			for _, c := range template.Spec.Containers {
				var foundVolumeMount bool
				for _, vm := range c.VolumeMounts {
					if vm.Name == ConfigVol && vm.MountPath == ConfigDir {
						foundVolumeMount = true
						break
					}
				}
				assert.Assert(t, foundVolumeMount)
			}
		})
	}
}

func TestAddSSHToPod(t *testing.T) {

	postgresClusterBase := &v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hippo",
		},
		Spec: v1beta1.PostgresClusterSpec{
			Backups: v1beta1.Backups{
				PGBackRest: v1beta1.PGBackRestArchive{
					RepoHost: &v1beta1.PGBackRestRepoHost{},
				},
			},
		},
	}

	resources := v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("250m"),
			v1.ResourceMemory: resource.MustParse("128Mi"),
		},
	}

	testCases := []struct {
		sshConfig               *v1.ConfigMapProjection
		sshSecret               *v1.SecretProjection
		additionalSSHContainers []v1.Container
	}{{
		sshConfig: &v1.ConfigMapProjection{
			LocalObjectReference: v1.LocalObjectReference{Name: "cust-ssh-config.conf"}},
		sshSecret: &v1.SecretProjection{
			LocalObjectReference: v1.LocalObjectReference{Name: "cust-ssh-secret.conf"}},
		additionalSSHContainers: []v1.Container{{Name: "database"}},
	}, {
		additionalSSHContainers: []v1.Container{{Name: "database"}},
	}}

	for _, tc := range testCases {

		customConfig := (tc.sshConfig != nil)
		customSecret := (tc.sshSecret != nil)
		testRunStr := fmt.Sprintf("customConfig=%t, customSecret=%t, additionalSSHContainers=%d",
			customConfig, customSecret, len(tc.additionalSSHContainers))

		postgresCluster := postgresClusterBase.DeepCopy()

		if customConfig {
			postgresCluster.Spec.Backups.PGBackRest.RepoHost.SSHConfiguration = tc.sshConfig
		}
		if customSecret {
			postgresCluster.Spec.Backups.PGBackRest.RepoHost.SSHSecret = tc.sshSecret
		}

		t.Run(testRunStr, func(t *testing.T) {

			template := &v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: tc.additionalSSHContainers,
				},
			}

			err := AddSSHToPod(postgresCluster, template, true, resources,
				getContainerNames(tc.additionalSSHContainers)...)
			assert.NilError(t, err)

			// verify the ssh volume
			var foundSSHVolume bool
			var sshVolume v1.Volume
			for _, v := range template.Spec.Volumes {
				if v.Name == naming.PGBackRestSSHVolume {
					foundSSHVolume = true
					sshVolume = v
					break
				}
			}
			assert.Assert(t, foundSSHVolume)

			// verify the ssh config and secret
			var foundSSHConfigVolume, foundSSHSecretVolume bool
			defaultConfigName := naming.PGBackRestSSHConfig(postgresCluster).Name
			defaultSecretName := naming.PGBackRestSSHSecret(postgresCluster).Name
			for _, s := range sshVolume.Projected.Sources {
				if s.ConfigMap != nil {
					if (!customConfig && s.ConfigMap.Name == defaultConfigName) ||
						(customConfig && s.ConfigMap.Name == tc.sshConfig.Name) {
						foundSSHConfigVolume = true
					}
				} else if s.Secret != nil {
					if (!customSecret && s.Secret.Name == defaultSecretName) ||
						(customSecret && s.Secret.Name == tc.sshSecret.Name) {
						foundSSHSecretVolume = true
					}
				}
			}
			assert.Assert(t, foundSSHConfigVolume)
			assert.Assert(t, foundSSHSecretVolume)

			// verify that pgbackrest container is present and that the proper SSH volume mount in
			// present in all containers
			var foundSSHContainer bool
			for _, c := range template.Spec.Containers {
				if c.Name == naming.PGBackRestRepoContainerName {
					foundSSHContainer = true
					// verify proper resources are present and correct
					assert.DeepEqual(t, c.Resources, resources)
				}
				var foundVolumeMount bool
				for _, vm := range c.VolumeMounts {
					if vm.Name == naming.PGBackRestSSHVolume && vm.MountPath == sshConfigPath &&
						vm.ReadOnly == true {
						foundVolumeMount = true
						break
					}
				}
				assert.Assert(t, foundVolumeMount)
			}
			assert.Assert(t, foundSSHContainer)
		})
	}
}

func getContainerNames(containers []v1.Container) []string {
	names := make([]string, len(containers))
	for i, c := range containers {
		names[i] = c.Name
	}
	return names
}

func TestReplicaCreateCommand(t *testing.T) {
	cluster := new(v1beta1.PostgresCluster)
	instance := new(v1beta1.PostgresInstanceSetSpec)

	t.Run("NoRepositories", func(t *testing.T) {
		assert.Equal(t, 0, len(ReplicaCreateCommand(cluster, instance)))
	})

	t.Run("NoReadyRepositories", func(t *testing.T) {
		cluster.Status.PGBackRest = &v1beta1.PGBackRestStatus{
			Repos: []v1beta1.RepoStatus{{
				Name: "repo2", ReplicaCreateBackupComplete: false,
			}},
		}

		assert.Equal(t, 0, len(ReplicaCreateCommand(cluster, instance)))
	})

	t.Run("SomeReadyRepositories", func(t *testing.T) {
		cluster.Status.PGBackRest = &v1beta1.PGBackRestStatus{
			Repos: []v1beta1.RepoStatus{{
				Name: "repo2", ReplicaCreateBackupComplete: true,
			}, {
				Name: "repo3", ReplicaCreateBackupComplete: true,
			}},
		}

		assert.DeepEqual(t, ReplicaCreateCommand(cluster, instance), []string{
			"pgbackrest", "restore", "--delta", "--stanza=db", "--repo=2",
			"--link-map=pg_wal=/pgdata/pg0_wal",
		})
	})

	t.Run("Standby", func(t *testing.T) {
		cluster := cluster.DeepCopy()
		cluster.Spec.Standby = &v1beta1.PostgresStandbySpec{
			Enabled:  true,
			RepoName: "repo7",
		}

		assert.DeepEqual(t, ReplicaCreateCommand(cluster, instance), []string{
			"pgbackrest", "restore", "--delta", "--stanza=db", "--repo=7",
			"--link-map=pg_wal=/pgdata/pg0_wal",
		})
	})
}
