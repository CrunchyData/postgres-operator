// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"fmt"
	"hash/fnv"
	"io"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
)

var tmpDirSizeLimit = resource.MustParse("16Mi")

const (
	// devSHMDir is the directory used for allocating shared memory segments,
	// which are needed by Postgres
	devSHMDir = "/dev/shm"
	// nssWrapperDir is the directory in a container for the nss_wrapper passwd and group files
	nssWrapperDir = "/tmp/nss_wrapper/%s/%s"
	// postgresNSSWrapperPrefix sets the required variables when running the NSS
	// wrapper script for the 'postgres' user
	postgresNSSWrapperPrefix = `export NSS_WRAPPER_SUBDIR=postgres CRUNCHY_NSS_USERNAME=postgres ` +
		`CRUNCHY_NSS_USER_DESC="postgres" `
	// pgAdminNSSWrapperPrefix sets the required variables when running the NSS
	// wrapper script for the 'pgadmin' user
	pgAdminNSSWrapperPrefix = `export NSS_WRAPPER_SUBDIR=pgadmin CRUNCHY_NSS_USERNAME=pgadmin ` +
		`CRUNCHY_NSS_USER_DESC="pgadmin" `
	// nssWrapperScript sets up an nss_wrapper environment in accordance with OpenShift
	// guidance for supporting arbitrary user ID's is the script for the configuration
	// and startup of the pgAdmin service.
	// It is based on the nss_wrapper.sh script from the Crunchy Containers Project.
	// - https://github.com/CrunchyData/crunchy-containers/blob/master/bin/common/nss_wrapper.sh
	nssWrapperScript = `
# Define nss_wrapper directory and passwd & group files that will be utilized by nss_wrapper.  The
# nss_wrapper_env.sh script (which also sets these vars) isn't sourced here since the nss_wrapper
# has not yet been setup, and we therefore don't yet want the nss_wrapper vars in the environment.
mkdir -p /tmp/nss_wrapper
chmod g+rwx /tmp/nss_wrapper

NSS_WRAPPER_DIR="/tmp/nss_wrapper/${NSS_WRAPPER_SUBDIR}"
NSS_WRAPPER_PASSWD="${NSS_WRAPPER_DIR}/passwd"
NSS_WRAPPER_GROUP="${NSS_WRAPPER_DIR}/group"

# create the nss_wrapper directory
mkdir -p "${NSS_WRAPPER_DIR}"

# grab the current user ID and group ID
USER_ID=$(id -u)
export USER_ID
GROUP_ID=$(id -g)
export GROUP_ID

# get copies of the passwd and group files
[[ -f "${NSS_WRAPPER_PASSWD}" ]] || cp "/etc/passwd" "${NSS_WRAPPER_PASSWD}"
[[ -f "${NSS_WRAPPER_GROUP}" ]] || cp "/etc/group" "${NSS_WRAPPER_GROUP}"

# if the username is missing from the passwd file, then add it
if [[ ! $(cat "${NSS_WRAPPER_PASSWD}") =~ ${CRUNCHY_NSS_USERNAME}:x:${USER_ID} ]]; then
    echo "nss_wrapper: adding user"
    passwd_tmp="${NSS_WRAPPER_DIR}/passwd_tmp"
    cp "${NSS_WRAPPER_PASSWD}" "${passwd_tmp}"
    sed -i "/${CRUNCHY_NSS_USERNAME}:x:/d" "${passwd_tmp}"
    # needed for OCP 4.x because crio updates /etc/passwd with an entry for USER_ID
    sed -i "/${USER_ID}:x:/d" "${passwd_tmp}"
    printf '${CRUNCHY_NSS_USERNAME}:x:${USER_ID}:${GROUP_ID}:${CRUNCHY_NSS_USER_DESC}:${HOME}:/bin/bash\n' >> "${passwd_tmp}"
    envsubst < "${passwd_tmp}" > "${NSS_WRAPPER_PASSWD}"
    rm "${passwd_tmp}"
else
    echo "nss_wrapper: user exists"
fi

# if the username (which will be the same as the group name) is missing from group file, then add it
if [[ ! $(cat "${NSS_WRAPPER_GROUP}") =~ ${CRUNCHY_NSS_USERNAME}:x:${USER_ID} ]]; then
    echo "nss_wrapper: adding group"
    group_tmp="${NSS_WRAPPER_DIR}/group_tmp"
    cp "${NSS_WRAPPER_GROUP}" "${group_tmp}"
    sed -i "/${CRUNCHY_NSS_USERNAME}:x:/d" "${group_tmp}"
    printf '${CRUNCHY_NSS_USERNAME}:x:${USER_ID}:${CRUNCHY_NSS_USERNAME}\n' >> "${group_tmp}"
    envsubst < "${group_tmp}" > "${NSS_WRAPPER_GROUP}"
    rm "${group_tmp}"
else
    echo "nss_wrapper: group exists"
fi

# export the nss_wrapper env vars
# define nss_wrapper directory and passwd & group files that will be utilized by nss_wrapper
NSS_WRAPPER_DIR="/tmp/nss_wrapper/${NSS_WRAPPER_SUBDIR}"
NSS_WRAPPER_PASSWD="${NSS_WRAPPER_DIR}/passwd"
NSS_WRAPPER_GROUP="${NSS_WRAPPER_DIR}/group"

export LD_PRELOAD=/usr/lib64/libnss_wrapper.so
export NSS_WRAPPER_PASSWD="${NSS_WRAPPER_PASSWD}"
export NSS_WRAPPER_GROUP="${NSS_WRAPPER_GROUP}"

echo "nss_wrapper: environment configured"
`
)

