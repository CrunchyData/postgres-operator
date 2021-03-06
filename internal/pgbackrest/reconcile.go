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

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// AddRepoVolumesToPod adds pgBackRest repository volumes to the provided Pod template spec, while
// also adding associated volume mounts to the containers specified.
func AddRepoVolumesToPod(postgresCluster *v1alpha1.PostgresCluster, template *v1.PodTemplateSpec,
	containerNames ...string) error {

	for _, repoVol := range postgresCluster.Spec.Archive.PGBackRest.Repos {
		template.Spec.Volumes = append(template.Spec.Volumes, v1.Volume{
			Name: repoVol.Name,
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					ClaimName: naming.PGBackRestRepoVolume(postgresCluster,
						repoVol.Name).Name},
			},
		})

		for _, name := range containerNames {
			var containerFound bool
			var index int
			for index = range template.Spec.Containers {
				if template.Spec.Containers[index].Name == name {
					containerFound = true
					break
				}
			}
			if !containerFound {
				return fmt.Errorf("Unable to find container %q when adding pgBackRest repo volumes" +
					name)
			}
			template.Spec.Containers[index].VolumeMounts =
				append(template.Spec.Containers[index].VolumeMounts, v1.VolumeMount{
					Name:      repoVol.Name,
					MountPath: "/pgbackrest/" + repoVol.Name,
				})
		}
	}

	return nil
}

// AddConfigsToPod populates a Pod template Spec with with pgBackRest configuration volumes while
// then mounting that configuration to the specified containers.
func AddConfigsToPod(postgresCluster *v1alpha1.PostgresCluster, template *v1.PodTemplateSpec,
	configName string, containerNames ...string) error {

	// grab user provided configs
	pgBackRestConfigs := postgresCluster.Spec.Archive.PGBackRest.Configuration
	// add default pgbackrest configs
	defaultConfig := v1.VolumeProjection{
		ConfigMap: &v1.ConfigMapProjection{
			LocalObjectReference: v1.LocalObjectReference{
				Name: naming.PGBackRestConfig(postgresCluster).Name,
			},
			Items: []v1.KeyToPath{{Key: configName, Path: configName}},
		},
	}
	pgBackRestConfigs = append(pgBackRestConfigs, defaultConfig)

	template.Spec.Volumes = append(template.Spec.Volumes, v1.Volume{
		Name: ConfigVol,
		VolumeSource: v1.VolumeSource{
			Projected: &v1.ProjectedVolumeSource{
				Sources: pgBackRestConfigs,
			},
		},
	})

	for _, name := range containerNames {
		var containerFound bool
		var index int
		for index = range template.Spec.Containers {
			if template.Spec.Containers[index].Name == name {
				containerFound = true
				break
			}
		}
		if !containerFound {
			return fmt.Errorf("Unable to find container %q when adding pgBackRest configration" +
				name)
		}
		template.Spec.Containers[index].VolumeMounts =
			append(template.Spec.Containers[index].VolumeMounts,
				v1.VolumeMount{
					Name:      ConfigVol,
					MountPath: ConfigDir,
				})
	}

	return nil
}

// AddSSHToPod populates a Pod template Spec with with the container and volumes needed to enable
// SSH within a Pod.  It will also mount the SSH configuration to any additional containers specified.
func AddSSHToPod(postgresCluster *v1alpha1.PostgresCluster, template *v1.PodTemplateSpec,
	additionalVolumeMountContainers ...string) error {

	sshConfigs := []v1.VolumeProjection{}
	// stores all SSH configurations (ConfigMaps & Secrets)
	if postgresCluster.Spec.Archive.PGBackRest.RepoHost.SSHConfiguration == nil {
		sshConfigs = append(sshConfigs, v1.VolumeProjection{
			ConfigMap: &v1.ConfigMapProjection{
				LocalObjectReference: v1.LocalObjectReference{
					Name: naming.PGBackRestSSHConfig(postgresCluster).Name,
				},
			},
		})
	} else {
		sshConfigs = append(sshConfigs, v1.VolumeProjection{
			ConfigMap: postgresCluster.Spec.Archive.PGBackRest.RepoHost.SSHConfiguration,
		})
	}
	if postgresCluster.Spec.Archive.PGBackRest.RepoHost.SSHSecret == nil {
		sshConfigs = append(sshConfigs, v1.VolumeProjection{
			Secret: &v1.SecretProjection{
				LocalObjectReference: v1.LocalObjectReference{
					Name: naming.PGBackRestSSHSecret(postgresCluster).Name,
				},
			},
		})
	} else {
		sshConfigs = append(sshConfigs, v1.VolumeProjection{
			Secret: postgresCluster.Spec.Archive.PGBackRest.RepoHost.SSHSecret,
		})
	}
	mode := int32(0040)
	template.Spec.Volumes = append(template.Spec.Volumes, v1.Volume{
		Name: naming.PGBackRestSSHVolume,
		VolumeSource: v1.VolumeSource{
			Projected: &v1.ProjectedVolumeSource{
				Sources:     sshConfigs,
				DefaultMode: &mode,
			},
		},
	})

	sshVolumeMount := v1.VolumeMount{
		Name:      naming.PGBackRestSSHVolume,
		MountPath: sshConfigPath,
		ReadOnly:  true,
	}

	template.Spec.Containers = append(template.Spec.Containers,
		v1.Container{
			Command: []string{"/usr/sbin/sshd", "-D", "-e"},
			Image:   postgresCluster.Spec.Archive.PGBackRest.RepoHost.Image,
			LivenessProbe: &v1.Probe{
				Handler: v1.Handler{
					TCPSocket: &v1.TCPSocketAction{
						Port: intstr.FromInt(2022),
					},
				},
			},
			Name:         naming.PGBackRestRepoContainerName,
			VolumeMounts: []v1.VolumeMount{sshVolumeMount},
		})

	for _, name := range additionalVolumeMountContainers {
		var containerFound bool
		var index int
		for index = range template.Spec.Containers {
			if template.Spec.Containers[index].Name == name {
				containerFound = true
				break
			}
		}
		if !containerFound {
			return fmt.Errorf("Unable to find container %q when adding pgBackRest to Pod" +
				name)
		}
		template.Spec.Containers[index].VolumeMounts =
			append(template.Spec.Containers[index].VolumeMounts, sshVolumeMount)
	}

	return nil
}
