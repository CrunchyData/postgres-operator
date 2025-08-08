// Copyright 2024 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package shell

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/internal/testing/require"
)

func TestCleanFileName(t *testing.T) {
	t.Parallel()

	t.Run("Empty", func(t *testing.T) {
		assert.Equal(t, CleanFileName(""), "")
	})

	t.Run("Dots", func(t *testing.T) {
		assert.Equal(t, CleanFileName("."), "")
		assert.Equal(t, CleanFileName(".."), "")
		assert.Equal(t, CleanFileName("..."), "...")
		assert.Equal(t, CleanFileName("././/.././../."), "")
		assert.Equal(t, CleanFileName("././/.././../.."), "")
		assert.Equal(t, CleanFileName("././/.././../../x.j"), "x.j")
	})

	t.Run("Directories", func(t *testing.T) {
		assert.Equal(t, CleanFileName("/"), "")
		assert.Equal(t, CleanFileName("//"), "")
		assert.Equal(t, CleanFileName("asdf/"), "")
		assert.Equal(t, CleanFileName("asdf//12.3"), "12.3")
		assert.Equal(t, CleanFileName("//////"), "")
		assert.Equal(t, CleanFileName("//////gg"), "gg")
	})

	t.Run("NoSeparators", func(t *testing.T) {
		assert.Equal(t, CleanFileName("asdf12.3.ssgg"), "asdf12.3.ssgg")
	})
}

func TestMakeDirectories(t *testing.T) {
	t.Parallel()

	t.Run("NoPaths", func(t *testing.T) {
		assert.Equal(t,
			MakeDirectories("/asdf/jklm"),
			`test -d '/asdf/jklm'`)
	})

	t.Run("Children", func(t *testing.T) {
		assert.DeepEqual(t,
			MakeDirectories("/asdf", "/asdf/jklm", "/asdf/qwerty"),
			`mkdir -p '/asdf/jklm' '/asdf/qwerty' && { chmod 0775 '/asdf/jklm' '/asdf/qwerty' || :; }`)
	})

	t.Run("Grandchild", func(t *testing.T) {
		script := MakeDirectories("/asdf", "/asdf/qwerty/boots")
		assert.DeepEqual(t, script,
			`mkdir -p '/asdf/qwerty/boots' && { chmod 0775 '/asdf/qwerty/boots' '/asdf/qwerty' || :; }`)

		t.Run("ShellCheckPOSIX", func(t *testing.T) {
			shellcheck := require.ShellCheck(t)

			dir := t.TempDir()
			file := filepath.Join(dir, "script.sh")
			assert.NilError(t, os.WriteFile(file, []byte(script), 0o600))

			// Expect ShellCheck for "sh" to be happy.
			// - https://www.shellcheck.net/wiki/SC2148
			cmd := exec.CommandContext(t.Context(), shellcheck, "--enable=all", "--shell=sh", file)
			output, err := cmd.CombinedOutput()
			assert.NilError(t, err, "%q\n%s", cmd.Args, output)
		})
	})

	t.Run("Long", func(t *testing.T) {
		script := MakeDirectories("/", strings.Repeat("/asdf", 20))

		t.Run("PrettyYAML", func(t *testing.T) {
			b, err := yaml.Marshal(script)
			s := string(b)
			assert.NilError(t, err)
			assert.Assert(t, !strings.HasPrefix(s, `"`) && !strings.HasPrefix(s, `'`),
				"expected plain unquoted scalar, got:\n%s", b)
		})
	})

	t.Run("Relative", func(t *testing.T) {
		assert.Equal(t,
			MakeDirectories("/x", "one", "two/three"),
			`mkdir -p '/x/one' '/x/two/three' && { chmod 0775 '/x/one' '/x/two/three' '/x/two' || :; }`,
			"expected paths relative to base")

		assert.Equal(t,
			MakeDirectories("/x/y/z", "../one", "./two", "../../../../three"),
			`mkdir -p '/x/y/one' '/x/y/z/two' '/three' && { chmod 0775 '/x/y/one' '/x/y/z/two' '/three' || :; }`,
			"expected paths relative to base")

		script := MakeDirectories("x/y", "../one", "./two", "../../../../three")
		assert.Equal(t, script,
			`mkdir -p 'x/one' 'x/y/two' '../../three' && { chmod 0775 'x/one' 'x/y/two' '../../three' || :; }`,
			"expected paths relative to base")

		// Calling `mkdir -p '../..'` works, but run it by ShellCheck as a precaution.
		t.Run("ShellCheckPOSIX", func(t *testing.T) {
			shellcheck := require.ShellCheck(t)

			dir := t.TempDir()
			file := filepath.Join(dir, "script.sh")
			assert.NilError(t, os.WriteFile(file, []byte(script), 0o600))

			// Expect ShellCheck for "sh" to be happy.
			// - https://www.shellcheck.net/wiki/SC2148
			cmd := exec.CommandContext(t.Context(), shellcheck, "--enable=all", "--shell=sh", file)
			output, err := cmd.CombinedOutput()
			assert.NilError(t, err, "%q\n%s", cmd.Args, output)
		})
	})

	t.Run("Unrelated", func(t *testing.T) {
		assert.Equal(t,
			MakeDirectories("/one", "/two/three/four"),
			`mkdir -p '/two/three/four' && { chmod 0775 '/two/three/four' || :; }`,
			"expected no chmod of parent directories")
	})
}
