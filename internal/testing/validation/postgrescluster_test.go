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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestPostgresAuthenticationRules(t *testing.T) {
	ctx := context.Background()
	cc := require.Kubernetes(t)
	t.Parallel()

	namespace := require.Namespace(t, cc)
	base := v1beta1.NewPostgresCluster()

	// Start with a bunch of required fields.
	require.UnmarshalInto(t, &base.Spec, `{
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
	}`)

	base.Namespace = namespace.Name
	base.Name = "postgres-authentication-rules"

	assert.NilError(t, cc.Create(ctx, base.DeepCopy(), client.DryRunAll),
		"expected this base cluster to be valid")

	t.Run("OneTopLevel", func(t *testing.T) {
		cluster := base.DeepCopy()
		require.UnmarshalInto(t, &cluster.Spec.Authentication, `{
			rules: [
				{ connection: host, hba: anything },
				{ users: [alice, bob], hba: anything },
			],
		}`)

		err := cc.Create(ctx, cluster, client.DryRunAll)
		assert.Assert(t, apierrors.IsInvalid(err))

		status := require.StatusError(t, err)
		assert.Assert(t, status.Details != nil)
		assert.Assert(t, cmp.Len(status.Details.Causes, 2))

		for i, cause := range status.Details.Causes {
			assert.Equal(t, cause.Field, fmt.Sprintf("spec.authentication.rules[%d]", i))
			assert.Assert(t, cmp.Contains(cause.Message, "cannot be combined"))
		}
	})

	t.Run("NoInclude", func(t *testing.T) {
		cluster := base.DeepCopy()
		require.UnmarshalInto(t, &cluster.Spec.Authentication, `{
			rules: [
				{ hba: 'include "/etc/passwd"' },
				{ hba: '   include_dir /tmp' },
				{ hba: 'include_if_exists postgresql.auto.conf' },
			],
		}`)

		err := cc.Create(ctx, cluster, client.DryRunAll)
		assert.Assert(t, apierrors.IsInvalid(err))

		status := require.StatusError(t, err)
		assert.Assert(t, status.Details != nil)
		assert.Assert(t, cmp.Len(status.Details.Causes, 3))

		for i, cause := range status.Details.Causes {
			assert.Equal(t, cause.Field, fmt.Sprintf("spec.authentication.rules[%d].hba", i))
			assert.Assert(t, cmp.Contains(cause.Message, "cannot include"))
		}
	})

	t.Run("NoStructuredTrust", func(t *testing.T) {
		cluster := base.DeepCopy()
		require.UnmarshalInto(t, &cluster.Spec.Authentication, `{
			rules: [
				{ connection: local, method: trust },
				{ connection: hostssl, method: trust },
				{ connection: hostgssenc, method: trust },
			],
		}`)

		err := cc.Create(ctx, cluster, client.DryRunAll)
		assert.Assert(t, apierrors.IsInvalid(err))

		status := require.StatusError(t, err)
		assert.Assert(t, status.Details != nil)
		assert.Assert(t, cmp.Len(status.Details.Causes, 3))

		for i, cause := range status.Details.Causes {
			assert.Equal(t, cause.Field, fmt.Sprintf("spec.authentication.rules[%d].method", i))
			assert.Assert(t, cmp.Contains(cause.Message, "unsafe"))
		}
	})
}

