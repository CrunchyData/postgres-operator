// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgaudit

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/internal/postgres"
)

func TestEnableInPostgreSQL(t *testing.T) {
	expected := errors.New("whoops")
	exec := func(
		_ context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
	) error {
		assert.Assert(t, stdout != nil, "should capture stdout")
		assert.Assert(t, stderr != nil, "should capture stderr")

		assert.Assert(t, strings.Contains(strings.Join(command, "\n"),
			`SELECT datname FROM pg_catalog.pg_database`,
		), "expected all databases and templates")

		b, err := io.ReadAll(stdin)
		assert.NilError(t, err)
		assert.Equal(t, string(b), strings.Trim(`
SET client_min_messages = WARNING; CREATE EXTENSION IF NOT EXISTS pgaudit;
		`, "\t\n"))

		return expected
	}

	ctx := context.Background()
	assert.Equal(t, expected, EnableInPostgreSQL(ctx, exec))
}

func TestPostgreSQLParameters(t *testing.T) {
	parameters := postgres.Parameters{
		Mandatory: postgres.NewParameterSet(),
	}

	// No comma when empty.
	PostgreSQLParameters(&parameters)

	assert.Assert(t, parameters.Default == nil)
	assert.DeepEqual(t, parameters.Mandatory.AsMap(), map[string]string{
		"shared_preload_libraries": "pgaudit",
	})

	// Appended when not empty.
	parameters.Mandatory.Add("shared_preload_libraries", "some,existing")
	PostgreSQLParameters(&parameters)

	assert.Assert(t, parameters.Default == nil)
	assert.DeepEqual(t, parameters.Mandatory.AsMap(), map[string]string{
		"shared_preload_libraries": "some,existing,pgaudit",
	})
}
