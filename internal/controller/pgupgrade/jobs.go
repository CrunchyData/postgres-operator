// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgupgrade

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/shell"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// pgUpgradeJob returns the ObjectMeta for the pg_upgrade Job utilized to
// upgrade from one major PostgreSQL version to another
func pgUpgradeJob(upgrade *v1beta1.PGUpgrade) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: upgrade.Namespace,
		Name:      upgrade.Name + "-pgdata",
	}
}

// upgradeCommand returns an entrypoint that prepares the filesystem for
// and performs a PostgreSQL major version upgrade using pg_upgrade.
func upgradeCommand(spec *v1beta1.PGUpgradeSettings, fetchKeyCommand string) []string {
	argJobs := fmt.Sprintf(` --jobs=%d`, max(1, spec.Jobs))
	argMethod := cmp.Or(map[string]string{
		"Clone":         ` --clone`,
		"Copy":          ` --copy`,
		"CopyFileRange": ` --copy-file-range`,
	}[spec.TransferMethod], ` --link`)

	oldVersion := spec.FromPostgresVersion
	newVersion := spec.ToPostgresVersion

	var argEncryptionKeyCommand string
	if fetchKeyCommand != "" {
		argEncryptionKeyCommand = ` --encryption-key-command=` + shell.QuoteWord(fetchKeyCommand)
	}

	args := []string{fmt.Sprint(oldVersion), fmt.Sprint(newVersion)}
	script := strings.Join([]string{
		// Exit immediately when a pipeline or subshell exits non-zero or when expanding an unset variable.
		`shopt -so errexit nounset`,

		`declare -r data_volume='/pgdata' old_version="$1" new_version="$2"`,
		`printf 'Performing PostgreSQL upgrade from version "%s" to "%s" ...\n' "$@"`,
		`section() { printf '\n\n%s\n' "$@"; }`,

		// NOTE: Rather than import the nss_wrapper init container, as we do in
		// the PostgresCluster controller, this job does the required nss_wrapper
		// settings here.
		`section 'Step 1 of 7: Ensuring username is postgres...'`,

		// Create a copy of the system group definitions, but remove the "postgres"
		// group or any group with the current GID. Replace them with our own that
		// has the current GID.
		`gid=$(id -G); NSS_WRAPPER_GROUP=$(mktemp)`,
		`(sed "/^postgres:x:/ d; /^[^:]*:x:${gid%% *}:/ d" /etc/group`,
		`echo "postgres:x:${gid%% *}:") > "${NSS_WRAPPER_GROUP}"`,

		// Create a copy of the system user definitions, but remove the "postgres"
		// user or any user with the current UID. Replace them with our own that
		// has the current UID and GID.
		`uid=$(id -u); NSS_WRAPPER_PASSWD=$(mktemp)`,
		`(sed "/^postgres:x:/ d; /^[^:]*:x:${uid}:/ d" /etc/passwd`,
		`echo "postgres:x:${uid}:${gid%% *}::${data_volume}:") > "${NSS_WRAPPER_PASSWD}"`,

		// Enable nss_wrapper so the current UID and GID resolve to "postgres".
		// - https://cwrap.org/nss_wrapper.html
		`export LD_PRELOAD='libnss_wrapper.so' NSS_WRAPPER_GROUP NSS_WRAPPER_PASSWD`,
		`id; [[ "$(id -nu)" == 'postgres' && "$(id -ng)" == 'postgres' ]]`,

		`section 'Step 2 of 7: Finding data and tools...'`,

		// Expect Postgres executables at the Red Hat paths.
		`old_bin="/usr/pgsql-${old_version}/bin" && [[ -x "${old_bin}/postgres" ]]`,
		`new_bin="/usr/pgsql-${new_version}/bin" && [[ -x "${new_bin}/initdb" ]]`,
		`old_data="${data_volume}/pg${old_version}" && [[ -d "${old_data}" ]]`,
		`new_data="${data_volume}/pg${new_version}"`,

		// pg_upgrade writes its files in "${new_data}/pg_upgrade_output.d" since PostgreSQL v15.
		// Change to a writable working directory to be compatible with PostgreSQL v14 and earlier.
		//
		// https://www.postgresql.org/docs/release/15#id-1.11.6.20.5.11.3
		`cd "${data_volume}"`,

		// Below is the pg_upgrade script used to upgrade a PostgresCluster from
		// one major version to another. Additional information concerning the
		// steps used and command flag specifics can be found in the documentation:
		// - https://www.postgresql.org/docs/current/pgupgrade.html

		`section 'Step 3 of 7: Initializing new data directory...'`,
		`PGDATA="${new_data}" "${new_bin}/initdb" --allow-group-access --data-checksums` + argEncryptionKeyCommand,

		// Read the configured value then quote it; every single-quote U+0027 is replaced by two.
		//
		// https://www.postgresql.org/docs/current/config-setting.html
		// https://www.gnu.org/software/bash/manual/bash.html#ANSI_002dC-Quoting
		`section 'Step 4 of 7: Copying shared_preload_libraries parameter...'`,
		`value=$(LC_ALL=C PGDATA="${old_data}" "${old_bin}/postgres" -C shared_preload_libraries)`,
		`echo >> "${new_data}/postgresql.conf" "shared_preload_libraries = '${value//$'\''/$'\'\''}'"`,

		`section 'Step 5 of 7: Checking for potential issues...'`,
		`"${new_bin}/pg_upgrade" --check` + argMethod + argJobs + ` \`,
		`--old-bindir="${old_bin}" --old-datadir="${old_data}" \`,
		`--new-bindir="${new_bin}" --new-datadir="${new_data}"`,

		`section 'Step 6 of 7: Performing upgrade...'`,
		`(set -x && time "${new_bin}/pg_upgrade"` + argMethod + argJobs + ` \`,
		`--old-bindir="${old_bin}" --old-datadir="${old_data}" \`,
		`--new-bindir="${new_bin}" --new-datadir="${new_data}")`,

		// https://patroni.readthedocs.io/en/latest/existing_data.html#major-upgrade-of-postgresql-version
		`section 'Step 7 of 7: Copying Patroni settings...'`,
		`(set -x && cp "${old_data}/patroni.dynamic.json" "${new_data}")`,

		`section 'Success!'`,
	}, "\n")

	return append([]string{"bash", "-c", "--", script, "upgrade"}, args...)
}

// largestWholeCPU returns the maximum CPU request or limit as a non-negative
// integer of CPUs. When resources lacks any CPU, the result is zero.
func largestWholeCPU(resources corev1.ResourceRequirements) int64 {
	// Read CPU quantities as millicores then divide to get the "floor."
	// NOTE: [resource.Quantity.Value] looks easier, but it rounds up.
	return max(
		resources.Limits.Cpu().ScaledValue(resource.Milli)/1000,
		resources.Requests.Cpu().ScaledValue(resource.Milli)/1000,
		0)
}

// generateUpgradeJob returns a Job that can upgrade the PostgreSQL data
// directory of the startup instance.
func (r *PGUpgradeReconciler) generateUpgradeJob(
	ctx context.Context, upgrade *v1beta1.PGUpgrade,
	startup *appsv1.StatefulSet, fetchKeyCommand string,
) *batchv1.Job {
	job := &batchv1.Job{}
	job.SetGroupVersionKind(batchv1.SchemeGroupVersion.WithKind("Job"))

	job.Namespace = upgrade.Namespace
	job.Name = pgUpgradeJob(upgrade).Name

	job.Labels = Merge(upgrade.Spec.Metadata.GetLabelsOrNil(),
		commonLabels(pgUpgrade, upgrade), //FIXME role pgupgrade
		map[string]string{
			LabelVersion: fmt.Sprint(upgrade.Spec.ToPostgresVersion),
		})

	// Find the database container.
	var database *corev1.Container
	for i := range startup.Spec.Template.Spec.Containers {
		container := startup.Spec.Template.Spec.Containers[i]
		if container.Name == ContainerDatabase {
			database = &container
		}
	}

	job.Annotations = Merge(upgrade.Spec.Metadata.GetAnnotationsOrNil(),
		map[string]string{
			naming.DefaultContainerAnnotation: database.Name,
		})

	// Copy the pod template from the startup instance StatefulSet. This includes
	// the service account, volumes, DNS policies, and scheduling constraints.
	startup.Spec.Template.DeepCopyInto(&job.Spec.Template)

	// Use the same labels and annotations as the job.
	job.Spec.Template.ObjectMeta = metav1.ObjectMeta{
		Annotations: job.Annotations,
		Labels:      job.Labels,
	}

	// Use the image pull secrets specified for the upgrade image.
	job.Spec.Template.Spec.ImagePullSecrets = upgrade.Spec.ImagePullSecrets

	// Attempt the upgrade exactly once.
	job.Spec.BackoffLimit = initialize.Int32(0)
	job.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyNever

	settings := upgrade.Spec.PGUpgradeSettings.DeepCopy()

	// When jobs is undefined, use one less than the number of CPUs.
	//nolint:gosec // The CPU count is clamped to MaxInt32.
	if settings.Jobs == 0 && feature.Enabled(ctx, feature.PGUpgradeCPUConcurrency) {
		wholeCPUs := int32(min(math.MaxInt32, largestWholeCPU(upgrade.Spec.Resources)))
		settings.Jobs = wholeCPUs - 1
	}

	// Replace all containers with one that does the upgrade.
	job.Spec.Template.Spec.EphemeralContainers = nil
	job.Spec.Template.Spec.InitContainers = nil
	job.Spec.Template.Spec.Containers = []corev1.Container{{
		// Copy volume mounts and the security context needed to access them
		// from the database container. There is a downward API volume that
		// refers back to the container by name, so use that same name here.
		Name:            database.Name,
		SecurityContext: database.SecurityContext,
		VolumeMounts:    database.VolumeMounts,

		// Use our upgrade command and the specified image and resources.
		Command:         upgradeCommand(settings, fetchKeyCommand),
		Image:           pgUpgradeContainerImage(upgrade),
		ImagePullPolicy: upgrade.Spec.ImagePullPolicy,
		Resources:       upgrade.Spec.Resources,
	}}

	// The following will set these fields to null if not set in the spec
	job.Spec.Template.Spec.Affinity = upgrade.Spec.Affinity
	job.Spec.Template.Spec.PriorityClassName =
		initialize.FromPointer(upgrade.Spec.PriorityClassName)
	job.Spec.Template.Spec.Tolerations = upgrade.Spec.Tolerations

	r.setControllerReference(upgrade, job)
	return job
}

// Remove data job

// removeDataCommand returns an entrypoint that removes certain directories.
// We currently target the `pgdata/pg{old_version}` and `pgdata/pg{old_version}_wal`
// directories for removal.
func removeDataCommand(upgrade *v1beta1.PGUpgrade) []string {
	oldVersion := upgrade.Spec.FromPostgresVersion

	// Before removing the directories (both data and wal), we check that
	// the directory is not in use by running `pg_controldata` and making sure
	// the server state is "shut down in recovery"
	args := []string{fmt.Sprint(oldVersion)}
	script := strings.Join([]string{
		// Exit immediately when a pipeline or subshell exits non-zero or when expanding an unset variable.
		`shopt -so errexit nounset`,

		`declare -r data_volume='/pgdata' old_version="$1"`,
		`printf 'Removing PostgreSQL %s data...\n\n' "$@"`,
		`delete() (set -x && rm -rf -- "$@")`,

		`old_data="${data_volume}/pg${old_version}"`,
		`control=$(LC_ALL=C /usr/pgsql-${old_version}/bin/pg_controldata "${old_data}")`,
		`read -r state <<< "${control##*cluster state:}"`,

		// We expect exactly one state for a replica that has been stopped.
		//
		// https://git.postgresql.org/gitweb/?p=postgresql.git;hb=refs/tags/REL_10_0;f=src/bin/pg_controldata/pg_controldata.c#l55
		// https://git.postgresql.org/gitweb/?p=postgresql.git;hb=refs/tags/REL_17_0;f=src/bin/pg_controldata/pg_controldata.c#l58
		`[[ "${state}" == 'shut down in recovery' ]] || { printf >&2 'Unexpected state! %q\n' "${state}"; exit 1; }`,

		// "rm" does not follow symbolic links.
		// Delete the old data directory after subdirectories that contain versioned data.
		`delete "${old_data}/pg_wal/"`,
		`delete "${old_data}" && echo 'Success!'`,
	}, "\n")

	return append([]string{"bash", "-c", "--", script, "remove"}, args...)
}

// generateRemoveDataJob returns a Job that can remove the data
// on the given replica StatefulSet
func (r *PGUpgradeReconciler) generateRemoveDataJob(
	_ context.Context, upgrade *v1beta1.PGUpgrade, sts *appsv1.StatefulSet,
) *batchv1.Job {
	job := &batchv1.Job{}
	job.SetGroupVersionKind(batchv1.SchemeGroupVersion.WithKind("Job"))

	job.Namespace = upgrade.Namespace
	job.Name = upgrade.Name + "-" + sts.Name

	job.Labels = labels.Merge(upgrade.Spec.Metadata.GetLabelsOrNil(),
		commonLabels(removeData, upgrade)) //FIXME role removedata

	// Find the database container.
	var database *corev1.Container
	for i := range sts.Spec.Template.Spec.Containers {
		container := sts.Spec.Template.Spec.Containers[i]
		if container.Name == ContainerDatabase {
			database = &container
		}
	}

	job.Annotations = Merge(upgrade.Spec.Metadata.GetAnnotationsOrNil(),
		map[string]string{
			naming.DefaultContainerAnnotation: database.Name,
		})

	// Copy the pod template from the sts instance StatefulSet. This includes
	// the service account, volumes, DNS policies, and scheduling constraints.
	sts.Spec.Template.DeepCopyInto(&job.Spec.Template)

	// Use the same labels and annotations as the job.
	job.Spec.Template.ObjectMeta = metav1.ObjectMeta{
		Annotations: job.Annotations,
		Labels:      job.Labels,
	}

	// Use the image pull secrets specified for the upgrade image.
	job.Spec.Template.Spec.ImagePullSecrets = upgrade.Spec.ImagePullSecrets

	// Attempt the removal exactly once.
	job.Spec.BackoffLimit = initialize.Int32(0)
	job.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyNever

	// Replace all containers with one that removes the data.
	job.Spec.Template.Spec.EphemeralContainers = nil
	job.Spec.Template.Spec.InitContainers = nil
	job.Spec.Template.Spec.Containers = []corev1.Container{{
		// Copy volume mounts and the security context needed to access them
		// from the database container. There is a downward API volume that
		// refers back to the container by name, so use that same name here.
		// We are using a PG image in order to check that the PG server is down.
		Name:            database.Name,
		SecurityContext: database.SecurityContext,
		VolumeMounts:    database.VolumeMounts,

		// Use our remove command and the specified resources.
		Command:         removeDataCommand(upgrade),
		Image:           pgUpgradeContainerImage(upgrade),
		ImagePullPolicy: upgrade.Spec.ImagePullPolicy,
		Resources:       upgrade.Spec.Resources,
	}}

	// The following will set these fields to null if not set in the spec
	job.Spec.Template.Spec.Affinity = upgrade.Spec.Affinity
	job.Spec.Template.Spec.PriorityClassName =
		initialize.FromPointer(upgrade.Spec.PriorityClassName)
	job.Spec.Template.Spec.Tolerations = upgrade.Spec.Tolerations

	r.setControllerReference(upgrade, job)
	return job
}

// Util functions

// pgUpgradeContainerImage returns the container image to use for pg_upgrade.
func pgUpgradeContainerImage(upgrade *v1beta1.PGUpgrade) string {
	var image string
	if upgrade.Spec.Image != nil {
		image = *upgrade.Spec.Image
	}
	return defaultFromEnv(image, "RELATED_IMAGE_PGUPGRADE")
}

// verifyUpgradeImageValue checks that the upgrade container image required by the
// spec is defined. If it is undefined, an error is returned.
func verifyUpgradeImageValue(upgrade *v1beta1.PGUpgrade) error {
	if pgUpgradeContainerImage(upgrade) == "" {
		return fmt.Errorf("missing crunchy-upgrade image")
	}
	return nil
}

// jobFailed returns "true" if the Job provided has failed.  Otherwise it returns "false".
func jobFailed(job *batchv1.Job) bool {
	conditions := job.Status.Conditions
	for i := range conditions {
		if conditions[i].Type == batchv1.JobFailed {
			return (conditions[i].Status == corev1.ConditionTrue)
		}
	}
	return false
}

// jobCompleted returns "true" if the Job provided completed successfully.  Otherwise it returns
// "false".
func jobCompleted(job *batchv1.Job) bool {
	conditions := job.Status.Conditions
	for i := range conditions {
		if conditions[i].Type == batchv1.JobComplete {
			return (conditions[i].Status == corev1.ConditionTrue)
		}
	}
	return false
}
