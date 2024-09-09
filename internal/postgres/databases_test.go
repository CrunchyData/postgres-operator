// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
)

func TestCreateDatabasesInPostgreSQL(t *testing.T) {
	ctx := context.Background()

	t.Run("Arguments", func(t *testing.T) {
		expected := errors.New("pass-through")
		exec := func(
			_ context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			assert.Assert(t, stdout != nil, "should capture stdout")
			assert.Assert(t, stderr != nil, "should capture stderr")
			return expected
		}

		assert.Equal(t, expected, CreateDatabasesInPostgreSQL(ctx, exec, nil))
	})

	t.Run("Empty", func(t *testing.T) {
		calls := 0
		exec := func(
			_ context.Context, stdin io.Reader, _, _ io.Writer, command ...string,
		) error {
			calls++

			b, err := io.ReadAll(stdin)
			assert.NilError(t, err)
			assert.Equal(t, string(b), strings.TrimLeft(`
SET search_path TO '';
CREATE TEMPORARY TABLE input (id serial, data json);
\copy input (data) from stdin with (format text)
\.

SELECT pg_catalog.format('CREATE DATABASE %I',
       pg_catalog.json_extract_path_text(input.data, 'database'))
  FROM input
 WHERE NOT EXISTS (
       SELECT 1 FROM pg_catalog.pg_database
       WHERE datname = pg_catalog.json_extract_path_text(input.data, 'database'))
 ORDER BY input.id
\gexec
`, "\n"))
			return nil
		}

		assert.NilError(t, CreateDatabasesInPostgreSQL(ctx, exec, nil))
		assert.Equal(t, calls, 1)

		assert.NilError(t, CreateDatabasesInPostgreSQL(ctx, exec, []string{}))
		assert.Equal(t, calls, 2)
	})

	t.Run("Full", func(t *testing.T) {
		calls := 0
		exec := func(
			_ context.Context, stdin io.Reader, _, _ io.Writer, command ...string,
		) error {
			calls++

			b, err := io.ReadAll(stdin)
			assert.NilError(t, err)
			assert.Assert(t, cmp.Contains(string(b), `
\copy input (data) from stdin with (format text)
{"database":"white space"}
{"database":"eXaCtLy"}
\.
`))
			return nil
		}

		assert.NilError(t, CreateDatabasesInPostgreSQL(ctx, exec,
			[]string{"white space", "eXaCtLy"},
		))
		assert.Equal(t, calls, 1)
	})
}
