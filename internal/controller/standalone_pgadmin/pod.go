// Copyright 2023 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package standalone_pgadmin

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

const (
	configMountPath        = "/etc/pgadmin/conf.d"
	configFilePath         = "~postgres-operator/" + settingsConfigMapKey
	clusterFilePath        = "~postgres-operator/" + settingsClusterMapKey
	configDatabaseURIPath  = "~postgres-operator/config-database-uri"
	ldapFilePath           = "~postgres-operator/ldap-bind-password"
	gunicornConfigFilePath = "~postgres-operator/" + gunicornConfigKey

	// Nothing should be mounted to this location except the script our initContainer writes
	scriptMountPath = "/etc/pgadmin"
)

// pod populates a PodSpec with the container and volumes needed to run pgAdmin.
func pod(
	inPGAdmin *v1beta1.PGAdmin,
	inConfigMap *corev1.ConfigMap,
	outPod *corev1.PodSpec,
	pgAdminVolume *corev1.PersistentVolumeClaim,
) {
	const (
		// config and data volume names
		configVolumeName = "pgadmin-config"
		dataVolumeName   = "pgadmin-data"
		logVolumeName    = "pgadmin-log"
		scriptVolumeName = "pgadmin-config-system"
		tempVolumeName   = "tmp"
	)

	// create the projected volume of config maps for use in
	// 1. dynamic server discovery
	// 2. adding the config variables during pgAdmin startup
	configVolume := corev1.Volume{Name: configVolumeName}
	configVolume.VolumeSource = corev1.VolumeSource{
		Projected: &corev1.ProjectedVolumeSource{
			Sources: podConfigFiles(inConfigMap, *inPGAdmin),
		},
	}

	// create the data volume for the persistent database
	dataVolume := corev1.Volume{Name: dataVolumeName}
	dataVolume.VolumeSource = corev1.VolumeSource{
		PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
			ClaimName: pgAdminVolume.Name,
			ReadOnly:  false,
		},
	}

	// create the temp volume for logs
	logVolume := corev1.Volume{Name: logVolumeName}
	logVolume.VolumeSource = corev1.VolumeSource{
		EmptyDir: &corev1.EmptyDirVolumeSource{
			Medium: corev1.StorageMediumMemory,
		},
	}

	// Volume used to write a custom config_system.py file in the initContainer
	// which then loads the configs found in the `configVolume`
	scriptVolume := corev1.Volume{Name: scriptVolumeName}
	scriptVolume.VolumeSource = corev1.VolumeSource{
		EmptyDir: &corev1.EmptyDirVolumeSource{
			Medium: corev1.StorageMediumMemory,

			// When this volume is too small, the Pod will be evicted and recreated
			// by the StatefulSet controller.
			// - https://kubernetes.io/docs/concepts/storage/volumes/#emptydir
			// NOTE: tmpfs blocks are PAGE_SIZE, usually 4KiB, and size rounds up.
			SizeLimit: resource.NewQuantity(32<<10, resource.BinarySI),
		},
	}

	// create a temp volume for restart pid/other/debugging use
	// TODO: discuss tmp vol vs. persistent vol
	tmpVolume := corev1.Volume{Name: tempVolumeName}
	tmpVolume.VolumeSource = corev1.VolumeSource{
		EmptyDir: &corev1.EmptyDirVolumeSource{
			Medium: corev1.StorageMediumMemory,
		},
	}

	// pgadmin container
	container := corev1.Container{
		Name:            naming.ContainerPGAdmin,
		Command:         startupScript(inPGAdmin),
		Image:           config.StandalonePGAdminContainerImage(inPGAdmin),
		ImagePullPolicy: inPGAdmin.Spec.ImagePullPolicy,
		Resources:       inPGAdmin.Spec.Resources,
		SecurityContext: initialize.RestrictedSecurityContext(),
		Ports: []corev1.ContainerPort{{
			Name:          naming.PortPGAdmin,
			ContainerPort: int32(pgAdminPort),
			Protocol:      corev1.ProtocolTCP,
		}},
		Env: []corev1.EnvVar{
			{
				Name:  "PGADMIN_SETUP_EMAIL",
				Value: fmt.Sprintf("admin@%s.%s.svc", inPGAdmin.Name, inPGAdmin.Namespace),
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
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      configVolumeName,
				MountPath: configMountPath,
				ReadOnly:  true,
			},
			{
				Name:      dataVolumeName,
				MountPath: "/var/lib/pgadmin",
			},
			{
				Name:      logVolumeName,
				MountPath: "/var/log/pgadmin",
			},
			{
				Name:      scriptVolumeName,
				MountPath: scriptMountPath,
				ReadOnly:  true,
			},
			{
				Name:      tempVolumeName,
				MountPath: "/tmp",
			},
		},
	}

	// Creating a readiness probe that will check that the pgAdmin `/login`
	// endpoint is reachable at the specified port
	readinessProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Port:   *initialize.IntOrStringInt32(pgAdminPort),
				Path:   "/login",
				Scheme: corev1.URISchemeHTTP,
			},
		},
	}
	gunicornData := inConfigMap.Data[gunicornConfigKey]
	// Check the configmap to see  if we think TLS is enabled
	// If so, update the readiness check scheme to HTTPS
	if strings.Contains(gunicornData, "certfile") && strings.Contains(gunicornData, "keyfile") {
		readinessProbe.ProbeHandler.HTTPGet.Scheme = corev1.URISchemeHTTPS
	}
	container.ReadinessProbe = readinessProbe

	startup := corev1.Container{
		Name:            naming.ContainerPGAdminStartup,
		Command:         startupCommand(),
		Image:           container.Image,
		ImagePullPolicy: container.ImagePullPolicy,
		Resources:       container.Resources,
		SecurityContext: initialize.RestrictedSecurityContext(),
		VolumeMounts: []corev1.VolumeMount{
			// Volume to write a custom `config_system.py` file to.
			{
				Name:      scriptVolumeName,
				MountPath: scriptMountPath,
				ReadOnly:  false,
			},
		},
	}

	// add volumes and containers
	outPod.Volumes = []corev1.Volume{
		configVolume,
		dataVolume,
		logVolume,
		scriptVolume,
		tmpVolume,
	}
	outPod.Containers = []corev1.Container{container}
	outPod.InitContainers = []corev1.Container{startup}
}

