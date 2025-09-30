// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"path"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/events"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestSanitizeLogDirectory(t *testing.T) {
	t.Parallel()

	cluster := new(v1beta1.PostgresCluster)
	cluster.Spec.PostgresVersion = 18
	cluster.UID = "doot"

	recorder := events.NewRecorder(t, runtime.Scheme)

	for _, tt := range []struct{ expected, from, event string }{
		// User wants logs inside the data directory.
		{expected: "/pgdata/pg18/log", from: "log"},

		// Other relative paths warn.
		{
			expected: "/pgdata/pg18/some/directory", from: "some/directory",
			event: `"log_directory" should be "log" or an absolute path`,
		},

		// Postgres interprets blank to mean root.
		// That's no good, so we choose better.
		{expected: "/pgdata/logs/postgres", from: "", event: `"log_directory" = ""`},
		{expected: "/pgdata/logs/postgres", from: "/", event: `"log_directory" = "/"`},

		// Paths into Postgres directories are replaced on the same volume.
		{
			expected: "/pgdata/logs/postgres", from: ".", event: `"log_directory" = "."`,
		}, {
			expected: "/pgdata/logs/postgres", from: "global", event: `"log_directory" = "global"`,
		}, {
			expected: "/pgdata/logs/postgres", from: "postgresql.conf", event: `"log_directory" = "postgresql.conf"`,
		}, {
			expected: "/pgdata/logs/postgres", from: "postgresql.auto.conf", event: `"log_directory" = "postgresql.auto.conf"`,
		}, {
			expected: "/pgdata/logs/postgres", from: "pg_wal", event: `"log_directory" = "pg_wal"`,
		}, {
			expected: "/pgdata/logs/postgres", from: "/pgdata/pg99/any", event: `"log_directory" = "/pgdata/pg99/any"`,
		}, {
			expected: "/pgdata/logs/postgres", from: "/pgdata/pg18_wal", event: `"log_directory" = "/pgdata/pg18_wal"`,
		}, {
			expected: "/pgdata/logs/postgres", from: "/pgdata/pgsql_tmp/ludicrous/speed", event: `"log_directory" = "/pgdata/pgsql_tmp/ludicrous/speed"`,
		}, {
			expected: "/pgwal/logs/postgres", from: "/pgwal/pg18_wal", event: `"log_directory" = "/pgwal/pg18_wal"`,
		}, {
			expected: "/pgtmp/logs/postgres", from: "/pgtmp/pg18_wal", event: `"log_directory" = "/pgtmp/pg18_wal"`,
		},
	} {
		recorder.Events = nil
		result := sanitizeLogDirectory(cluster, tt.from, recorder)

		assert.Equal(t, tt.expected, result, "from: %s", tt.from)
		assert.Assert(t, path.IsAbs(result))

		if len(tt.event) > 0 {
			assert.Assert(t, cmp.Len(recorder.Events, 1))
			assert.Equal(t, recorder.Events[0].Type, "Warning")
			assert.Equal(t, recorder.Events[0].Reason, "InvalidParameter")
			assert.Equal(t, recorder.Events[0].Regarding.Kind, "PostgresCluster")
			assert.Assert(t, cmp.Equal(recorder.Events[0].Regarding.UID, "doot"))
			assert.Assert(t, cmp.Contains(recorder.Events[0].Note, tt.event))
		}
	}
}
