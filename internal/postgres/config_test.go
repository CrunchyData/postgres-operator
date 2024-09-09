// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestConfigDirectory(t *testing.T) {
	cluster := new(v1beta1.PostgresCluster)
	cluster.Spec.PostgresVersion = 11

	assert.Equal(t, ConfigDirectory(cluster), "/pgdata/pg11")
}

func TestDataDirectory(t *testing.T) {
	cluster := new(v1beta1.PostgresCluster)
	cluster.Spec.PostgresVersion = 12

	assert.Equal(t, DataDirectory(cluster), "/pgdata/pg12")
}

func TestWALDirectory(t *testing.T) {
	cluster := new(v1beta1.PostgresCluster)
	cluster.Spec.PostgresVersion = 13

	// without WAL volume
	instance := new(v1beta1.PostgresInstanceSetSpec)
	assert.Equal(t, WALDirectory(cluster, instance), "/pgdata/pg13_wal")

	// with WAL volume
	instance.WALVolumeClaimSpec = new(corev1.PersistentVolumeClaimSpec)
	assert.Equal(t, WALDirectory(cluster, instance), "/pgwal/pg13_wal")
}

func TestBashHalt(t *testing.T) {
	t.Run("NoPipeline", func(t *testing.T) {
		cmd := exec.Command("bash")
		cmd.Args = append(cmd.Args, "-c", "--", bashHalt+`; halt ab cd e`)

		var exit *exec.ExitError
		stdout, err := cmd.Output()
		assert.Assert(t, errors.As(err, &exit))
		assert.Equal(t, string(stdout), "", "expected no stdout")
		assert.Equal(t, string(exit.Stderr), "ab cd e\n")
		assert.Equal(t, exit.ExitCode(), 1)
	})

	t.Run("PipelineZeroStatus", func(t *testing.T) {
		cmd := exec.Command("bash")
		cmd.Args = append(cmd.Args, "-c", "--", bashHalt+`; true && halt message`)

		var exit *exec.ExitError
		stdout, err := cmd.Output()
		assert.Assert(t, errors.As(err, &exit))
		assert.Equal(t, string(stdout), "", "expected no stdout")
		assert.Equal(t, string(exit.Stderr), "message\n")
		assert.Equal(t, exit.ExitCode(), 1)
	})

	t.Run("PipelineNonZeroStatus", func(t *testing.T) {
		cmd := exec.Command("bash")
		cmd.Args = append(cmd.Args, "-c", "--", bashHalt+`; (exit 99) || halt $'multi\nline'`)

		var exit *exec.ExitError
		stdout, err := cmd.Output()
		assert.Assert(t, errors.As(err, &exit))
		assert.Equal(t, string(stdout), "", "expected no stdout")
		assert.Equal(t, string(exit.Stderr), "multi\nline\n")
		assert.Equal(t, exit.ExitCode(), 99)
	})

	t.Run("Subshell", func(t *testing.T) {
		cmd := exec.Command("bash")
		cmd.Args = append(cmd.Args, "-c", "--", bashHalt+`; (halt 'err') || echo 'after'`)

		stderr := new(bytes.Buffer)
		cmd.Stderr = stderr

		stdout, err := cmd.Output()
		assert.NilError(t, err)
		assert.Equal(t, string(stdout), "after\n")
		assert.Equal(t, stderr.String(), "err\n")
		assert.Equal(t, cmd.ProcessState.ExitCode(), 0)
	})
}

func TestBashPermissions(t *testing.T) {
	// macOS `stat` takes different arguments than BusyBox and GNU coreutils.
	if output, err := exec.Command("stat", "--help").CombinedOutput(); err != nil {
		t.Skip(`requires "stat" executable`)
	} else if !strings.Contains(string(output), "%A") {
		t.Skip(`requires "stat" with access format sequence`)
	}

	dir := t.TempDir()
	assert.NilError(t, os.Mkdir(filepath.Join(dir, "sub"), 0o751))
	assert.NilError(t, os.Chmod(filepath.Join(dir, "sub"), 0o751))
	assert.NilError(t, os.WriteFile(filepath.Join(dir, "sub", "fn"), nil, 0o624)) // #nosec G306 OK permissions for a temp dir in a test
	assert.NilError(t, os.Chmod(filepath.Join(dir, "sub", "fn"), 0o624))

	cmd := exec.Command("bash")
	cmd.Args = append(cmd.Args, "-c", "--",
		bashPermissions+`; permissions "$@"`, "-",
		filepath.Join(dir, "sub", "fn"))

	stdout, err := cmd.Output()
	assert.NilError(t, err)
	assert.Assert(t, cmp.Regexp(``+
		`drwxr-x--x\s+\d+\s+\d+\s+[^ ]+/sub\n`+
		`-rw--w-r--\s+\d+\s+\d+\s+[^ ]+/sub/fn\n`+
		`$`, string(stdout)))
}