func TestPostgresConfigParameters(t *testing.T) {
	ctx := context.Background()
	cc := require.Kubernetes(t)
	t.Parallel()

	namespace := require.Namespace(t, cc)
	base := v1beta1.NewPostgresCluster()

	// Start with a bunch of required fields.
	require.UnmarshalInto(t, &base.Spec, `{
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
	}`)

	base.Namespace = namespace.Name
	base.Name = "postgres-config-parameters"

	assert.NilError(t, cc.Create(ctx, base.DeepCopy(), client.DryRunAll),
		"expected this base cluster to be valid")

	t.Run("Allowed", func(t *testing.T) {
		for _, tt := range []struct {
			key   string
			value any
		}{
			{"archive_timeout", int64(100)},
			{"archive_timeout", "20s"},
		} {
			t.Run(tt.key, func(t *testing.T) {
				cluster := require.Value(runtime.ToUnstructuredObject(base))
				assert.NilError(t, unstructured.SetNestedField(cluster.Object,
					tt.value, "spec", "config", "parameters", tt.key))

				assert.NilError(t, cc.Create(ctx, cluster, client.DryRunAll))
			})
		}
	})

	t.Run("Disallowed", func(t *testing.T) {
		for _, tt := range []struct {
			key   string
			value any
		}{
			{key: "cluster_name", value: "asdf"},
			{key: "config_file", value: "asdf"},
			{key: "data_directory", value: ""},
			{key: "external_pid_file", value: ""},
			{key: "hba_file", value: "one"},
			{key: "hot_standby", value: "off"},
			{key: "ident_file", value: "two"},
			{key: "listen_addresses", value: ""},
			{key: "log_file_mode", value: ""},
			{key: "logging_collector", value: "off"},
			{key: "port", value: int64(5)},
			{key: "wal_log_hints", value: "off"},
		} {
			t.Run(tt.key, func(t *testing.T) {
				cluster := require.Value(runtime.ToUnstructuredObject(base))
				assert.NilError(t, unstructured.SetNestedField(cluster.Object,
					tt.value, "spec", "config", "parameters", tt.key))

				err := cc.Create(ctx, cluster, client.DryRunAll)
				assert.Assert(t, apierrors.IsInvalid(err))

				status := require.StatusError(t, err)
				assert.Assert(t, status.Details != nil)
				assert.Assert(t, cmp.Len(status.Details.Causes, 1))

				// TODO(k8s-1.30) TODO(validation): Move the parameter name from the message to the field path.
				assert.Equal(t, status.Details.Causes[0].Field, "spec.config.parameters")
				assert.Assert(t, cmp.Contains(status.Details.Causes[0].Message, tt.key))
			})
		}
	})

	t.Run("NoConnections", func(t *testing.T) {
		for _, tt := range []struct {
			key   string
			value any
		}{
			{key: "ssl", value: "off"},
			{key: "ssl_ca_file", value: ""},
			{key: "unix_socket_directories", value: "one"},
			{key: "unix_socket_group", value: "two"},
		} {
			t.Run(tt.key, func(t *testing.T) {
				cluster := require.Value(runtime.ToUnstructuredObject(base))
				assert.NilError(t, unstructured.SetNestedField(cluster.Object,
					tt.value, "spec", "config", "parameters", tt.key))

				err := cc.Create(ctx, cluster, client.DryRunAll)
				assert.Assert(t, apierrors.IsInvalid(err))
			})
		}
	})

	t.Run("NoWriteAheadLog", func(t *testing.T) {
		for _, tt := range []struct {
			key   string
			value any
		}{
			{key: "archive_mode", value: "off"},
			{key: "archive_command", value: "true"},
			{key: "restore_command", value: "true"},
			{key: "recovery_target", value: "immediate"},
			{key: "recovery_target_name", value: "doot"},
		} {
			t.Run(tt.key, func(t *testing.T) {
				cluster := require.Value(runtime.ToUnstructuredObject(base))
				assert.NilError(t, unstructured.SetNestedField(cluster.Object,
					tt.value, "spec", "config", "parameters", tt.key))

				err := cc.Create(ctx, cluster, client.DryRunAll)
				assert.Assert(t, apierrors.IsInvalid(err))
			})
		}
	})

	t.Run("wal_level", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			cluster := base.DeepCopy()

			cluster.Spec.Config = &v1beta1.PostgresConfigSpec{
				Parameters: map[string]intstr.IntOrString{
					"wal_level": intstr.FromString("logical"),
				},
			}
			assert.NilError(t, cc.Create(ctx, cluster, client.DryRunAll))
		})

		t.Run("Invalid", func(t *testing.T) {
			cluster := base.DeepCopy()

			cluster.Spec.Config = &v1beta1.PostgresConfigSpec{
				Parameters: map[string]intstr.IntOrString{
					"wal_level": intstr.FromString("minimal"),
				},
			}

			err := cc.Create(ctx, cluster, client.DryRunAll)
			assert.Assert(t, apierrors.IsInvalid(err))
			assert.ErrorContains(t, err, `"replica" or higher`)

			status := require.StatusError(t, err)
			assert.Assert(t, status.Details != nil)
			assert.Assert(t, cmp.Len(status.Details.Causes, 1))
			assert.Equal(t, status.Details.Causes[0].Field, "spec.config.parameters")
			assert.Assert(t, cmp.Contains(status.Details.Causes[0].Message, "wal_level"))
		})
	})

	t.Run("NoReplication", func(t *testing.T) {
		for _, tt := range []struct {
			key   string
			value any
		}{
			{key: "synchronous_standby_names", value: ""},
			{key: "primary_conninfo", value: ""},
			{key: "primary_slot_name", value: ""},
			{key: "recovery_min_apply_delay", value: ""},
		} {
			t.Run(tt.key, func(t *testing.T) {
				cluster := require.Value(runtime.ToUnstructuredObject(base))
				assert.NilError(t, unstructured.SetNestedField(cluster.Object,
					tt.value, "spec", "config", "parameters", tt.key))

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
	require.UnmarshalInto(t, &base.Spec, `{
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
	}`)

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

		status := require.StatusError(t, err)
		assert.Assert(t, status.Details != nil)
		assert.Assert(t, cmp.Len(status.Details.Causes, 3))

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

		status := require.StatusError(t, err)
		assert.Assert(t, status.Details != nil)
		assert.Assert(t, cmp.Len(status.Details.Causes, 2))

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

		status := require.StatusError(t, err)
		assert.Assert(t, status.Details != nil)
		assert.Assert(t, cmp.Len(status.Details.Causes, 1))
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
