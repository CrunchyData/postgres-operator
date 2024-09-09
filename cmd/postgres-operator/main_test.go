// Copyright 2017 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"reflect"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestInitManager(t *testing.T) {
	t.Run("Defaults", func(t *testing.T) {
		options, err := initManager()
		assert.NilError(t, err)

		if assert.Check(t, options.Cache.SyncPeriod != nil) {
			assert.Equal(t, *options.Cache.SyncPeriod, time.Hour)
		}

		assert.Assert(t, options.HealthProbeBindAddress == ":8081")

		assert.DeepEqual(t, options.Controller.GroupKindConcurrency,
			map[string]int{
				"PostgresCluster.postgres-operator.crunchydata.com": 2,
			})

		assert.Assert(t, options.Cache.DefaultNamespaces == nil)
		assert.Assert(t, options.LeaderElection == false)

		{
			options.Cache.SyncPeriod = nil
			options.Controller.GroupKindConcurrency = nil
			options.HealthProbeBindAddress = ""

			assert.Assert(t, reflect.ValueOf(options).IsZero(),
				"expected remaining fields to be unset:\n%+v", options)
		}
	})

	t.Run("PGO_CONTROLLER_LEASE_NAME", func(t *testing.T) {
		t.Setenv("PGO_NAMESPACE", "test-namespace")

		t.Run("Invalid", func(t *testing.T) {
			t.Setenv("PGO_CONTROLLER_LEASE_NAME", "INVALID_NAME")

			options, err := initManager()
			assert.ErrorContains(t, err, "PGO_CONTROLLER_LEASE_NAME")
			assert.ErrorContains(t, err, "invalid")

			assert.Assert(t, options.LeaderElection == false)
			assert.Equal(t, options.LeaderElectionNamespace, "")
		})

		t.Run("Valid", func(t *testing.T) {
			t.Setenv("PGO_CONTROLLER_LEASE_NAME", "valid-name")

			options, err := initManager()
			assert.NilError(t, err)
			assert.Assert(t, options.LeaderElection == true)
			assert.Equal(t, options.LeaderElectionNamespace, "test-namespace")
			assert.Equal(t, options.LeaderElectionID, "valid-name")
		})
	})

	t.Run("PGO_TARGET_NAMESPACE", func(t *testing.T) {
		t.Setenv("PGO_TARGET_NAMESPACE", "some-such")

		options, err := initManager()
		assert.NilError(t, err)
		assert.Assert(t, cmp.Len(options.Cache.DefaultNamespaces, 1),
			"expected only one configured namespace")

		assert.Assert(t, cmp.Contains(options.Cache.DefaultNamespaces, "some-such"))
	})

	t.Run("PGO_TARGET_NAMESPACES", func(t *testing.T) {
		t.Setenv("PGO_TARGET_NAMESPACES", "some-such,another-one")

		options, err := initManager()
		assert.NilError(t, err)
		assert.Assert(t, cmp.Len(options.Cache.DefaultNamespaces, 2),
			"expect two configured namespaces")

		assert.Assert(t, cmp.Contains(options.Cache.DefaultNamespaces, "some-such"))
		assert.Assert(t, cmp.Contains(options.Cache.DefaultNamespaces, "another-one"))
	})

	t.Run("PGO_WORKERS", func(t *testing.T) {
		t.Run("Invalid", func(t *testing.T) {
			for _, v := range []string{"-3", "0", "3.14"} {
				t.Setenv("PGO_WORKERS", v)

				options, err := initManager()
				assert.NilError(t, err)
				assert.DeepEqual(t, options.Controller.GroupKindConcurrency,
					map[string]int{
						"PostgresCluster.postgres-operator.crunchydata.com": 2,
					})
			}
		})

		t.Run("Valid", func(t *testing.T) {
			t.Setenv("PGO_WORKERS", "19")

			options, err := initManager()
			assert.NilError(t, err)
			assert.DeepEqual(t, options.Controller.GroupKindConcurrency,
				map[string]int{
					"PostgresCluster.postgres-operator.crunchydata.com": 19,
				})
		})
	})
}
