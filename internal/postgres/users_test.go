// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"errors"
	"io"
	"regexp"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestSanitizeAlterRoleOptions(t *testing.T) {
	assert.Equal(t, sanitizeAlterRoleOptions(""), "")
	assert.Equal(t, sanitizeAlterRoleOptions(" login  other stuff"), "",
		"expected non-options to be removed")

	t.Run("RemovesPassword", func(t *testing.T) {
		assert.Equal(t, sanitizeAlterRoleOptions("password 'anything'"), "")
		assert.Equal(t, sanitizeAlterRoleOptions("password $wild$ dollar quoting $wild$ login"), "LOGIN")
		assert.Equal(t, sanitizeAlterRoleOptions(" login password '' replication "), "LOGIN REPLICATION")
	})

	t.Run("RemovesComments", func(t *testing.T) {
		assert.Equal(t, sanitizeAlterRoleOptions("login -- asdf"), "LOGIN")
		assert.Equal(t, sanitizeAlterRoleOptions("login /*"), "")
		assert.Equal(t, sanitizeAlterRoleOptions("login /* createdb */ createrole"), "LOGIN CREATEROLE")
	})
}

func TestWriteUsersInPostgreSQL(t *testing.T) {
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

		cluster := new(v1beta1.PostgresCluster)
		assert.Equal(t, expected, WriteUsersInPostgreSQL(ctx, cluster, exec, nil, nil))
	})

	t.Run("Empty", func(t *testing.T) {
		calls := 0
		exec := func(
			_ context.Context, stdin io.Reader, _, _ io.Writer, command ...string,
		) error {
			calls++

			b, err := io.ReadAll(stdin)
			assert.NilError(t, err)
			assert.Equal(t, string(b), strings.TrimSpace(`
SET search_path TO '';
CREATE TEMPORARY TABLE input (id serial, data json);
\copy input (data) from stdin with (format text)
\.
BEGIN;
SELECT pg_catalog.format('CREATE USER %I',
       pg_catalog.json_extract_path_text(input.data, 'username'))
  FROM input
 WHERE NOT EXISTS (
       SELECT 1 FROM pg_catalog.pg_roles
       WHERE rolname = pg_catalog.json_extract_path_text(input.data, 'username'))
 ORDER BY input.id
\gexec

SELECT pg_catalog.format('ALTER ROLE %I WITH %s PASSWORD %L',
       pg_catalog.json_extract_path_text(input.data, 'username'),
       pg_catalog.json_extract_path_text(input.data, 'options'),
       pg_catalog.json_extract_path_text(input.data, 'verifier'))
  FROM input ORDER BY input.id
\gexec

SELECT pg_catalog.format('GRANT ALL PRIVILEGES ON DATABASE %I TO %I',
       pg_catalog.json_array_elements_text(
       pg_catalog.json_extract_path(
       pg_catalog.json_strip_nulls(input.data), 'databases')),
       pg_catalog.json_extract_path_text(input.data, 'username'))
  FROM input ORDER BY input.id
\gexec
COMMIT;`))
			return nil
		}

		cluster := new(v1beta1.PostgresCluster)
		assert.NilError(t, WriteUsersInPostgreSQL(ctx, cluster, exec, nil, nil))
		assert.Equal(t, calls, 1)

		assert.NilError(t, WriteUsersInPostgreSQL(ctx, cluster, exec, []v1beta1.PostgresUserSpec{}, nil))
		assert.Equal(t, calls, 2)

		assert.NilError(t, WriteUsersInPostgreSQL(ctx, cluster, exec, nil, map[string]string{}))
		assert.Equal(t, calls, 3)
	})

	t.Run("OptionalFields", func(t *testing.T) {
		cluster := new(v1beta1.PostgresCluster)
		calls := 0
		exec := func(
			_ context.Context, stdin io.Reader, _, _ io.Writer, command ...string,
		) error {
			calls++

			b, err := io.ReadAll(stdin)
			assert.NilError(t, err)
			assert.Assert(t, cmp.Contains(string(b), `
\copy input (data) from stdin with (format text)
{"databases":["db1"],"options":"","username":"user-no-options","verifier":""}
{"databases":null,"options":"CREATEDB CREATEROLE","username":"user-no-databases","verifier":""}
{"databases":null,"options":"","username":"user-with-verifier","verifier":"some$verifier"}
{"databases":null,"options":"LOGIN","username":"user-invalid-options","verifier":""}
\.
`))
			return nil
		}

		assert.NilError(t, WriteUsersInPostgreSQL(ctx, cluster, exec,
			[]v1beta1.PostgresUserSpec{
				{
					Name:      "user-no-options",
					Databases: []v1beta1.PostgresIdentifier{"db1"},
				},
				{
					Name:    "user-no-databases",
					Options: "createdb createrole",
				},
				{
					Name: "user-with-verifier",
				},
				{
					Name:    "user-invalid-options",
					Options: "login password 'doot' --",
				},
			},
			map[string]string{
				"no-user":            "ignored",
				"user-with-verifier": "some$verifier",
			},
		))
		assert.Equal(t, calls, 1)
	})

	t.Run("PostgresSuperuser", func(t *testing.T) {
		calls := 0
		cluster := new(v1beta1.PostgresCluster)
		exec := func(
			_ context.Context, stdin io.Reader, _, _ io.Writer, command ...string,
		) error {
			calls++

			b, err := io.ReadAll(stdin)
			assert.NilError(t, err)
			assert.Assert(t, cmp.Contains(string(b), `
\copy input (data) from stdin with (format text)
{"databases":["postgres"],"options":"LOGIN SUPERUSER","username":"postgres","verifier":"allowed"}
\.
`))
			return nil
		}

		assert.NilError(t, WriteUsersInPostgreSQL(ctx, cluster, exec,
			[]v1beta1.PostgresUserSpec{
				{
					Name:      "postgres",
					Databases: []v1beta1.PostgresIdentifier{"all", "ignored"},
					Options:   "NOLOGIN CONNECTION LIMIT 0",
				},
			},
			map[string]string{
				"postgres": "allowed",
			},
		))
		assert.Equal(t, calls, 1)
	})
}

