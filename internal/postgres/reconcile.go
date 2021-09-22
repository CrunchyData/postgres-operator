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

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// AddCertVolumeToPod adds the secret containing the TLS certificate, key and the CA certificate
// as a volume to the provided Pod template spec, while also adding associated volume mounts to
// the database container specified.
func AddCertVolumeToPod(postgresCluster *v1beta1.PostgresCluster, template *v1.PodTemplateSpec,
	sidecarContainerName string) error {

	var sidecarContainerFound bool
	var index int
	for index = range template.Spec.Containers {
		if template.Spec.Containers[index].Name == sidecarContainerName {
			sidecarContainerFound = true

			template.Spec.Containers[index].VolumeMounts =
				append(template.Spec.Containers[index].VolumeMounts, v1.VolumeMount{
					Name:      naming.CertVolume,
					MountPath: naming.CertMountPath,
					ReadOnly:  true,
				})
		}
		if sidecarContainerFound {
			break
		}
	}
	if !sidecarContainerFound {
		return errors.Errorf("Unable to find container %q when adding certificate volumes",
			sidecarContainerName)
	}

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
	inClusterCertificates, inClientCertificates *corev1.SecretProjection,
	inDataVolume, inWALVolume *corev1.PersistentVolumeClaim,
	outInstancePod *corev1.PodSpec,
) {
	certVolumeMount := corev1.VolumeMount{
		Name:      naming.CertVolume,
		MountPath: naming.CertMountPath,
		ReadOnly:  true,
	}
	certVolume := corev1.Volume{
		Name: certVolumeMount.Name,
		VolumeSource: corev1.VolumeSource{
			Projected: &corev1.ProjectedVolumeSource{
				// PostgreSQL expects client certificate keys to not be readable
				// by any other user.
				// - https://www.postgresql.org/docs/current/libpq-ssl.html
				DefaultMode: initialize.Int32(0o600),
				Sources: []corev1.VolumeProjection{
					{Secret: inClusterCertificates},
					{Secret: inClientCertificates},
				},
			},
		},
	}

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

		Env:             Environment(inCluster),
		Image:           config.PostgresContainerImage(inCluster),
		ImagePullPolicy: inCluster.Spec.ImagePullPolicy,
		Resources:       inInstanceSpec.Resources,

		Ports: []corev1.ContainerPort{{
			Name:          naming.PortPostgreSQL,
			ContainerPort: *inCluster.Spec.Port,
			Protocol:      corev1.ProtocolTCP,
		}},

		SecurityContext: initialize.RestrictedSecurityContext(),
		VolumeMounts:    []corev1.VolumeMount{certVolumeMount, dataVolumeMount},
	}

	startup := corev1.Container{
		Name: naming.ContainerPostgresStartup,

		Command: startupCommand(inCluster, inInstanceSpec),
		Env:     Environment(inCluster),

		Image:           container.Image,
		ImagePullPolicy: container.ImagePullPolicy,
		Resources:       container.Resources,
		SecurityContext: initialize.RestrictedSecurityContext(),

		VolumeMounts: []corev1.VolumeMount{certVolumeMount, dataVolumeMount},
	}

	outInstancePod.Volumes = []corev1.Volume{certVolume, dataVolume}

	// Mount the WAL PVC whenever it exists. The startup command will move WAL
	// files to or from this volume according to inInstanceSpec.
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

// PodSecurityContext returns a v1.PodSecurityContext for cluster that can write
// to PersistentVolumes.
func PodSecurityContext(cluster *v1beta1.PostgresCluster) *corev1.PodSecurityContext {
	podSecurityContext := initialize.RestrictedPodSecurityContext()

	// Use the specified supplementary groups except for root. The CRD has
	// similar validation, but we should never emit a PodSpec with that group.
	// - https://docs.k8s.io/concepts/security/pod-security-standards/
	for i := range cluster.Spec.SupplementalGroups {
		if gid := cluster.Spec.SupplementalGroups[i]; gid > 0 {
			podSecurityContext.SupplementalGroups =
				append(podSecurityContext.SupplementalGroups, gid)
		}
	}

	// OpenShift assigns a filesystem group based on a SecurityContextConstraint.
	// Otherwise, set a filesystem group so PostgreSQL can write to files
	// regardless of the UID or GID of a container.
	// - https://cloud.redhat.com/blog/a-guide-to-openshift-and-uids
	// - https://docs.k8s.io/tasks/configure-pod-container/security-context/
	// - https://docs.openshift.com/container-platform/4.8/authentication/managing-security-context-constraints.html
	if cluster.Spec.OpenShift == nil || !*cluster.Spec.OpenShift {
		podSecurityContext.FSGroup = initialize.Int64(26)
	}

	return podSecurityContext
}
