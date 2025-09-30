// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestPGAdminDataVolume(t *testing.T) {
	ctx := context.Background()
	cc := require.Kubernetes(t)
	t.Parallel()

	namespace := require.Namespace(t, cc)
	base := v1beta1.NewPGAdmin()
	base.Namespace = namespace.Name
	base.Name = "pgadmin-data-volume"
	require.UnmarshalInto(t, &base.Spec, `{
		dataVolumeClaimSpec: {
			accessModes: [ReadWriteOnce],
			resources: { requests: { storage: 1Gi } },
		},
	}`)

	assert.NilError(t, cc.Create(ctx, base.DeepCopy(), client.DryRunAll),
		"expected this base to be valid")

	t.Run("Required", func(t *testing.T) {
		u := require.Value(runtime.ToUnstructuredObject(base))
		unstructured.RemoveNestedField(u.Object, "spec", "dataVolumeClaimSpec")

		err := cc.Create(ctx, u, client.DryRunAll)
		assert.Assert(t, apierrors.IsInvalid(err))
		assert.ErrorContains(t, err, "dataVolumeClaimSpec")
		assert.ErrorContains(t, err, "Required")

		details := require.StatusErrorDetails(t, err)
		assert.Assert(t, cmp.Len(details.Causes, 2))

		assert.Equal(t, details.Causes[0].Field, "spec.dataVolumeClaimSpec")
		assert.Assert(t, cmp.Contains(details.Causes[0].Message, "Required"))

		assert.Equal(t, string(details.Causes[1].Type), "FieldValueInvalid")
		assert.Assert(t, cmp.Contains(details.Causes[1].Message, "rules were not checked"))
	})

	t.Run("AccessModes", func(t *testing.T) {
		t.Run("Missing", func(t *testing.T) {
			u := require.Value(runtime.ToUnstructuredObject(base))
			unstructured.RemoveNestedField(u.Object, "spec", "dataVolumeClaimSpec", "accessModes")

			err := cc.Create(ctx, u, client.DryRunAll)
			assert.Assert(t, apierrors.IsInvalid(err))
			assert.ErrorContains(t, err, "dataVolumeClaimSpec")
			assert.ErrorContains(t, err, "accessModes")
		})

		t.Run("Empty", func(t *testing.T) {
			pgadmin := base.DeepCopy()
			require.UnmarshalInto(t, &pgadmin.Spec.DataVolumeClaimSpec, `{
				accessModes: [],
			}`)

			err := cc.Create(ctx, pgadmin, client.DryRunAll)
			assert.Assert(t, apierrors.IsInvalid(err))
			assert.ErrorContains(t, err, "dataVolumeClaimSpec")
			assert.ErrorContains(t, err, "accessModes")
		})
	})

	t.Run("Resources", func(t *testing.T) {
		t.Run("Missing", func(t *testing.T) {
			for _, tt := range [][]string{
				{"spec", "dataVolumeClaimSpec", "resources"},
				{"spec", "dataVolumeClaimSpec", "resources", "requests"},
				{"spec", "dataVolumeClaimSpec", "resources", "requests", "storage"},
			} {
				u := require.Value(runtime.ToUnstructuredObject(base))
				unstructured.RemoveNestedField(u.Object, tt...)

				err := cc.Create(ctx, u, client.DryRunAll)
				assert.Assert(t, apierrors.IsInvalid(err))
				assert.ErrorContains(t, err, "dataVolumeClaimSpec")
				assert.ErrorContains(t, err, "storage request")
			}
		})
	})
}

