// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgadmin

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestPodConfigFiles(t *testing.T) {
	configmap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "some-cm"}}

	spec := v1beta1.PGAdminPodSpec{
		Config: v1beta1.PGAdminConfiguration{Files: []corev1.VolumeProjection{{
			Secret: &corev1.SecretProjection{LocalObjectReference: corev1.LocalObjectReference{
				Name: "test-secret",
			}},
		}, {
			ConfigMap: &corev1.ConfigMapProjection{LocalObjectReference: corev1.LocalObjectReference{
				Name: "test-cm",
			}},
		}}},
	}

	projections := podConfigFiles(configmap, spec)
	assert.Assert(t, cmp.MarshalMatches(projections, `
- secret:
    name: test-secret
- configMap:
    name: test-cm
- configMap:
    items:
    - key: pgadmin-settings.json
      path: ~postgres-operator/pgadmin.json
    name: some-cm
	`))
}

func TestStartupCommand(t *testing.T) {
	assert.Assert(t, cmp.MarshalMatches(startupCommand(), `
- bash
- -ceu
- --
- (umask a-w && echo "$1" > /etc/pgadmin/config_system.py)
- startup
- |
  import glob, json, re, os
  DEFAULT_BINARY_PATHS = {'pg': sorted([''] + glob.glob('/usr/pgsql-*/bin')).pop()}
  with open('/etc/pgadmin/conf.d/~postgres-operator/pgadmin.json') as _f:
      _conf, _data = re.compile(r'[A-Z_0-9]+'), json.load(_f)
      if type(_data) is dict:
          globals().update({k: v for k, v in _data.items() if _conf.fullmatch(k)})
  if os.path.isfile('/etc/pgadmin/conf.d/~postgres-operator/ldap-bind-password'):
      with open('/etc/pgadmin/conf.d/~postgres-operator/ldap-bind-password') as _f:
          LDAP_BIND_PASSWORD = _f.read()
`))

	t.Run("ShellCheck", func(t *testing.T) {
		command := startupCommand()
		shellcheck := require.ShellCheck(t)

		assert.Assert(t, len(command) > 3)
		dir := t.TempDir()
		file := filepath.Join(dir, "script.bash")
		assert.NilError(t, os.WriteFile(file, []byte(command[3]), 0o600))

		// Expect shellcheck to be happy.
		cmd := exec.Command(shellcheck, "--enable=all", file)
		output, err := cmd.CombinedOutput()
		assert.NilError(t, err, "%q\n%s", cmd.Args, output)
	})

	t.Run("ConfigSystemFlake8", func(t *testing.T) {
		command := startupCommand()
		flake8 := require.Flake8(t)

		assert.Assert(t, len(command) > 5)
		dir := t.TempDir()
		file := filepath.Join(dir, "script.py")
		assert.NilError(t, os.WriteFile(file, []byte(command[5]), 0o600))

		// Expect flake8 to be happy. Ignore "E401 multiple imports on one line"
		// in addition to the defaults. The file contents appear in PodSpec, so
		// allow lines longer than the default to save some vertical space.
		cmd := exec.Command(flake8, "--extend-ignore=E401", "--max-line-length=99", file)
		output, err := cmd.CombinedOutput()
		assert.NilError(t, err, "%q\n%s", cmd.Args, output)
	})
}

func TestSystemSettings(t *testing.T) {
	spec := new(v1beta1.PGAdminPodSpec)
	assert.Assert(t, cmp.MarshalMatches(systemSettings(spec), `
SERVER_MODE: true
	`))

	spec.Config.Settings = map[string]interface{}{
		"ALLOWED_HOSTS": []interface{}{"225.0.0.0/8", "226.0.0.0/7", "228.0.0.0/6"},
	}
	assert.Assert(t, cmp.MarshalMatches(systemSettings(spec), `
ALLOWED_HOSTS:
- 225.0.0.0/8
- 226.0.0.0/7
- 228.0.0.0/6
SERVER_MODE: true
	`))
}
