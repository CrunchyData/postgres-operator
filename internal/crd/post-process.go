// Copyright 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

//go:build generate

//go:generate go run post-process.go

package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"regexp"

	"github.com/itchyny/gojq"
	"sigs.k8s.io/yaml"
)

func main() {
	cwd := need(os.Getwd())
	dir := filepath.Join(cwd, "..", "..", "config", "dev", "crd", "bases")
	query := "post-process.jq"

	slog.Info("Reading", "file", query)
	code := need(gojq.Compile(
		need(gojq.Parse(string(need(os.ReadFile(query))))),
		gojq.WithModuleLoader(gojq.NewModuleLoader([]string{"$ORIGIN/../lib"})),
	))

	slog.Info("Reading", "directory", dir)
	yamlPaths := need(filepath.Glob(filepath.Join(dir, "*.yaml")))

	for _, yamlPath := range yamlPaths {
		yamlName := filepath.Base(yamlPath)
		yamlValue := any(nil)

		slog.Info("Reading", "file", yamlName)
		must(yaml.UnmarshalStrict(need(os.ReadFile(yamlPath)), &yamlValue))

		result := code.Run(yamlValue)
		if v, ok := result.Next(); ok {
			if err, ok := v.(error); ok {
				panic(err)
			}

			// Turn top-level strings that start with octothorpe U+0023 into YAML comments by removing their quotes.
			yamlData := need(yaml.Marshal(v))
			yamlData = regexp.MustCompile(`(?m)^'(#[^']*)'(.*)$`).ReplaceAll(yamlData, []byte("$1$2"))

			slog.Info("Writing", "file", yamlName)
			must(os.WriteFile(yamlPath, append([]byte("---\n"), yamlData...), 0o644))
		}

		if _, ok := result.Next(); ok {
			panic("unexpected second result")
		}
	}
}

func must(err error) { need(0, err) }
func need[V any](v V, err error) V {
	if err != nil {
		panic(err)
	}
	return v
}
