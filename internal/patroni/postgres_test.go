// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package patroni

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestPostgresParameters(t *testing.T) {
	t.Run("Zero", func(t *testing.T) {
		result := PostgresParameters(nil)

		assert.Assert(t, result != nil)
		assert.DeepEqual(t, result.AsMap(), map[string]string{})
	})

	t.Run("NoDynamicConfig", func(t *testing.T) {
		spec := new(v1beta1.PatroniSpec)
		result := PostgresParameters(spec)

		assert.Assert(t, result != nil)
		assert.DeepEqual(t, result.AsMap(), map[string]string{})
	})

	t.Run("NoPostgreSQL", func(t *testing.T) {
		spec := new(v1beta1.PatroniSpec)
		require.UnmarshalInto(t, spec, `{
			dynamicConfiguration: {},
		}`)
		result := PostgresParameters(spec)

		assert.Assert(t, result != nil)
		assert.DeepEqual(t, result.AsMap(), map[string]string{})
	})

	t.Run("WrongPostgreSQLType", func(t *testing.T) {
		spec := new(v1beta1.PatroniSpec)
		require.UnmarshalInto(t, spec, `{
			dynamicConfiguration: {
				postgresql: asdf,
			},
		}`)
		result := PostgresParameters(spec)

		assert.Assert(t, result != nil)
		assert.DeepEqual(t, result.AsMap(), map[string]string{})
	})

	t.Run("NoParameters", func(t *testing.T) {
		spec := new(v1beta1.PatroniSpec)
		require.UnmarshalInto(t, spec, `{
			dynamicConfiguration: {
				postgresql: {
					use_pg_rewind: true,
				},
			},
		}`)
		result := PostgresParameters(spec)

		assert.Assert(t, result != nil)
		assert.DeepEqual(t, result.AsMap(), map[string]string{})
	})

	t.Run("WrongParametersType", func(t *testing.T) {
		spec := new(v1beta1.PatroniSpec)
		require.UnmarshalInto(t, spec, `{
			dynamicConfiguration: {
				postgresql: {
					parameters: [1,2],
				},
			},
		}`)
		result := PostgresParameters(spec)

		assert.Assert(t, result != nil)
		assert.DeepEqual(t, result.AsMap(), map[string]string{})
	})

	t.Run("Parameters", func(t *testing.T) {
		spec := new(v1beta1.PatroniSpec)
		require.UnmarshalInto(t, spec, `{
			dynamicConfiguration: {
				postgresql: {
					parameters: {
						log_statement_sample_rate: 0.98,
						max_connections: 1000,
						wal_log_hints: true,
						wal_level: replica,
						strange.though: [ 1, 2.3, yes ],
					},
				},
			},
		}`)
		result := PostgresParameters(spec)

		assert.Assert(t, result != nil)
		assert.DeepEqual(t, result.AsMap(), map[string]string{
			"log_statement_sample_rate": "0.98",
			"max_connections":           "1000",
			"wal_log_hints":             "true",
			"wal_level":                 "replica",
			"strange.though":            "[1,2.3,true]",
		})
	})
}