func TestWriteUsersSchemasInPostgreSQL(t *testing.T) {
	ctx := context.Background()

	t.Run("Mixed users", func(t *testing.T) {
		calls := 0
		exec := func(
			_ context.Context, stdin io.Reader, _, _ io.Writer, command ...string,
		) error {
			calls++

			b, err := io.ReadAll(stdin)
			assert.NilError(t, err)

			// The command strings will contain either of two possibilities, depending on the user called.
			commands := strings.Join(command, ",")
			re := regexp.MustCompile("--set=databases=\\[\"db1\"\\],--set=username=user-single-db|--set=databases=\\[\"db1\",\"db2\"\\],--set=username=user-multi-db")
			assert.Assert(t, cmp.Regexp(re, commands))

			assert.Assert(t, cmp.Contains(string(b), `CREATE SCHEMA IF NOT EXISTS :"username" AUTHORIZATION :"username";`))
			return nil
		}

		assert.NilError(t, WriteUsersSchemasInPostgreSQL(ctx, exec,
			[]v1beta1.PostgresUserSpec{
				{
					Name:      "user-single-db",
					Databases: []v1beta1.PostgresIdentifier{"db1"},
				},
				{
					Name: "user-no-databases",
				},
				{
					Name:      "user-multi-dbs",
					Databases: []v1beta1.PostgresIdentifier{"db1", "db2"},
				},
				{
					Name:      "public",
					Databases: []v1beta1.PostgresIdentifier{"db3"},
				},
			},
		))
		// The spec.users has four elements, but two will be skipped:
		// 	* the user with the reserved name `public`
		// 	* the user with 0 databases
		assert.Equal(t, calls, 2)
	})

}
