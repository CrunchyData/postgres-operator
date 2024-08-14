// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgbackrest

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/api/resource"

	corev1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/internal/testing/require"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestStanzaCreateOrUpgrade(t *testing.T) {
	shellcheck := require.ShellCheck(t)

	ctx := context.Background()
	configHash := "7f5d4d5bdc"
	expectedCommand := []string{"bash", "-ceu", "--", `
declare -r hash="$1" stanza="$2" hash_msg="$3" vol_msg="$4" check_repo_cmd="$5"
if [[ "$(< /etc/pgbackrest/conf.d/config-hash)" != "${hash}" ]]; then
    printf >&2 "%s" "${hash_msg}"; exit 1;
elif ! bash -c "${check_repo_cmd}"; then
 	 printf >&2 "%s" "${vol_msg}"; exit 1;
else
    pgbackrest stanza-create --stanza="${stanza}" || pgbackrest stanza-upgrade --stanza="${stanza}"
fi
`,
		"-", "7f5d4d5bdc", "db", "postgres operator error: pgBackRest config hash mismatch",
		"postgres operator error: pgBackRest stale volume-backed repo configuration",
		"grep repo1-path /etc/pgbackrest/conf.d/pgbackrest_instance.conf",
	}

	var shellCheckScript string
	stanzaExec := func(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer,
		command ...string) error {

		// verify the command created by StanzaCreate() matches the expected command
		assert.DeepEqual(t, command, expectedCommand)

		assert.Assert(t, len(command) > 3)
		shellCheckScript = command[3]

		return nil
	}
	postgresCluster := &v1beta1.PostgresCluster{
		Spec: v1beta1.PostgresClusterSpec{
			Backups: v1beta1.Backups{
				PGBackRest: v1beta1.PGBackRestArchive{
					Repos: []v1beta1.PGBackRestRepo{{
						Name: "repo1",
						Volume: &v1beta1.RepoPVC{
							VolumeClaimSpec: corev1.PersistentVolumeClaimSpec{
								AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
								Resources: corev1.VolumeResourceRequirements{
									Requests: map[corev1.ResourceName]resource.Quantity{
										corev1.ResourceStorage: resource.MustParse("1Gi"),
									},
								},
							},
						},
					}, {
						Name: "repo2",
						S3: &v1beta1.RepoS3{
							Bucket:   "bucket",
							Endpoint: "endpoint",
							Region:   "region",
						},
					}},
				},
			},
		},
	}

	configHashMismatch, err := Executor(stanzaExec).StanzaCreateOrUpgrade(ctx, configHash, postgresCluster)
	assert.NilError(t, err)
	assert.Assert(t, !configHashMismatch)

	// shell check the script
	// Write out that inline script.
	dir := t.TempDir()
	file := filepath.Join(dir, "script.bash")
	assert.NilError(t, os.WriteFile(file, []byte(shellCheckScript), 0o600))

	// Expect shellcheck to be happy.
	cmd := exec.Command(shellcheck, "--enable=all", file)
	output, err := cmd.CombinedOutput()
	assert.NilError(t, err, "%q\n%s", cmd.Args, output)
}
