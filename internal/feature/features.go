// Copyright 2017 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

/*
Package feature provides types and functions to enable and disable features
of the Postgres Operator.

To add a new feature, export its name as a constant string and configure it
in [NewGate]. Choose a name that is clear to end users, as they will use it
to enable or disable the feature.

# Stages

Each feature must be configured with a maturity called a stage. We follow the
Kubernetes convention that features in the "Alpha" stage are disabled by default,
while those in the "Beta" stage are enabled by default.
  - https://docs.k8s.io/reference/command-line-tools-reference/feature-gates/#feature-stages

NOTE: Since Kubernetes 1.24, APIs (not features) in the "Beta" stage are disabled by default:
  - https://blog.k8s.io/2022/05/03/kubernetes-1-24-release-announcement/#beta-apis-off-by-default
  - https://git.k8s.io/enhancements/keps/sig-architecture/3136-beta-apis-off-by-default#goals

# Using Features

We initialize and configure one [MutableGate] in main() and add it to the Context
passed to Reconcilers and other Runnables. Those can then interrogate it using [Enabled]:

	if !feature.Enabled(ctx, feature.Excellent) { return }

Tests should create and configure their own [MutableGate] and inject it using
[NewContext]. For example, the following enables one feature and disables another:

	gate := feature.NewGate()
	assert.NilError(t, gate.SetFromMap(map[string]bool{
		feature.Excellent: true,
		feature.Uncommon: false,
	}))
	ctx := feature.NewContext(context.Background(), gate)
*/
package feature

import (
	"context"

	"k8s.io/component-base/featuregate"
)

type Feature = featuregate.Feature

// Gate indicates what features exist and which are enabled.
type Gate interface {
	Enabled(Feature) bool
	String() string
}

// MutableGate contains features that can be enabled or disabled.
type MutableGate interface {
	Gate
	// Set enables or disables features by parsing a string like "feature1=true,feature2=false".
	Set(string) error
	// SetFromMap enables or disables features by boolean values.
	SetFromMap(map[string]bool) error
}

const (
	// Support appending custom queries to default PGMonitor queries
	AppendCustomQueries = "AppendCustomQueries"

	// Enables automatic creation of user schema
	AutoCreateUserSchema = "AutoCreateUserSchema"

	// Support automatically growing volumes
	AutoGrowVolumes = "AutoGrowVolumes"

	BridgeIdentifiers = "BridgeIdentifiers"

	// Support custom sidecars for PostgreSQL instance Pods
	InstanceSidecars = "InstanceSidecars"

	// Support custom sidecars for pgBouncer Pods
	PGBouncerSidecars = "PGBouncerSidecars"

	// Support tablespace volumes
	TablespaceVolumes = "TablespaceVolumes"

	// Support VolumeSnapshots
	VolumeSnapshots = "VolumeSnapshots"
)

// NewGate returns a MutableGate with the Features defined in this package.
func NewGate() MutableGate {
	gate := featuregate.NewFeatureGate()

	if err := gate.Add(map[Feature]featuregate.FeatureSpec{
		AppendCustomQueries:  {Default: false, PreRelease: featuregate.Alpha},
		AutoCreateUserSchema: {Default: false, PreRelease: featuregate.Alpha},
		AutoGrowVolumes:      {Default: false, PreRelease: featuregate.Alpha},
		BridgeIdentifiers:    {Default: false, PreRelease: featuregate.Alpha},
		InstanceSidecars:     {Default: false, PreRelease: featuregate.Alpha},
		PGBouncerSidecars:    {Default: false, PreRelease: featuregate.Alpha},
		TablespaceVolumes:    {Default: false, PreRelease: featuregate.Alpha},
		VolumeSnapshots:      {Default: false, PreRelease: featuregate.Alpha},
	}); err != nil {
		panic(err)
	}

	return gate
}

type contextKey struct{}

// Enabled indicates if a Feature is enabled in the Gate contained in ctx. It
// returns false when there is no Gate.
func Enabled(ctx context.Context, f Feature) bool {
	gate, ok := ctx.Value(contextKey{}).(Gate)
	return ok && gate.Enabled(f)
}

// NewContext returns a copy of ctx containing gate. Check it using [Enabled].
func NewContext(ctx context.Context, gate Gate) context.Context {
	return context.WithValue(ctx, contextKey{}, gate)
}