// addDevSHM adds the shared memory "directory" to a Pod, which is needed by
// Postgres to allocate shared memory segments. This is a special directory
// called "/dev/shm", and is mounted as an emptyDir over a "memory" medium. This
// is mounted only to the database container.
func addDevSHM(template *corev1.PodTemplateSpec) {

	// do not set a size limit on shared memory. This will be handled by the OS
	// layer
	template.Spec.Volumes = append(template.Spec.Volumes, corev1.Volume{
		Name: "dshm",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumMemory,
			},
		},
	})

	// only give the database container access to shared memory
	for i := range template.Spec.Containers {
		if template.Spec.Containers[i].Name == naming.ContainerDatabase {
			template.Spec.Containers[i].VolumeMounts = append(template.Spec.Containers[i].VolumeMounts,
				corev1.VolumeMount{
					Name:      "dshm",
					MountPath: devSHMDir,
				})
		}
	}
}

// addTMPEmptyDir adds a "tmp" EmptyDir volume to the provided Pod template, while then also adding a
// volume mount at /tmp for all containers defined within the Pod template
// The '/tmp' directory is currently utilized for the following:
//   - As the pgBackRest lock directory (this is the default lock location for pgBackRest)
//   - The location where the replication client certificates can be loaded with the proper
//     permissions set
func addTMPEmptyDir(template *corev1.PodTemplateSpec) {

	template.Spec.Volumes = append(template.Spec.Volumes, corev1.Volume{
		Name: "tmp",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				SizeLimit: &tmpDirSizeLimit,
			},
		},
	})

	for i := range template.Spec.Containers {
		template.Spec.Containers[i].VolumeMounts = append(template.Spec.Containers[i].VolumeMounts,
			corev1.VolumeMount{
				Name:      "tmp",
				MountPath: "/tmp",
			})
	}

	for i := range template.Spec.InitContainers {
		template.Spec.InitContainers[i].VolumeMounts = append(template.Spec.InitContainers[i].VolumeMounts,
			corev1.VolumeMount{
				Name:      "tmp",
				MountPath: "/tmp",
			})
	}
}

