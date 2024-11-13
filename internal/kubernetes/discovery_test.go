// Copyright 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/crunchydata/postgres-operator/internal/testing/require"
)

func TestDiscoveryRunnerInterfaces(t *testing.T) {
	var _ APIs = new(DiscoveryRunner)
	var _ manager.Runnable = new(DiscoveryRunner)

	var runnable manager.LeaderElectionRunnable = new(DiscoveryRunner)
	assert.Assert(t, false == runnable.NeedLeaderElection())
}

func TestDiscoveryRunnerAPIs(t *testing.T) {
	ctx := context.Background()
	cfg, _ := require.Kubernetes2(t)
	require.ParallelCapacity(t, 0)

	runner, err := NewDiscoveryRunner(cfg)
	assert.NilError(t, err)

	// Search for an API that should always exist.
	runner.relevant = append(runner.relevant, API{Kind: "Pod"})
	assert.NilError(t, runner.readAPIs(ctx))

	assert.Assert(t, runner.Has(API{Kind: "Pod"}))
	assert.Assert(t, runner.HasAll(API{Kind: "Pod"}, API{Kind: "Secret"}))
	assert.Assert(t, runner.HasAny(API{Kind: "Pod"}, API{Kind: "NotGonnaExist"}))
	assert.Assert(t, !runner.Has(API{Kind: "NotGonnaExist"}))
}

func TestDiscoveryRunnerVersion(t *testing.T) {
	cfg, _ := require.Kubernetes2(t)
	require.ParallelCapacity(t, 0)

	runner, err := NewDiscoveryRunner(cfg)
	assert.NilError(t, err)
	assert.NilError(t, runner.readVersion())

	version := runner.Version()
	assert.Assert(t, version.Major != "", "got %#v", version)
	assert.Assert(t, version.Minor != "", "got %#v", version)
	assert.Assert(t, version.String() != "", "got %q", version.String())
}
