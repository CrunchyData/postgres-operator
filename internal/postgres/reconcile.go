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

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

const (
	// https://www.postgresql.org/docs/current/ssl-tcp.html
	clusterCertFile = "tls.crt"
	clusterKeyFile  = "tls.key"
	rootCertFile    = "ca.crt"
)

// AddPGDATAVolumeToPod adds pgBackRest repository volumes to the provided Pod template spec, while
// also adding associated volume mounts to the containers and/or init containers specified.
func AddPGDATAVolumeToPod(postgresCluster *v1beta1.PostgresCluster, template *v1.PodTemplateSpec,
	claimName string, containerNames, initContainerNames []string) error {

	if claimName == "" {
		return errors.WithStack(errors.New("claimName must not be empty"))
	}

	pgdataVolume := v1.Volume{
		Name: naming.PGDATAVolume,
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
				ClaimName: claimName,
			},
		},
	}
	template.Spec.Volumes = append(template.Spec.Volumes, pgdataVolume)

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
			return fmt.Errorf("Unable to find container %q when adding pgBackRest repo volumes",
				name)
		}
		template.Spec.Containers[index].VolumeMounts =
			append(template.Spec.Containers[index].VolumeMounts, v1.VolumeMount{
				Name:      naming.PGDATAVolume,
				MountPath: naming.PGDATAVMountPath,
			})
	}

	for _, name := range initContainerNames {
		var initContainerFound bool
		var initIndex int
		for initIndex = range template.Spec.InitContainers {
			if template.Spec.InitContainers[initIndex].Name == name {
				initContainerFound = true
				break
			}
		}
		if !initContainerFound {
			return fmt.Errorf("Unable to find init container %q when adding pgBackRest repo volumes",
				name)
		}
		template.Spec.InitContainers[initIndex].VolumeMounts =
			append(template.Spec.InitContainers[initIndex].VolumeMounts, v1.VolumeMount{
				Name:      naming.PGDATAVolume,
				MountPath: naming.PGDATAVMountPath,
			})
	}

	return nil
}

// AddPGDATAInitToPod adds an initialization container to the Pod template that is responsible
// for properly initializing the PGDATA directory.
func AddPGDATAInitToPod(postgresCluster *v1beta1.PostgresCluster,
	template *v1.PodTemplateSpec) {

	pgdata := naming.GetPGDATADirectory(postgresCluster)
	cmd := fmt.Sprintf(`mkdir -p "%s" && chmod 0700 "%s"`, pgdata, pgdata)
	template.Spec.InitContainers = append(template.Spec.InitContainers,
		v1.Container{
			Command: []string{"bash", "-c", cmd},
			Image:   postgresCluster.Spec.Image,
			Name:    naming.ContainerDatabasePGDATAInit,
		})
}

// AddCertVolumeToPod adds the secret containing the TLS certificate, key and the CA certificate
// as a volume to the provided Pod template spec, while also adding associated volume mounts to
// the database container specified.
func AddCertVolumeToPod(postgresCluster *v1beta1.PostgresCluster, template *v1.PodTemplateSpec,
	containerName string, inClusterCertificates *v1.SecretProjection) error {

	certVolume := v1.Volume{Name: naming.CertVolume}
	certVolume.Projected = &v1.ProjectedVolumeSource{
		DefaultMode: initialize.Int32(0o600),
	}

	// Add the certificate volume projection
	certVolume.Projected.Sources = append(append(
		certVolume.Projected.Sources, []v1.VolumeProjection(nil)...),
		[]v1.VolumeProjection{{
			Secret: inClusterCertificates}}...)

	template.Spec.Volumes = append(template.Spec.Volumes, certVolume)

	var containerFound bool
	var index int
	for index = range template.Spec.Containers {
		if template.Spec.Containers[index].Name == containerName {
			containerFound = true
			break
		}
	}
	if !containerFound {
		return fmt.Errorf("Unable to find container %q when adding certificate volumes",
			containerName)
	}

	template.Spec.Containers[index].VolumeMounts =
		append(template.Spec.Containers[index].VolumeMounts, v1.VolumeMount{
			Name:      naming.CertVolume,
			MountPath: naming.CertMountPath,
			ReadOnly:  true,
		})

	return nil
}
