// Copyright 2024 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

//go:build generate

//go:generate go run generate_json.go

package main

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"
)

func main() {
	cwd := need(os.Getwd())
	yamlFileNames := []string{}

	slog.Info("Reading", "directory", cwd)
	for _, entry := range need(os.ReadDir(cwd)) {
		if entry.Type() == 0 && strings.HasSuffix(entry.Name(), ".yaml") {
			yamlFileNames = append(yamlFileNames, entry.Name())
		}
	}

	for _, yamlName := range yamlFileNames {
		slog.Info("Reading", "file", yamlName)
		jsonData := need(yaml.YAMLToJSONStrict(need(os.ReadFile(yamlName))))
		jsonPath := filepath.Join("generated", strings.TrimSuffix(yamlName, ".yaml")+".json")

		slog.Info("Writing", "file", jsonPath)
		must(os.WriteFile(jsonPath, append(bytes.TrimSpace(jsonData), '\n'), 0o644))
	}
}

func must(err error) { need(0, err) }
func need[V any](v V, err error) V {
	if err != nil {
		panic(err)
	}
	return v
}
