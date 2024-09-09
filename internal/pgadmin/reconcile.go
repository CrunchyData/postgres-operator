// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgadmin

import (
	"bytes"
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// startupScript is the script for the configuration and startup of the pgAdmin service.
// It is based on the start-pgadmin4.sh script from the Crunchy Containers Project.
// Any required functions from common_lib.sh are added as required.
// - https://github.com/CrunchyData/crunchy-containers/blob/master/bin/pgadmin4/start-pgadmin4.sh
// - https://github.com/CrunchyData/crunchy-containers/blob/master/bin/common/common_lib.sh
const startupScript = `CRUNCHY_DIR=${CRUNCHY_DIR:-'/opt/crunchy'}
PGADMIN_DIR=/usr/lib/python3.6/site-packages/pgadmin4-web
APACHE_PIDFILE='/tmp/httpd.pid'
export PATH=$PATH:/usr/pgsql-*/bin

RED="\033[0;31m"
GREEN="\033[0;32m"
RESET="\033[0m"

function enable_debugging() {
    if [[ ${CRUNCHY_DEBUG:-false} == "true" ]]
    then
        echo_info "Turning debugging on.."
        export PS4='+(${BASH_SOURCE}:${LINENO})> ${FUNCNAME[0]:+${FUNCNAME[0]}(): }'
        set -x
    fi
}

function env_check_err() {
    if [[ -z ${!1} ]]
    then
        echo_err "$1 environment variable is not set, aborting."
        exit 1
    fi
}

function echo_info() {
    echo -e "${GREEN?}$(date) INFO: ${1?}${RESET?}"
}

function echo_err() {
    echo -e "${RED?}$(date) ERROR: ${1?}${RESET?}"
}

function err_check {
    RC=${1?}
    CONTEXT=${2?}
    ERROR=${3?}

    if [[ ${RC?} != 0 ]]
    then
        echo_err "${CONTEXT?}: ${ERROR?}"
        exit ${RC?}
    fi
}

function trap_sigterm() {
    echo_info "Doing trap logic.."
    echo_warn "Clean shutdown of Apache.."
    /usr/sbin/httpd -k stop
    kill -SIGINT $(head -1 $APACHE_PIDFILE)
}

enable_debugging
trap 'trap_sigterm' SIGINT SIGTERM

env_check_err "PGADMIN_SETUP_EMAIL"
env_check_err "PGADMIN_SETUP_PASSWORD"

if [[ ${ENABLE_TLS:-false} == 'true' ]]
then
    echo_info "TLS enabled. Applying https configuration.."
    if [[ ( ! -f /certs/server.key ) || ( ! -f /certs/server.crt ) ]]
    then
        echo_err "ENABLE_TLS true but /certs/server.key or /certs/server.crt not found, aborting"
        exit 1
    fi
    cp "${CRUNCHY_DIR}/conf/pgadmin-https.conf" /var/lib/pgadmin/pgadmin.conf
else
    echo_info "TLS disabled. Applying http configuration.."
    cp "${CRUNCHY_DIR}/conf/pgadmin-http.conf" /var/lib/pgadmin/pgadmin.conf
fi

cp "${CRUNCHY_DIR}/conf/config_local.py" /var/lib/pgadmin/config_local.py

if [[ -z "${SERVER_PATH}" ]]
then
    sed -i "/RedirectMatch/d" /var/lib/pgadmin/pgadmin.conf
fi

sed -i "s|SERVER_PATH|${SERVER_PATH:-/}|g" /var/lib/pgadmin/pgadmin.conf
sed -i "s|SERVER_PORT|${SERVER_PORT:-5050}|g" /var/lib/pgadmin/pgadmin.conf
sed -i "s/^DEFAULT_SERVER_PORT.*/DEFAULT_SERVER_PORT = ${SERVER_PORT:-5050}/" /var/lib/pgadmin/config_local.py
sed -i "s|\"pg\":.*|\"pg\": \"/usr/pgsql-${PGVERSION?}/bin\",|g" /var/lib/pgadmin/config_local.py

cd ${PGADMIN_DIR?}

if [[ ! -f /var/lib/pgadmin/pgadmin4.db ]]
then
    echo_info "Setting up pgAdmin4 database.."
    python3 setup.py > /tmp/pgadmin4.stdout 2> /tmp/pgadmin4.stderr
    err_check "$?" "pgAdmin4 Database Setup" "Could not create pgAdmin4 database: \n$(cat /tmp/pgadmin4.stderr)"
fi

echo_info "Starting Apache web server.."
/usr/sbin/httpd -D FOREGROUND &
echo $! > $APACHE_PIDFILE

wait`

// ConfigMap populates a ConfigMap with the configuration needed to run pgAdmin.
func ConfigMap(
	inCluster *v1beta1.PostgresCluster,
	outConfigMap *corev1.ConfigMap,
) error {
	if inCluster.Spec.UserInterface == nil || inCluster.Spec.UserInterface.PGAdmin == nil {
		// pgAdmin is disabled; there is nothing to do.
		return nil
	}

	initialize.StringMap(&outConfigMap.Data)

	// To avoid spurious reconciles, the following value must not change when
	// the spec does not change. [json.Encoder] and [json.Marshal] do this by
	// emitting map keys in sorted order. Indent so the value is not rendered
	// as one long line by `kubectl`.
	buffer := new(bytes.Buffer)
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(systemSettings(inCluster.Spec.UserInterface.PGAdmin))
	if err == nil {
		outConfigMap.Data[settingsConfigMapKey] = buffer.String()
	}
	return err
}

// Pod populates a PodSpec with the container and volumes needed to run pgAdmin.
func Pod(
	inCluster *v1beta1.PostgresCluster,
	inConfigMap *corev1.ConfigMap,
	outPod *corev1.PodSpec, pgAdminVolume *corev1.PersistentVolumeClaim,
) {
	if inCluster.Spec.UserInterface == nil || inCluster.Spec.UserInterface.PGAdmin == nil {
		// pgAdmin is disabled; there is nothing to do.
		return
	}

	// create the pgAdmin Pod volumes
	tmp := corev1.Volume{Name: tmpVolume}
	tmp.EmptyDir = &corev1.EmptyDirVolumeSource{
		Medium: corev1.StorageMediumMemory,
	}

	pgAdminLog := corev1.Volume{Name: logVolume}
	pgAdminLog.EmptyDir = &corev1.EmptyDirVolumeSource{
		Medium: corev1.StorageMediumMemory,
	}

	pgAdminData := corev1.Volume{Name: dataVolume}
	pgAdminData.VolumeSource = corev1.VolumeSource{
		PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
			ClaimName: pgAdminVolume.Name,
			ReadOnly:  false,
		},
	}

	configVolumeMount := corev1.VolumeMount{
		Name: "pgadmin-config", MountPath: configMountPath, ReadOnly: true,
	}
	configVolume := corev1.Volume{Name: configVolumeMount.Name}
	configVolume.Projected = &corev1.ProjectedVolumeSource{
		Sources: podConfigFiles(inConfigMap, *inCluster.Spec.UserInterface.PGAdmin),
	}

	startupVolumeMount := corev1.VolumeMount{
		Name: "pgadmin-startup", MountPath: startupMountPath, ReadOnly: true,
	}
	startupVolume := corev1.Volume{Name: startupVolumeMount.Name}
	startupVolume.EmptyDir = &corev1.EmptyDirVolumeSource{
		Medium: corev1.StorageMediumMemory,

		// When this volume is too small, the Pod will be evicted and recreated
		// by the StatefulSet controller.
		// - https://kubernetes.io/docs/concepts/storage/volumes/#emptydir
		// NOTE: tmpfs blocks are PAGE_SIZE, usually 4KiB, and size rounds up.
		SizeLimit: resource.NewQuantity(32<<10, resource.BinarySI),
	}

	// pgadmin container
	container := corev1.Container{
		Name: naming.ContainerPGAdmin,
		Env: []corev1.EnvVar{
			{
				Name:  "PGADMIN_SETUP_EMAIL",
				Value: loginEmail,
			},
			{
				Name:  "PGADMIN_SETUP_PASSWORD",
				Value: loginPassword,
			},
			// Setting the KRB5_CONFIG for kerberos
			// - https://web.mit.edu/kerberos/krb5-current/doc/admin/conf_files/krb5_conf.html
			{
				Name:  "KRB5_CONFIG",
				Value: configMountPath + "/krb5.conf",
			},
			// In testing it was determined that we need to set this env var for the replay cache
			// otherwise it defaults to the read-only location `/var/tmp/`
			// - https://web.mit.edu/kerberos/krb5-current/doc/basic/rcache_def.html#replay-cache-types
			{
				Name:  "KRB5RCACHEDIR",
				Value: "/tmp",
			},
		},
		Command:         []string{"bash", "-c", startupScript},
		Image:           config.PGAdminContainerImage(inCluster),
		ImagePullPolicy: inCluster.Spec.ImagePullPolicy,
		Resources:       inCluster.Spec.UserInterface.PGAdmin.Resources,

		SecurityContext: initialize.RestrictedSecurityContext(),

		Ports: []corev1.ContainerPort{{
			Name:          naming.PortPGAdmin,
			ContainerPort: int32(pgAdminPort),
			Protocol:      corev1.ProtocolTCP,
		}},
		VolumeMounts: []corev1.VolumeMount{
			startupVolumeMount,
			configVolumeMount,
			{
				Name:      tmpVolume,
				MountPath: runMountPath,
			},
			{
				Name:      logVolume,
				MountPath: logMountPath,
			},
			{
				Name:      dataVolume,
				MountPath: dataMountPath,
			},
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt(pgAdminPort),
				},
			},
			InitialDelaySeconds: 20,
			PeriodSeconds:       10,
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt(pgAdminPort),
				},
			},
			InitialDelaySeconds: 15,
			PeriodSeconds:       20,
		},
	}

	startup := corev1.Container{
		Name:    naming.ContainerPGAdminStartup,
		Command: startupCommand(),

		Image:           container.Image,
		ImagePullPolicy: container.ImagePullPolicy,
		Resources:       container.Resources,
		SecurityContext: initialize.RestrictedSecurityContext(),
		VolumeMounts: []corev1.VolumeMount{
			startupVolumeMount,
			configVolumeMount,
		},
	}

	// The startup container is the only one allowed to write to the startup volume.
	startup.VolumeMounts[0].ReadOnly = false

	outPod.InitContainers = []corev1.Container{startup}
	// add all volumes other than 'tmp' as that is added later
	outPod.Volumes = []corev1.Volume{pgAdminLog, pgAdminData, configVolume, startupVolume}

	outPod.Containers = []corev1.Container{container}
}
