/*
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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
	tmpVolume    = "tmp"
	tmpMountPath = "/tmp"
	runMountPath = "/etc/httpd/run"

	// log volume and path where the pgadmin4.log is located
	logVolume    = "pgadmin-log"
	logMountPath = "/var/log/pgadmin"

	// data volume and path to hold persistent pgAdmin data
	dataVolume    = "pgadmin-data"
	dataMountPath = "/var/lib/pgadmin"

	// TODO(tjmoore4): The login and password implementation will be updated in
	// upcoming enhancement work.

	// initial pgAdmin login email address
	loginEmail = "admin"

	// initial pgAdmin login password
	loginPassword = "admin"

	// default pgAdmin port
	defaultPort = 5050

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
	// - https://git.postgresql.org/gitweb/?p=pgadmin4.git;f=docs/en_US/config_py.rst;hb=REL-4_30
	configSystemAbsolutePath = startupMountPath + "/config_system.py"
)

// podConfigFiles returns projections of pgAdmin's configuration files to
// include in the configuration volume.
func podConfigFiles(configmap *corev1.ConfigMap, spec v1beta1.PGAdminPodSpec) []corev1.VolumeProjection {
	return append(append([]corev1.VolumeProjection{}, spec.Config.Files...),
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
}

// startupCommand returns an entrypoint that prepares the filesystem for pgAdmin.
func startupCommand() []string {
	// pgAdmin reads from the following file by importing its public names.
	// Make sure to assign only to variables that begin with underscore U+005F.
	// - https://git.postgresql.org/gitweb/?p=pgadmin4.git;f=web/config.py;hb=REL-4_30#l669
	// - https://docs.python.org/3/reference/simple_stmts.html#import
	//
	// DEFAULT_BINARY_PATHS contains the paths to various client tools. The "pg"
	// key is for PostgreSQL. Use the latest version found in "/usr" or fallback
	// to the default of empty string.
	// - https://git.postgresql.org/gitweb/?p=pgadmin4.git;f=web/config.py;hb=REL-4_30#l415
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
	const configSystem = `
import glob, json, re
DEFAULT_BINARY_PATHS = {'pg': sorted([''] + glob.glob('/usr/pgsql-*/bin')).pop()}
with open('` + settingsAbsolutePath + `') as _f:
    _conf, _data = re.compile(r'[A-Z_]+'), json.load(_f)
    if type(_data) is dict:
        globals().update({k: v for k, v in _data.items() if _conf.fullmatch(k)})
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
	// - https://git.postgresql.org/gitweb/?p=pgadmin4.git;f=web/config.py;hb=REL-4_30#l105
	settings["SERVER_MODE"] = true

	return settings
}
