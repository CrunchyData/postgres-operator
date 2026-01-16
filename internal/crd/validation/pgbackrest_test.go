// Copyright 2021 - 2026 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/testing/require"
	v1 "github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1"
)

func TestV1PGBackRestLogging(t *testing.T) {
	ctx := context.Background()
	cc := require.Kubernetes(t)
	t.Parallel()

	namespace := require.Namespace(t, cc)

	base := v1.NewPostgresCluster()
	base.Namespace = namespace.Name
	base.Name = "pgbackrest-logging"
	// required fields
	require.UnmarshalInto(t, &base.Spec, `{
		postgresVersion: 16,
		instances: [{
			dataVolumeClaimSpec: {
				accessModes: [ReadWriteOnce],
				resources: { requests: { storage: 1Mi } },
			},
		}],
		backups: {
			pgbackrest: {
				repos: [{
					name: repo1,
				}]
			},
		},
	}`)

	assert.NilError(t, cc.Create(ctx, base.DeepCopy(), client.DryRunAll),
		"expected this base to be valid")

	t.Run("Cannot set log-path via global", func(t *testing.T) {
		tmp := base.DeepCopy()

		require.UnmarshalInto(t, &tmp.Spec.Backups.PGBackRest, `{
			global: {
				log-path: "/anything"
			}
		}`)
		err := cc.Create(ctx, tmp.DeepCopy(), client.DryRunAll)
		assert.Assert(t, apierrors.IsInvalid(err))
		assert.ErrorContains(t, err, "pgbackrest log-path must be set via the various log.path fields in the spec")
	})

	t.Run("Cannot set pgbackrest sidecar's log.path without correct subdir", func(t *testing.T) {
		tmp := base.DeepCopy()

		t.Run("Wrong subdir", func(t *testing.T) {
			require.UnmarshalInto(t, &tmp.Spec.Backups.PGBackRest, `{
				log: {
					path: "/something/wrong"
				}
			}`)

			err := cc.Create(ctx, tmp.DeepCopy(), client.DryRunAll)
			assert.Assert(t, apierrors.IsInvalid(err))
			assert.ErrorContains(t, err, "pgbackrest sidecar log path is restricted to an existing additional volume")
		})

		t.Run("Single instance - missing additional volume", func(t *testing.T) {
			require.UnmarshalInto(t, &tmp.Spec.Backups.PGBackRest, `{
				log: {
					path: "/volumes/test"
				}
			}`)

			err := cc.Create(ctx, tmp.DeepCopy(), client.DryRunAll)
			assert.Assert(t, apierrors.IsInvalid(err))
			assert.ErrorContains(t, err, `all instances need an additional volume for pgbackrest sidecar to log in "/volumes"`)
		})

		t.Run("Multiple instances - one missing additional volume", func(t *testing.T) {
			require.UnmarshalInto(t, &tmp.Spec.Backups.PGBackRest, `{
				log: {
					path: "/volumes/test"
				}
			}`)

			require.UnmarshalInto(t, &tmp.Spec.InstanceSets, `[{
				dataVolumeClaimSpec: {
					accessModes: [ReadWriteOnce],
					resources: { requests: { storage: 1Mi } },
				},
				volumes: {
					additional: [{
						name: "test",
						claimName: "pvc-claim"
					}]
				}
			},{
				name: "instance2",
				dataVolumeClaimSpec: {
					accessModes: [ReadWriteOnce],
					resources: { requests: { storage: 1Mi } },
				},
			}]`)

			err := cc.Create(ctx, tmp.DeepCopy(), client.DryRunAll)
			assert.Assert(t, apierrors.IsInvalid(err))
			assert.ErrorContains(t, err, `all instances need an additional volume for pgbackrest sidecar to log in "/volumes"`)
		})

		t.Run("Single instance - additional volume present", func(t *testing.T) {
			require.UnmarshalInto(t, &tmp.Spec.Backups.PGBackRest, `{
				log: {
					path: "/volumes/test"
				}
			}`)

			require.UnmarshalInto(t, &tmp.Spec.InstanceSets, `[{
				dataVolumeClaimSpec: {
					accessModes: [ReadWriteOnce],
					resources: { requests: { storage: 1Mi } },
				},
				volumes: {
					additional: [{
						name: "test",
						claimName: "pvc-claim"
					}]
				}
			}]`)

			assert.NilError(t, cc.Create(ctx, tmp.DeepCopy(), client.DryRunAll), "expected log.path to be valid")
		})

		t.Run("Multiple instances - additional volume present but not matching path", func(t *testing.T) {
			require.UnmarshalInto(t, &tmp.Spec.Backups.PGBackRest, `{
				log: {
					path: "/volumes/test"
				}
			}`)

			require.UnmarshalInto(t, &tmp.Spec.InstanceSets, `[{
				dataVolumeClaimSpec: {
					accessModes: [ReadWriteOnce],
					resources: { requests: { storage: 1Mi } },
				},
				volumes: {
					additional: [{
						name: "test",
						claimName: "pvc-claim"
					}]
				}
			},{
				name: "instance2",
				dataVolumeClaimSpec: {
					accessModes: [ReadWriteOnce],
					resources: { requests: { storage: 1Mi } },
				},
				volumes: {
					additional: [{
						name: "another",
						claimName: "another-pvc-claim"
					}]
				}
			}]`)

			err := cc.Create(ctx, tmp.DeepCopy(), client.DryRunAll)
			assert.Assert(t, apierrors.IsInvalid(err))
			assert.ErrorContains(t, err, `all instances need an additional volume for pgbackrest sidecar to log in "/volumes"`)
		})

		t.Run("Multiple instances - additional volumes present and matching log path", func(t *testing.T) {
			require.UnmarshalInto(t, &tmp.Spec.Backups.PGBackRest, `{
				log: {
					path: "/volumes/test"
				}
			}`)

			require.UnmarshalInto(t, &tmp.Spec.InstanceSets, `[{
				dataVolumeClaimSpec: {
					accessModes: [ReadWriteOnce],
					resources: { requests: { storage: 1Mi } },
				},
				volumes: {
					additional: [{
						name: "test",
						claimName: "pvc-claim"
					}]
				}
			},{
				name: "instance2",
				dataVolumeClaimSpec: {
					accessModes: [ReadWriteOnce],
					resources: { requests: { storage: 1Mi } },
				},
				volumes: {
					additional: [{
						name: "test",
						claimName: "another-pvc-claim"
					}]
				}
			}]`)

			assert.NilError(t, cc.Create(ctx, tmp.DeepCopy(), client.DryRunAll), "expected log.path to be valid")
		})
	})

	t.Run("Cannot set logging on volumes that don't exist", func(t *testing.T) {
		t.Run("Repo Host", func(t *testing.T) {
			tmp := base.DeepCopy()

			require.UnmarshalInto(t, &tmp.Spec.Backups.PGBackRest, `{
				repoHost: {
					log: {
						path: "/volumes/wrong"
					},
					volumes: {
		 				additional: [
		 				{
		 					name: logging,
		     				claimName: required-1
		 				}]
					}
				}
			}`)

			err := cc.Create(ctx, tmp.DeepCopy(), client.DryRunAll)
			assert.Assert(t, apierrors.IsInvalid(err))
			assert.ErrorContains(t, err, "repo host log path is restricted to an existing additional volume")
		})

		t.Run("Backup Jobs", func(t *testing.T) {
			tmp := base.DeepCopy()

			require.UnmarshalInto(t, &tmp.Spec.Backups.PGBackRest, `{
				jobs: {
					log: {
						path: "/volumes/wrong"
					},
					volumes: {
		 				additional: [
		 				{
		 					name: logging,
		     				claimName: required-1
		 				}]
					}
				}
			}`)

			err := cc.Create(ctx, tmp.DeepCopy(), client.DryRunAll)
			assert.Assert(t, apierrors.IsInvalid(err))
			assert.ErrorContains(t, err, "backup jobs log path is restricted to an existing additional volume")
		})
	})

	t.Run("Can set logging on volumes that do exist", func(t *testing.T) {
		t.Run("Repo Host", func(t *testing.T) {
			tmp := base.DeepCopy()

			require.UnmarshalInto(t, &tmp.Spec.Backups.PGBackRest, `{
				repoHost: {
					log: {
						path: "/volumes/logging/logs"
					},
					volumes: {
		 				additional: [
		 				{
		 					name: logging,
		     				claimName: required-1
		 				}]
					}
				}
			}`)

			assert.NilError(t, cc.Create(ctx, tmp.DeepCopy(), client.DryRunAll),
				"expected this configuration to be valid")
		})

		t.Run("Backup Jobs", func(t *testing.T) {
			tmp := base.DeepCopy()

			require.UnmarshalInto(t, &tmp.Spec.Backups.PGBackRest, `{
				jobs: {
					log: {
						path: "/volumes/logging/logs"
					},
					volumes: {
		 				additional: [
		 				{
		 					name: logging,
		     				claimName: required-1
		 				}]
					}
				}
			}`)

			assert.NilError(t, cc.Create(ctx, tmp.DeepCopy(), client.DryRunAll),
				"expected this configuration to be valid")
		})
	})
}
