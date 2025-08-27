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

func TestPostgresAuthenticationV1beta1(t *testing.T) {
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
	base.Name = "postgres-authentication-rules"

	assert.NilError(t, cc.Create(ctx, base.DeepCopy(), client.DryRunAll),
		"expected this base cluster to be valid")

	var u unstructured.Unstructured
	require.UnmarshalInto(t, &u, require.Value(yaml.Marshal(base)))
	assert.Equal(t, u.GetAPIVersion(), "postgres-operator.crunchydata.com/v1beta1")

	testPostgresAuthenticationCommon(t, cc, u)
}

func TestPostgresAuthenticationV1(t *testing.T) {
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
	base.Name = "postgres-authentication-rules"

	assert.NilError(t, cc.Create(ctx, base.DeepCopy(), client.DryRunAll),
		"expected this base cluster to be valid")

	var u unstructured.Unstructured
	require.UnmarshalInto(t, &u, require.Value(yaml.Marshal(base)))
	assert.Equal(t, u.GetAPIVersion(), "postgres-operator.crunchydata.com/v1")

	testPostgresAuthenticationCommon(t, cc, u)
}

func testPostgresAuthenticationCommon(t *testing.T, cc client.Client, base unstructured.Unstructured) {
	ctx := t.Context()

	t.Run("OneTopLevel", func(t *testing.T) {
		cluster := base.DeepCopy()
		require.UnmarshalIntoField(t, cluster, `{
			rules: [
				{ connection: host, hba: anything },
				{ users: [alice, bob], hba: anything },
			],
		}`, "spec", "authentication")

		err := cc.Create(ctx, cluster, client.DryRunAll)
		assert.Assert(t, apierrors.IsInvalid(err))

		details := require.StatusErrorDetails(t, err)
		assert.Assert(t, cmp.Len(details.Causes, 2))

		for i, cause := range details.Causes {
			assert.Equal(t, cause.Field, fmt.Sprintf("spec.authentication.rules[%d]", i))
			assert.Assert(t, cmp.Contains(cause.Message, "cannot be combined"))
		}
	})

	t.Run("NoInclude", func(t *testing.T) {
		cluster := base.DeepCopy()
		require.UnmarshalIntoField(t, cluster, `{
			rules: [
				{ hba: 'include "/etc/passwd"' },
				{ hba: '   include_dir /tmp' },
				{ hba: 'include_if_exists postgresql.auto.conf' },
			],
		}`, "spec", "authentication")

		err := cc.Create(ctx, cluster, client.DryRunAll)
		assert.Assert(t, apierrors.IsInvalid(err))

		details := require.StatusErrorDetails(t, err)
		assert.Assert(t, cmp.Len(details.Causes, 3))

		for i, cause := range details.Causes {
			assert.Equal(t, cause.Field, fmt.Sprintf("spec.authentication.rules[%d].hba", i))
			assert.Assert(t, cmp.Contains(cause.Message, "cannot include"))
		}
	})

	t.Run("NoStructuredTrust", func(t *testing.T) {
		cluster := base.DeepCopy()
		require.UnmarshalIntoField(t, cluster, `{
			rules: [
				{ connection: local, method: trust },
				{ connection: hostssl, method: trust },
				{ connection: hostgssenc, method: trust },
			],
		}`, "spec", "authentication")

		err := cc.Create(ctx, cluster, client.DryRunAll)
		assert.Assert(t, apierrors.IsInvalid(err))

		details := require.StatusErrorDetails(t, err)
		assert.Assert(t, cmp.Len(details.Causes, 3))

		for i, cause := range details.Causes {
			assert.Equal(t, cause.Field, fmt.Sprintf("spec.authentication.rules[%d].method", i))
			assert.Assert(t, cmp.Contains(cause.Message, "unsafe"))
		}
	})

	t.Run("LDAP", func(t *testing.T) {
		t.Run("Required", func(t *testing.T) {
			cluster := base.DeepCopy()
			require.UnmarshalIntoField(t, cluster, `{
				rules: [
					{ connection: hostssl, method: ldap },
					{ connection: hostssl, method: ldap, options: {} },
					{ connection: hostssl, method: ldap, options: { ldapbinddn: any } },
				],
			}`, "spec", "authentication")

			err := cc.Create(ctx, cluster, client.DryRunAll)
			assert.Assert(t, apierrors.IsInvalid(err))

			details := require.StatusErrorDetails(t, err)
			assert.Assert(t, cmp.Len(details.Causes, 3))

			for i, cause := range details.Causes {
				assert.Equal(t, cause.Field, fmt.Sprintf("spec.authentication.rules[%d]", i), "%#v", cause)
				assert.Assert(t, cmp.Contains(cause.Message, `"ldap" method requires`))
			}

			// These are valid.

			unstructured.RemoveNestedField(cluster.Object, "spec", "authentication")
			require.UnmarshalIntoField(t, cluster, `{
				rules: [
					{ connection: hostssl, method: ldap, options: { ldapbasedn: any } },
					{ connection: hostssl, method: ldap, options: { ldapprefix: any } },
					{ connection: hostssl, method: ldap, options: { ldapsuffix: any } },
				],
			}`, "spec", "authentication")
			assert.NilError(t, cc.Create(ctx, cluster, client.DryRunAll))
		})

		t.Run("Mixed", func(t *testing.T) {
			// Some options cannot be combined with others.

			cluster := base.DeepCopy()
			require.UnmarshalIntoField(t, cluster, `{
				rules: [
					{ connection: hostssl, method: ldap, options: { ldapbinddn: any, ldapprefix: other } },
					{ connection: hostssl, method: ldap, options: { ldapbasedn: any, ldapsuffix: other } },
				],
			}`, "spec", "authentication")

			err := cc.Create(ctx, cluster, client.DryRunAll)
			assert.Assert(t, apierrors.IsInvalid(err))

			details := require.StatusErrorDetails(t, err)
			assert.Assert(t, cmp.Len(details.Causes, 2))

			for i, cause := range details.Causes {
				assert.Equal(t, cause.Field, fmt.Sprintf("spec.authentication.rules[%d]", i), "%#v", cause)
				assert.Assert(t, cmp.Regexp(`cannot use .+? options with .+? options`, cause.Message))
			}

			// These combinations are allowed.

			unstructured.RemoveNestedField(cluster.Object, "spec", "authentication")
			require.UnmarshalIntoField(t, cluster, `{
				rules: [
					{ connection: hostssl, method: ldap, options: { ldapprefix: one, ldapsuffix: two } },
					{ connection: hostssl, method: ldap, options: { ldapbasedn: one, ldapbinddn: two } },
					{ connection: hostssl, method: ldap, options: {
						ldapbasedn: one, ldapsearchattribute: two, ldapsearchfilter: three,
					} },
				],
			}`, "spec", "authentication")
			assert.NilError(t, cc.Create(ctx, cluster, client.DryRunAll))
		})
	})

	t.Run("RADIUS", func(t *testing.T) {
		t.Run("Required", func(t *testing.T) {
			cluster := base.DeepCopy()
			require.UnmarshalIntoField(t, cluster, `{
				rules: [
					{ connection: hostssl, method: radius },
					{ connection: hostssl, method: radius, options: {} },
					{ connection: hostssl, method: radius, options: { radiusidentifiers: any } },
					{ connection: hostssl, method: radius, options: { radiusservers: any } },
					{ connection: hostssl, method: radius, options: { radiussecrets: any } },
				],
			}`, "spec", "authentication")

			err := cc.Create(ctx, cluster, client.DryRunAll)
			assert.Assert(t, apierrors.IsInvalid(err))

			details := require.StatusErrorDetails(t, err)
			assert.Assert(t, cmp.Len(details.Causes, 5))

			for i, cause := range details.Causes {
				assert.Equal(t, cause.Field, fmt.Sprintf("spec.authentication.rules[%d]", i), "%#v", cause)
				assert.Assert(t, cmp.Contains(cause.Message, `"radius" method requires`))
			}

			// These are valid.

			unstructured.RemoveNestedField(cluster.Object, "spec", "authentication")
			require.UnmarshalIntoField(t, cluster, `{
				rules: [
					{ connection: hostssl, method: radius, options: { radiusservers: one, radiussecrets: two } },
					{ connection: hostssl, method: radius, options: {
						radiusservers: one, radiussecrets: two, radiusports: three,
					} },
				],
			}`, "spec", "authentication")
			assert.NilError(t, cc.Create(ctx, cluster, client.DryRunAll))
		})
	})
}
