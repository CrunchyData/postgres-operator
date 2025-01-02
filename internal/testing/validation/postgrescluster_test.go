// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"fmt"
	"testing"

	"gotest.tools/v3/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestPostgresConfigParameters(t *testing.T) {
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
	base.Name = "postgres-config-parameters"

	assert.NilError(t, cc.Create(ctx, base.DeepCopy(), client.DryRunAll),
		"expected this base cluster to be valid")

	t.Run("Disallowed", func(t *testing.T) {
		for _, tt := range []struct {
			key   string
			value intstr.IntOrString
		}{
			{key: "cluster_name", value: intstr.FromString("asdf")},
			{key: "config_file", value: intstr.FromString("asdf")},
			{key: "data_directory", value: intstr.FromString("")},
			{key: "external_pid_file", value: intstr.FromString("")},
			{key: "hba_file", value: intstr.FromString("one")},
			{key: "hot_standby", value: intstr.FromString("off")},
			{key: "ident_file", value: intstr.FromString("two")},
			{key: "listen_addresses", value: intstr.FromString("")},
			{key: "log_file_mode", value: intstr.FromString("")},
			{key: "logging_collector", value: intstr.FromString("off")},
			{key: "port", value: intstr.FromInt(5)},
			{key: "wal_log_hints", value: intstr.FromString("off")},
		} {
			t.Run(tt.key, func(t *testing.T) {
				cluster := base.DeepCopy()
				cluster.Spec.Config.Parameters = map[string]intstr.IntOrString{
					tt.key: tt.value,
				}

				err := cc.Create(ctx, cluster, client.DryRunAll)
				assert.Assert(t, apierrors.IsInvalid(err))

				//nolint:errorlint // This is a test, and a panic is unlikely.
				status := err.(apierrors.APIStatus).Status()
				assert.Assert(t, status.Details != nil)
				assert.Equal(t, len(status.Details.Causes), 1)

				// TODO(k8s-1.30) TODO(validation): Move the parameter name from the message to the field path.
				assert.Equal(t, status.Details.Causes[0].Field, "spec.config.parameters")
				assert.Assert(t, cmp.Contains(status.Details.Causes[0].Message, tt.key))
			})
		}
	})

	t.Run("NoConnections", func(t *testing.T) {
		for _, tt := range []struct {
			key   string
			value intstr.IntOrString
		}{
			{key: "ssl", value: intstr.FromString("off")},
			{key: "ssl_ca_file", value: intstr.FromString("")},
			{key: "unix_socket_directories", value: intstr.FromString("one")},
			{key: "unix_socket_group", value: intstr.FromString("two")},
		} {
			t.Run(tt.key, func(t *testing.T) {
				cluster := base.DeepCopy()
				cluster.Spec.Config.Parameters = map[string]intstr.IntOrString{
					tt.key: tt.value,
				}

				err := cc.Create(ctx, cluster, client.DryRunAll)
				assert.Assert(t, apierrors.IsInvalid(err))
			})
		}
	})

	t.Run("NoWriteAheadLog", func(t *testing.T) {
		for _, tt := range []struct {
			key   string
			value intstr.IntOrString
		}{
			{key: "archive_mode", value: intstr.FromString("off")},
			{key: "archive_command", value: intstr.FromString("true")},
			{key: "restore_command", value: intstr.FromString("true")},
			{key: "recovery_target", value: intstr.FromString("immediate")},
			{key: "recovery_target_name", value: intstr.FromString("doot")},
		} {
			t.Run(tt.key, func(t *testing.T) {
				cluster := base.DeepCopy()
				cluster.Spec.Config.Parameters = map[string]intstr.IntOrString{
					tt.key: tt.value,
				}

				err := cc.Create(ctx, cluster, client.DryRunAll)
				assert.Assert(t, apierrors.IsInvalid(err))
			})
		}
	})

	t.Run("wal_level", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			cluster := base.DeepCopy()

			cluster.Spec.Config.Parameters = map[string]intstr.IntOrString{
				"wal_level": intstr.FromString("logical"),
			}
			assert.NilError(t, cc.Create(ctx, cluster, client.DryRunAll))

			cluster.Spec.Config.Parameters = map[string]intstr.IntOrString{
				"wal_level": intstr.FromString("replica"),
			}
			assert.NilError(t, cc.Create(ctx, cluster, client.DryRunAll))
		})

		t.Run("Invalid", func(t *testing.T) {
			cluster := base.DeepCopy()

			cluster.Spec.Config.Parameters = map[string]intstr.IntOrString{
				"wal_level": intstr.FromString("minimal"),
			}

			err := cc.Create(ctx, cluster, client.DryRunAll)
			assert.Assert(t, apierrors.IsInvalid(err))
			assert.ErrorContains(t, err, `"replica" or higher`)

			//nolint:errorlint // This is a test, and a panic is unlikely.
			status := err.(apierrors.APIStatus).Status()
			assert.Assert(t, status.Details != nil)
			assert.Equal(t, len(status.Details.Causes), 1)
			assert.Equal(t, status.Details.Causes[0].Field, "spec.config.parameters")
			assert.Assert(t, cmp.Contains(status.Details.Causes[0].Message, "wal_level"))
		})
	})

	t.Run("NoReplication", func(t *testing.T) {
		for _, tt := range []struct {
			key   string
			value intstr.IntOrString
		}{
			{key: "synchronous_standby_names", value: intstr.FromString("")},
			{key: "primary_conninfo", value: intstr.FromString("")},
			{key: "primary_slot_name", value: intstr.FromString("")},
			{key: "recovery_min_apply_delay", value: intstr.FromString("")},
		} {
			t.Run(tt.key, func(t *testing.T) {
				cluster := base.DeepCopy()
				cluster.Spec.Config.Parameters = map[string]intstr.IntOrString{
					tt.key: tt.value,
				}

				err := cc.Create(ctx, cluster, client.DryRunAll)
				assert.Assert(t, apierrors.IsInvalid(err))
			})
		}
	})
}

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
