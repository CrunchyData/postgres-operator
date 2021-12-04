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

package pgbackrest

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/internal/testing/require"
)

func TestStanzaCreateOrUpgrade(t *testing.T) {
	shellcheck := require.ShellCheck(t)

	ctx := context.Background()
	configHash := "7f5d4d5bdc"
	expectedCommand := []string{"bash", "-ceu", "--", `
declare -r hash="$1" stanza="$2" message="$3" cmd="$4"
if [[ "$(< /etc/pgbackrest/conf.d/config-hash)" != "${hash}" ]]; then
    printf >&2 "%s" "${message}"; exit 1;
else
    pgbackrest "${cmd}" --stanza="${stanza}"
fi
`,
		"-", "7f5d4d5bdc", "db", "postgres operator error: pgBackRest config hash mismatch",
		"stanza-create"}

	var shellCheckScript string
	stanzaExec := func(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer,
		command ...string) error {

		// verify the command created by StanzaCreate() matches the expected command
		assert.DeepEqual(t, command, expectedCommand)

		assert.Assert(t, len(command) > 3)
		shellCheckScript = command[3]

		return nil
	}

	configHashMismatch, err := Executor(stanzaExec).StanzaCreateOrUpgrade(ctx, configHash, false)
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