func TestBashRecreateDirectory(t *testing.T) {
	// macOS `stat` takes different arguments than BusyBox and GNU coreutils.
	if output, err := exec.Command("stat", "--help").CombinedOutput(); err != nil {
		t.Skip(`requires "stat" executable`)
	} else if !strings.Contains(string(output), "%a") {
		t.Skip(`requires "stat" with access format sequence`)
	}

	dir := t.TempDir()
	assert.NilError(t, os.Mkdir(filepath.Join(dir, "d"), 0o755))
	assert.NilError(t, os.WriteFile(filepath.Join(dir, "d", ".hidden"), nil, 0o644)) // #nosec G306 OK permissions for a temp dir in a test
	assert.NilError(t, os.WriteFile(filepath.Join(dir, "d", "file"), nil, 0o644))    // #nosec G306 OK permissions for a temp dir in a test

	stat := func(args ...string) string {
		cmd := exec.Command("stat", "-c", "%i %#a %N")
		cmd.Args = append(cmd.Args, args...)
		out, err := cmd.CombinedOutput()

		t.Helper()
		assert.NilError(t, err, string(out))
		return string(out)
	}

	var before, after struct{ d, f, dInode, dPerms string }

	before.d = stat(filepath.Join(dir, "d"))
	before.f = stat(
		filepath.Join(dir, "d", ".hidden"),
		filepath.Join(dir, "d", "file"),
	)

	cmd := exec.Command("bash")
	cmd.Args = append(cmd.Args, "-ceu", "--",
		bashRecreateDirectory+` recreate "$@"`, "-",
		filepath.Join(dir, "d"), "0740")
	// The assertion below expects alphabetically sorted filenames.
	// Set an empty environment to always use the default/standard locale.
	cmd.Env = []string{}
	output, err := cmd.CombinedOutput()
	assert.NilError(t, err, string(output))
	assert.Assert(t, cmp.Regexp(`^`+
		`[+] chmod 0740 [^ ]+/tmp.[^ /]+\n`+
		`[+] mv [^ ]+/d/.hidden [^ ]+/d/file [^ ]+/tmp.[^ /]+\n`+
		`[+] rmdir [^ ]+/d\n`+
		`[+] mv [^ ]+/tmp.[^ /]+ [^ ]+/d\n`+
		`$`, string(output)))

	after.d = stat(filepath.Join(dir, "d"))
	after.f = stat(
		filepath.Join(dir, "d", ".hidden"),
		filepath.Join(dir, "d", "file"),
	)

	_, err = fmt.Sscan(before.d, &before.dInode, &before.dPerms)
	assert.NilError(t, err)
	_, err = fmt.Sscan(after.d, &after.dInode, &after.dPerms)
	assert.NilError(t, err)

	// New directory is new.
	assert.Assert(t, after.dInode != before.dInode)

	// New directory has the requested permissions.
	assert.Equal(t, after.dPerms, "0740")

	// Files are in the new directory and unchanged.
	assert.DeepEqual(t, after.f, before.f)
}

