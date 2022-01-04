/*
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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
