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
	"strconv"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/internal/config"
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

		VolumeMounts: []corev1.VolumeMount{certVolumeMount},
	}

	if inInstanceSpec.Sidecars != nil &&
		inInstanceSpec.Sidecars.ReplicaCertCopy != nil &&
		inInstanceSpec.Sidecars.ReplicaCertCopy.Resources != nil {
		reloader.Resources = *inInstanceSpec.Sidecars.ReplicaCertCopy.Resources
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

	outInstancePod.Volumes = []corev1.Volume{
		certVolume,
		dataVolume,
		downwardAPIVolume,
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

// GenerateUpgradeJobIntent creates an pg_upgrade Job to perform a major
// PostgreSQL upgrade.
func GenerateUpgradeJobIntent(
	cluster *v1beta1.PostgresCluster, sa string,
	spec *v1beta1.PostgresInstanceSetSpec,
	inClusterCertificates, inClientCertificates *corev1.SecretProjection,
	inDataVolume, inWALVolume *corev1.PersistentVolumeClaim,
) (batchv1.Job, error) {

	// create the pg_upgrade Job
	upgradeJob := &batchv1.Job{}
	upgradeJob.ObjectMeta = naming.PGUpgradeJob(cluster)

	// set labels and annotations
	var labels, annotations map[string]string
	labels = naming.Merge(cluster.Spec.Metadata.GetLabelsOrNil(),
		cluster.Spec.Upgrade.Metadata.GetLabelsOrNil(),
		naming.PGUpgradeJobLabels(cluster.Name))
	annotations = naming.Merge(cluster.Spec.Metadata.GetAnnotationsOrNil(),
		cluster.Spec.Upgrade.Metadata.GetAnnotationsOrNil(),
		map[string]string{naming.PGUpgradeVersion: strconv.Itoa(cluster.Spec.PostgresVersion)})
	upgradeJob.ObjectMeta.Labels = labels
	upgradeJob.ObjectMeta.Annotations = annotations

	volumeMounts := []corev1.VolumeMount{}
	volumes := []corev1.Volume{}

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
	volumeMounts = append(volumeMounts, certVolumeMount)
	volumes = append(volumes, certVolume)

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
	volumeMounts = append(volumeMounts, dataVolumeMount)
	volumes = append(volumes, dataVolume)

	var walVolume corev1.Volume
	if inWALVolume != nil {
		walVolumeMount := WALVolumeMount()
		walVolume = corev1.Volume{
			Name: walVolumeMount.Name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: inWALVolume.Name,
					ReadOnly:  false,
				},
			},
		}
		volumeMounts = append(volumeMounts, walVolumeMount)
		volumes = append(volumes, walVolume)
	}

	container := corev1.Container{
		Command:         upgradeCommand(cluster),
		Image:           config.PGUpgradeContainerImage(cluster),
		ImagePullPolicy: cluster.Spec.ImagePullPolicy,
		Name:            naming.ContainerPGUpgrade,
		SecurityContext: initialize.RestrictedSecurityContext(),
		VolumeMounts:    volumeMounts,
	}

	if cluster.Spec.Backups.PGBackRest.Jobs != nil {
		container.Resources = cluster.Spec.Backups.PGBackRest.Jobs.Resources
	}

	jobSpec := &batchv1.JobSpec{
		// Set the BackoffLimit to zero because we do not want to attempt the
		// major upgrade more than once.
		BackoffLimit: initialize.Int32(0),
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: labels, Annotations: annotations},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{container},
				// Set RestartPolicy to "Never" since we want a new Pod to be
				// created by the Job controller when there is a failure
				// (instead of the container simply restarting).
				RestartPolicy:      corev1.RestartPolicyNever,
				ServiceAccountName: sa,
				Volumes:            volumes,
				SecurityContext:    PodSecurityContext(cluster),
			},
		},
	}

	// set the priority class name, if it exists
	if spec.PriorityClassName != nil {
		jobSpec.Template.Spec.PriorityClassName = *spec.PriorityClassName
	}

	// Set the image pull secrets, if any exist.
	// This is set here rather than using the service account due to the lack
	// of propagation to existing pods when the CRD is updated:
	// https://github.com/kubernetes/kubernetes/issues/88456
	jobSpec.Template.Spec.ImagePullSecrets = cluster.Spec.ImagePullSecrets

	upgradeJob.Spec = *jobSpec

	return *upgradeJob, nil
}