// addNSSWrapper adds nss_wrapper environment variables to the database and pgBackRest
// containers in the Pod template.  Additionally, an init container is added to the Pod template
// as needed to setup the nss_wrapper. Please note that the nss_wrapper is required for
// compatibility with OpenShift: https://access.redhat.com/articles/4859371.
func addNSSWrapper(image string, imagePullPolicy corev1.PullPolicy, template *corev1.PodTemplateSpec) {

	nssWrapperCmd := postgresNSSWrapperPrefix + nssWrapperScript
	for i, c := range template.Spec.Containers {
		switch c.Name {
		case naming.ContainerDatabase, naming.PGBackRestRepoContainerName,
			naming.PGBackRestRestoreContainerName:
			passwd := fmt.Sprintf(nssWrapperDir, "postgres", "passwd")
			group := fmt.Sprintf(nssWrapperDir, "postgres", "group")
			template.Spec.Containers[i].Env = append(template.Spec.Containers[i].Env, []corev1.EnvVar{
				{Name: "LD_PRELOAD", Value: "/usr/lib64/libnss_wrapper.so"},
				{Name: "NSS_WRAPPER_PASSWD", Value: passwd},
				{Name: "NSS_WRAPPER_GROUP", Value: group},
			}...)
		case naming.ContainerPGAdmin:
			nssWrapperCmd = pgAdminNSSWrapperPrefix + nssWrapperScript
			passwd := fmt.Sprintf(nssWrapperDir, "pgadmin", "passwd")
			group := fmt.Sprintf(nssWrapperDir, "pgadmin", "group")
			template.Spec.Containers[i].Env = append(template.Spec.Containers[i].Env, []corev1.EnvVar{
				{Name: "LD_PRELOAD", Value: "/usr/lib64/libnss_wrapper.so"},
				{Name: "NSS_WRAPPER_PASSWD", Value: passwd},
				{Name: "NSS_WRAPPER_GROUP", Value: group},
			}...)
		}
	}

	container := corev1.Container{
		Command:         []string{"bash", "-c", nssWrapperCmd},
		Image:           image,
		ImagePullPolicy: imagePullPolicy,
		Name:            naming.ContainerNSSWrapperInit,
		SecurityContext: initialize.RestrictedSecurityContext(),
	}

	// Here we set the NSS wrapper container resources to the 'database', 'pgadmin'
	// or 'pgbackrest' container configuration, as appropriate.

	// First, we'll set the NSS wrapper container configuration for any pgAdmin
	// containers because pgAdmin Pods won't contain any other containers
	containsPGAdmin := false
	for i, c := range template.Spec.Containers {
		if c.Name == naming.ContainerPGAdmin {
			containsPGAdmin = true
			container.Resources = template.Spec.Containers[i].Resources
			break
		}
	}

	// If this was a pgAdmin Pod, we don't need to check anything else.
	if !containsPGAdmin {
		// Because the instance Pod has both a 'database' and 'pgbackrest' container,
		// we'll first check for the 'database' container and use those resource
		// settings for any instance pods.
		containsDatabase := false
		for i, c := range template.Spec.Containers {
			if c.Name == naming.ContainerDatabase {
				containsDatabase = true
				container.Resources = template.Spec.Containers[i].Resources
				break
			}
			if c.Name == naming.PGBackRestRestoreContainerName {
				container.Resources = template.Spec.Containers[i].Resources
				break
			}
		}
		// If 'database' is not found, we need to use the 'pgbackrest' resource
		// configuration settings instead
		if !containsDatabase {
			for i, c := range template.Spec.Containers {
				if c.Name == naming.PGBackRestRepoContainerName {
					container.Resources = template.Spec.Containers[i].Resources
					break
				}
			}
		}
	}
	template.Spec.InitContainers = append(template.Spec.InitContainers, container)
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

// safeHash32 runs content and returns a short alphanumeric string that
// represents everything written to w. The string is unlikely to have bad words
// and is safe to store in the Kubernetes API. This is the same algorithm used
// by ControllerRevision's "controller.kubernetes.io/hash".
func safeHash32(content func(w io.Writer) error) (string, error) {
	hash := fnv.New32()
	if err := content(hash); err != nil {
		return "", err
	}
	return rand.SafeEncodeString(fmt.Sprint(hash.Sum32())), nil
}
