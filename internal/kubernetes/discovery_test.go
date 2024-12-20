// Copyright 2024 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/version"
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

func TestIsOpenShift(t *testing.T) {
	ctx := context.Background()
	assert.Assert(t, !IsOpenShift(ctx))

	runner := new(DiscoveryRunner)
	runner.have.APISet = NewAPISet(
		API{Group: "security.openshift.io", Kind: "SecurityContextConstraints"},
	)
	assert.Assert(t, IsOpenShift(NewAPIContext(ctx, runner)))
}

func TestVersionString(t *testing.T) {
	ctx := context.Background()
	assert.Equal(t, "", VersionString(ctx))

	runner := new(DiscoveryRunner)
	runner.have.Version = version.Info{
		Major: "1", Minor: "2", GitVersion: "asdf",
	}
	assert.Equal(t, "asdf", VersionString(NewAPIContext(ctx, runner)))
}
