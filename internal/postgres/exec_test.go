// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/internal/testing/require"
)

// This example demonstrates how Executor can work with exec.Cmd.
func ExampleExecutor_execCmd() {
	_ = Executor(func(
		ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
	) error {
		// #nosec G204 Nothing calls the function defined in this example.
		cmd := exec.CommandContext(ctx, command[0], command[1:]...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
		return cmd.Run()
	})
}

func TestExecutorExec(t *testing.T) {
	expected := errors.New("pass-through")
	fn := func(
		_ context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
	) error {
		b, err := io.ReadAll(stdin)
		assert.NilError(t, err)
		assert.Equal(t, string(b), `statements; to run;`)

		assert.DeepEqual(t, command, []string{
			"psql", "-Xw", "--file=-",
			"--set=CASE=sEnSiTiVe",
			"--set=different=vars",
			"--set=lots=of",
		})

		_, _ = io.WriteString(stdout, "some stdout")
		_, _ = io.WriteString(stderr, "and stderr")
		return expected
	}

	stdout, stderr, err := Executor(fn).Exec(
		context.Background(),
		strings.NewReader(`statements; to run;`),
		map[string]string{
			"lots":      "of",
			"different": "vars",
			"CASE":      "sEnSiTiVe",
		})

	assert.Equal(t, expected, err, "expected function to be called")
	assert.Equal(t, stdout, "some stdout")
	assert.Equal(t, stderr, "and stderr")
}

func TestExecutorExecInAllDatabases(t *testing.T) {
	expected := errors.New("exact")
	fn := func(
		_ context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
	) error {
		b, err := io.ReadAll(stdin)
		assert.NilError(t, err)
		assert.Equal(t, string(b), `the; stuff;`)

		assert.DeepEqual(t, command, []string{
			"bash", "-ceu", "--", `
sql_target=$(< /dev/stdin)
sql_databases="$1"
shift 1

databases=$(psql "$@" -Xw -Aqt --file=- <<< "${sql_databases}")
while IFS= read -r database; do
	PGDATABASE="${database}" psql "$@" -Xw --file=- <<< "${sql_target}"
done <<< "${databases}"
`,
			"-",
			`SET search_path = '';SELECT datname FROM pg_catalog.pg_database WHERE datallowconn AND datname NOT IN ('template0')`,
			"--set=CASE=sEnSiTiVe",
			"--set=different=vars",
			"--set=lots=of",
		})

		_, _ = io.WriteString(stdout, "some stdout")
		_, _ = io.WriteString(stderr, "and stderr")
		return expected
	}

	stdout, stderr, err := Executor(fn).ExecInAllDatabases(
		context.Background(),
		`the; stuff;`,
		map[string]string{
			"lots":      "of",
			"different": "vars",
			"CASE":      "sEnSiTiVe",
		})

	assert.Equal(t, expected, err, "expected function to be called")
	assert.Equal(t, stdout, "some stdout")
	assert.Equal(t, stderr, "and stderr")
}

func TestExecutorExecInDatabasesFromQuery(t *testing.T) {
	expected := errors.New("splat")
	fn := func(
		_ context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
	) error {
		b, err := io.ReadAll(stdin)
		assert.NilError(t, err)
		assert.Equal(t, string(b), `statements; to run;`)

		assert.DeepEqual(t, command, []string{
			"bash", "-ceu", "--", `
sql_target=$(< /dev/stdin)
sql_databases="$1"
shift 1

databases=$(psql "$@" -Xw -Aqt --file=- <<< "${sql_databases}")
while IFS= read -r database; do
	PGDATABASE="${database}" psql "$@" -Xw --file=- <<< "${sql_target}"
done <<< "${databases}"
`,
			"-",
			`db query`,
			"--set=CASE=sEnSiTiVe",
			"--set=different=vars",
			"--set=lots=of",
		})

		// Use the PGDATABASE environment variable to ensure the value is not
		// interpreted as a connection string.
		//
		// > $ psql -Xw -d 'host=127.0.0.1'
		// > psql: error: fe_sendauth: no password supplied
		// >
		// > $ PGDATABASE='host=127.0.0.1' psql -Xw
		// > psql: error: FATAL:  database "host=127.0.0.1" does not exist
		//
		// TODO(cbandy): Create a test that actually runs psql.
		assert.Assert(t, strings.Contains(command[3], `PGDATABASE="${database}" psql`))

		_, _ = io.WriteString(stdout, "some stdout")
		_, _ = io.WriteString(stderr, "and stderr")
		return expected
	}

	stdout, stderr, err := Executor(fn).ExecInDatabasesFromQuery(
		context.Background(), `db query`, `statements; to run;`, map[string]string{
			"lots":      "of",
			"different": "vars",
			"CASE":      "sEnSiTiVe",
		})

	assert.Equal(t, expected, err, "expected function to be called")
	assert.Equal(t, stdout, "some stdout")
	assert.Equal(t, stderr, "and stderr")

	t.Run("ShellCheck", func(t *testing.T) {
		shellcheck := require.ShellCheck(t)

		_, _, _ = Executor(func(
			_ context.Context, _ io.Reader, _, _ io.Writer, command ...string,
		) error {
			// Expect a bash command with an inline script.
			assert.DeepEqual(t, command[:3], []string{"bash", "-ceu", "--"})
			assert.Assert(t, len(command) > 3)
			script := command[3]

			// Write out that inline script.
			dir := t.TempDir()
			file := filepath.Join(dir, "script.bash")
			assert.NilError(t, os.WriteFile(file, []byte(script), 0o600))

			// Expect shellcheck to be happy.
			cmd := exec.Command(shellcheck, "--enable=all", file)
			output, err := cmd.CombinedOutput()
			assert.NilError(t, err, "%q\n%s", cmd.Args, output)

			return nil
		}).ExecInDatabasesFromQuery(context.Background(), "", "", nil)
	})
}
