/*
Copyright 2017 - 2024 Crunchy Data Solutions, Inc.
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

package main

import (
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

		assert.Assert(t, options.Cache.DefaultNamespaces == nil)
		assert.Assert(t, options.LeaderElection == false)
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

		for k := range options.Cache.DefaultNamespaces {
			assert.Equal(t, k, "some-such")
		}
	})
}
