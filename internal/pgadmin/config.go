// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgadmin

import (
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

const (
	// tmp volume to hold the nss_wrapper, process and socket files
	// both the '/tmp' mount path and '/etc/httpd/run' mount path
	// mount the 'tmp' volume
	tmpVolume = "tmp"

	// runMountPath holds the pgAdmin run path, which mounts the 'tmp' volume
	runMountPath = "/etc/httpd/run"

	// log volume and path where the pgadmin4.log is located
	logVolume    = "pgadmin-log"
	logMountPath = "/var/log/pgadmin"

	// data volume and path to hold persistent pgAdmin data
	dataVolume    = "pgadmin-data"
	dataMountPath = "/var/lib/pgadmin"

	// ldapPasswordPath is the path for mounting the LDAP Bind Password
	ldapPasswordPath         = "~postgres-operator/ldap-bind-password" /* #nosec */
	ldapPasswordAbsolutePath = configMountPath + "/" + ldapPasswordPath

	// TODO(tjmoore4): The login and password implementation will be updated in
	// upcoming enhancement work.

	// initial pgAdmin login email address
	loginEmail = "admin"

	// initial pgAdmin login password
	loginPassword = "admin"

	// default pgAdmin port
	pgAdminPort = 5050

	// configMountPath is where to mount configuration files, secrets, etc.
	configMountPath = "/etc/pgadmin/conf.d"

	settingsAbsolutePath   = configMountPath + "/" + settingsProjectionPath
	settingsConfigMapKey   = "pgadmin-settings.json"
	settingsProjectionPath = "~postgres-operator/pgadmin.json"

	// startupMountPath is where to mount a temporary directory that is only
	// writable during Pod initialization.
	//
	// NOTE: No ConfigMap nor Secret should ever be mounted here because they
	// could be used to inject code through "config_system.py".
	startupMountPath = "/etc/pgadmin"

	// configSystemAbsolutePath is imported by pgAdmin after all other config files.
	// - https://github.com/pgadmin-org/pgadmin4/blob/REL-4_30/docs/en_US/config_py.rst
	configSystemAbsolutePath = startupMountPath + "/config_system.py"
)

// podConfigFiles returns projections of pgAdmin's configuration files to
// include in the configuration volume.
func podConfigFiles(configmap *corev1.ConfigMap, spec v1beta1.PGAdminPodSpec) []corev1.VolumeProjection {
	config := append(append([]corev1.VolumeProjection{}, spec.Config.Files...),
		[]corev1.VolumeProjection{
			{
				ConfigMap: &corev1.ConfigMapProjection{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configmap.Name,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  settingsConfigMapKey,
							Path: settingsProjectionPath,
						},
					},
				},
			},
		}...)

	// To enable LDAP authentication for pgAdmin, various LDAP settings must be configured.
	// While most of the required configuration can be set using the 'settings'
	// feature on the spec (.Spec.UserInterface.PGAdmin.Config.Settings), those
	// values are stored in a ConfigMap in plaintext.
	// As a special case, here we mount a provided Secret containing the LDAP_BIND_PASSWORD
	// for use with the other pgAdmin LDAP configuration.
	// - https://www.pgadmin.org/docs/pgadmin4/latest/config_py.html
	// - https://www.pgadmin.org/docs/pgadmin4/development/enabling_ldap_authentication.html
	if spec.Config.LDAPBindPassword != nil {
		config = append(config, corev1.VolumeProjection{
			Secret: &corev1.SecretProjection{
				LocalObjectReference: spec.Config.LDAPBindPassword.LocalObjectReference,
				Optional:             spec.Config.LDAPBindPassword.Optional,
				Items: []corev1.KeyToPath{
					{
						Key:  spec.Config.LDAPBindPassword.Key,
						Path: ldapPasswordPath,
					},
				},
			},
		})
	}

	return config
}

// startupCommand returns an entrypoint that prepares the filesystem for pgAdmin.
func startupCommand() []string {
	// pgAdmin reads from the following file by importing its public names.
	// Make sure to assign only to variables that begin with underscore U+005F.
	// - https://github.com/pgadmin-org/pgadmin4/blob/REL-4_30/web/config.py#L669
	// - https://docs.python.org/3/reference/simple_stmts.html#import
	//
	// DEFAULT_BINARY_PATHS contains the paths to various client tools. The "pg"
	// key is for PostgreSQL. Use the latest version found in "/usr" or fallback
	// to the default of empty string.
	// - https://github.com/pgadmin-org/pgadmin4/blob/REL-4_30/web/config.py#L415
	//
	//     Python 3.6.8 (default, Sep 10 2021, 09:13:53)
	//     >>> sorted(['']+[]).pop()
	//     ''
	//     >>> sorted(['']+['/pg13','/pg10']).pop()
	//     '/pg13'
	//
	// Set all remaining variables from the JSON in settingsAbsolutePath. All
	// pgAdmin settings are uppercase with underscores, so ignore any keys/names
	// that are not.
	//
	// Lastly, set pgAdmin's LDAP_BIND_PASSWORD setting, if the value was provided
	// via Secret. As this assignment happens after any values provided via the
	// 'Settings' ConfigMap loaded above, this value will overwrite any previous
	// configuration of LDAP_BIND_PASSWORD (that is, last write wins).
	const configSystem = `
import glob, json, re, os
DEFAULT_BINARY_PATHS = {'pg': sorted([''] + glob.glob('/usr/pgsql-*/bin')).pop()}
with open('` + settingsAbsolutePath + `') as _f:
    _conf, _data = re.compile(r'[A-Z_0-9]+'), json.load(_f)
    if type(_data) is dict:
        globals().update({k: v for k, v in _data.items() if _conf.fullmatch(k)})
if os.path.isfile('` + ldapPasswordAbsolutePath + `'):
    with open('` + ldapPasswordAbsolutePath + `') as _f:
        LDAP_BIND_PASSWORD = _f.read()
`

	args := []string{strings.TrimLeft(configSystem, "\n")}

	script := strings.Join([]string{
		// Write the system configuration into a read-only file.
		`(umask a-w && echo "$1" > ` + configSystemAbsolutePath + `)`,
	}, "\n")

	return append([]string{"bash", "-ceu", "--", script, "startup"}, args...)
}

// systemSettings returns pgAdmin settings as a value that can be marshaled to JSON.
func systemSettings(spec *v1beta1.PGAdminPodSpec) map[string]interface{} {
	settings := *spec.Config.Settings.DeepCopy()
	if settings == nil {
		settings = make(map[string]interface{})
	}

	// SERVER_MODE must always be enabled when running on a webserver.
	// - https://github.com/pgadmin-org/pgadmin4/blob/REL-4_30/web/config.py#L105
	settings["SERVER_MODE"] = true

	return settings
}
