// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
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

func TestV1PGBouncerLogging(t *testing.T) {
	ctx := context.Background()
	cc := require.Kubernetes(t)
	t.Parallel()

	namespace := require.Namespace(t, cc)

	base := v1.NewPostgresCluster()
	base.Namespace = namespace.Name
	base.Name = "pgbouncer-logging"
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

	t.Run("Can set logging on tmp with .log", func(t *testing.T) {
		tmp := base.DeepCopy()

		require.UnmarshalInto(t, &tmp.Spec.Proxy, `{
			pgBouncer: {
				config: {
					global: {
						logfile: "/tmp/logs/pgbouncer/log.log"
					}
				}
			}
		}`)
		assert.NilError(t, cc.Create(ctx, base.DeepCopy(), client.DryRunAll),
			"expected this base to be valid")
	})

	t.Run("Cannot set logging on tmp without .log", func(t *testing.T) {
		tmp := base.DeepCopy()

		require.UnmarshalInto(t, &tmp.Spec.Proxy, `{
			pgBouncer: {
				config: {
					global: {
						logfile: "/tmp/logs/pgbouncer/log.txt"
					}
				}
			}
		}`)

		err := cc.Create(ctx, tmp.DeepCopy(), client.DryRunAll)
		assert.Assert(t, apierrors.IsInvalid(err))
		assert.ErrorContains(t, err, "logfile config must end with '.log'")
	})
}