func TestBashSafeLink(t *testing.T) {
	// macOS `mv` takes different arguments than GNU coreutils.
	if output, err := exec.Command("mv", "--help").CombinedOutput(); err != nil {
		t.Skip(`requires "mv" executable`)
	} else if !strings.Contains(string(output), "no-target-directory") {
		t.Skip(`requires "mv" that overwrites a directory symlink`)
	}

	// execute calls the bash function with args.
	execute := func(args ...string) (string, error) {
		cmd := exec.Command("bash")
		cmd.Args = append(cmd.Args, "-ceu", "--", bashSafeLink+`safelink "$@"`, "-")
		cmd.Args = append(cmd.Args, args...)
		output, err := cmd.CombinedOutput()
		return string(output), err
	}

	t.Run("CurrentIsFullDirectory", func(t *testing.T) {
		// setupDirectory creates a non-empty directory.
		setupDirectory := func(t testing.TB) (root, current string) {
			t.Helper()
			root = t.TempDir()
			current = filepath.Join(root, "original")
			assert.NilError(t, os.MkdirAll(current, 0o700))
			file, err := os.Create(filepath.Join(current, "original.file"))
			assert.NilError(t, err)
			assert.NilError(t, file.Close())
			return
		}

		// assertSetupContents ensures that directory contents match setupDirectory.
		assertSetupContents := func(t testing.TB, directory string) {
			t.Helper()
			entries, err := os.ReadDir(directory)
			assert.NilError(t, err)
			assert.Equal(t, len(entries), 1)
			assert.Equal(t, entries[0].Name(), "original.file")
		}

		// This situation is unexpected and succeeds.
		t.Run("DesiredIsEmptyDirectory", func(t *testing.T) {
			root, current := setupDirectory(t)

			// desired is an empty directory.
			desired := filepath.Join(root, "desired")
			assert.NilError(t, os.MkdirAll(desired, 0o700))

			output, err := execute(desired, current)
			assert.NilError(t, err, "\n%s", output)

			result, err := os.Readlink(current)
			assert.NilError(t, err, "expected symlink")
			assert.Equal(t, result, desired)
			assertSetupContents(t, desired)
		})

		// This situation is unexpected and aborts.
		t.Run("DesiredIsFullDirectory", func(t *testing.T) {
			root, current := setupDirectory(t)

			// desired is a non-empty directory.
			desired := filepath.Join(root, "desired")
			assert.NilError(t, os.MkdirAll(desired, 0o700))
			file, err := os.Create(filepath.Join(desired, "existing.file"))
			assert.NilError(t, err)
			assert.NilError(t, file.Close())

			// The function should fail and leave the original directory alone.
			output, err := execute(desired, current)
			assert.ErrorContains(t, err, "exit status 1")
			assert.Assert(t, strings.Contains(output, "cannot"), "\n%v", output)
			assertSetupContents(t, current)
		})

		// This situation is unexpected and aborts.
		t.Run("DesiredIsFile", func(t *testing.T) {
			root, current := setupDirectory(t)

			// desired is an empty file.
			desired := filepath.Join(root, "desired")
			file, err := os.Create(desired)
			assert.NilError(t, err)
			assert.NilError(t, file.Close())

			// The function should fail and leave the original directory alone.
			output, err := execute(desired, current)
			assert.ErrorContains(t, err, "exit status 1")
			assert.Assert(t, strings.Contains(output, "cannot"), "\n%v", output)
			assertSetupContents(t, current)
		})

		// This covers a legacy WAL directory that is still inside the data directory.
		t.Run("DesiredIsMissing", func(t *testing.T) {
			root, current := setupDirectory(t)

			// desired does not exist.
			desired := filepath.Join(root, "desired")

			output, err := execute(desired, current)
			assert.NilError(t, err, "\n%s", output)

			result, err := os.Readlink(current)
			assert.NilError(t, err, "expected symlink")
			assert.Equal(t, result, desired)
			assertSetupContents(t, desired)
		})
	})

	t.Run("CurrentIsFile", func(t *testing.T) {
		// setupFile creates an non-empty file.
		setupFile := func(t testing.TB) (root, current string) {
			t.Helper()
			root = t.TempDir()
			current = filepath.Join(root, "original")
			assert.NilError(t, os.WriteFile(current, []byte(`treasure`), 0o600))
			return
		}

		// assertSetupContents ensures that file contents match setupFile.
		assertSetupContents := func(t testing.TB, file string) {
			t.Helper()
			content, err := os.ReadFile(file)
			assert.NilError(t, err)
			assert.Equal(t, string(content), `treasure`)
		}

		// This is situation is unexpected and aborts.
		t.Run("DesiredIsEmptyDirectory", func(t *testing.T) {
			root, current := setupFile(t)

			// desired is an empty directory.
			desired := filepath.Join(root, "desired")
			assert.NilError(t, os.MkdirAll(desired, 0o700))

			// The function should fail and leave the original directory alone.
			output, err := execute(desired, current)
			assert.ErrorContains(t, err, "exit status 1")
			assert.Assert(t, strings.Contains(output, "cannot"), "\n%v", output)
			assertSetupContents(t, current)
		})

		// This situation is unexpected and succeeds.
		t.Run("DesiredIsFile", func(t *testing.T) {
			root, current := setupFile(t)

			// desired is an empty file.
			desired := filepath.Join(root, "desired")
			file, err := os.Create(desired)
			assert.NilError(t, err)
			assert.NilError(t, file.Close())

			output, err := execute(desired, current)
			assert.NilError(t, err, "\n%s", output)

			result, err := os.Readlink(current)
			assert.NilError(t, err, "expected symlink")
			assert.Equal(t, result, desired)
			assertSetupContents(t, desired)
		})

		// This situation is normal and succeeds.
		t.Run("DesiredIsMissing", func(t *testing.T) {
			root, current := setupFile(t)

			// desired does not exist.
			desired := filepath.Join(root, "desired")

			output, err := execute(desired, current)
			assert.NilError(t, err, "\n%s", output)

			result, err := os.Readlink(current)
			assert.NilError(t, err, "expected symlink")
			assert.Equal(t, result, desired)
			assertSetupContents(t, desired)
		})
	})

	// This is the steady state and must be a successful no-op.
	t.Run("CurrentIsLinkToDesired", func(t *testing.T) {
		root := t.TempDir()

		// current is a non-empty directory.
		current := filepath.Join(root, "original")
		assert.NilError(t, os.MkdirAll(current, 0o700))
		file, err := os.Create(filepath.Join(current, "original.file"))
		assert.NilError(t, err)
		assert.NilError(t, file.Close())
		symlink := filepath.Join(root, "symlink")
		assert.NilError(t, os.Symlink(current, symlink))

		output, err := execute(current, symlink)
		assert.NilError(t, err, "\n%s", output)

		result, err := os.Readlink(symlink)
		assert.NilError(t, err, "expected symlink")
		assert.Equal(t, result, current)

		entries, err := os.ReadDir(current)
		assert.NilError(t, err)
		assert.Equal(t, len(entries), 1)
		assert.Equal(t, entries[0].Name(), "original.file")
	})

	// This covers a WAL directory that is a symbolic link.
	t.Run("CurrentIsLinkToExisting", func(t *testing.T) {
		root := t.TempDir()

		// desired does not exist.
		desired := filepath.Join(root, "desired")

		// current is a non-empty directory.
		current := filepath.Join(root, "original")
		assert.NilError(t, os.MkdirAll(current, 0o700))
		file, err := os.Create(filepath.Join(current, "original.file"))
		assert.NilError(t, err)
		assert.NilError(t, file.Close())
		symlink := filepath.Join(root, "symlink")
		assert.NilError(t, os.Symlink(current, symlink))

		output, err := execute(desired, symlink)
		assert.NilError(t, err, "\n%s", output)

		result, err := os.Readlink(symlink)
		assert.NilError(t, err, "expected symlink")
		assert.Equal(t, result, desired)

		entries, err := os.ReadDir(desired)
		assert.NilError(t, err)
		assert.Equal(t, len(entries), 1)
		assert.Equal(t, entries[0].Name(), "original.file")
	})

	// This is situation is unexpected and aborts.
	t.Run("CurrentIsLinkToMissing", func(t *testing.T) {
		root := t.TempDir()

		// desired does not exist.
		desired := filepath.Join(root, "desired")

		// current does not exist.
		current := filepath.Join(root, "original")
		symlink := filepath.Join(root, "symlink")
		assert.NilError(t, os.Symlink(current, symlink))

		// The function should fail and leave the symlink alone.
		output, err := execute(desired, symlink)
		assert.ErrorContains(t, err, "exit status 1")
		assert.Assert(t, strings.Contains(output, "cannot"), "\n%v", output)

		result, err := os.Readlink(symlink)
		assert.NilError(t, err, "expected symlink")
		assert.Equal(t, result, current)
	})
}

