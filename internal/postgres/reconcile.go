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
	"context"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// InitCopyReplicationTLS copies the mounted client certificate, key and CA certificate files
// from the /pgconf/tls/replication directory to the /tmp/replication directory in order
// to set proper file permissions. This is required because the group permission settings
// applied via the defaultMode option are not honored as expected, resulting in incorrect
// group read permissions.
// See https://github.com/kubernetes/kubernetes/issues/57923
// TODO(tjmoore4): remove this implementation when/if defaultMode permissions are set as
// expected for the mounted volume.
func InitCopyReplicationTLS(postgresCluster *v1beta1.PostgresCluster,
	template *v1.PodTemplateSpec) {

	cmd := fmt.Sprintf(`mkdir -p %s && install -m 0600 %s/{%s,%s,%s} %s`,
		naming.ReplicationTmp, naming.CertMountPath+naming.ReplicationDirectory,
		naming.ReplicationCert, naming.ReplicationPrivateKey,
		naming.ReplicationCACert, naming.ReplicationTmp)
	template.Spec.InitContainers = append(template.Spec.InitContainers,
		v1.Container{
			Command:         []string{"bash", "-c", cmd},
			Image:           postgresCluster.Spec.Image,
			Name:            naming.ContainerClientCertInit,
			SecurityContext: initialize.RestrictedSecurityContext(),
		})
}

// AddCertVolumeToPod adds the secret containing the TLS certificate, key and the CA certificate
// as a volume to the provided Pod template spec, while also adding associated volume mounts to
// the database container specified.
func AddCertVolumeToPod(postgresCluster *v1beta1.PostgresCluster, template *v1.PodTemplateSpec,
	initContainerName, dbContainerName, sidecarContainerName string, inClusterCertificates,
	inClientCertificates *v1.SecretProjection) error {

	certVolume := v1.Volume{Name: naming.CertVolume}
	certVolume.Projected = &v1.ProjectedVolumeSource{
		DefaultMode: initialize.Int32(0o600),
	}

	// Add the certificate volume projection
	certVolume.Projected.Sources = append(append(
		certVolume.Projected.Sources, []v1.VolumeProjection(nil)...),
		[]v1.VolumeProjection{
			{Secret: inClusterCertificates},
			{Secret: inClientCertificates}}...)

	template.Spec.Volumes = append(template.Spec.Volumes, certVolume)

	var dbContainerFound bool
	var sidecarContainerFound bool
	var index int
	for index = range template.Spec.Containers {
		if template.Spec.Containers[index].Name == dbContainerName {
			dbContainerFound = true

			template.Spec.Containers[index].VolumeMounts =
				append(template.Spec.Containers[index].VolumeMounts, v1.VolumeMount{
					Name:      naming.CertVolume,
					MountPath: naming.CertMountPath,
					ReadOnly:  true,
				})
		}
		if template.Spec.Containers[index].Name == sidecarContainerName {
			sidecarContainerFound = true

			template.Spec.Containers[index].VolumeMounts =
				append(template.Spec.Containers[index].VolumeMounts, v1.VolumeMount{
					Name:      naming.CertVolume,
					MountPath: naming.CertMountPath,
					ReadOnly:  true,
				})
		}
		if dbContainerFound && sidecarContainerFound {
			break
		}
	}
	if !dbContainerFound {
		return errors.Errorf("Unable to find container %q when adding certificate volumes",
			dbContainerName)
	}
	if !sidecarContainerFound {
		return errors.Errorf("Unable to find container %q when adding certificate volumes",
			sidecarContainerName)
	}

	var initContainerFound bool
	var initIndex int
	for initIndex = range template.Spec.InitContainers {
		if template.Spec.InitContainers[initIndex].Name == initContainerName {
			initContainerFound = true
			break
		}
	}
	if !initContainerFound {
		return fmt.Errorf("Unable to find init container %q when adding certificate volumes",
			initContainerName)
	}

	template.Spec.InitContainers[initIndex].VolumeMounts =
		append(template.Spec.InitContainers[initIndex].VolumeMounts, v1.VolumeMount{
			Name:      naming.CertVolume,
			MountPath: naming.CertMountPath,
			ReadOnly:  true,
		})

	return nil
}

// DataVolumeMount returns the name and mount path of the PostgreSQL data volume.
func DataVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{Name: "postgres-data", MountPath: dataMountPath}
}

// WALVolumeMount returns the name and mount path of the PostgreSQL WAL volume.
func WALVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{Name: "postgres-wal", MountPath: walMountPath}
}

// InstancePod initializes outInstancePod with the database container and the
// volumes needed by PostgreSQL.
func InstancePod(ctx context.Context,
	inCluster *v1beta1.PostgresCluster,
	inInstanceSpec *v1beta1.PostgresInstanceSetSpec,
	inDataVolume, inWALVolume *corev1.PersistentVolumeClaim,
	outInstancePod *corev1.PodSpec,
) {
	dataVolumeMount := DataVolumeMount()
	dataVolume := corev1.Volume{
		Name: dataVolumeMount.Name,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: inDataVolume.Name,
				ReadOnly:  false,
			},
		},
	}

	container := corev1.Container{
		Name: naming.ContainerDatabase,

		// Patroni will set the command and probes.

		Env:       Environment(inCluster),
		Image:     inCluster.Spec.Image,
		Resources: inInstanceSpec.Resources,

		Ports: []corev1.ContainerPort{{
			Name:          naming.PortPostgreSQL,
			ContainerPort: *inCluster.Spec.Port,
			Protocol:      corev1.ProtocolTCP,
		}},

		VolumeMounts:    []corev1.VolumeMount{dataVolumeMount},
		SecurityContext: initialize.RestrictedSecurityContext(),
	}

	startup := corev1.Container{
		Name: naming.ContainerPostgresStartup,

		Command:   startupCommand(inCluster, inInstanceSpec),
		Env:       Environment(inCluster),
		Image:     inCluster.Spec.Image,
		Resources: inInstanceSpec.Resources,

		SecurityContext: initialize.RestrictedSecurityContext(),
		VolumeMounts:    []corev1.VolumeMount{dataVolumeMount},
	}

	outInstancePod.Volumes = []corev1.Volume{dataVolume}

	if inWALVolume != nil {
		walVolumeMount := WALVolumeMount()
		walVolume := corev1.Volume{
			Name: walVolumeMount.Name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: inWALVolume.Name,
					ReadOnly:  false,
				},
			},
		}

		container.VolumeMounts = append(container.VolumeMounts, walVolumeMount)
		startup.VolumeMounts = append(startup.VolumeMounts, walVolumeMount)
		outInstancePod.Volumes = append(outInstancePod.Volumes, walVolume)
	}

	outInstancePod.Containers = []corev1.Container{container}
	outInstancePod.InitContainers = []corev1.Container{startup}
}
