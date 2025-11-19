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

func TestPostgresConfigParametersV1beta1(t *testing.T) {
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
	base.Name = "postgres-config-parameters"

	assert.NilError(t, cc.Create(ctx, base.DeepCopy(), client.DryRunAll),
		"expected this base cluster to be valid")

	var u unstructured.Unstructured
	require.UnmarshalInto(t, &u, require.Value(yaml.Marshal(base)))
	assert.Equal(t, u.GetAPIVersion(), "postgres-operator.crunchydata.com/v1beta1")

	testPostgresConfigParametersCommon(t, cc, u)

	t.Run("Logging", func(t *testing.T) {
		t.Run("Allowed", func(t *testing.T) {
			for _, tt := range []struct {
				key   string
				value any
			}{
				{key: "log_directory", value: "anything"},
			} {
				t.Run(tt.key, func(t *testing.T) {
					cluster := u.DeepCopy()
					require.UnmarshalIntoField(t, cluster,
						require.Value(yaml.Marshal(tt.value)),
						"spec", "config", "parameters", tt.key)

					assert.NilError(t, cc.Create(ctx, cluster, client.DryRunAll))
				})
			}
		})
	})

	t.Run("ssl_groups and ssl_ecdh_curve", func(t *testing.T) {
		t.Run("ssl_groups not allowed for pg17", func(t *testing.T) {
			for _, tt := range []struct {
				key   string
				value any
			}{
				{key: "ssl_groups", value: "anything"},
			} {
				t.Run(tt.key, func(t *testing.T) {
					cluster := u.DeepCopy()
					require.UnmarshalIntoField(t, cluster,
						require.Value(yaml.Marshal(17)),
						"spec", "postgresVersion")
					require.UnmarshalIntoField(t, cluster,
						require.Value(yaml.Marshal(tt.value)),
						"spec", "config", "parameters", tt.key)

					err := cc.Create(ctx, cluster, client.DryRunAll)
					assert.Assert(t, apierrors.IsInvalid(err))

					details := require.StatusErrorDetails(t, err)
					assert.Assert(t, cmp.Len(details.Causes, 1))
				})
			}
		})

		t.Run("ssl_groups allowed for pg18", func(t *testing.T) {
			for _, tt := range []struct {
				key   string
				value any
			}{
				{key: "ssl_groups", value: "anything"},
			} {
				t.Run(tt.key, func(t *testing.T) {
					cluster := u.DeepCopy()
					require.UnmarshalIntoField(t, cluster,
						require.Value(yaml.Marshal(18)),
						"spec", "postgresVersion")
					require.UnmarshalIntoField(t, cluster,
						require.Value(yaml.Marshal(tt.value)),
						"spec", "config", "parameters", tt.key)

					assert.NilError(t, cc.Create(ctx, cluster, client.DryRunAll))
				})
			}
		})

		t.Run("ssl_ecdh_curve allowed for both", func(t *testing.T) {
			for _, tt := range []struct {
				key   string
				value any
			}{
				{key: "ssl_ecdh_curve", value: "anything"},
			} {
				t.Run(tt.key, func(t *testing.T) {
					cluster := u.DeepCopy()
					require.UnmarshalIntoField(t, cluster,
						require.Value(yaml.Marshal(17)),
						"spec", "postgresVersion")
					require.UnmarshalIntoField(t, cluster,
						require.Value(yaml.Marshal(tt.value)),
						"spec", "config", "parameters", tt.key)

					assert.NilError(t, cc.Create(ctx, cluster, client.DryRunAll))

					cluster2 := u.DeepCopy()
					require.UnmarshalIntoField(t, cluster2,
						require.Value(yaml.Marshal(18)),
						"spec", "postgresVersion")
					require.UnmarshalIntoField(t, cluster2,
						require.Value(yaml.Marshal(tt.value)),
						"spec", "config", "parameters", tt.key)

					assert.NilError(t, cc.Create(ctx, cluster2, client.DryRunAll))
				})
			}
		})

		t.Run("other ssl_* parameters not allowed for any pg version", func(t *testing.T) {
			for _, tt := range []struct {
				key   string
				value any
			}{
				{key: "ssl_anything", value: "anything"},
			} {
				t.Run(tt.key, func(t *testing.T) {
					cluster := u.DeepCopy()
					require.UnmarshalIntoField(t, cluster,
						require.Value(yaml.Marshal(17)),
						"spec", "postgresVersion")
					require.UnmarshalIntoField(t, cluster,
						require.Value(yaml.Marshal(tt.value)),
						"spec", "config", "parameters", tt.key)

					err := cc.Create(ctx, cluster, client.DryRunAll)
					assert.Assert(t, apierrors.IsInvalid(err))

					details := require.StatusErrorDetails(t, err)
					assert.Assert(t, cmp.Len(details.Causes, 1))

					cluster1 := u.DeepCopy()
					require.UnmarshalIntoField(t, cluster1,
						require.Value(yaml.Marshal(18)),
						"spec", "postgresVersion")
					require.UnmarshalIntoField(t, cluster1,
						require.Value(yaml.Marshal(tt.value)),
						"spec", "config", "parameters", tt.key)

					err = cc.Create(ctx, cluster1, client.DryRunAll)
					assert.Assert(t, apierrors.IsInvalid(err))

					details = require.StatusErrorDetails(t, err)
					assert.Assert(t, cmp.Len(details.Causes, 1))
				})
			}
		})
	})
}