func TestStartupCommand(t *testing.T) {
	shellcheck := require.ShellCheck(t)
	t.Parallel()

	cluster := new(v1beta1.PostgresCluster)
	cluster.Spec.PostgresVersion = 13
	instance := new(v1beta1.PostgresInstanceSetSpec)

	ctx := context.Background()
	command := startupCommand(ctx, cluster, instance)

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

	t.Run("PrettyYAML", func(t *testing.T) {
		b, err := yaml.Marshal(script)
		assert.NilError(t, err)
		assert.Assert(t, strings.HasPrefix(string(b), `|`),
			"expected literal block scalar, got:\n%s", b)
	})

	t.Run("EnableTDE", func(t *testing.T) {

		cluster.Spec.Patroni = &v1beta1.PatroniSpec{
			DynamicConfiguration: map[string]any{
				"postgresql": map[string]any{
					"parameters": map[string]any{
						"encryption_key_command": "echo test",
					},
				},
			},
		}
		command := startupCommand(ctx, cluster, instance)
		assert.Assert(t, len(command) > 3)
		assert.Assert(t, strings.Contains(command[3], `cat << "EOF" > /tmp/pg_rewind_tde.sh
#!/bin/sh
pg_rewind -K "$(postgres -C encryption_key_command)" "$@"
EOF
chmod +x /tmp/pg_rewind_tde.sh`))
	})
}