// podConfigFiles returns projections of pgAdmin's configuration files to
// include in the configuration volume.
func podConfigFiles(configmap *corev1.ConfigMap, pgadmin v1beta1.PGAdmin) []corev1.VolumeProjection {

	config := append(append([]corev1.VolumeProjection{}, pgadmin.Spec.Config.Files...),
		[]corev1.VolumeProjection{
			{
				ConfigMap: &corev1.ConfigMapProjection{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configmap.Name,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  settingsConfigMapKey,
							Path: configFilePath,
						},
						{
							Key:  settingsClusterMapKey,
							Path: clusterFilePath,
						},
						{
							Key:  gunicornConfigKey,
							Path: gunicornConfigFilePath,
						},
					},
				},
			},
		}...)

	if pgadmin.Spec.Config.ConfigDatabaseURI != nil {
		config = append(config, corev1.VolumeProjection{
			Secret: &corev1.SecretProjection{
				LocalObjectReference: pgadmin.Spec.Config.ConfigDatabaseURI.LocalObjectReference,
				Optional:             pgadmin.Spec.Config.ConfigDatabaseURI.Optional,
				Items: []corev1.KeyToPath{
					{
						Key:  pgadmin.Spec.Config.ConfigDatabaseURI.Key,
						Path: configDatabaseURIPath,
					},
				},
			},
		})
	}

	// To enable LDAP authentication for pgAdmin, various LDAP settings must be configured.
	// While most of the required configuration can be set using the 'settings'
	// feature on the spec (.Spec.UserInterface.PGAdmin.Config.Settings), those
	// values are stored in a ConfigMap in plaintext.
	// As a special case, here we mount a provided Secret containing the LDAP_BIND_PASSWORD
	// for use with the other pgAdmin LDAP configuration.
	// - https://www.pgadmin.org/docs/pgadmin4/latest/config_py.html
	// - https://www.pgadmin.org/docs/pgadmin4/development/enabling_ldap_authentication.html
	if pgadmin.Spec.Config.LDAPBindPassword != nil {
		config = append(config, corev1.VolumeProjection{
			Secret: &corev1.SecretProjection{
				LocalObjectReference: pgadmin.Spec.Config.LDAPBindPassword.LocalObjectReference,
				Optional:             pgadmin.Spec.Config.LDAPBindPassword.Optional,
				Items: []corev1.KeyToPath{
					{
						Key:  pgadmin.Spec.Config.LDAPBindPassword.Key,
						Path: ldapFilePath,
					},
				},
			},
		})
	}

	return config
}