// upgradeCommand returns an entrypoint that prepares the filesystem for
// and performs a PostgreSQL major version upgrade using pg_upgrade.
func upgradeCommand(cluster *v1beta1.PostgresCluster) []string {
	oldVersion := fmt.Sprint(cluster.Spec.Upgrade.FromPostgresVersion)
	newVersion := fmt.Sprint(cluster.Spec.PostgresVersion)

	args := []string{oldVersion, newVersion, cluster.Name}
	script := strings.Join([]string{
		// Below is the pg_upgrade script used to upgrade a PostgresCluster from
		// one major verson to another. Additional information concerning the
		// steps used and command flag specifics can be found in the documentation:
		// - https://www.postgresql.org/docs/current/pgupgrade.html
		`declare -r old_version="$1" new_version="$2" cluster_name="$3"`,
		`echo -e "Performing PostgreSQL upgrade from version ""${old_version}"" to\`,
		` ""${new_version}"" for cluster ""${cluster_name}"".\n"`,

		// To begin, we first move to the mounted /pgdata directory and create a
		// new version directory which is then initialized with the initdb command.
		`cd /pgdata || exit`,
		`echo -e "Step 1: Making new pgdata directory...\n"`,
		`mkdir /pgdata/pg"${new_version}"`,
		`echo -e "Step 2: Initializing new pgdata directory...\n"`,
		`/usr/pgsql-"${new_version}"/bin/initdb -k -D /pgdata/pg"${new_version}"`,

		// Before running the upgrade check, which ensures the clusters are compatible,
		// proper permissions have to be set on the old pgdata directory and the
		// preload library settings must be copied over.
		`echo -e "\nStep 3: Setting the expected permissions on the old pgdata directory...\n"`,
		`chmod 700 /pgdata/pg"${old_version}"`,
		`echo -e "Step 4: Copying shared_preload_libraries setting to new postgresql.conf file...\n"`,
		`echo "shared_preload_libraries = '$(/usr/pgsql-"""${old_version}"""/bin/postgres -D \`,
		`/pgdata/pg"""${old_version}""" -C shared_preload_libraries)'" >> /pgdata/pg"${new_version}"/postgresql.conf`,

		// Before the actual upgrade is run, we will run the upgrade --check to
		// verify everything before actually changing any data.
		`echo -e "Step 5: Running pg_upgrade check...\n"`,
		`time /usr/pgsql-"${new_version}"/bin/pg_upgrade --old-bindir /usr/pgsql-"${old_version}"/bin \`,
		`--new-bindir /usr/pgsql-"${new_version}"/bin --old-datadir /pgdata/pg"${old_version}"\`,
		` --new-datadir /pgdata/pg"${new_version}" --link --check`,

		// Assuming the check completes successfully, the pg_upgrade command will
		// be run that actually prepares the upgraded pgdata directory.
		`echo -e "\nStep 6: Running pg_upgrade...\n"`,
		`time /usr/pgsql-"${new_version}"/bin/pg_upgrade --old-bindir /usr/pgsql-"${old_version}"/bin \`,
		`--new-bindir /usr/pgsql-"${new_version}"/bin --old-datadir /pgdata/pg"${old_version}" \`,
		`--new-datadir /pgdata/pg"${new_version}" --link`,

		// Since we have cleared the Patroni cluster step by removing the EndPoints, we copy patroni.dynamic.json
		// from the old data dir to help retain PostgreSQL parameters you had set before.
		// - https://patroni.readthedocs.io/en/latest/existing_data.html#major-upgrade-of-postgresql-version
		`echo -e "\nStep 7: Copying patroni.dynamic.json...\n"`,
		`cp /pgdata/pg"${old_version}"/patroni.dynamic.json /pgdata/pg"${new_version}"`,

		// Rename the new data directory for the "existing" bootstrap method
		`mv /pgdata/pg"${new_version}" /pgdata/pg"${new_version}"_bootstrap`,

		`echo -e "\npg_upgrade Job Complete!"`,
	}, "\n")

	return append([]string{"bash", "-ceu", "--", script, "upgrade"}, args...)
}
