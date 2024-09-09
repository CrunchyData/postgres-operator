// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgupgrade

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// Upgrade job

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
func upgradeCommand(upgrade *v1beta1.PGUpgrade, fetchKeyCommand string) []string {
	oldVersion := fmt.Sprint(upgrade.Spec.FromPostgresVersion)
	newVersion := fmt.Sprint(upgrade.Spec.ToPostgresVersion)

	// if the fetch key command is set for TDE, provide the value during initialization
	initdb := `/usr/pgsql-"${new_version}"/bin/initdb -k -D /pgdata/pg"${new_version}"`
	if fetchKeyCommand != "" {
		initdb += ` --encryption-key-command "` + fetchKeyCommand + `"`
	}

	args := []string{oldVersion, newVersion}
	script := strings.Join([]string{
		`declare -r data_volume='/pgdata' old_version="$1" new_version="$2"`,
		`printf 'Performing PostgreSQL upgrade from version "%s" to "%s" ...\n\n' "$@"`,

		// Note: Rather than import the nss_wrapper init container, as we do in
		// the main postgres-operator, this job does the required nss_wrapper
		// settings here.

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

		// Below is the pg_upgrade script used to upgrade a PostgresCluster from
		// one major version to another. Additional information concerning the
		// steps used and command flag specifics can be found in the documentation:
		// - https://www.postgresql.org/docs/current/pgupgrade.html

		// To begin, we first move to the mounted /pgdata directory and create a
		// new version directory which is then initialized with the initdb command.
		`cd /pgdata || exit`,
		`echo -e "Step 1: Making new pgdata directory...\n"`,
		`mkdir /pgdata/pg"${new_version}"`,
		`echo -e "Step 2: Initializing new pgdata directory...\n"`,
		initdb,

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

		`echo -e "\npg_upgrade Job Complete!"`,
	}, "\n")

	return append([]string{"bash", "-ceu", "--", script, "upgrade"}, args...)
}

// generateUpgradeJob returns a Job that can upgrade the PostgreSQL data
// directory of the startup instance.
func (r *PGUpgradeReconciler) generateUpgradeJob(
	_ context.Context, upgrade *v1beta1.PGUpgrade,
	startup *appsv1.StatefulSet, fetchKeyCommand string,
) *batchv1.Job {
	job := &batchv1.Job{}
	job.SetGroupVersionKind(batchv1.SchemeGroupVersion.WithKind("Job"))

	job.Namespace = upgrade.Namespace
	job.Name = pgUpgradeJob(upgrade).Name

	job.Annotations = upgrade.Spec.Metadata.GetAnnotationsOrNil()
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
		Command:         upgradeCommand(upgrade, fetchKeyCommand),
		Image:           pgUpgradeContainerImage(upgrade),
		ImagePullPolicy: upgrade.Spec.ImagePullPolicy,
		Resources:       upgrade.Spec.Resources,
	}}

	// The following will set these fields to null if not set in the spec
	job.Spec.Template.Spec.Affinity = upgrade.Spec.Affinity
	job.Spec.Template.Spec.PriorityClassName = initialize.FromPointer(
		upgrade.Spec.PriorityClassName)
	job.Spec.Template.Spec.Tolerations = upgrade.Spec.Tolerations

	r.setControllerReference(upgrade, job)
	return job
}

// Remove data job

// removeDataCommand returns an entrypoint that removes certain directories.
// We currently target the `pgdata/pg{old_version}` and `pgdata/pg{old_version}_wal`
// directories for removal.
func removeDataCommand(upgrade *v1beta1.PGUpgrade) []string {
	oldVersion := fmt.Sprint(upgrade.Spec.FromPostgresVersion)

	// Before removing the directories (both data and wal), we check that
	// the directory is not in use by running `pg_controldata` and making sure
	// the server state is "shut down in recovery"
	// TODO(benjaminjb): pg_controldata seems pretty stable, but might want to
	// experiment with a few more versions.
	args := []string{oldVersion}
	script := strings.Join([]string{
		`declare -r old_version="$1"`,
		`printf 'Removing PostgreSQL data dir for pg%s...\n\n' "$@"`,
		`echo -e "Checking the directory exists and isn't being used...\n"`,
		`cd /pgdata || exit`,
		// The string `shut down in recovery` is the dbstate that postgres sets from
		// at least version 10 to 14 when a replica has been shut down.
		// - https://git.postgresql.org/gitweb/?p=postgresql.git;a=blob;f=src/bin/pg_controldata/pg_controldata.c;h=f911f98d946d83f1191abf35239d9b4455c5f52a;hb=HEAD#l59
		// Note: `pg_controldata` is actually used by `pg_upgrade` before upgrading
		// to make sure that the server in question is shut down as a primary;
		// that aligns with our use here, where we're making sure that the server in question
		// was shut down as a replica.
		// - https://git.postgresql.org/gitweb/?p=postgresql.git;a=blob;f=src/bin/pg_upgrade/controldata.c;h=41b8f69b8cbe4f40e6098ad84c2e8e987e24edaf;hb=HEAD#l122
		`if [ "$(/usr/pgsql-"${old_version}"/bin/pg_controldata /pgdata/pg"${old_version}" | grep -c "shut down in recovery")" -ne 1 ]; then echo -e "Directory in use, cannot remove..."; exit 1; fi`,
		`echo -e "Removing old pgdata directory...\n"`,
		// When deleting the wal directory, use `realpath` to resolve the symlink from
		// the pgdata directory. This is necessary because the wal directory can be
		// mounted at different places depending on if an external wal PVC is used,
		// i.e. `/pgdata/pg14_wal` vs `/pgwal/pg14_wal`
		`rm -rf /pgdata/pg"${old_version}" "$(realpath /pgdata/pg${old_version}/pg_wal)"`,
		`echo -e "Remove Data Job Complete!"`,
	}, "\n")

	return append([]string{"bash", "-ceu", "--", script, "remove"}, args...)
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

	job.Annotations = upgrade.Spec.Metadata.GetAnnotationsOrNil()
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
	job.Spec.Template.Spec.PriorityClassName = initialize.FromPointer(
		upgrade.Spec.PriorityClassName)
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
		return fmt.Errorf("Missing crunchy-upgrade image")
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
