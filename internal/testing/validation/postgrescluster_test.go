// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"fmt"
	"testing"

	"gotest.tools/v3/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestPostgresUserOptions(t *testing.T) {
	ctx := context.Background()
	cc := require.Kubernetes(t)
	t.Parallel()

	namespace := require.Namespace(t, cc)
	base := v1beta1.NewPostgresCluster()

	// Start with a bunch of required fields.
	assert.NilError(t, yaml.Unmarshal([]byte(`{
		postgresVersion: 16,
		backups: {
			pgbackrest: {
				repos: [{ name: repo1 }],
			},
		},
		instances: [{
			dataVolumeClaimSpec: {
				accessModes: [ReadWriteOnce],
				resources: { requests: { storage: 1Mi } },
			},
		}],
	}`), &base.Spec))

	base.Namespace = namespace.Name
	base.Name = "postgres-user-options"

	assert.NilError(t, cc.Create(ctx, base.DeepCopy(), client.DryRunAll),
		"expected this base cluster to be valid")

	// See [internal/controller/postgrescluster.TestValidatePostgresUsers]

	t.Run("NoComments", func(t *testing.T) {
		cluster := base.DeepCopy()
		cluster.Spec.Users = []v1beta1.PostgresUserSpec{
			{Name: "dashes", Options: "ANY -- comment"},
			{Name: "block-open", Options: "/* asdf"},
			{Name: "block-close", Options: " qw */ rt"},
		}

		err := cc.Create(ctx, cluster, client.DryRunAll)
		assert.Assert(t, apierrors.IsInvalid(err))
		assert.ErrorContains(t, err, "cannot contain comments")

		//nolint:errorlint // This is a test, and a panic is unlikely.
		status := err.(apierrors.APIStatus).Status()
		assert.Assert(t, status.Details != nil)
		assert.Equal(t, len(status.Details.Causes), 3)

		for i, cause := range status.Details.Causes {
			assert.Equal(t, cause.Field, fmt.Sprintf("spec.users[%d].options", i))
			assert.Assert(t, cmp.Contains(cause.Message, "cannot contain comments"))
		}
	})

	t.Run("NoPassword", func(t *testing.T) {
		cluster := base.DeepCopy()
		cluster.Spec.Users = []v1beta1.PostgresUserSpec{
			{Name: "uppercase", Options: "SUPERUSER PASSWORD ''"},
			{Name: "lowercase", Options: "password 'asdf'"},
		}

		err := cc.Create(ctx, cluster, client.DryRunAll)
		assert.Assert(t, apierrors.IsInvalid(err))
		assert.ErrorContains(t, err, "cannot assign password")

		//nolint:errorlint // This is a test, and a panic is unlikely.
		status := err.(apierrors.APIStatus).Status()
		assert.Assert(t, status.Details != nil)
		assert.Equal(t, len(status.Details.Causes), 2)

		for i, cause := range status.Details.Causes {
			assert.Equal(t, cause.Field, fmt.Sprintf("spec.users[%d].options", i))
			assert.Assert(t, cmp.Contains(cause.Message, "cannot assign password"))
		}
	})

	t.Run("NoTerminators", func(t *testing.T) {
		cluster := base.DeepCopy()
		cluster.Spec.Users = []v1beta1.PostgresUserSpec{
			{Name: "semicolon", Options: "some ;where"},
		}

		err := cc.Create(ctx, cluster, client.DryRunAll)
		assert.Assert(t, apierrors.IsInvalid(err))
		assert.ErrorContains(t, err, "should match")

		//nolint:errorlint // This is a test, and a panic is unlikely.
		status := err.(apierrors.APIStatus).Status()
		assert.Assert(t, status.Details != nil)
		assert.Equal(t, len(status.Details.Causes), 1)
		assert.Equal(t, status.Details.Causes[0].Field, "spec.users[0].options")
	})

	t.Run("Valid", func(t *testing.T) {
		cluster := base.DeepCopy()
		cluster.Spec.Users = []v1beta1.PostgresUserSpec{
			{Name: "normal", Options: "CREATEDB valid until '2006-01-02'"},
			{Name: "very-full", Options: "NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOLOGIN NOREPLICATION NOBYPASSRLS CONNECTION LIMIT 5"},
		}

		assert.NilError(t, cc.Create(ctx, cluster, client.DryRunAll))
	})
}
