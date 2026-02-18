// Copyright 2021 - 2026 Crunchy Data Solutions, Inc.
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
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	v1 "github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestPostgresUserInterfaceAcrossVersions(t *testing.T) {
	ctx := context.Background()
	cc := require.Kubernetes(t)
	t.Parallel()

	namespace := require.Namespace(t, cc)

	base := v1beta1.NewPostgresCluster()
	// Start with a bunch of required fields.
	base.Namespace = namespace.Name
	base.Name = "postgres-pgadmin"
	require.UnmarshalInto(t, &base.Spec, `{
		userInterface: {
			pgAdmin: {
				dataVolumeClaimSpec: {
					accessModes: [ReadWriteOnce],
					resources: { requests: { storage: 1Mi } },
				},
			},
		},
		postgresVersion: 16,
		instances: [{
			dataVolumeClaimSpec: {
				accessModes: [ReadWriteOnce],
				resources: { requests: { storage: 1Mi } },
			},
		}],
	}`)

	v1base := v1.NewPostgresCluster()
	// Start with a bunch of required fields.
	v1base.Namespace = namespace.Name
	v1base.Name = "postgres-pgadmin"
	require.UnmarshalInto(t, &v1base.Spec, `{
		userInterface: {
			pgAdmin: {
				dataVolumeClaimSpec: {
					accessModes: [ReadWriteOnce],
					resources: { requests: { storage: 1Mi } },
				},
			},
		},
		postgresVersion: 16,
		instances: [{
			dataVolumeClaimSpec: {
				accessModes: [ReadWriteOnce],
				resources: { requests: { storage: 1Mi } },
			},
		}],
	}`)

	t.Run("v1beta1 is valid with pgadmin", func(t *testing.T) {
		assert.NilError(t, cc.Create(ctx, base.DeepCopy(), client.DryRunAll),
			"expected this base cluster to be valid")
	})
	t.Run("v1 is invalid with pgadmin", func(t *testing.T) {
		assert.ErrorContains(t, cc.Create(ctx, v1base.DeepCopy(), client.DryRunAll),
			"userInterface not available in v1")
	})

	t.Run("v1 is valid with pgadmin but only if unchanged from v1beta1", func(t *testing.T) {
		// Validation ratcheting is enabled starting in Kubernetes 1.30
		require.KubernetesAtLeast(t, "1.30")

		// A v1 that has been updated from a v1beta1 with no change to the userInterface is valid
		assert.NilError(t, cc.Create(ctx, base),
			"expected this base cluster to be valid")
		v1base.ResourceVersion = base.ResourceVersion
		assert.NilError(t, cc.Update(ctx, v1base),
			"expected this v1 cluster to be a valid update")

		// But will not be valid if there's a change to the userInterface
		require.UnmarshalInto(t, &v1base.Spec, `{
			userInterface: {
				pgAdmin: {
					dataVolumeClaimSpec: {
						accessModes: [ReadWriteOnce, ReadWriteMany],
						resources: { requests: { storage: 2Mi } },
					},
				},
			},
		}`)

		assert.ErrorContains(t, cc.Update(ctx, v1base),
			"userInterface not available in v1")
	})
}

