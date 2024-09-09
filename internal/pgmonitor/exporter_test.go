// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgmonitor

import (
	"context"
	"os"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestGenerateDefaultExporterQueries(t *testing.T) {
	if os.Getenv("QUERIES_CONFIG_DIR") == "" {
		t.Skip("QUERIES_CONFIG_DIR must be set")
	}

	ctx := context.Background()
	cluster := &v1beta1.PostgresCluster{}

	t.Run("PG<=11", func(t *testing.T) {
		cluster.Spec.PostgresVersion = 11
		queries := GenerateDefaultExporterQueries(ctx, cluster)
		assert.Assert(t, !strings.Contains(queries, "ccp_pg_stat_statements_reset"),
			"Queries contain 'ccp_pg_stat_statements_reset' query when they should not.")
	})

	t.Run("PG>=12", func(t *testing.T) {
		cluster.Spec.PostgresVersion = 12
		queries := GenerateDefaultExporterQueries(ctx, cluster)
		assert.Assert(t, strings.Contains(queries, "ccp_pg_stat_statements_reset"),
			"Queries do not contain 'ccp_pg_stat_statements_reset' query when they should.")
	})
}

func TestExporterStartCommand(t *testing.T) {
	for _, tt := range []struct {
		Name       string
		Collectors bool
		Flags      []string
		Expect     func(t *testing.T, command []string, script string)
	}{
		{
			Name: "NoCollectorsNoFlags",
			Expect: func(t *testing.T, _ []string, script string) {
				assert.Assert(t, cmp.Contains(script, "--[no-]collector"))
			},
		},
		{
			Name:       "WithCollectorsNoFlags",
			Collectors: true,
			Expect: func(t *testing.T, _ []string, script string) {
				assert.Assert(t, !strings.Contains(script, "collector"))
			},
		},
		{
			Name:  "MultipleFlags",
			Flags: []string{"--firstTestFlag", "--secondTestFlag"},
			Expect: func(t *testing.T, command []string, _ string) {
				assert.DeepEqual(t, command[4:], []string{"postgres_exporter_watcher", "--firstTestFlag", "--secondTestFlag"})
			},
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			command := ExporterStartCommand(tt.Collectors, tt.Flags...)
			assert.DeepEqual(t, command[:3], []string{"bash", "-ceu", "--"})
			assert.Assert(t, len(command) > 3)
			script := command[3]

			assert.Assert(t, cmp.Contains(script, "'--extend.query-path=/tmp/queries.yml'"))
			assert.Assert(t, cmp.Contains(script, "'--web.listen-address=:9187'"))

			tt.Expect(t, command, script)

			t.Run("PrettyYAML", func(t *testing.T) {
				b, err := yaml.Marshal(script)
				assert.NilError(t, err)
				assert.Assert(t, strings.HasPrefix(string(b), `|`),
					"expected literal block scalar, got:\n%s", b)
			})
		})
	}
}