func startupScript(pgadmin *v1beta1.PGAdmin) []string {
	// loadServerCommandV7 is a python command leveraging the pgadmin v7 setup.py script
	// with the `--load-servers` flag to replace the servers registered to the admin user
	// with the contents of the `settingsClusterMapKey` file
	var loadServerCommandV7 = fmt.Sprintf(`python3 ${PGADMIN_DIR}/setup.py --load-servers %s/%s --user %s --replace`,
		configMountPath,
		clusterFilePath,
		fmt.Sprintf("admin@%s.%s.svc", pgadmin.Name, pgadmin.Namespace))

	// loadServerCommandV8 is a python command leveraging the pgadmin v8 setup.py script
	// with the `load-servers` sub-command to replace the servers registered to the admin user
	// with the contents of the `settingsClusterMapKey` file
	var loadServerCommandV8 = fmt.Sprintf(`python3 ${PGADMIN_DIR}/setup.py load-servers %s/%s --user %s --replace`,
		configMountPath,
		clusterFilePath,
		fmt.Sprintf("admin@%s.%s.svc", pgadmin.Name, pgadmin.Namespace))

	// setupCommands (v8 requires the 'setup-db' sub-command)
	var setupCommandV7 = "python3 ${PGADMIN_DIR}/setup.py"
	var setupCommandV8 = setupCommandV7 + " setup-db"

	// startCommands (v8 image includes Gunicorn)
	var startCommandV7 = "pgadmin4 &"
	var startCommandV8 = "gunicorn -c /etc/pgadmin/gunicorn_config.py --chdir $PGADMIN_DIR pgAdmin4:app &"

	// This script sets up, starts pgadmin, and runs the appropriate `loadServerCommand` to register the discovered servers.
	// pgAdmin is hosted by Gunicorn and uses a config file.
	// - https://www.pgadmin.org/docs/pgadmin4/development/server_deployment.html#standalone-gunicorn-configuration
	// - https://docs.gunicorn.org/en/latest/configure.html
	var startScript = fmt.Sprintf(`
export PGADMIN_SETUP_PASSWORD="$(date +%%s | sha256sum | base64 | head -c 32)"
PGADMIN_DIR=%s
APP_RELEASE=$(cd $PGADMIN_DIR && python3 -c "import config; print(config.APP_RELEASE)")

echo "Running pgAdmin4 Setup"
if [ $APP_RELEASE -eq 7 ]; then
    %s
else
    %s
fi

echo "Starting pgAdmin4"
PGADMIN4_PIDFILE=/tmp/pgadmin4.pid
if [ $APP_RELEASE -eq 7 ]; then
    %s
else
    %s
fi
echo $! > $PGADMIN4_PIDFILE

loadServerCommand() {
    if [ $APP_RELEASE -eq 7 ]; then
        %s
    else
        %s
    fi
}
loadServerCommand
`, pgAdminDir, setupCommandV7, setupCommandV8, startCommandV7, startCommandV8, loadServerCommandV7, loadServerCommandV8)

	// Use a Bash loop to periodically check:
	// 1. the mtime of the mounted configuration volume for shared/discovered servers.
	//   When it changes, reload the shared server configuration.
	// 2. that the pgadmin process is still running on the saved proc id.
	//	 When it isn't, we consider pgadmin stopped.
	//   Restart pgadmin and continue watching.

	// Coreutils `sleep` uses a lot of memory, so the following opens a file
	// descriptor and uses the timeout of the builtin `read` to wait. That same
	// descriptor gets closed and reopened to use the builtin `[ -nt` to check mtimes.
	// - https://unix.stackexchange.com/a/407383
	var reloadScript = `
exec {fd}<> <(:||:)
while read -r -t 5 -u "${fd}" ||:; do
    if [[ "${cluster_file}" -nt "/proc/self/fd/${fd}" ]] && loadServerCommand
    then
        exec {fd}>&- && exec {fd}<> <(:||:)
        stat --format='Loaded shared servers dated %y' "${cluster_file}"
    fi
    if [[ ! -d /proc/$(cat $PGADMIN4_PIDFILE) ]]
    then
        if [[ $APP_RELEASE -eq 7 ]]; then
            ` + startCommandV7 + `
        else
            ` + startCommandV8 + `
        fi
        echo $! > $PGADMIN4_PIDFILE
        echo "Restarting pgAdmin4"
    fi
done
`

	wrapper := `monitor() {` + startScript + reloadScript + `}; export cluster_file="$1"; export -f monitor; exec -a "$0" bash -ceu monitor`

	return []string{"bash", "-ceu", "--", wrapper, "pgadmin", fmt.Sprintf("%s/%s", configMountPath, clusterFilePath)}
}

