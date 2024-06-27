/*
 Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
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

package runtime

import (
	"testing"

	"gotest.tools/v3/assert"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func TestAddLeaderElectionOptions(t *testing.T) {
	t.Setenv("PGO_NAMESPACE", "test-namespace")

	t.Run("PGO_CONTROLLER_LEASE_NAME is not set", func(t *testing.T) {
		opts := manager.Options{HealthProbeBindAddress: "0"}

		opts, err := addLeaderElectionOptions(opts)

		assert.NilError(t, err)
		assert.Assert(t, opts.HealthProbeBindAddress == "0")
		assert.Assert(t, !opts.LeaderElection)
		assert.Assert(t, opts.LeaderElectionNamespace == "")
		assert.Assert(t, opts.LeaderElectionID == "")
	})

	t.Run("PGO_CONTROLLER_LEASE_NAME is invalid", func(t *testing.T) {
		t.Setenv("PGO_CONTROLLER_LEASE_NAME", "INVALID_NAME")
		opts := manager.Options{HealthProbeBindAddress: "0"}

		opts, err := addLeaderElectionOptions(opts)

		assert.ErrorContains(t, err, "value for PGO_CONTROLLER_LEASE_NAME is invalid:")
		assert.Assert(t, opts.HealthProbeBindAddress == "0")
		assert.Assert(t, !opts.LeaderElection)
		assert.Assert(t, opts.LeaderElectionNamespace == "")
		assert.Assert(t, opts.LeaderElectionID == "")
	})

	t.Run("PGO_CONTROLLER_LEASE_NAME is valid", func(t *testing.T) {
		t.Setenv("PGO_CONTROLLER_LEASE_NAME", "valid-name")
		opts := manager.Options{HealthProbeBindAddress: "0"}

		opts, err := addLeaderElectionOptions(opts)

		assert.NilError(t, err)
		assert.Assert(t, opts.HealthProbeBindAddress == "0")
		assert.Assert(t, opts.LeaderElection)
		assert.Assert(t, opts.LeaderElectionNamespace == "test-namespace")
		assert.Assert(t, opts.LeaderElectionID == "valid-name")
	})
}
