//go:build envtest
// +build envtest

/*
 Copyright 2021 - 2023 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package pgmonitor

import (
	"context"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestGenerateDefaultExporterQueries(t *testing.T) {
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
	t.Run("OneFlag", func(t *testing.T) {
		commandSlice := ExporterStartCommand([]string{"--testFlag"})
		assert.DeepEqual(t, commandSlice[:3], []string{"bash", "-ceu", "--"})
		assert.DeepEqual(t, commandSlice[4:], []string{"postgres_exporter_watcher", "--testFlag"})
	})

	t.Run("MultipleFlags", func(t *testing.T) {
		commandSlice := ExporterStartCommand([]string{"--firstTestFlag", "--secondTestFlag"})
		assert.DeepEqual(t, commandSlice[:3], []string{"bash", "-ceu", "--"})
		assert.DeepEqual(t, commandSlice[4:], []string{"postgres_exporter_watcher", "--firstTestFlag", "--secondTestFlag"})
	})
}
