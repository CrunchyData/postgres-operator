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
	"sigs.k8s.io/yaml"

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
		err := cc.Create(ctx, tmp.DeepCopy(), client.DryRunAll)
		assert.Assert(t, apierrors.IsInvalid(err))
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
		err := cc.Create(ctx, tmp.DeepCopy(), client.DryRunAll)
		assert.Assert(t, apierrors.IsInvalid(err))
		assert.ErrorContains(t, err, "readOnly cannot be set false when using an ImageVolumeSource")
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
		err := cc.Create(ctx, tmp.DeepCopy(), client.DryRunAll)
		assert.Assert(t, apierrors.IsInvalid(err))
		assert.ErrorContains(t, err, "if using an ImageVolumeSource, you must set a reference")
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
		err := cc.Create(ctx, tmp.DeepCopy(), client.DryRunAll)
		assert.Assert(t, apierrors.IsInvalid(err))
		assert.ErrorContains(t, err, "if using an ImageVolumeSource, you must set a reference")
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
		assert.NilError(t, cc.Create(ctx, tmp.DeepCopy(), client.DryRunAll))
	})
}
