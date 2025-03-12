// Copyright 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"

	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

type CrunchyCRD interface {
	*v1beta1.PostgresCluster | *v1beta1.PGAdmin | *v1beta1.InstrumentationSpec
}

func OpenTelemetrySpecPresent[T CrunchyCRD](object T) bool {

	switch v := any(object).(type) {
	case *v1beta1.InstrumentationSpec:
		return v != nil
	case *v1beta1.PostgresCluster:
		return v.Spec.Instrumentation != nil
	case *v1beta1.PGAdmin:
		return v.Spec.Instrumentation != nil
	default:
		return false
	}

}

func OpenTelemetryLogsOrMetricsEnabled[T CrunchyCRD](
	ctx context.Context,
	object T,
) bool {
	return OpenTelemetrySpecPresent(object) &&
		(feature.Enabled(ctx, feature.OpenTelemetryLogs) ||
			feature.Enabled(ctx, feature.OpenTelemetryMetrics))
}

func OpenTelemetryLogsEnabled[T CrunchyCRD](
	ctx context.Context,
	object T,
) bool {
	return OpenTelemetrySpecPresent(object) &&
		feature.Enabled(ctx, feature.OpenTelemetryLogs)
}

func OpenTelemetryMetricsEnabled[T CrunchyCRD](
	ctx context.Context,
	object T,
) bool {
	return OpenTelemetrySpecPresent(object) &&
		feature.Enabled(ctx, feature.OpenTelemetryMetrics)
}
