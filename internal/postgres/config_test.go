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

package postgres

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

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

func TestBashSafeLink(t *testing.T) {
	// macOS lacks `realpath` which is part of GNU coreutils.
	if _, err := exec.LookPath("realpath"); err != nil {
		t.Skip(`requires "realpath" executable`)
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

func TestBashSafeLinkPrettyYAML(t *testing.T) {
	b, err := yaml.Marshal(bashSafeLink)
	assert.NilError(t, err)
	assert.Assert(t, strings.HasPrefix(string(b), `|`),
		"expected literal block scalar, got:\n%s", b)
}

func TestStartupCommand(t *testing.T) {
	shellcheck := require.ShellCheck(t)

	cluster := new(v1beta1.PostgresCluster)
	cluster.Spec.PostgresVersion = 13
	instance := new(v1beta1.PostgresInstanceSetSpec)

	command := startupCommand(cluster, instance)

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
