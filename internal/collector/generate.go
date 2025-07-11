// Copyright 2024 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

// [pg_query.Parse] requires CGO to compile and call https://github.com/pganalyze/libpg_query
//go:build cgo && generate

//go:generate go run generate.go

package main

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"
	"sigs.k8s.io/yaml"
)

func main() {
	cwd := need(os.Getwd())
	fileNames := map[string][]string{}

	slog.Info("Reading", "directory", cwd)
	for _, entry := range need(os.ReadDir(cwd)) {
		if entry.Type() == 0 {
			ext := filepath.Ext(entry.Name())
			fileNames[ext] = append(fileNames[ext], entry.Name())
		}
	}

	for _, sqlName := range fileNames[".sql"] {
		slog.Info("Reading", "file", sqlName)
		sqlData := need(pg_query.Parse(string(need(os.ReadFile(sqlName)))))
		sqlPath := filepath.Join("generated", sqlName)

		slog.Info("Writing", "file", sqlPath)
		must(os.WriteFile(sqlPath, []byte(need(pg_query.Deparse(sqlData))+"\n"), 0o644))
	}

	for _, yamlName := range fileNames[".yaml"] {
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
