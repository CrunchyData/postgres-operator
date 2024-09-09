// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgis

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
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
		assert.Equal(t, string(b), `SET client_min_messages = WARNING;
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS postgis_topology;
CREATE EXTENSION IF NOT EXISTS fuzzystrmatch;
CREATE EXTENSION IF NOT EXISTS postgis_tiger_geocoder;`)

		return expected
	}

	ctx := context.Background()
	assert.Equal(t, expected, EnableInPostgreSQL(ctx, exec))
}
