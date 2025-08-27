// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"fmt"
	"testing"

	"gotest.tools/v3/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	v1 "github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestPostgresUserOptionsV1beta1(t *testing.T) {
	ctx := t.Context()
	cc := require.Kubernetes(t)
	t.Parallel()

	namespace := require.Namespace(t, cc)
	base := v1beta1.NewPostgresCluster()

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

	base.Namespace = namespace.Name
	base.Name = "postgres-user-options"

	assert.NilError(t, cc.Create(ctx, base.DeepCopy(), client.DryRunAll),
		"expected this base cluster to be valid")

	var u unstructured.Unstructured
	require.UnmarshalInto(t, &u, require.Value(yaml.Marshal(base)))
	assert.Equal(t, u.GetAPIVersion(), "postgres-operator.crunchydata.com/v1beta1")

	testPostgresUserOptionsCommon(t, cc, u)
}

func TestPostgresUserOptionsV1(t *testing.T) {
	ctx := t.Context()
	cc := require.KubernetesAtLeast(t, "1.30")
	t.Parallel()

	namespace := require.Namespace(t, cc)
	base := v1.NewPostgresCluster()

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

	base.Namespace = namespace.Name
	base.Name = "postgres-user-options"

	assert.NilError(t, cc.Create(ctx, base.DeepCopy(), client.DryRunAll),
		"expected this base cluster to be valid")

	var u unstructured.Unstructured
	require.UnmarshalInto(t, &u, require.Value(yaml.Marshal(base)))
	assert.Equal(t, u.GetAPIVersion(), "postgres-operator.crunchydata.com/v1")

	testPostgresUserOptionsCommon(t, cc, u)
}

func testPostgresUserOptionsCommon(t *testing.T, cc client.Client, base unstructured.Unstructured) {
	ctx := t.Context()

	// See [internal/controller/postgrescluster.TestValidatePostgresUsers]

	t.Run("NoComments", func(t *testing.T) {
		cluster := base.DeepCopy()
		require.UnmarshalIntoField(t, cluster,
			require.Value(yaml.Marshal([]v1beta1.PostgresUserSpec{
				{Name: "dashes", Options: "ANY -- comment"},
				{Name: "block-open", Options: "/* asdf"},
				{Name: "block-close", Options: " qw */ rt"},
			})),
			"spec", "users")

		err := cc.Create(ctx, cluster, client.DryRunAll)
		assert.Assert(t, apierrors.IsInvalid(err))
		assert.ErrorContains(t, err, "cannot contain comments")

		details := require.StatusErrorDetails(t, err)
		assert.Assert(t, cmp.Len(details.Causes, 3))

		for i, cause := range details.Causes {
			assert.Equal(t, cause.Field, fmt.Sprintf("spec.users[%d].options", i))
			assert.Assert(t, cmp.Contains(cause.Message, "cannot contain comments"))
		}
	})

	t.Run("NoPassword", func(t *testing.T) {
		cluster := base.DeepCopy()
		require.UnmarshalIntoField(t, cluster,
			require.Value(yaml.Marshal([]v1beta1.PostgresUserSpec{
				{Name: "uppercase", Options: "SUPERUSER PASSWORD ''"},
				{Name: "lowercase", Options: "password 'asdf'"},
			})),
			"spec", "users")

		err := cc.Create(ctx, cluster, client.DryRunAll)
		assert.Assert(t, apierrors.IsInvalid(err))
		assert.ErrorContains(t, err, "cannot assign password")

		details := require.StatusErrorDetails(t, err)
		assert.Assert(t, cmp.Len(details.Causes, 2))

		for i, cause := range details.Causes {
			assert.Equal(t, cause.Field, fmt.Sprintf("spec.users[%d].options", i))
			assert.Assert(t, cmp.Contains(cause.Message, "cannot assign password"))
		}
	})

	t.Run("NoTerminators", func(t *testing.T) {
		cluster := base.DeepCopy()
		require.UnmarshalIntoField(t, cluster,
			require.Value(yaml.Marshal([]v1beta1.PostgresUserSpec{
				{Name: "semicolon", Options: "some ;where"},
			})),
			"spec", "users")

		err := cc.Create(ctx, cluster, client.DryRunAll)
		assert.Assert(t, apierrors.IsInvalid(err))
		assert.ErrorContains(t, err, "should match")

		details := require.StatusErrorDetails(t, err)
		assert.Assert(t, cmp.Len(details.Causes, 1))
		assert.Equal(t, details.Causes[0].Field, "spec.users[0].options")
	})

	t.Run("Valid", func(t *testing.T) {
		cluster := base.DeepCopy()
		require.UnmarshalIntoField(t, cluster,
			require.Value(yaml.Marshal([]v1beta1.PostgresUserSpec{
				{Name: "normal", Options: "CREATEDB valid until '2006-01-02'"},
				{Name: "very-full", Options: "NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOLOGIN NOREPLICATION NOBYPASSRLS CONNECTION LIMIT 5"},
			})),
			"spec", "users")

		assert.NilError(t, cc.Create(ctx, cluster, client.DryRunAll))
	})
}
