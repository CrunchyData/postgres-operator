// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

var (
	oneMillicore = resource.MustParse("1m")
	oneMebibyte  = resource.MustParse("1Mi")
)

// DataVolumeMount returns the name and mount path of the PostgreSQL data volume.
func DataVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{Name: "postgres-data", MountPath: dataMountPath}
}

// TablespaceVolumeMount returns the name and mount path of the PostgreSQL tablespace data volume.
func TablespaceVolumeMount(tablespaceName string) corev1.VolumeMount {
	return corev1.VolumeMount{Name: "tablespace-" + tablespaceName, MountPath: tablespaceMountPath + "/" + tablespaceName}
}

// WALVolumeMount returns the name and mount path of the PostgreSQL WAL volume.
func WALVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{Name: "postgres-wal", MountPath: walMountPath}
}

// DownwardAPIVolumeMount returns the name and mount path of the DownwardAPI volume.
func DownwardAPIVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      "database-containerinfo",
		MountPath: downwardAPIPath,
		ReadOnly:  true,
	}
}

// AdditionalConfigVolumeMount returns the name and mount path of the additional config files.
func AdditionalConfigVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      "postgres-config",
		MountPath: configMountPath,
		ReadOnly:  true,
	}
}

// InstancePod initializes outInstancePod with the database container and the
// volumes needed by PostgreSQL.
func InstancePod(ctx context.Context,
	inCluster *v1beta1.PostgresCluster,
	inInstanceSpec *v1beta1.PostgresInstanceSetSpec,
	inClusterCertificates, inClientCertificates *corev1.SecretProjection,
	inDataVolume, inWALVolume *corev1.PersistentVolumeClaim,
	inTablespaceVolumes []*corev1.PersistentVolumeClaim,
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

	downwardAPIVolumeMount := DownwardAPIVolumeMount()
	downwardAPIVolume := corev1.Volume{
		Name: downwardAPIVolumeMount.Name,
		VolumeSource: corev1.VolumeSource{
			DownwardAPI: &corev1.DownwardAPIVolumeSource{
				// The paths defined in Items (cpu_limit, cpu_request, etc.)
				// are hard coded in the pgnodemx queries defined by
				// pgMonitor configuration (queries_nodemx.yml)
				// https://github.com/CrunchyData/pgmonitor/blob/master/postgres_exporter/common/queries_nodemx.yml
				Items: []corev1.DownwardAPIVolumeFile{{
					Path: "cpu_limit",
					ResourceFieldRef: &corev1.ResourceFieldSelector{
						ContainerName: naming.ContainerDatabase,
						Resource:      "limits.cpu",
						Divisor:       oneMillicore,
					},
				}, {
					Path: "cpu_request",
					ResourceFieldRef: &corev1.ResourceFieldSelector{
						ContainerName: naming.ContainerDatabase,
						Resource:      "requests.cpu",
						Divisor:       oneMillicore,
					},
				}, {
					Path: "mem_limit",
					ResourceFieldRef: &corev1.ResourceFieldSelector{
						ContainerName: naming.ContainerDatabase,
						Resource:      "limits.memory",
						Divisor:       oneMebibyte,
					},
				}, {
					Path: "mem_request",
					ResourceFieldRef: &corev1.ResourceFieldSelector{
						ContainerName: naming.ContainerDatabase,
						Resource:      "requests.memory",
						Divisor:       oneMebibyte,
					},
				}, {
					Path: "labels",
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: corev1.SchemeGroupVersion.Version,
						FieldPath:  "metadata.labels",
					},
				}, {
					Path: "annotations",
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: corev1.SchemeGroupVersion.Version,
						FieldPath:  "metadata.annotations",
					},
				}},
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
		VolumeMounts: []corev1.VolumeMount{
			certVolumeMount,
			dataVolumeMount,
			downwardAPIVolumeMount,
		},
	}

	reloader := corev1.Container{
		Name: naming.ContainerClientCertCopy,

		Command: reloadCommand(naming.ContainerClientCertCopy),

		Image:           container.Image,
		ImagePullPolicy: container.ImagePullPolicy,
		SecurityContext: initialize.RestrictedSecurityContext(),

		VolumeMounts: []corev1.VolumeMount{certVolumeMount, dataVolumeMount},
	}

	if inInstanceSpec.Sidecars != nil &&
		inInstanceSpec.Sidecars.ReplicaCertCopy != nil &&
		inInstanceSpec.Sidecars.ReplicaCertCopy.Resources != nil {
		reloader.Resources = *inInstanceSpec.Sidecars.ReplicaCertCopy.Resources
	}

	startup := corev1.Container{
		Name: naming.ContainerPostgresStartup,

		Command: startupCommand(ctx, inCluster, inInstanceSpec),
		Env:     Environment(inCluster),

		Image:           container.Image,
		ImagePullPolicy: container.ImagePullPolicy,
		Resources:       container.Resources,
		SecurityContext: initialize.RestrictedSecurityContext(),

		VolumeMounts: []corev1.VolumeMount{certVolumeMount, dataVolumeMount},
	}

	outInstancePod.Volumes = []corev1.Volume{
		certVolume,
		dataVolume,
		downwardAPIVolume,
	}

	// If `TablespaceVolumes` FeatureGate is enabled, `inTablespaceVolumes` may not be nil.
	// In that case, add any tablespace volumes to the pod, and
	// add volumeMounts to the database and startup containers
	for _, vol := range inTablespaceVolumes {
		tablespaceVolumeMount := TablespaceVolumeMount(vol.Labels[naming.LabelData])
		tablespaceVolume := corev1.Volume{
			Name: tablespaceVolumeMount.Name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: vol.Name,
					ReadOnly:  false,
				},
			},
		}
		outInstancePod.Volumes = append(outInstancePod.Volumes, tablespaceVolume)
		container.VolumeMounts = append(container.VolumeMounts, tablespaceVolumeMount)
		startup.VolumeMounts = append(startup.VolumeMounts, tablespaceVolumeMount)
	}

	if len(inCluster.Spec.Config.Files) != 0 {
		additionalConfigVolumeMount := AdditionalConfigVolumeMount()
		additionalConfigVolume := corev1.Volume{Name: additionalConfigVolumeMount.Name}
		additionalConfigVolume.Projected = &corev1.ProjectedVolumeSource{
			Sources: append([]corev1.VolumeProjection{}, inCluster.Spec.Config.Files...),
		}
		container.VolumeMounts = append(container.VolumeMounts, additionalConfigVolumeMount)
		outInstancePod.Volumes = append(outInstancePod.Volumes, additionalConfigVolume)
	}

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

	outInstancePod.Containers = []corev1.Container{container, reloader}

	// If the InstanceSidecars feature gate is enabled and instance sidecars are
	// defined, add the defined container to the Pod.
	if feature.Enabled(ctx, feature.InstanceSidecars) &&
		inInstanceSpec.Containers != nil {
		outInstancePod.Containers = append(outInstancePod.Containers, inInstanceSpec.Containers...)
	}

	outInstancePod.InitContainers = []corev1.Container{startup}
}

// PodSecurityContext returns a v1.PodSecurityContext for cluster that can write
// to PersistentVolumes.
func PodSecurityContext(cluster *v1beta1.PostgresCluster) *corev1.PodSecurityContext {
	podSecurityContext := initialize.PodSecurityContext()

	// Use the specified supplementary groups except for root. The CRD has
	// similar validation, but we should never emit a PodSpec with that group.
	// - https://docs.k8s.io/concepts/security/pod-security-standards/
	for i := range cluster.Spec.SupplementalGroups {
		if gid := cluster.Spec.SupplementalGroups[i]; gid > 0 {
			podSecurityContext.SupplementalGroups = append(podSecurityContext.SupplementalGroups, gid)
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
