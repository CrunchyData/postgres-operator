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

package pgbackrest

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestCreatePGBackRestConfigMapIntent(t *testing.T) {
	cluster := v1beta1.PostgresCluster{}
	cluster.Namespace = "ns1"
	cluster.Name = "hippo-dance"

	cluster.Spec.Port = initialize.Int32(2345)
	cluster.Spec.PostgresVersion = 12

	domain := naming.KubernetesClusterDomain(context.Background())

	t.Run("NoVolumeRepo", func(t *testing.T) {
		cluster := cluster.DeepCopy()
		cluster.Spec.Backups.PGBackRest.Repos = nil

		configmap := CreatePGBackRestConfigMapIntent(cluster,
			"", "number", "pod-service-name", "test-ns",
			[]string{"some-instance"})

		assert.Equal(t, configmap.Data["config-hash"], "number")
		assert.Equal(t, configmap.Data["pgbackrest-server.conf"], "")
	})

	t.Run("DedicatedRepoHost", func(t *testing.T) {
		cluster := cluster.DeepCopy()
		cluster.Spec.Backups.PGBackRest.Global = map[string]string{
			"repo3-test": "something",
		}
		cluster.Spec.Backups.PGBackRest.Repos = []v1beta1.PGBackRestRepo{
			{
				Name:   "repo1",
				Volume: &v1beta1.RepoPVC{},
			},
			{
				Name:  "repo2",
				Azure: &v1beta1.RepoAzure{Container: "a-container"},
			},
			{
				Name: "repo3",
				GCS:  &v1beta1.RepoGCS{Bucket: "g-bucket"},
			},
			{
				Name: "repo4",
				S3: &v1beta1.RepoS3{
					Bucket: "s-bucket", Endpoint: "endpoint-s", Region: "earth",
				},
			},
		}

		configmap := CreatePGBackRestConfigMapIntent(cluster,
			"repo-hostname", "abcde12345", "pod-service-name", "test-ns",
			[]string{"some-instance"})

		assert.DeepEqual(t, configmap.Annotations, map[string]string{})
		assert.DeepEqual(t, configmap.Labels, map[string]string{
			"postgres-operator.crunchydata.com/cluster":           "hippo-dance",
			"postgres-operator.crunchydata.com/pgbackrest":        "",
			"postgres-operator.crunchydata.com/pgbackrest-config": "",
		})

		assert.Equal(t, configmap.Data["config-hash"], "abcde12345")
		assert.Equal(t, configmap.Data["pgbackrest_repo.conf"], strings.Trim(`
# Generated by postgres-operator. DO NOT EDIT.
# Your changes will not be saved.

[global]
log-path = /tmp
repo1-path = /pgbackrest/repo1
repo2-azure-container = a-container
repo2-path = /pgbackrest/repo2
repo2-type = azure
repo3-gcs-bucket = g-bucket
repo3-path = /pgbackrest/repo3
repo3-test = something
repo3-type = gcs
repo4-path = /pgbackrest/repo4
repo4-s3-bucket = s-bucket
repo4-s3-endpoint = endpoint-s
repo4-s3-region = earth
repo4-type = s3

[db]
pg1-host = some-instance-0.pod-service-name.test-ns.svc.`+domain+`
pg1-host-ca-file = /etc/pgbackrest/conf.d/~postgres-operator/tls-ca.crt
pg1-host-cert-file = /etc/pgbackrest/conf.d/~postgres-operator/client-tls.crt
pg1-host-key-file = /etc/pgbackrest/conf.d/~postgres-operator/client-tls.key
pg1-host-type = tls
pg1-path = /pgdata/pg12
pg1-port = 2345
pg1-socket-path = /tmp/postgres
		`, "\t\n")+"\n")

		assert.Equal(t, configmap.Data["pgbackrest_instance.conf"], strings.Trim(`
# Generated by postgres-operator. DO NOT EDIT.
# Your changes will not be saved.

[global]
log-path = /tmp
repo1-host = repo-hostname-0.pod-service-name.test-ns.svc.`+domain+`
repo1-host-ca-file = /etc/pgbackrest/conf.d/~postgres-operator/tls-ca.crt
repo1-host-cert-file = /etc/pgbackrest/conf.d/~postgres-operator/client-tls.crt
repo1-host-key-file = /etc/pgbackrest/conf.d/~postgres-operator/client-tls.key
repo1-host-type = tls
repo1-host-user = postgres
repo1-path = /pgbackrest/repo1
repo2-azure-container = a-container
repo2-path = /pgbackrest/repo2
repo2-type = azure
repo3-gcs-bucket = g-bucket
repo3-path = /pgbackrest/repo3
repo3-test = something
repo3-type = gcs
repo4-path = /pgbackrest/repo4
repo4-s3-bucket = s-bucket
repo4-s3-endpoint = endpoint-s
repo4-s3-region = earth
repo4-type = s3

[db]
pg1-path = /pgdata/pg12
pg1-port = 2345
pg1-socket-path = /tmp/postgres
		`, "\t\n")+"\n")
	})

	t.Run("CustomMetadata", func(t *testing.T) {
		cluster := cluster.DeepCopy()
		cluster.Spec.Metadata = &v1beta1.Metadata{
			Annotations: map[string]string{
				"ak1": "cluster-av1",
				"ak2": "cluster-av2",
			},
			Labels: map[string]string{
				"lk1": "cluster-lv1",
				"lk2": "cluster-lv2",

				"postgres-operator.crunchydata.com/cluster": "cluster-ignored",
			},
		}
		cluster.Spec.Backups.PGBackRest.Metadata = &v1beta1.Metadata{
			Annotations: map[string]string{
				"ak2": "backups-av2",
				"ak3": "backups-av3",
			},
			Labels: map[string]string{
				"lk2": "backups-lv2",
				"lk3": "backups-lv3",

				"postgres-operator.crunchydata.com/cluster": "backups-ignored",
			},
		}

		configmap := CreatePGBackRestConfigMapIntent(cluster,
			"any", "any", "any", "any", nil)

		assert.DeepEqual(t, configmap.Annotations, map[string]string{
			"ak1": "cluster-av1",
			"ak2": "backups-av2",
			"ak3": "backups-av3",
		})
		assert.DeepEqual(t, configmap.Labels, map[string]string{
			"lk1": "cluster-lv1",
			"lk2": "backups-lv2",
			"lk3": "backups-lv3",

			"postgres-operator.crunchydata.com/cluster":           "hippo-dance",
			"postgres-operator.crunchydata.com/pgbackrest":        "",
			"postgres-operator.crunchydata.com/pgbackrest-config": "",
		})
	})
}