func TestPostgresConfigParametersV1(t *testing.T) {
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
	base.Name = "postgres-config-parameters"

	assert.NilError(t, cc.Create(ctx, base.DeepCopy(), client.DryRunAll),
		"expected this base cluster to be valid")

	var u unstructured.Unstructured
	require.UnmarshalInto(t, &u, require.Value(yaml.Marshal(base)))
	assert.Equal(t, u.GetAPIVersion(), "postgres-operator.crunchydata.com/v1")

	testPostgresConfigParametersCommon(t, cc, u)

	t.Run("Logging", func(t *testing.T) {
		t.Run("log_directory", func(t *testing.T) {
			volume := `{ accessModes: [ReadWriteOnce], resources: { requests: { storage: 1Mi } } }`

			for _, tt := range []struct {
				name      string
				value     any
				valid     bool
				message   string
				instances string
			}{
				// Only a few prefixes are allowed.
				{valid: false, value: 99, message: `must start with "/`},
				{valid: false, value: "relative", message: `must start with "/`},
				{valid: false, value: "/absolute", message: `must start with "/pg`},

				// These are the two acceptable directories inside /pgdata.
				{valid: true, value: "log"},
				{valid: true, value: "/pgdata/logs/postgres"},
				{valid: false, value: "/pgdata", message: `"/pgdata/logs/postgres"`},
				{valid: false, value: "/pgdata/elsewhere", message: `"/pgdata/logs/postgres"`},
				{valid: false, value: "/pgdata/sub/dir", message: `"/pgdata/logs/postgres"`},

				// There is one acceptable directory inside /pgtmp, but every instance set needs a temp volume.
				{
					name:  "two instance sets and two temp volumes",
					value: "/pgtmp/logs/postgres",
					instances: `[
						{ name: one, dataVolumeClaimSpec: ` + volume + `, volumes: { temp: ` + volume + ` } },
						{ name: two, dataVolumeClaimSpec: ` + volume + `, volumes: { temp: ` + volume + ` } },
					]`,
					valid: true,
				},
				{
					name:  "two instance sets and one temp volume",
					value: "/pgtmp/logs/postgres",
					instances: `[
						{ name: one, dataVolumeClaimSpec: ` + volume + `, volumes: { temp: ` + volume + ` } },
						{ name: two, dataVolumeClaimSpec: ` + volume + ` },
					]`,
					valid:   false,
					message: `all instances need "volumes.temp"`,
				},
				{
					name:  "two instance sets and no temp volumes",
					value: "/pgtmp/logs/postgres",
					instances: `[
						{ name: one, dataVolumeClaimSpec: ` + volume + ` },
						{ name: two, dataVolumeClaimSpec: ` + volume + ` },
					]`,
					valid:   false,
					message: `all instances need "volumes.temp"`,
				},

				// These directories inside /pgtmp are unacceptable, regardless of volumes.
				{
					name:      "no temp volumes",
					value:     "/pgtmp/elsewhere",
					instances: `[{ name: any, dataVolumeClaimSpec: ` + volume + ` }]`,
					valid:     false,
					message:   `"/pgtmp/logs/postgres"`,
				},
				{
					name:  "two temp volumes",
					value: "/pgtmp",
					instances: `[
						{ name: one, dataVolumeClaimSpec: ` + volume + `, volumes: { temp: ` + volume + ` } },
						{ name: two, dataVolumeClaimSpec: ` + volume + `, volumes: { temp: ` + volume + ` } },
					]`,
					valid:   false,
					message: `"/pgtmp/logs/postgres"`,
				},

				// There is one acceptable directory inside /pgwal, but every instance set needs a WAL volume.
				{
					name:  "two instance sets and two WAL volumes",
					value: "/pgwal/logs/postgres",
					instances: `[
						{ name: one, dataVolumeClaimSpec: ` + volume + `, walVolumeClaimSpec: ` + volume + ` },
						{ name: two, dataVolumeClaimSpec: ` + volume + `, walVolumeClaimSpec: ` + volume + ` },
					]`,
					valid: true,
				},
				{
					name:  "two instance sets and one WAL volume",
					value: "/pgwal/logs/postgres",
					instances: `[
						{ name: one, dataVolumeClaimSpec: ` + volume + `, walVolumeClaimSpec: ` + volume + ` },
						{ name: two, dataVolumeClaimSpec: ` + volume + ` },
					]`,
					valid:   false,
					message: `all instances need "walVolumeClaimSpec"`,
				},
				{
					name:  "two instance sets and no WAL volumes",
					value: "/pgwal/logs/postgres",
					instances: `[
						{ name: one, dataVolumeClaimSpec: ` + volume + ` },
						{ name: two, dataVolumeClaimSpec: ` + volume + ` },
					]`,
					valid:   false,
					message: `all instances need "walVolumeClaimSpec"`,
				},

				// These directories inside /pgwal are unacceptable, regardless of volumes.
				{
					name:      "no WAL volumes",
					value:     "/pgwal/elsewhere",
					instances: `[{ name: any, dataVolumeClaimSpec: ` + volume + ` }]`,
					valid:     false,
					message:   `"/pgwal/logs/postgres"`,
				},
				{
					name:  "two WAL volumes",
					value: "/pgwal",
					instances: `[
						{ name: one, dataVolumeClaimSpec: ` + volume + `, walVolumeClaimSpec: ` + volume + ` },
						{ name: two, dataVolumeClaimSpec: ` + volume + `, walVolumeClaimSpec: ` + volume + ` },
					]`,
					valid:   false,
					message: `"/pgwal/logs/postgres"`,
				},

				// Directories inside /volumes are acceptable, but every instance set needs the correct additional volume.
				{
					name:  "two instance sets and two correct additional volumes",
					value: "/volumes/yep",
					instances: `[
						{ name: one, dataVolumeClaimSpec: ` + volume + `, volumes: { additional: [{ name: yep, claimName: a }] } },
						{ name: two, dataVolumeClaimSpec: ` + volume + `, volumes: { additional: [{ name: yep, claimName: b }] } },
					]`,
					valid: true,
				},
				{
					name:  "two instance sets and one correct additional volume",
					value: "/volumes/yep",
					instances: `[
						{ name: one, dataVolumeClaimSpec: ` + volume + `, volumes: { additional: [{ name: yep, claimName: a }] } },
						{ name: two, dataVolumeClaimSpec: ` + volume + `, volumes: { additional: [{ name: diff, claimName: b }] } },
					]`,
					valid:   false,
					message: `all instances need an additional volume`,
				},
				{
					name:  "two instance sets and one additional volume",
					value: "/volumes/yep",
					instances: `[
						{ name: one, dataVolumeClaimSpec: ` + volume + `, volumes: { additional: [{ name: yep, claimName: a }] } },
						{ name: two, dataVolumeClaimSpec: ` + volume + ` },
					]`,
					valid:   false,
					message: `all instances need an additional volume`,
				},
				{
					name:  "two instance sets and no additional volumes",
					value: "/volumes/yep",
					instances: `[
						{ name: one, dataVolumeClaimSpec: ` + volume + ` },
						{ name: two, dataVolumeClaimSpec: ` + volume + ` },
					]`,
					valid:   false,
					message: `all instances need an additional volume`,
				},
			} {
				t.Run(cmp.Or(tt.name, fmt.Sprint(tt.valid, tt.value)), func(t *testing.T) {
					cluster := u.DeepCopy()
					if tt.instances != "" {
						require.UnmarshalIntoField(t, cluster, tt.instances, "spec", "instances")
					}
					require.UnmarshalIntoField(t, cluster, require.Value(yaml.Marshal(tt.value)),
						"spec", "config", "parameters", "log_directory")

					err := cc.Create(ctx, cluster, client.DryRunAll)

					if tt.valid {
						assert.NilError(t, err)
						assert.Equal(t, "", tt.message, "BUG IN TEST: no message expected when valid")
					} else {
						assert.Assert(t, apierrors.IsInvalid(err))

						details := require.StatusErrorDetails(t, err)
						assert.Assert(t, cmp.Len(details.Causes, 1))

						// https://issue.k8s.io/133761
						if details.Causes[0].Field != "spec.config.parameters.[log_directory]" {
							assert.Check(t, cmp.Equal(details.Causes[0].Field, "spec.config.parameters[log_directory]"))
						}
						assert.Assert(t, cmp.Contains(details.Causes[0].Message, tt.message))
					}
				})
			}
		})
	})
}

