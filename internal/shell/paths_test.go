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

func TestMakeDirectories(t *testing.T) {
	t.Parallel()

	t.Run("NoPaths", func(t *testing.T) {
		assert.Equal(t,
			MakeDirectories(0o755, "/asdf/jklm"),
			`test -d '/asdf/jklm'`)
	})

	t.Run("Children", func(t *testing.T) {
		assert.DeepEqual(t,
			MakeDirectories(0o775, "/asdf", "/asdf/jklm", "/asdf/qwerty"),
			`mkdir -p '/asdf/jklm' '/asdf/qwerty' && chmod 0775 '/asdf/jklm' '/asdf/qwerty'`)
	})

	t.Run("Grandchild", func(t *testing.T) {
		script := MakeDirectories(0o775, "/asdf", "/asdf/qwerty/boots")
		assert.DeepEqual(t, script,
			`mkdir -p '/asdf/qwerty/boots' && chmod 0775 '/asdf/qwerty/boots' '/asdf/qwerty'`)

		t.Run("ShellCheckPOSIX", func(t *testing.T) {
			shellcheck := require.ShellCheck(t)

			dir := t.TempDir()
			file := filepath.Join(dir, "script.sh")
			assert.NilError(t, os.WriteFile(file, []byte(script), 0o600))

			// Expect ShellCheck for "sh" to be happy.
			// - https://www.shellcheck.net/wiki/SC2148
			cmd := exec.Command(shellcheck, "--enable=all", "--shell=sh", file)
			output, err := cmd.CombinedOutput()
			assert.NilError(t, err, "%q\n%s", cmd.Args, output)
		})
	})

	t.Run("Long", func(t *testing.T) {
		script := MakeDirectories(0o700, "/", strings.Repeat("/asdf", 20))

		t.Run("PrettyYAML", func(t *testing.T) {
			b, err := yaml.Marshal(script)
			s := string(b)
			assert.NilError(t, err)
			assert.Assert(t, !strings.HasPrefix(s, `"`) && !strings.HasPrefix(s, `'`),
				"expected plain unquoted scalar, got:\n%s", b)
		})
	})
}
