// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"path"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestSanitizeLogDirectory(t *testing.T) {
	t.Parallel()

	cluster := new(v1beta1.PostgresCluster)
	cluster.Spec.PostgresVersion = 18

	for _, tt := range []struct{ expected, from string }{
		// User wants logs inside the data directory.
		{expected: "/pgdata/pg18/log", from: "log"},

		// Postgres interprets blank to mean root.
		// That's no good, so we choose better.
		{expected: "/pgdata/logs/postgres", from: ""},
		{expected: "/pgdata/logs/postgres", from: "/"},

		// Paths into Postgres directories are replaced on the same volume.
		{expected: "/pgdata/logs/postgres", from: "."},
		{expected: "/pgdata/logs/postgres", from: "global"},
		{expected: "/pgdata/logs/postgres", from: "postgresql.conf"},
		{expected: "/pgdata/logs/postgres", from: "postgresql.auto.conf"},
		{expected: "/pgdata/logs/postgres", from: "pg_wal"},
		{expected: "/pgdata/logs/postgres", from: "/pgdata/pg99/any"},
		{expected: "/pgdata/logs/postgres", from: "/pgdata/pg18_wal"},
		{expected: "/pgdata/logs/postgres", from: "/pgdata/pgsql_tmp/1/2"},
		{expected: "/pgwal/logs/postgres", from: "/pgwal/pg18_wal"},
		{expected: "/pgtmp/logs/postgres", from: "/pgtmp/pg18_wal"},
	} {
		result := sanitizeLogDirectory(cluster, tt.from)

		assert.Equal(t, tt.expected, result, "from: %s", tt.from)
		assert.Assert(t, path.IsAbs(result))
	}
}
