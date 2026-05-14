// Copyright 2026 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package bin_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestPgBackRestInfoScript(t *testing.T) {
	const script = "pgbackrest-info.sh"

	t.Run("Failure", func(t *testing.T) {
		dir := t.TempDir()
		pgbackrest := filepath.Join(dir, "pgbackrest")
		assert.NilError(t, os.WriteFile(pgbackrest, []byte("#!/bin/sh\nexit 42\n"), 0o700))

		cmd := exec.CommandContext(t.Context(), "bash", script)
		cmd.Env = append(os.Environ(), "PATH="+dir)

		output, err := cmd.CombinedOutput()
		assert.ErrorContains(t, err, "exit status 42")
		assert.Equal(t, string(output), "|")
	})

	t.Run("Success", func(t *testing.T) {
		dir := t.TempDir()
		pgbackrest := filepath.Join(dir, "pgbackrest")
		assert.NilError(t, os.WriteFile(pgbackrest, []byte("#!/bin/sh\nprintf '[{}]'\n"), 0o700))

		cmd := exec.CommandContext(t.Context(), "bash", script)
		cmd.Env = append(os.Environ(), "PATH="+dir)

		output, err := cmd.CombinedOutput()
		assert.NilError(t, err)
		assert.Equal(t, string(output), "|[{}]\n")
	})
}