func TestReloadCommand(t *testing.T) {
	shellcheck, err := exec.LookPath("shellcheck")
	if err != nil {
		t.Skip(`requires "shellcheck" executable`)
	} else {
		output, err := exec.Command(shellcheck, "--version").CombinedOutput()
		assert.NilError(t, err)
		t.Logf("using %q:\n%s", shellcheck, output)
	}

	command := reloadCommand("some-name")

	// Expect a bash command with an inline script.
	assert.DeepEqual(t, command[:3], []string{"bash", "-ceu", "--"})
	assert.Assert(t, len(command) > 3)

	// Write out that inline script.
	dir := t.TempDir()
	file := filepath.Join(dir, "script.bash")
	assert.NilError(t, os.WriteFile(file, []byte(command[3]), 0o600))

	// Expect shellcheck to be happy.
	cmd := exec.Command(shellcheck, "--enable=all", file)
	output, err := cmd.CombinedOutput()
	assert.NilError(t, err, "%q\n%s", cmd.Args, output)
}

func TestReloadCommandPrettyYAML(t *testing.T) {
	b, err := yaml.Marshal(reloadCommand("any"))
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(string(b), "\n- |"),
		"expected literal block scalar, got:\n%s", b)
}

func TestRestoreCommand(t *testing.T) {
	shellcheck, err := exec.LookPath("shellcheck")
	if err != nil {
		t.Skip(`requires "shellcheck" executable`)
	} else {
		output, err := exec.Command(shellcheck, "--version").CombinedOutput()
		assert.NilError(t, err)
		t.Logf("using %q:\n%s", shellcheck, output)
	}

	pgdata := "/pgdata/pg13"
	opts := []string{
		"--stanza=" + DefaultStanzaName, "--pg1-path=" + pgdata,
		"--repo=1"}
	command := RestoreCommand(pgdata, strings.Join(opts, " "))

	assert.DeepEqual(t, command[:3], []string{"bash", "-ceu", "--"})
	assert.Assert(t, len(command) > 3)

	dir := t.TempDir()
	file := filepath.Join(dir, "script.bash")
	assert.NilError(t, ioutil.WriteFile(file, []byte(command[3]), 0o600))

	cmd := exec.Command(shellcheck, "--enable=all", file)
	output, err := cmd.CombinedOutput()
	assert.NilError(t, err, "%q\n%s", cmd.Args, output)
}

func TestRestoreCommandPrettyYAML(t *testing.T) {
	b, err := yaml.Marshal(RestoreCommand("/dir", "--options"))
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(string(b), "\n- |"),
		"expected literal block scalar, got:\n%s", b)
}

func TestServerConfig(t *testing.T) {
	cluster := &v1beta1.PostgresCluster{}
	cluster.UID = "shoe"

	assert.Equal(t, serverConfig(cluster).String(), `
[global]
tls-server-address = ::
tls-server-auth = pgbackrest@shoe=*
tls-server-ca-file = /etc/pgbackrest/conf.d/~postgres-operator/tls-ca.crt
tls-server-cert-file = /etc/pgbackrest/server/server-tls.crt
tls-server-key-file = /etc/pgbackrest/server/server-tls.key

[global:server-start]
log-level-console = detail
log-level-file = off
log-level-stderr = error
log-timestamp = n
`)
}
