// Copyright 2024 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

// We want the [filepath] package to behave correctly for Linux containers.
//go:build unix

package shell

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// CleanFileName returns the suffix of path after its last slash U+002F.
// This is similar to "basename" except this returns empty string when:
//   - The final character of path is slash U+002F, or
//   - The result would be "." or ".."
//
// See:
//   - https://pubs.opengroup.org/onlinepubs/9799919799/utilities/basename.html
func CleanFileName(path string) string {
	if i := strings.LastIndexByte(path, '/'); i < 0 {
		return path
	} else if path = path[i+1:]; path != "." && path != ".." {
		return path
	}
	return ""
}

// MakeDirectories returns a list of POSIX shell commands that ensure each path
// exists. It creates every directory leading to path from (but not including)
// base and sets their permissions to exactly perms, regardless of umask.
//
// See:
//   - https://pubs.opengroup.org/onlinepubs/9799919799/utilities/chmod.html
//   - https://pubs.opengroup.org/onlinepubs/9799919799/utilities/mkdir.html
//   - https://pubs.opengroup.org/onlinepubs/9799919799/utilities/test.html
//   - https://pubs.opengroup.org/onlinepubs/9799919799/utilities/umask.html
func MakeDirectories(perms fs.FileMode, base string, paths ...string) string {
	// Without any paths, return a command that succeeds when the base path
	// exists.
	if len(paths) == 0 {
		return `test -d ` + QuoteWord(base)
	}

	allPaths := append([]string(nil), paths...)
	for _, p := range paths {
		if r, err := filepath.Rel(base, p); err == nil && filepath.IsLocal(r) {
			// The result of [filepath.Rel] is a shorter representation
			// of the full path; skip it.
			r = filepath.Dir(r)

			for r != "." {
				allPaths = append(allPaths, filepath.Join(base, r))
				r = filepath.Dir(r)
			}
		}
	}

	return `` +
		// Create all the paths and any missing parents.
		`mkdir -p ` + strings.Join(QuoteWords(paths...), " ") +

		// Set the permissions of every path and each parent.
		// NOTE: FileMode bits other than file permissions are ignored.
		fmt.Sprintf(` && chmod %#o %s`,
			perms&fs.ModePerm,
			strings.Join(QuoteWords(allPaths...), " "),
		)
}