func testPostgresConfigParametersCommon(t *testing.T, cc client.Client, base unstructured.Unstructured) {
	ctx := t.Context()

	t.Run("Allowed", func(t *testing.T) {
		for _, tt := range []struct {
			key   string
			value any
		}{
			{"archive_timeout", 100},
			{"archive_timeout", "20s"},
		} {
			t.Run(tt.key, func(t *testing.T) {
				cluster := base.DeepCopy()
				require.UnmarshalIntoField(t, cluster,
					require.Value(yaml.Marshal(tt.value)),
					"spec", "config", "parameters", tt.key)

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
			{key: "port", value: 5},
			{key: "wal_log_hints", value: "off"},
		} {
			t.Run(tt.key, func(t *testing.T) {
				cluster := base.DeepCopy()
				require.UnmarshalIntoField(t, cluster,
					require.Value(yaml.Marshal(tt.value)),
					"spec", "config", "parameters", tt.key)

				err := cc.Create(ctx, cluster, client.DryRunAll)
				assert.Assert(t, apierrors.IsInvalid(err))

				details := require.StatusErrorDetails(t, err)
				assert.Assert(t, cmp.Len(details.Causes, 1))

				// TODO(k8s-1.30) TODO(validation): Move the parameter name from the message to the field path.
				assert.Equal(t, details.Causes[0].Field, "spec.config.parameters")
				assert.Assert(t, cmp.Contains(details.Causes[0].Message, tt.key))
			})
		}
	})

	t.Run("Logging", func(t *testing.T) {
		for _, tt := range []struct {
			valid   bool
			key     string
			value   any
			message string
		}{
			{valid: false, key: "log_file_mode", value: "", message: "cannot be changed"},
			{valid: false, key: "log_file_mode", value: "any", message: "cannot be changed"},
			{valid: false, key: "logging_collector", value: "", message: "unsafe"},
			{valid: false, key: "logging_collector", value: "off", message: "unsafe"},
			{valid: false, key: "logging_collector", value: "on", message: "unsafe"},

			{valid: true, key: "log_destination", value: "anything"},
			{valid: true, key: "log_filename", value: "anything"},
			{valid: true, key: "log_filename", value: "percent-%s-too"},
			{valid: true, key: "log_rotation_age", value: "7d"},
			{valid: true, key: "log_rotation_age", value: 5},
			{valid: true, key: "log_rotation_size", value: "100MB"},
			{valid: true, key: "log_rotation_size", value: 13},
			{valid: true, key: "log_timezone", value: ""},
			{valid: true, key: "log_timezone", value: "nonsense"},
		} {
			t.Run(fmt.Sprint(tt), func(t *testing.T) {
				cluster := base.DeepCopy()
				require.UnmarshalIntoField(t, cluster,
					require.Value(yaml.Marshal(tt.value)),
					"spec", "config", "parameters", tt.key)

				err := cc.Create(ctx, cluster, client.DryRunAll)

				if tt.valid {
					assert.NilError(t, err)
					assert.Equal(t, "", tt.message, "BUG IN TEST: no message expected when valid")
				} else {
					assert.Assert(t, apierrors.IsInvalid(err))

					details := require.StatusErrorDetails(t, err)
					assert.Assert(t, cmp.Len(details.Causes, 1))

					// TODO(k8s-1.30) TODO(validation): Move the parameter name from the message to the field path.
					assert.Equal(t, details.Causes[0].Field, "spec.config.parameters")
					assert.Assert(t, cmp.Contains(details.Causes[0].Message, tt.key))
					assert.Assert(t, cmp.Contains(details.Causes[0].Message, tt.message))
				}
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
				cluster := base.DeepCopy()
				require.UnmarshalIntoField(t, cluster,
					require.Value(yaml.Marshal(tt.value)),
					"spec", "config", "parameters", tt.key)

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
				cluster := base.DeepCopy()
				require.UnmarshalIntoField(t, cluster,
					require.Value(yaml.Marshal(tt.value)),
					"spec", "config", "parameters", tt.key)

				err := cc.Create(ctx, cluster, client.DryRunAll)
				assert.Assert(t, apierrors.IsInvalid(err))
			})
		}
	})

	t.Run("wal_level", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			cluster := base.DeepCopy()
			require.UnmarshalIntoField(t, cluster,
				`logical`, "spec", "config", "parameters", "wal_level")

			assert.NilError(t, cc.Create(ctx, cluster, client.DryRunAll))
		})

		t.Run("Invalid", func(t *testing.T) {
			cluster := base.DeepCopy()
			require.UnmarshalIntoField(t, cluster,
				`minimal`, "spec", "config", "parameters", "wal_level")

			err := cc.Create(ctx, cluster, client.DryRunAll)
			assert.Assert(t, apierrors.IsInvalid(err))
			assert.ErrorContains(t, err, `"replica" or higher`)

			details := require.StatusErrorDetails(t, err)
			assert.Assert(t, cmp.Len(details.Causes, 1))
			assert.Equal(t, details.Causes[0].Field, "spec.config.parameters")
			assert.Assert(t, cmp.Contains(details.Causes[0].Message, "wal_level"))
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
				cluster := base.DeepCopy()
				require.UnmarshalIntoField(t, cluster,
					require.Value(yaml.Marshal(tt.value)),
					"spec", "config", "parameters", tt.key)

				err := cc.Create(ctx, cluster, client.DryRunAll)
				assert.Assert(t, apierrors.IsInvalid(err))
			})
		}
	})
}
