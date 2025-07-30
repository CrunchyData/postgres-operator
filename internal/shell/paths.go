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
	"slices"
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
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		path = path[i+1:]
	}
	if path != "." && path != ".." {
		return path
	}
	return ""
}

// MakeDirectories returns a list of POSIX shell commands that ensure each path
// exists. It creates every directory leading to path from (but not including)
// base and sets their permissions for Kubernetes, regardless of umask.
//
// Relative paths are expanded relative to base.
//
// See:
//   - https://pubs.opengroup.org/onlinepubs/9799919799/utilities/chmod.html
//   - https://pubs.opengroup.org/onlinepubs/9799919799/utilities/mkdir.html
//   - https://pubs.opengroup.org/onlinepubs/9799919799/utilities/test.html
//   - https://pubs.opengroup.org/onlinepubs/9799919799/utilities/umask.html
func MakeDirectories(base string, paths ...string) string {
	// Without any paths, return a command that succeeds when the base path exists.
	if len(paths) == 0 {
		return `test -d ` + QuoteWord(base)
	}

	// Expand each path relative to the base path.
	expandedPaths := make([]string, len(paths))
	for i, p := range paths {
		if filepath.IsAbs(p) {
			expandedPaths[i] = p
		} else {
			expandedPaths[i] = filepath.Join(base, p)
		}
	}

	// Gather parent directories of each path.
	allPaths := slices.Clone(expandedPaths)
	for _, p := range expandedPaths {
		if r, err := filepath.Rel(base, p); err == nil && filepath.IsLocal(r) {
			// The result of [filepath.Rel] is a shorter representation of the full path; skip it.
			r = filepath.Dir(r)

			for r != "." {
				allPaths = append(allPaths, filepath.Join(base, r))
				r = filepath.Dir(r)
			}
		}
	}

	// Pod "securityContext.fsGroup" ensures processes and filesystems agree on a GID.
	// Use the same permissions for group and owner.
	const perms fs.FileMode = 0 |
		// S_IRWXU: enable owner read, write, and execute permissions.
		0o0700 |
		// S_IRWXG: enable group read, write, and execute permissions.
		0o0070 |
		// S_IXOTH, S_IROTH: enable other read and execute permissions.
		0o0001 | 0o0004

	return `` +
		// Create all the paths and any missing parents.
		`mkdir -p ` + strings.Join(QuoteWords(expandedPaths...), " ") +

		// Try to set the permissions of every path and each parent.
		// This swallows the exit status of `chmod` because not all filesystems
		// tolerate the operation; CIFS and NFS are notable examples.
		fmt.Sprintf(` && { chmod %#o %s || :; }`,
			perms, strings.Join(QuoteWords(allPaths...), " "),
		)
}