func TestPGAdminInstrumentation(t *testing.T) {
	ctx := context.Background()
	cc := require.Kubernetes(t)
	t.Parallel()

	namespace := require.Namespace(t, cc)
	base := v1beta1.NewPGAdmin()
	base.Namespace = namespace.Name
	base.Name = "pgadmin-instrumentation"
	require.UnmarshalInto(t, &base.Spec, `{
		dataVolumeClaimSpec: {
			accessModes: [ReadWriteOnce],
			resources: { requests: { storage: 1Gi } },
		},
	}`)

	assert.NilError(t, cc.Create(ctx, base.DeepCopy(), client.DryRunAll),
		"expected this base to be valid")

	t.Run("LogsBatches", func(t *testing.T) {
		t.Run("Disable", func(t *testing.T) {
			for _, tt := range []struct {
				batches string
				valid   bool
			}{
				{valid: true, batches: ``},              // both null
				{valid: true, batches: `minRecords: 1`}, // one null
				{valid: true, batches: `maxDelay: 1s`},  // other null

				{valid: false, batches: `minRecords: 0`}, // one zero
				{valid: false, batches: `maxDelay: 0m`},  // other zero

				{valid: true, batches: `minRecords: 0, maxDelay: 0m`}, // both zero
				{valid: true, batches: `minRecords: 1, maxDelay: 1s`}, // both non-zero
			} {
				pgadmin := base.DeepCopy()
				require.UnmarshalInto(t, &pgadmin.Spec.Instrumentation, `{
					logs: { batches: { `+tt.batches+` } }
				}`)

				err := cc.Create(ctx, pgadmin, client.DryRunAll)
				if tt.valid {
					assert.NilError(t, err)
				} else {
					assert.Assert(t, apierrors.IsInvalid(err))
					assert.ErrorContains(t, err, "disable")
					assert.ErrorContains(t, err, "minRecords")
					assert.ErrorContains(t, err, "maxDelay")

					details := require.StatusErrorDetails(t, err)
					assert.Assert(t, cmp.Len(details.Causes, 1))

					for _, cause := range details.Causes {
						assert.Equal(t, cause.Field, "spec.instrumentation.logs.batches")
						assert.Assert(t, cmp.Contains(cause.Message, "disable batching"))
						assert.Assert(t, cmp.Contains(cause.Message, "minRecords and maxDelay must be zero"))
					}
				}
			}
		})

		t.Run("MaxDelay", func(t *testing.T) {
			pgadmin := base.DeepCopy()
			require.UnmarshalInto(t, &pgadmin.Spec.Instrumentation, `{
				logs: {
					batches: { maxDelay: 100min },
				},
			}`)

			err := cc.Create(ctx, pgadmin, client.DryRunAll)
			assert.Assert(t, apierrors.IsInvalid(err))
			assert.ErrorContains(t, err, "maxDelay")
			assert.ErrorContains(t, err, "5m")

			details := require.StatusErrorDetails(t, err)
			assert.Assert(t, cmp.Len(details.Causes, 1))

			for _, cause := range details.Causes {
				assert.Equal(t, cause.Field, "spec.instrumentation.logs.batches.maxDelay")
			}
		})

		t.Run("MinMaxRecords", func(t *testing.T) {
			pgadmin := base.DeepCopy()
			require.UnmarshalInto(t, &pgadmin.Spec.Instrumentation, `{
				logs: {
					batches: { minRecords: -11, maxRecords: 0 },
				},
			}`)

			err := cc.Create(ctx, pgadmin, client.DryRunAll)
			assert.Assert(t, apierrors.IsInvalid(err))
			assert.ErrorContains(t, err, "minRecords")
			assert.ErrorContains(t, err, "greater than or equal to 0")
			assert.ErrorContains(t, err, "maxRecords")
			assert.ErrorContains(t, err, "greater than or equal to 1")

			details := require.StatusErrorDetails(t, err)
			assert.Assert(t, cmp.Len(details.Causes, 2))

			for _, cause := range details.Causes {
				switch cause.Field {
				case "spec.instrumentation.logs.batches.maxRecords":
					assert.Assert(t, cmp.Contains(cause.Message, "0"))
					assert.Assert(t, cmp.Contains(cause.Message, "greater than or equal to 1"))

				case "spec.instrumentation.logs.batches.minRecords":
					assert.Assert(t, cmp.Contains(cause.Message, "-11"))
					assert.Assert(t, cmp.Contains(cause.Message, "greater than or equal to 0"))
				}
			}

			t.Run("Reversed", func(t *testing.T) {
				for _, batches := range []string{
					`maxRecords: 99`,                 // default minRecords
					`minRecords: 99, maxRecords: 21`, //
				} {
					pgadmin := base.DeepCopy()
					require.UnmarshalInto(t, &pgadmin.Spec.Instrumentation, `{
						logs: {
							batches: { `+batches+` },
						},
					}`)

					err := cc.Create(ctx, pgadmin, client.DryRunAll)
					assert.Assert(t, apierrors.IsInvalid(err))
					assert.ErrorContains(t, err, "minRecords")
					assert.ErrorContains(t, err, "maxRecords")

					details := require.StatusErrorDetails(t, err)
					assert.Assert(t, cmp.Len(details.Causes, 1))

					for _, cause := range details.Causes {
						assert.Equal(t, cause.Field, "spec.instrumentation.logs.batches")
						assert.Assert(t, cmp.Contains(cause.Message, "minRecords cannot be larger than maxRecords"))
					}
				}
			})
		})
	})

	t.Run("LogsRetentionPeriod", func(t *testing.T) {
		pgadmin := base.DeepCopy()
		require.UnmarshalInto(t, &pgadmin.Spec, `{
			instrumentation: {
				logs: { retentionPeriod: 5m },
			},
		}`)

		err := cc.Create(ctx, pgadmin, client.DryRunAll)
		assert.Assert(t, apierrors.IsInvalid(err))
		assert.ErrorContains(t, err, "retentionPeriod")
		assert.ErrorContains(t, err, "hour|day|week")
		assert.ErrorContains(t, err, "one hour")

		details := require.StatusErrorDetails(t, err)
		assert.Assert(t, cmp.Len(details.Causes, 2))

		for _, cause := range details.Causes {
			assert.Equal(t, cause.Field, "spec.instrumentation.logs.retentionPeriod")
		}

		t.Run("Valid", func(t *testing.T) {
			for _, tt := range []string{
				"28 weeks",
				"90 DAY",
				"1 hr",
				"PT1D2H",
				"1 week 2 days",
			} {
				u, err := runtime.ToUnstructuredObject(pgadmin)
				assert.NilError(t, err)
				assert.NilError(t, unstructured.SetNestedField(u.Object,
					tt, "spec", "instrumentation", "logs", "retentionPeriod"))

				assert.NilError(t, cc.Create(ctx, u, client.DryRunAll), tt)
			}
		})

		t.Run("Invalid", func(t *testing.T) {
			for _, tt := range []string{
				// Amount too small
				"0 days",
				"0",

				// Text too long
				"2 weeks 3 days 4 hours",
			} {
				u, err := runtime.ToUnstructuredObject(pgadmin)
				assert.NilError(t, err)
				assert.NilError(t, unstructured.SetNestedField(u.Object,
					tt, "spec", "instrumentation", "logs", "retentionPeriod"))

				err = cc.Create(ctx, u, client.DryRunAll)
				assert.Assert(t, apierrors.IsInvalid(err), tt)
				assert.ErrorContains(t, err, "retentionPeriod")
			}
		})
	})
}
