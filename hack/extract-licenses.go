//go:build go1.21

package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
)

func main() {
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), strings.TrimSpace(`
Usage: `+flags.Name()+` {directory} {executables...}

This program downloads and extracts the licenses of Go modules used to build
Go executables.

The first argument is a directory that will receive license files. It will be
created if it does not exist. This program will overwrite existing files but
not delete them. Remaining arguments must be Go executables.

Go modules are downloaded to the Go module cache which can be changed via
the environment: https://go.dev/ref/mod#module-cache`,
		))
	}
	if _ = flags.Parse(os.Args[1:]); flags.NArg() < 2 || slices.ContainsFunc(
		os.Args, func(arg string) bool { return arg == "-help" || arg == "--help" },
	) {
		flags.Usage()
		os.Exit(2)
	}

	ctx, cancel := context.WithCancel(context.Background())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() { <-signals; cancel() }()

	// Create the target directory.
	if err := os.MkdirAll(flags.Arg(0), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Extract module information from remaining arguments.
	modules := identifyModules(ctx, flags.Args()[1:]...)

	// Ignore packages from Crunchy Data. Most are not available in any [proxy],
	// and we handle their licenses elsewhere.
	//
	// This is also a quick fix to avoid the [replace] directive in our projects.
	// The logic below cannot handle them. Showing xxhash versus a replace:
	//
	//  dep	github.com/cespare/xxhash/v2	v2.3.0	h1:UL815xU9SqsFlibzuggzjXhog7bL6oX9BbNZnL2UFvs=
	//  dep	github.com/crunchydata/postgres-operator	v0.0.0-00010101000000-000000000000
	//  =>	./postgres-operator	(devel)
	//
	// [proxy]: https://go.dev/ref/mod#module-proxy
	// [replace]: https://go.dev/ref/mod#go-mod-file-replace
	modules = slices.DeleteFunc(modules, func(s string) bool {
		return strings.HasPrefix(s, "git.crunchydata.com/") ||
			strings.HasPrefix(s, "github.com/crunchydata/")
	})

	// Download modules to the Go module cache.
	directories := downloadModules(ctx, modules...)

	// Gather license files from every module into the target directory.
	for module, directory := range directories {
		for _, license := range findLicenses(ctx, directory) {
			relative := module + strings.TrimPrefix(license, directory)
			destination := filepath.Join(flags.Arg(0), relative)

			var data []byte
			err := ctx.Err()

			if err == nil {
				err = os.MkdirAll(filepath.Dir(destination), 0o755)
			}
			if err == nil {
				data, err = os.ReadFile(license)
			}
			if err == nil {
				err = os.WriteFile(destination, data, 0o644)
			}
			if err == nil {
				fmt.Println(license, "=>", destination)
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
	}
}

func downloadModules(ctx context.Context, modules ...string) map[string]string {
	var stdout bytes.Buffer

	// Download modules and read their details into a series of JSON objects.
	// - https://go.dev/ref/mod#go-mod-download
	cmd := exec.CommandContext(ctx, os.Getenv("GO"), append([]string{"mod", "download", "-json"}, modules...)...)
	if cmd.Path == "" {
		cmd.Path, cmd.Err = exec.LookPath("go")
	}
	cmd.Stderr = os.Stderr
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cmd.ProcessState.ExitCode())
	}

	decoder := json.NewDecoder(&stdout)
	results := make(map[string]string, len(modules))

	// NOTE: The directory in the cache is a normalized spelling of the module path;
	// ask Go for the directory; do not try to spell it yourself.
	// - https://go.dev/ref/mod#module-cache
	// - https://go.dev/ref/mod#module-path
	for {
		var module struct{ Path, Version, Dir string }
		err := decoder.Decode(&module)

		if err == nil {
			results[module.Path+"@"+module.Version] = module.Dir
			continue
		}
		if errors.Is(err, io.EOF) {
			break
		}

		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return results
}

func findLicenses(ctx context.Context, directory string) []string {
	var results []string

	// Syft maintains a list of license filenames that began as a list maintained by
	// Go. We gather a similar list by matching on "copying" and "license" filenames.
	// - https://pkg.go.dev/github.com/anchore/syft@v1.3.0/internal/licenses#FileNames
	//
	// Ignore Go files and anything in the special "testdata" directory.
	// - https://go.dev/cmd/go
	err := filepath.WalkDir(directory, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() && d.Name() == "testdata" {
			return fs.SkipDir
		}
		if d.IsDir() || strings.HasSuffix(path, ".go") {
			return err
		}

		lower := strings.ToLower(d.Name())
		if strings.Contains(lower, "copying") || strings.Contains(lower, "license") {
			results = append(results, path)
		}

		return err
	})

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return results
}

func identifyModules(ctx context.Context, executables ...string) []string {
	var stdout bytes.Buffer

	// Use `go version -m` to read the embedded module information as a text table.
	// - https://go.dev/ref/mod#go-version-m
	cmd := exec.CommandContext(ctx, os.Getenv("GO"), append([]string{"version", "-m"}, executables...)...)
	if cmd.Path == "" {
		cmd.Path, cmd.Err = exec.LookPath("go")
	}
	cmd.Stderr = os.Stderr
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cmd.ProcessState.ExitCode())
	}

	// Parse the tab-separated table without checking row lengths.
	reader := csv.NewReader(&stdout)
	reader.Comma = '\t'
	reader.FieldsPerRecord = -1

	lines, _ := reader.ReadAll()
	result := make([]string, 0, len(lines))

	for _, fields := range lines {
		if len(fields) > 3 && fields[1] == "dep" {
			result = append(result, fields[2]+"@"+fields[3])
		}
		if len(fields) > 4 && fields[1] == "mod" && fields[4] != "" {
			result = append(result, fields[2]+"@"+fields[3])
		}
	}

	// The `go version -m` command returns no information for empty files, and it
	// is possible for a Go executable to have no main module and no dependencies.
	if len(result) == 0 {
		fmt.Fprintf(os.Stderr, "no Go modules in %v\n", executables)
		os.Exit(0)
	}

	return result
}