func TestAdditionalVolumes(t *testing.T) {
	ctx := context.Background()
	cc := require.KubernetesAtLeast(t, "1.30")
	dryrun := client.NewDryRunClient(cc)
	t.Parallel()

	namespace := require.Namespace(t, cc)
	base := v1beta1.NewPostgresCluster()

	base.Namespace = namespace.Name
	base.Name = "image-volume-source-test"
	// required fields
	require.UnmarshalInto(t, &base.Spec, `{
		postgresVersion: 16,
		instances: [{
			dataVolumeClaimSpec: {
				accessModes: [ReadWriteOnce],
				resources: { requests: { storage: 1Mi } },
			},
		}],
	}`)

	assert.NilError(t, cc.Create(ctx, base.DeepCopy(), client.DryRunAll),
		"expected this base to be valid")

	var unstructuredBase unstructured.Unstructured
	require.UnmarshalInto(t, &unstructuredBase, require.Value(yaml.Marshal(base)))

	t.Run("Cannot set both image and claimName", func(t *testing.T) {
		tmp := unstructuredBase.DeepCopy()

		require.UnmarshalIntoField(t, tmp, `[{
				dataVolumeClaimSpec: {
					accessModes: [ReadWriteOnce],
					resources: { requests: { storage: 1Mi } },
				},
				volumes: {
					additional: [{
						name: "test",
						claimName: "pvc-claim",
						image: {
							reference: "test-image",
							pullPolicy: Always
						},
						readOnly: true
					}]
				}
			}]`, "spec", "instances")

		err := dryrun.Create(ctx, tmp.DeepCopy())
		assert.Assert(t, apierrors.IsInvalid(err))

		details := require.StatusErrorDetails(t, err)
		assert.Assert(t, cmp.Len(details.Causes, 1))
		assert.Equal(t, details.Causes[0].Field, "spec.instances[0].volumes.additional[0]")
		assert.ErrorContains(t, err, "you must set only one of image or claimName")
	})

	t.Run("Cannot set readOnly to false when using image volume", func(t *testing.T) {
		tmp := unstructuredBase.DeepCopy()

		require.UnmarshalIntoField(t, tmp, `[{
				dataVolumeClaimSpec: {
					accessModes: [ReadWriteOnce],
					resources: { requests: { storage: 1Mi } },
				},
				volumes: {
					additional: [{
						name: "test",
						image: {
							reference: "test-image",
							pullPolicy: Always
						},
						readOnly: false
					}]
				}
			}]`, "spec", "instances")

		err := dryrun.Create(ctx, tmp.DeepCopy())
		assert.Assert(t, apierrors.IsInvalid(err))

		details := require.StatusErrorDetails(t, err)
		assert.Assert(t, cmp.Len(details.Causes, 1))
		assert.Equal(t, details.Causes[0].Field, "spec.instances[0].volumes.additional[0]")
		assert.ErrorContains(t, err, "image volumes must be readOnly")
	})

	t.Run("Reference must be set when using image volume", func(t *testing.T) {
		tmp := unstructuredBase.DeepCopy()

		require.UnmarshalIntoField(t, tmp, `[{
				dataVolumeClaimSpec: {
					accessModes: [ReadWriteOnce],
					resources: { requests: { storage: 1Mi } },
				},
				volumes: {
					additional: [{
						name: "test",
						image: {
							pullPolicy: Always
						},
						readOnly: true
					}]
				}
			}]`, "spec", "instances")

		err := dryrun.Create(ctx, tmp.DeepCopy())
		assert.Assert(t, apierrors.IsInvalid(err))

		details := require.StatusErrorDetails(t, err)
		assert.Assert(t, cmp.Len(details.Causes, 2))
		assert.Assert(t, cmp.Equal(details.Causes[0].Field, "spec.instances[0].volumes.additional[0].image.reference"))
		assert.Assert(t, cmp.Equal(details.Causes[0].Type, "FieldValueRequired"))
		assert.ErrorContains(t, err, "Required")
	})

	t.Run("Reference cannot be an empty string when using image volume", func(t *testing.T) {
		tmp := unstructuredBase.DeepCopy()

		require.UnmarshalIntoField(t, tmp, `[{
				dataVolumeClaimSpec: {
					accessModes: [ReadWriteOnce],
					resources: { requests: { storage: 1Mi } },
				},
				volumes: {
					additional: [{
						name: "test",
						image: {
							reference: "",
							pullPolicy: Always
						},
						readOnly: true
					}]
				}
			}]`, "spec", "instances")

		err := dryrun.Create(ctx, tmp.DeepCopy())
		assert.Assert(t, apierrors.IsInvalid(err))

		details := require.StatusErrorDetails(t, err)
		assert.Assert(t, cmp.Len(details.Causes, 1))
		assert.Assert(t, cmp.Equal(details.Causes[0].Field, "spec.instances[0].volumes.additional[0].image.reference"))
		assert.Assert(t, cmp.Equal(details.Causes[0].Type, "FieldValueInvalid"))
		assert.ErrorContains(t, err, "at least 1 chars long")
	})

	t.Run("ReadOnly can be omitted or set true when using image volume", func(t *testing.T) {
		tmp := unstructuredBase.DeepCopy()

		require.UnmarshalIntoField(t, tmp, `[{
				name: "test-instance",
				dataVolumeClaimSpec: {
					accessModes: [ReadWriteOnce],
					resources: { requests: { storage: 1Mi } },
				},
				volumes: {
					additional: [{
						name: "test",
						image: {
							reference: "test-image",
							pullPolicy: Always
						},
					}]
				}
			}, {
				name: "another-instance",
				dataVolumeClaimSpec: {
					accessModes: [ReadWriteOnce],
					resources: { requests: { storage: 1Mi } },
				},
				volumes: {
					additional: [{
						name: "another",
						image: {
							reference: "another-image",
							pullPolicy: Always
						},
						readOnly: true
					}]
				}
			}]`, "spec", "instances")
		assert.NilError(t, dryrun.Create(ctx, tmp.DeepCopy()))
	})
}

func TestPostgresClusterInstrumentation(t *testing.T) {
	ctx := context.Background()
	cc := require.Kubernetes(t)
	t.Parallel()

	namespace := require.Namespace(t, cc)
	base := v1beta1.NewPostgresCluster()
	base.Namespace = namespace.Name
	base.Name = "postgres-instrumentation"
	require.UnmarshalInto(t, &base.Spec, `{
		postgresVersion: 16,
		instances: [{
			dataVolumeClaimSpec: {
				accessModes: [ReadWriteOnce],
				resources: { requests: { storage: 1Mi } },
			},
		}],
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
				cluster := base.DeepCopy()
				require.UnmarshalInto(t, &cluster.Spec.Instrumentation, `{
					logs: { batches: { `+tt.batches+` } }
				}`)

				err := cc.Create(ctx, cluster, client.DryRunAll)
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
			cluster := base.DeepCopy()
			require.UnmarshalInto(t, &cluster.Spec.Instrumentation, `{
				logs: {
					batches: { maxDelay: 100min },
				},
			}`)

			err := cc.Create(ctx, cluster, client.DryRunAll)
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
			cluster := base.DeepCopy()
			require.UnmarshalInto(t, &cluster.Spec.Instrumentation, `{
				logs: {
					batches: { minRecords: -11, maxRecords: 0 },
				},
			}`)

			err := cc.Create(ctx, cluster, client.DryRunAll)
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
					cluster := base.DeepCopy()
					require.UnmarshalInto(t, &cluster.Spec.Instrumentation, `{
						logs: {
							batches: { `+batches+` },
						},
					}`)

					err := cc.Create(ctx, cluster, client.DryRunAll)
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
		cluster := base.DeepCopy()
		require.UnmarshalInto(t, &cluster.Spec, `{
			instrumentation: {
				logs: { retentionPeriod: 5m },
			},
		}`)

		err := cc.Create(ctx, cluster, client.DryRunAll)
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
				u, err := runtime.ToUnstructuredObject(cluster)
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
				u, err := runtime.ToUnstructuredObject(cluster)
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
