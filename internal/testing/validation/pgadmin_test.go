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

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestPGAdminInstrumentation(t *testing.T) {
	ctx := context.Background()
	cc := require.Kubernetes(t)
	t.Parallel()

	namespace := require.Namespace(t, cc)
	base := v1beta1.NewPGAdmin()
	base.Namespace = namespace.Name
	base.Name = "pgadmin-instrumentation"

	assert.NilError(t, cc.Create(ctx, base.DeepCopy(), client.DryRunAll),
		"expected this base to be valid")

	t.Run("LogsRetentionPeriod", func(t *testing.T) {
		pgadmin := base.DeepCopy()
		assert.NilError(t, yaml.UnmarshalStrict([]byte(`{
			instrumentation: {
				logs: { retentionPeriod: 5m },
			},
		}`), &pgadmin.Spec))

		err := cc.Create(ctx, pgadmin, client.DryRunAll)
		assert.Assert(t, apierrors.IsInvalid(err))
		assert.ErrorContains(t, err, "retentionPeriod")
		assert.ErrorContains(t, err, "hour|day|week")
		assert.ErrorContains(t, err, "one hour")

		//nolint:errorlint // This is a test, and a panic is unlikely.
		status := err.(apierrors.APIStatus).Status()
		assert.Assert(t, status.Details != nil)
		assert.Equal(t, len(status.Details.Causes), 2)

		for _, cause := range status.Details.Causes {
			assert.Equal(t, cause.Field, "spec.instrumentation.logs.retentionPeriod")
		}

		t.Run("Valid", func(t *testing.T) {
			for _, tt := range []string{
				"28 weeks",
				"90 DAY",
				"1 hr",
				"PT1D",
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
				"0 days",
				"0",
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