// startupCommand returns an entrypoint that prepares the filesystem for pgAdmin.
func startupCommand() []string {
	// pgAdmin reads from the `/etc/pgadmin/config_system.py` file during startup
	// after all other config files.
	// - https://github.com/pgadmin-org/pgadmin4/blob/REL-7_7/docs/en_US/config_py.rst
	//
	// This command writes a script in `/etc/pgadmin/config_system.py` that reads from
	// the `pgadmin-settings.json` file and the config-database-uri and/or
	// `ldap-bind-password` files (if either exists) and sets those variables globally.
	// That way those values are available as pgAdmin configurations when pgAdmin starts.
	//
	// Note: All pgAdmin settings are uppercase alphanumeric with underscores, so ignore
	// any keys/names that are not.
	//
	// Note: set the pgAdmin LDAP_BIND_PASSWORD and CONFIG_DATABASE_URI settings from the
	// Secrets last in order to overwrite the respective configurations set via ConfigMap JSON.

	const (
		// ldapFilePath is the path for mounting the LDAP Bind Password
		ldapPasswordAbsolutePath = configMountPath + "/" + ldapFilePath

		// configDatabaseURIPath is the path for mounting the database URI connection string
		configDatabaseURIPathAbsolutePath = configMountPath + "/" + configDatabaseURIPath

		configSystem = `
import glob, json, re, os
DEFAULT_BINARY_PATHS = {'pg': sorted([''] + glob.glob('/usr/pgsql-*/bin')).pop()}
with open('` + configMountPath + `/` + configFilePath + `') as _f:
    _conf, _data = re.compile(r'[A-Z_0-9]+'), json.load(_f)
    if type(_data) is dict:
        globals().update({k: v for k, v in _data.items() if _conf.fullmatch(k)})
if os.path.isfile('` + ldapPasswordAbsolutePath + `'):
    with open('` + ldapPasswordAbsolutePath + `') as _f:
        LDAP_BIND_PASSWORD = _f.read()
if os.path.isfile('` + configDatabaseURIPathAbsolutePath + `'):
    with open('` + configDatabaseURIPathAbsolutePath + `') as _f:
        CONFIG_DATABASE_URI = _f.read()
`
		// gunicorn reads from the `/etc/pgadmin/gunicorn_config.py` file during startup
		// after all other config files.
		// - https://docs.gunicorn.org/en/latest/configure.html#configuration-file
		//
		// This command writes a script in `/etc/pgadmin/gunicorn_config.py` that reads
		// from the `gunicorn-config.json` file and sets those variables globally.
		// That way those values are available as settings when gunicorn starts.
		//
		// Note: All gunicorn settings are lowercase with underscores, so ignore
		// any keys/names that are not.
		gunicornConfig = `
import json, re
with open('` + configMountPath + `/` + gunicornConfigFilePath + `') as _f:
    _conf, _data = re.compile(r'[a-z_]+'), json.load(_f)
    if type(_data) is dict:
        globals().update({k: v for k, v in _data.items() if _conf.fullmatch(k)})
`
	)

	args := []string{strings.TrimLeft(configSystem, "\n"), strings.TrimLeft(gunicornConfig, "\n")}

	script := strings.Join([]string{
		// Use the initContainer to create this path to avoid the error noted here:
		// - https://github.com/kubernetes/kubernetes/issues/121294
		`mkdir -p /etc/pgadmin/conf.d`,
		// Write the system configuration into a read-only file.
		`(umask a-w && echo "$1" > ` + scriptMountPath + `/config_system.py` + `)`,
		// Write the server configuration into a read-only file.
		`(umask a-w && echo "$2" > ` + scriptMountPath + `/gunicorn_config.py` + `)`,
	}, "\n")

	return append([]string{"bash", "-ceu", "--", script, "startup"}, args...)
}

// podSecurityContext returns a v1.PodSecurityContext for pgadmin that can write
// to PersistentVolumes.
func podSecurityContext(r *PGAdminReconciler) *corev1.PodSecurityContext {
	podSecurityContext := initialize.PodSecurityContext()

	// TODO (dsessler7): Add ability to add supplemental groups

	// OpenShift assigns a filesystem group based on a SecurityContextConstraint.
	// Otherwise, set a filesystem group so pgAdmin can write to files
	// regardless of the UID or GID of a container.
	// - https://cloud.redhat.com/blog/a-guide-to-openshift-and-uids
	// - https://docs.k8s.io/tasks/configure-pod-container/security-context/
	// - https://docs.openshift.com/container-platform/4.14/authentication/managing-security-context-constraints.html
	if !r.IsOpenShift {
		podSecurityContext.FSGroup = initialize.Int64(2)
	}

	return podSecurityContext
}
