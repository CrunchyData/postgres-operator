// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/events"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestGeneratePostgresHBA(t *testing.T) {
	reconciler := &Reconciler{}

	assert.Assert(t, reconciler.generatePostgresHBA(nil) == nil,
		"expected nil to return nil")

	for _, tt := range []struct {
		rule, expected string
	}{
		{
			rule:     `{ connection: host, method: scram }`,
			expected: `"host" all all all "scram"`,
		},
		{
			rule:     `{ connection: local, method: peer, databases: [one, two] }`,
			expected: `"local" "one","two" all all "peer"`,
		},
		{
			rule:     `{ connection: local, method: peer, users: [alice, bob] }`,
			expected: `"local" all "alice","bob" all "peer"`,
		},
		{
			rule:     `{ connection: hostssl, method: md5, options: { clientcert: verify-ca } }`,
			expected: `"hostssl" all all all "md5"  "clientcert"="verify-ca"`,
		},
		// "password" input should be "scram-sha-256" output
		{
			rule:     `{ connection: hostssl, method: password }`,
			expected: `"hostssl" all all all "scram-sha-256"`,
		},
	} {
		var rule *v1beta1.PostgresHBARule
		require.UnmarshalInto(t, &rule, tt.rule)

		hba := reconciler.generatePostgresHBA(rule)
		assert.Equal(t, hba.String(), tt.expected, "\n%#v", rule)
	}
}

func TestGeneratePostgresHBAs(t *testing.T) {
	ctx := context.Background()
	reconciler := &Reconciler{}

	builtin := reconciler.generatePostgresHBAs(ctx, v1beta1.NewPostgresCluster()).AsStrings()
	assert.Assert(t, len(builtin) > 0,
		"expected an empty cluster to have some builtin rules")

	defaults := builtin[len(builtin)-1:]
	assert.Assert(t, len(defaults) > 0,
		"expected at least one default rule")

	required := builtin[:len(builtin)-len(defaults)]
	assert.Assert(t, len(required) > 0,
		"expected at least one mandatory rule")

	t.Run("Authentication", func(t *testing.T) {
		cluster := v1beta1.NewPostgresCluster()
		require.UnmarshalInto(t, &cluster.Spec.Authentication, `{
			rules: [
				{ connection: host, method: scram },
				{ connection: local, method: peer, users: [alice, bob] },
			],
		}`)

		result := reconciler.generatePostgresHBAs(ctx, cluster).AsStrings()
		assert.Assert(t, cmp.Len(result, len(required)+2),
			"expected two rules from the Authentication section and no defaults")

		// mandatory rules should be first
		assert.DeepEqual(t, result[:len(required)], required)

		// specified rules should be last and in their original order
		assert.DeepEqual(t, result[len(required):], []string{
			`"host" all all all "scram"`,
			`"local" all "alice","bob" all "peer"`,
		})
	})

	t.Run("Patroni", func(t *testing.T) {
		cluster := v1beta1.NewPostgresCluster()
		require.UnmarshalInto(t, &cluster.Spec.Patroni, `{
			dynamicConfiguration: {
				postgresql: { pg_hba: [ "first custom", "another" ] },
			},
		}`)

		result := reconciler.generatePostgresHBAs(ctx, cluster).AsStrings()
		assert.Assert(t, cmp.Len(result, len(required)+2),
			"expected two rules from the Patroni section and no defaults")

		// mandatory rules should be first
		assert.DeepEqual(t, result[:len(required)], required)

		// specified rules should be last and in their original order
		assert.DeepEqual(t, result[len(required):], []string{`first custom`, `another`})
	})

	t.Run("Precedence", func(t *testing.T) {
		cluster := v1beta1.NewPostgresCluster()
		require.UnmarshalInto(t, &cluster.Spec.Authentication, `{
			rules: [
				{ connection: host, method: scram },
				{ connection: local, method: peer, users: [alice, bob] },
			],
		}`)
		require.UnmarshalInto(t, &cluster.Spec.Patroni, `{
			dynamicConfiguration: {
				postgresql: { pg_hba: [ "another" ] },
			},
		}`)

		result := reconciler.generatePostgresHBAs(ctx, cluster).AsStrings()
		assert.Assert(t, cmp.Len(result, len(required)+2+1),
			"expected two rules from the Authentication section"+
				" plus one from the Patroni section")

		// mandatory rules should be first
		assert.DeepEqual(t, result[:len(required)], required)

		// specified rules are next, no defaults
		assert.DeepEqual(t, result[len(required):], []string{
			`"host" all all all "scram"`,           // Authentication
			`"local" all "alice","bob" all "peer"`, // Authentication
			`another`,                              // Patroni
		})
	})
}

func TestGeneratePostgresParameters(t *testing.T) {
	ctx := context.Background()
	reconciler := &Reconciler{}

	builtin := reconciler.generatePostgresParameters(ctx, v1beta1.NewPostgresCluster(), false)
	assert.Assert(t, len(builtin.AsMap()) > 0,
		"expected an empty cluster to have some builtin parameters")

	assert.Equal(t, builtin.Value("jit"), "off",
		"BUG IN TEST: expected JIT to be disabled")

	assert.Equal(t, builtin.Value("shared_preload_libraries"), "pgaudit",
		"BUG IN TEST: expected pgAudit to be mandatory")

	t.Run("Config", func(t *testing.T) {
		cluster := v1beta1.NewPostgresCluster()
		require.UnmarshalInto(t, &cluster.Spec.Config, `{
			parameters: {
				something: str,
				another: 5,
			},
		}`)

		result := reconciler.generatePostgresParameters(ctx, cluster, false)
		assert.Assert(t, cmp.LenMap(result.AsMap(), len(builtin.AsMap())+2),
			"expected two parameters from the Config section")

		assert.Equal(t, result.Value("another"), "5")
		assert.Equal(t, result.Value("something"), "str")
	})

	t.Run("Patroni", func(t *testing.T) {
		cluster := v1beta1.NewPostgresCluster()
		require.UnmarshalInto(t, &cluster.Spec.Patroni, `{
			dynamicConfiguration: {
				postgresql: { parameters: {
					something: str,
					another: 5.1,
				} },
			},
		}`)

		result := reconciler.generatePostgresParameters(ctx, cluster, false)
		assert.Assert(t, cmp.LenMap(result.AsMap(), len(builtin.AsMap())+2),
			"expected two parameters from the Patroni section")

		assert.Equal(t, result.Value("another"), "5.1")
		assert.Equal(t, result.Value("something"), "str")
	})

	t.Run("Precedence", func(t *testing.T) {
		cluster := v1beta1.NewPostgresCluster()
		require.UnmarshalInto(t, &cluster.Spec.Config, `{
			parameters: {
				something: replaced,
				unrelated: used,
				jit: "on",
			},
		}`)
		require.UnmarshalInto(t, &cluster.Spec.Patroni, `{
			dynamicConfiguration: {
				postgresql: { parameters: {
					something: str,
					another: 5.1,
				} },
			},
		}`)

		result := reconciler.generatePostgresParameters(ctx, cluster, false)
		assert.Assert(t, cmp.LenMap(result.AsMap(), len(builtin.AsMap())+3+1-1),
			"expected three parameters from the Config section,"+
				"plus one from the Patroni section, minus one default")

		assert.Equal(t, result.Value("another"), "5.1")        // Patroni
		assert.Equal(t, result.Value("something"), "replaced") // Config
		assert.Equal(t, result.Value("unrelated"), "used")     // Config
		assert.Equal(t, result.Value("jit"), "on")             // Config
	})

	t.Run("shared_preload_libraries", func(t *testing.T) {
		t.Run("NumericIncluded", func(t *testing.T) {
			cluster := v1beta1.NewPostgresCluster()
			require.UnmarshalInto(t, &cluster.Spec.Config, `{
				parameters: {
					shared_preload_libraries: 123,
				},
			}`)

			result := reconciler.generatePostgresParameters(ctx, cluster, false)
			assert.Assert(t, cmp.Contains(result.Value("shared_preload_libraries"), "123"))
		})

		t.Run("Precedence", func(t *testing.T) {
			cluster := v1beta1.NewPostgresCluster()
			require.UnmarshalInto(t, &cluster.Spec.Config, `{
				parameters: {
					shared_preload_libraries: given,
				},
			}`)

			result := reconciler.generatePostgresParameters(ctx, cluster, false)
			assert.Equal(t, result.Value("shared_preload_libraries"), "pgaudit,given",
				"expected mandatory ahead of specified")

			require.UnmarshalInto(t, &cluster.Spec.Config, `{
				parameters: {
					shared_preload_libraries: 'given, citus,other'
				},
			}`)

			result = reconciler.generatePostgresParameters(ctx, cluster, false)
			assert.Equal(t, result.Value("shared_preload_libraries"), "citus,pgaudit,given, citus,other",
				"expected citus in front")
		})
	})
}

func TestGeneratePostgresUserSecret(t *testing.T) {
	_, tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	reconciler := &Reconciler{Client: tClient}

	cluster := &v1beta1.PostgresCluster{}
	cluster.Namespace = "ns1"
	cluster.Name = "hippo2"
	cluster.Spec.Port = initialize.Int32(9999)

	spec := &v1beta1.PostgresUserSpec{Name: "some-user-name"}

	t.Run("ObjectMeta", func(t *testing.T) {
		secret, err := reconciler.generatePostgresUserSecret(cluster, spec, nil)
		assert.NilError(t, err)

		if assert.Check(t, secret != nil) {
			assert.Equal(t, secret.Namespace, cluster.Namespace)
			assert.Assert(t, metav1.IsControlledBy(secret, cluster))
			assert.DeepEqual(t, secret.Labels, map[string]string{
				"postgres-operator.crunchydata.com/cluster": "hippo2",
				"postgres-operator.crunchydata.com/role":    "pguser",
				"postgres-operator.crunchydata.com/pguser":  "some-user-name",
			})
		}
	})

	t.Run("Primary", func(t *testing.T) {
		secret, err := reconciler.generatePostgresUserSecret(cluster, spec, nil)
		assert.NilError(t, err)

		if assert.Check(t, secret != nil) {
			assert.Equal(t, string(secret.Data["host"]), "hippo2-primary.ns1.svc")
			assert.Equal(t, string(secret.Data["port"]), "9999")
			assert.Equal(t, string(secret.Data["user"]), "some-user-name")
		}
	})

	t.Run("Password", func(t *testing.T) {
		// Generated when no existing Secret.
		secret, err := reconciler.generatePostgresUserSecret(cluster, spec, nil)
		assert.NilError(t, err)

		if assert.Check(t, secret != nil) {
			assert.Assert(t, len(secret.Data["password"]) > 16, "got %v", len(secret.Data["password"]))
			assert.Assert(t, len(secret.Data["verifier"]) > 90, "got %v", len(secret.Data["verifier"]))
		}

		// Generated when existing Secret is lacking.
		secret, err = reconciler.generatePostgresUserSecret(cluster, spec, new(corev1.Secret))
		assert.NilError(t, err)

		if assert.Check(t, secret != nil) {
			assert.Assert(t, len(secret.Data["password"]) > 16, "got %v", len(secret.Data["password"]))
			assert.Assert(t, len(secret.Data["verifier"]) > 90, "got %v", len(secret.Data["verifier"]))
		}

		t.Run("Policy", func(t *testing.T) {
			spec := spec.DeepCopy()

			// ASCII when unspecified.
			spec.Password = nil
			secret, err = reconciler.generatePostgresUserSecret(cluster, spec, new(corev1.Secret))
			assert.NilError(t, err)

			if assert.Check(t, secret != nil) {
				// This assertion is lacking, but distinguishing between "alphanumeric"
				// and "alphanumeric+symbols" is hard. If our generator changes to
				// guarantee at least one symbol, we can check for symbols here.
				assert.Assert(t, len(secret.Data["password"]) != 0)
			}

			// AlphaNumeric when specified.
			spec.Password = &v1beta1.PostgresPasswordSpec{
				Type: v1beta1.PostgresPasswordTypeAlphaNumeric,
			}

			secret, err = reconciler.generatePostgresUserSecret(cluster, spec, new(corev1.Secret))
			assert.NilError(t, err)

			if assert.Check(t, secret != nil) {
				assert.Assert(t, cmp.Regexp(`^[A-Za-z0-9]+$`, string(secret.Data["password"])))
			}
		})

		// Verifier is generated when existing Secret contains only a password.
		secret, err = reconciler.generatePostgresUserSecret(cluster, spec, &corev1.Secret{
			Data: map[string][]byte{
				"password": []byte(`asdf`),
			},
		})
		assert.NilError(t, err)

		if assert.Check(t, secret != nil) {
			assert.Equal(t, string(secret.Data["password"]), "asdf")
			assert.Assert(t, len(secret.Data["verifier"]) > 90, "got %v", len(secret.Data["verifier"]))
		}

		// Copied when existing Secret is full.
		secret, err = reconciler.generatePostgresUserSecret(cluster, spec, &corev1.Secret{
			Data: map[string][]byte{
				"password": []byte(`asdf`),
				"verifier": []byte(`some$thing`),
			},
		})
		assert.NilError(t, err)

		if assert.Check(t, secret != nil) {
			assert.Equal(t, string(secret.Data["password"]), "asdf")
			assert.Equal(t, string(secret.Data["verifier"]), "some$thing")
		}
	})

	t.Run("Database", func(t *testing.T) {
		spec := *spec

		// Missing when none specified.
		secret, err := reconciler.generatePostgresUserSecret(cluster, &spec, nil)
		assert.NilError(t, err)

		if assert.Check(t, secret != nil) {
			assert.Assert(t, secret.Data["dbname"] == nil)
			assert.Assert(t, secret.Data["uri"] == nil)
			assert.Assert(t, secret.Data["jdbc-uri"] == nil)
		}

		// Present when specified.
		spec.Databases = []string{"db1"}

		secret, err = reconciler.generatePostgresUserSecret(cluster, &spec, nil)
		assert.NilError(t, err)

		if assert.Check(t, secret != nil) {
			assert.Equal(t, string(secret.Data["dbname"]), "db1")
			assert.Assert(t, cmp.Regexp(
				`^postgresql://some-user-name:[^@]+@hippo2-primary.ns1.svc:9999/db1$`,
				string(secret.Data["uri"])))
			assert.Assert(t, cmp.Regexp(
				`^jdbc:postgresql://hippo2-primary.ns1.svc:9999/db1`+
					`[?]password=[^&]+&user=some-user-name$`,
				string(secret.Data["jdbc-uri"])))
		}

		// Only the first in the list.
		spec.Databases = []string{"first", "asdf"}

		secret, err = reconciler.generatePostgresUserSecret(cluster, &spec, nil)
		assert.NilError(t, err)

		if assert.Check(t, secret != nil) {
			assert.Equal(t, string(secret.Data["dbname"]), "first")
			assert.Assert(t, cmp.Regexp(
				`^postgresql://some-user-name:[^@]+@hippo2-primary.ns1.svc:9999/first$`,
				string(secret.Data["uri"])))
			assert.Assert(t, cmp.Regexp(
				`^jdbc:postgresql://hippo2-primary.ns1.svc:9999/first[?].+$`,
				string(secret.Data["jdbc-uri"])))

		}
	})

	t.Run("PgBouncer", func(t *testing.T) {
		require.UnmarshalInto(t, &cluster.Spec, `{
			proxy: { pgBouncer: { port: 10220 } },
		}`)

		secret, err := reconciler.generatePostgresUserSecret(cluster, spec, nil)
		assert.NilError(t, err)

		if assert.Check(t, secret != nil) {
			assert.Equal(t, string(secret.Data["pgbouncer-host"]), "hippo2-pgbouncer.ns1.svc")
			assert.Equal(t, string(secret.Data["pgbouncer-port"]), "10220")
			assert.Assert(t, secret.Data["pgbouncer-uri"] == nil)
			assert.Assert(t, secret.Data["pgbouncer-jdbc-uri"] == nil)
		}

		// Includes a URI when possible.
		spec := *spec
		spec.Databases = []string{"yes", "no"}

		secret, err = reconciler.generatePostgresUserSecret(cluster, &spec, nil)
		assert.NilError(t, err)

		if assert.Check(t, secret != nil) {
			assert.Assert(t, cmp.Regexp(
				`^postgresql://some-user-name:[^@]+@hippo2-pgbouncer.ns1.svc:10220/yes$`,
				string(secret.Data["pgbouncer-uri"])))
			assert.Assert(t, cmp.Regexp(
				`^jdbc:postgresql://hippo2-pgbouncer.ns1.svc:10220/yes`+
					`[?]password=[^&]+&prepareThreshold=0&user=some-user-name$`,
				string(secret.Data["pgbouncer-jdbc-uri"])))
		}
	})
}

func TestReconcilePostgresVolumes(t *testing.T) {
	ctx := context.Background()
	_, tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 1)

	reconciler := &Reconciler{
		Client: tClient,
		Owner:  client.FieldOwner(t.Name()),
	}

	t.Run("DataVolumeNoSourceCluster", func(t *testing.T) {
		cluster := testCluster()
		ns := setupNamespace(t, tClient)
		cluster.Namespace = ns.Name

		assert.NilError(t, tClient.Create(ctx, cluster))
		t.Cleanup(func() { assert.Check(t, tClient.Delete(ctx, cluster)) })

		spec := &v1beta1.PostgresInstanceSetSpec{}
		require.UnmarshalInto(t, spec, `{
			name: "some-instance",
			dataVolumeClaimSpec: {
				accessModes: [ReadWriteOnce],
				resources: { requests: { storage: 1Gi } },
				storageClassName: "storage-class-for-data",
			},
		}`)
		instance := &appsv1.StatefulSet{ObjectMeta: naming.GenerateInstance(cluster, spec)}

		pvc, err := reconciler.reconcilePostgresDataVolume(ctx, cluster, spec, instance, nil, nil)
		assert.NilError(t, err)

		assert.Assert(t, metav1.IsControlledBy(pvc, cluster))

		assert.Equal(t, pvc.Labels[naming.LabelCluster], cluster.Name)
		assert.Equal(t, pvc.Labels[naming.LabelInstance], instance.Name)
		assert.Equal(t, pvc.Labels[naming.LabelInstanceSet], spec.Name)
		assert.Equal(t, pvc.Labels[naming.LabelRole], "pgdata")

		assert.Assert(t, cmp.MarshalMatches(pvc.Spec, `
accessModes:
- ReadWriteOnce
resources:
  requests:
    storage: 1Gi
storageClassName: storage-class-for-data
volumeMode: Filesystem
		`))
	})

	t.Run("DataVolumeSourceClusterWithGoodSnapshot", func(t *testing.T) {
		if strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
			t.Skip("requires mocking of Job conditions")
		}
		cluster := testCluster()
		ns := setupNamespace(t, tClient)
		cluster.Namespace = ns.Name

		assert.NilError(t, tClient.Create(ctx, cluster))
		t.Cleanup(func() { assert.Check(t, tClient.Delete(ctx, cluster)) })

		spec := &v1beta1.PostgresInstanceSetSpec{}
		require.UnmarshalInto(t, spec, `{
			name: "some-instance",
			dataVolumeClaimSpec: {
				accessModes: [ReadWriteOnce],
				resources: { requests: { storage: 1Gi } },
				storageClassName: "storage-class-for-data",
			},
		}`)
		instance := &appsv1.StatefulSet{ObjectMeta: naming.GenerateInstance(cluster, spec)}

		recorder := events.NewRecorder(t, runtime.Scheme)
		reconciler.Recorder = recorder

		// Turn on VolumeSnapshots feature gate
		gate := feature.NewGate()
		assert.NilError(t, gate.SetFromMap(map[string]bool{
			feature.VolumeSnapshots: true,
		}))
		ctx := feature.NewContext(ctx, gate)

		// Create source cluster and enable snapshots
		sourceCluster := testCluster()
		sourceCluster.Namespace = ns.Name
		sourceCluster.Name = "rhino"
		sourceCluster.Spec.Backups.Snapshots = &v1beta1.VolumeSnapshots{
			VolumeSnapshotClassName: "some-class-name",
		}

		// Create a snapshot
		snapshot := &volumesnapshotv1.VolumeSnapshot{
			TypeMeta: metav1.TypeMeta{
				APIVersion: volumesnapshotv1.SchemeGroupVersion.String(),
				Kind:       "VolumeSnapshot",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "some-snapshot",
				Namespace: ns.Name,
				Labels: map[string]string{
					naming.LabelCluster: "rhino",
				},
			},
		}
		snapshot.Spec.Source.PersistentVolumeClaimName = initialize.String("some-pvc-name")
		snapshot.Spec.VolumeSnapshotClassName = initialize.String("some-class-name")
		err := reconciler.apply(ctx, snapshot)
		assert.NilError(t, err)

		// Get snapshot and update Status.ReadyToUse and CreationTime
		err = reconciler.Client.Get(ctx, client.ObjectKeyFromObject(snapshot), snapshot)
		assert.NilError(t, err)

		currentTime := metav1.Now()
		snapshot.Status = &volumesnapshotv1.VolumeSnapshotStatus{
			ReadyToUse:   initialize.Bool(true),
			CreationTime: &currentTime,
		}
		err = reconciler.Client.Status().Update(ctx, snapshot)
		assert.NilError(t, err)

		// Reconcile volume
		pvc, err := reconciler.reconcilePostgresDataVolume(ctx, cluster, spec, instance, nil, sourceCluster)
		assert.NilError(t, err)

		assert.Assert(t, metav1.IsControlledBy(pvc, cluster))

		assert.Equal(t, pvc.Labels[naming.LabelCluster], cluster.Name)
		assert.Equal(t, pvc.Labels[naming.LabelInstance], instance.Name)
		assert.Equal(t, pvc.Labels[naming.LabelInstanceSet], spec.Name)
		assert.Equal(t, pvc.Labels[naming.LabelRole], "pgdata")

		assert.Assert(t, cmp.MarshalMatches(pvc.Spec, `
accessModes:
- ReadWriteOnce
dataSource:
  apiGroup: snapshot.storage.k8s.io
  kind: VolumeSnapshot
  name: some-snapshot
dataSourceRef:
  apiGroup: snapshot.storage.k8s.io
  kind: VolumeSnapshot
  name: some-snapshot
resources:
  requests:
    storage: 1Gi
storageClassName: storage-class-for-data
volumeMode: Filesystem
		`))
		assert.Equal(t, len(recorder.Events), 1)
		assert.Equal(t, recorder.Events[0].Regarding.Name, cluster.Name)
		assert.Equal(t, recorder.Events[0].Reason, "BootstrappingWithSnapshot")
		assert.Equal(t, recorder.Events[0].Note, "Snapshot found for rhino; bootstrapping cluster with snapshot.")
	})

	t.Run("DataVolumeSourceClusterSnapshotsEnabledNoSnapshots", func(t *testing.T) {
		cluster := testCluster()
		ns := setupNamespace(t, tClient)
		cluster.Namespace = ns.Name

		assert.NilError(t, tClient.Create(ctx, cluster))
		t.Cleanup(func() { assert.Check(t, tClient.Delete(ctx, cluster)) })

		spec := &v1beta1.PostgresInstanceSetSpec{}
		require.UnmarshalInto(t, spec, `{
			name: "some-instance",
			dataVolumeClaimSpec: {
				accessModes: [ReadWriteOnce],
				resources: { requests: { storage: 1Gi } },
				storageClassName: "storage-class-for-data",
			},
		}`)
		instance := &appsv1.StatefulSet{ObjectMeta: naming.GenerateInstance(cluster, spec)}

		recorder := events.NewRecorder(t, runtime.Scheme)
		reconciler.Recorder = recorder

		// Turn on VolumeSnapshots feature gate
		gate := feature.NewGate()
		assert.NilError(t, gate.SetFromMap(map[string]bool{
			feature.VolumeSnapshots: true,
		}))
		ctx := feature.NewContext(ctx, gate)

		// Create source cluster and enable snapshots
		sourceCluster := testCluster()
		sourceCluster.Namespace = ns.Name
		sourceCluster.Name = "rhino"
		sourceCluster.Spec.Backups.Snapshots = &v1beta1.VolumeSnapshots{
			VolumeSnapshotClassName: "some-class-name",
		}

		// Reconcile volume
		pvc, err := reconciler.reconcilePostgresDataVolume(ctx, cluster, spec, instance, nil, sourceCluster)
		assert.NilError(t, err)

		assert.Assert(t, metav1.IsControlledBy(pvc, cluster))

		assert.Equal(t, pvc.Labels[naming.LabelCluster], cluster.Name)
		assert.Equal(t, pvc.Labels[naming.LabelInstance], instance.Name)
		assert.Equal(t, pvc.Labels[naming.LabelInstanceSet], spec.Name)
		assert.Equal(t, pvc.Labels[naming.LabelRole], "pgdata")

		assert.Assert(t, cmp.MarshalMatches(pvc.Spec, `
accessModes:
- ReadWriteOnce
resources:
  requests:
    storage: 1Gi
storageClassName: storage-class-for-data
volumeMode: Filesystem
		`))
		assert.Equal(t, len(recorder.Events), 1)
		assert.Equal(t, recorder.Events[0].Regarding.Name, cluster.Name)
		assert.Equal(t, recorder.Events[0].Reason, "SnapshotNotFound")
		assert.Equal(t, recorder.Events[0].Note, "No ReadyToUse snapshots were found for rhino; proceeding with typical restore process.")
	})

	t.Run("WALVolume", func(t *testing.T) {
		cluster := testCluster()
		ns := setupNamespace(t, tClient)
		cluster.Namespace = ns.Name

		assert.NilError(t, tClient.Create(ctx, cluster))
		t.Cleanup(func() { assert.Check(t, tClient.Delete(ctx, cluster)) })

		spec := &v1beta1.PostgresInstanceSetSpec{}
		require.UnmarshalInto(t, spec, `{
			name: "some-instance",
			dataVolumeClaimSpec: {
				accessModes: [ReadWriteOnce],
				resources: { requests: { storage: 1Gi } },
				storageClassName: "storage-class-for-data",
			},
		}`)
		instance := &appsv1.StatefulSet{ObjectMeta: naming.GenerateInstance(cluster, spec)}

		observed := &Instance{}

		t.Run("None", func(t *testing.T) {
			pvc, err := reconciler.reconcilePostgresWALVolume(ctx, cluster, spec, instance, observed, nil)
			assert.NilError(t, err)
			assert.Assert(t, pvc == nil)
		})

		t.Run("Specified", func(t *testing.T) {
			spec := spec.DeepCopy()
			require.UnmarshalInto(t, spec, `{
				walVolumeClaimSpec: {
					accessModes: [ReadWriteMany],
					resources: { requests: { storage: 2Gi } },
					storageClassName: "storage-class-for-wal",
				},
			}`)

			pvc, err := reconciler.reconcilePostgresWALVolume(ctx, cluster, spec, instance, observed, nil)
			assert.NilError(t, err)

			assert.Assert(t, metav1.IsControlledBy(pvc, cluster))

			assert.Equal(t, pvc.Labels[naming.LabelCluster], cluster.Name)
			assert.Equal(t, pvc.Labels[naming.LabelInstance], instance.Name)
			assert.Equal(t, pvc.Labels[naming.LabelInstanceSet], spec.Name)
			assert.Equal(t, pvc.Labels[naming.LabelRole], "pgwal")

			assert.Assert(t, cmp.MarshalMatches(pvc.Spec, `
accessModes:
- ReadWriteMany
resources:
  requests:
    storage: 2Gi
storageClassName: storage-class-for-wal
volumeMode: Filesystem
			`))

			t.Run("Removed", func(t *testing.T) {
				spec := spec.DeepCopy()
				spec.WALVolumeClaimSpec = nil

				ignoreTypeMeta := cmpopts.IgnoreFields(corev1.PersistentVolumeClaim{}, "TypeMeta")

				t.Run("FilesAreNotSafe", func(t *testing.T) {
					// No pods; expect no changes to the PVC.
					returned, err := reconciler.reconcilePostgresWALVolume(ctx, cluster, spec, instance, observed, nil)
					assert.NilError(t, err)
					assert.DeepEqual(t, returned, pvc, ignoreTypeMeta)

					// Not running; expect no changes to the PVC.
					observed.Pods = []*corev1.Pod{{}}

					returned, err = reconciler.reconcilePostgresWALVolume(ctx, cluster, spec, instance, observed, nil)
					assert.NilError(t, err)
					assert.DeepEqual(t, returned, pvc, ignoreTypeMeta)

					// Cannot find WAL files; expect no changes to the PVC.
					observed.Pods[0].Namespace, observed.Pods[0].Name = "pod-ns", "pod-name"
					observed.Pods[0].Status.ContainerStatuses = []corev1.ContainerStatus{{
						Name: naming.ContainerDatabase,
					}}
					observed.Pods[0].Status.ContainerStatuses[0].State.Running =
						new(corev1.ContainerStateRunning)

					expected := errors.New("flop")
					reconciler.PodExec = func(
						ctx context.Context, namespace, pod, container string,
						_ io.Reader, _, _ io.Writer, command ...string,
					) error {
						assert.Equal(t, namespace, "pod-ns")
						assert.Equal(t, pod, "pod-name")
						assert.Equal(t, container, "database")
						assert.DeepEqual(t, command,
							[]string{"bash", "-ceu", "--", `exec realpath "${PGDATA}/pg_wal"`})
						return expected
					}

					returned, err = reconciler.reconcilePostgresWALVolume(ctx, cluster, spec, instance, observed, nil)
					assert.Equal(t, expected, errors.Unwrap(err), "expected pod exec")
					assert.DeepEqual(t, returned, pvc, ignoreTypeMeta)

					// Files are in the wrong place; expect no changes to the PVC.
					reconciler.PodExec = func(
						ctx context.Context, _, _, _ string, _ io.Reader, stdout, _ io.Writer, _ ...string,
					) error {
						assert.Assert(t, stdout != nil)
						_, err := stdout.Write([]byte("some-place\n"))
						assert.NilError(t, err)
						return nil
					}

					returned, err = reconciler.reconcilePostgresWALVolume(ctx, cluster, spec, instance, observed, nil)
					assert.NilError(t, err)
					assert.DeepEqual(t, returned, pvc, ignoreTypeMeta)
				})

				t.Run("FilesAreSafe", func(t *testing.T) {
					// Files are seen in the directory intended by the specification.
					observed.Pods = []*corev1.Pod{{}}
					observed.Pods[0].Status.ContainerStatuses = []corev1.ContainerStatus{{
						Name: naming.ContainerDatabase,
					}}
					observed.Pods[0].Status.ContainerStatuses[0].State.Running =
						new(corev1.ContainerStateRunning)

					reconciler.PodExec = func(
						ctx context.Context, _, _, _ string, _ io.Reader, stdout, _ io.Writer, _ ...string,
					) error {
						assert.Assert(t, stdout != nil)
						_, err := stdout.Write([]byte(postgres.WALDirectory(cluster, spec) + "\n"))
						assert.NilError(t, err)
						return nil
					}

					returned, err := reconciler.reconcilePostgresWALVolume(ctx, cluster, spec, instance, observed, nil)
					assert.NilError(t, err)
					assert.Assert(t, returned == nil)

					key, fetched := client.ObjectKeyFromObject(pvc), &corev1.PersistentVolumeClaim{}
					if err := tClient.Get(ctx, key, fetched); err == nil {
						assert.Assert(t, fetched.DeletionTimestamp != nil, "expected deleted")
					} else {
						assert.Assert(t, apierrors.IsNotFound(err), "expected NotFound, got %v", err)
					}

					// Pods will redeploy while the PVC is scheduled for deletion.
					observed.Pods = nil
					returned, err = reconciler.reconcilePostgresWALVolume(ctx, cluster, spec, instance, observed, nil)
					assert.NilError(t, err)
					assert.Assert(t, returned == nil)
				})
			})
		})
	})
}

func TestReconcileDatabaseInitSQL(t *testing.T) {
	ctx := context.Background()
	var called bool

	// Test Environment Setup
	_, client := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	r := &Reconciler{
		Client: client,

		// Overwrite the PodExec function with a check to ensure the exec
		// call would have been made
		PodExec: func(ctx context.Context, namespace, pod, container string, stdin io.Reader,
			stdout, stderr io.Writer, command ...string) error {
			called = true
			return nil
		},
	}

	// Test Resources Setup
	ns := setupNamespace(t, client)

	// Define a status to be set if sql has already been run
	status := "set"

	// reconcileDatabaseInitSQL expects to find a pod that is running with a
	// writable database container. Define this pod in an observed instance so
	// we can simulate a podExec call into the database
	instances := []*Instance{
		{
			Name: "instance",
			Pods: []*corev1.Pod{{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns.Name,
					Name:      "pod",
					Annotations: map[string]string{
						"status": `{"role":"primary"}`,
					},
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{{
						Name: naming.ContainerDatabase,
						State: corev1.ContainerState{
							Running: new(corev1.ContainerStateRunning),
						},
					}},
				},
			}},
			Runner: &appsv1.StatefulSet{},
		},
	}
	observed := &observedInstances{forCluster: instances}

	// Create a ConfigMap containing SQL to be defined in the spec
	path := "test-path"
	cm := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cm",
			Namespace: ns.Name,
		},
		Data: map[string]string{
			path: "stuff",
		},
	}
	assert.NilError(t, client.Create(ctx, cm.DeepCopy()))

	// Define a fully configured cluster that would lead to SQL being run in
	// the database. This test cluster will be modified as needed for testing
	testCluster := testCluster()
	testCluster.Namespace = ns.Name
	testCluster.Spec.DatabaseInitSQL = &v1beta1.DatabaseInitSQL{
		Name: cm.Name,
		Key:  path,
	}

	// Start Tests
	t.Run("not defined", func(t *testing.T) {
		// Custom SQL is not defined in the spec and status is unset
		cluster := testCluster.DeepCopy()
		cluster.Spec.DatabaseInitSQL = nil

		assert.NilError(t, r.reconcileDatabaseInitSQL(ctx, cluster, observed))
		assert.Assert(t, !called, "PodExec should not have been called")
		assert.Assert(t, cluster.Status.DatabaseInitSQL == nil, "Status should not be set")
	})
	t.Run("not defined with status", func(t *testing.T) {
		// Custom SQL is not defined in the spec and status is set
		cluster := testCluster.DeepCopy()
		cluster.Spec.DatabaseInitSQL = nil
		cluster.Status.DatabaseInitSQL = &status

		assert.NilError(t, r.reconcileDatabaseInitSQL(ctx, cluster, observed))
		assert.Assert(t, !called, "PodExec should not have been called")
		assert.Assert(t, cluster.Status.DatabaseInitSQL == nil, "Status was set and should have been removed")
	})
	t.Run("status set", func(t *testing.T) {
		// Custom SQL is defined and status is set
		cluster := testCluster.DeepCopy()
		cluster.Status.DatabaseInitSQL = &status

		assert.NilError(t, r.reconcileDatabaseInitSQL(ctx, cluster, observed))
		assert.Assert(t, !called, "PodExec should  not have been called")
		assert.Equal(t, cluster.Status.DatabaseInitSQL, &status, "Status should not have changed")
	})
	t.Run("No writable pod", func(t *testing.T) {
		cluster := testCluster.DeepCopy()

		assert.NilError(t, r.reconcileDatabaseInitSQL(ctx, cluster, nil))
		assert.Assert(t, !called, "PodExec should not have been called")
		assert.Assert(t, cluster.Status.DatabaseInitSQL == nil, "SQL couldn't be executed so status should be unset")
	})
	t.Run("Fully Configured", func(t *testing.T) {
		cluster := testCluster.DeepCopy()

		assert.NilError(t, r.reconcileDatabaseInitSQL(ctx, cluster, observed))
		assert.Assert(t, called, "PodExec should be called")
		assert.Equal(t,
			*cluster.Status.DatabaseInitSQL,
			cluster.Spec.DatabaseInitSQL.Name,
			"Status should be set to the custom configmap name")
	})
}

func TestReconcileDatabaseInitSQLConfigMap(t *testing.T) {
	ctx := context.Background()
	var called bool

	// Test Environment Setup
	_, client := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	r := &Reconciler{
		Client: client,

		// Overwrite the PodExec function with a check to ensure the exec
		// call would have been made
		PodExec: func(ctx context.Context, namespace, pod, container string, stdin io.Reader,
			stdout, stderr io.Writer, command ...string) error {
			called = true
			return nil
		},
	}

	// Test Resources Setup
	ns := setupNamespace(t, client)

	// reconcileDatabaseInitSQL expects to find a pod that is running with a writable
	// database container. Define this pod in an observed instance so that
	// we can simulate a podExec call into the database
	instances := []*Instance{
		{
			Name: "instance",
			Pods: []*corev1.Pod{{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns.Name,
					Name:      "pod",
					Annotations: map[string]string{
						"status": `{"role":"primary"}`,
					},
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{{
						Name: naming.ContainerDatabase,
						State: corev1.ContainerState{
							Running: new(corev1.ContainerStateRunning),
						},
					}},
				},
			}},
			Runner: &appsv1.StatefulSet{},
		},
	}
	observed := &observedInstances{forCluster: instances}

	// Define fully configured cluster that would lead to sql being run in the
	// database. This cluster will be modified for testing
	testCluster := testCluster()
	testCluster.Namespace = ns.Name
	testCluster.Spec.DatabaseInitSQL = new(v1beta1.DatabaseInitSQL)

	t.Run("not found", func(t *testing.T) {
		cluster := testCluster.DeepCopy()
		cluster.Spec.DatabaseInitSQL = &v1beta1.DatabaseInitSQL{
			Name: "not-found",
		}

		err := r.reconcileDatabaseInitSQL(ctx, cluster, observed)
		assert.Assert(t, apierrors.IsNotFound(err), err)
		assert.Assert(t, !called)
	})

	t.Run("found no data", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "found-no-data",
				Namespace: ns.Name,
			},
		}
		assert.NilError(t, client.Create(ctx, cm))

		cluster := testCluster.DeepCopy()
		cluster.Spec.DatabaseInitSQL = &v1beta1.DatabaseInitSQL{
			Name: cm.Name,
			Key:  "bad-path",
		}

		err := r.reconcileDatabaseInitSQL(ctx, cluster, observed)
		assert.Equal(t, err.Error(), "ConfigMap did not contain expected key: bad-path")
		assert.Assert(t, !called)
	})

	t.Run("found with data", func(t *testing.T) {
		path := "test-path"

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "found-with-data",
				Namespace: ns.Name,
			},
			Data: map[string]string{
				path: "string",
			},
		}
		assert.NilError(t, client.Create(ctx, cm))

		cluster := testCluster.DeepCopy()
		cluster.Spec.DatabaseInitSQL = &v1beta1.DatabaseInitSQL{
			Name: cm.Name,
			Key:  path,
		}

		assert.NilError(t, r.reconcileDatabaseInitSQL(ctx, cluster, observed))
		assert.Assert(t, called)
	})
}

func TestValidatePostgresUsers(t *testing.T) {
	t.Parallel()

	t.Run("Empty", func(t *testing.T) {
		cluster := v1beta1.NewPostgresCluster()
		recorder := events.NewRecorder(t, runtime.Scheme)
		reconciler := &Reconciler{Recorder: recorder}

		cluster.Spec.Users = nil
		reconciler.validatePostgresUsers(cluster)
		assert.Equal(t, len(recorder.Events), 0)

		cluster.Spec.Users = []v1beta1.PostgresUserSpec{}
		reconciler.validatePostgresUsers(cluster)
		assert.Equal(t, len(recorder.Events), 0)
	})

	// See [internal/testing/validation.TestPostgresUserOptions]

	t.Run("NoComments", func(t *testing.T) {
		cluster := v1beta1.NewPostgresCluster()
		cluster.Name = "pg1"
		cluster.Spec.Users = []v1beta1.PostgresUserSpec{
			{Name: "dashes", Options: "ANY -- comment"},
			{Name: "block-open", Options: "/* asdf"},
			{Name: "block-close", Options: " qw */ rt"},
		}

		recorder := events.NewRecorder(t, runtime.Scheme)
		reconciler := &Reconciler{Recorder: recorder}

		reconciler.validatePostgresUsers(cluster)
		assert.Equal(t, len(recorder.Events), 3)

		for i, event := range recorder.Events {
			assert.Equal(t, event.Regarding.Name, cluster.Name)
			assert.Equal(t, event.Reason, "InvalidUser")
			assert.Assert(t, cmp.Contains(event.Note, "cannot contain comments"))
			assert.Assert(t, cmp.Contains(event.Note,
				fmt.Sprintf("spec.users[%d].options", i)))
		}
	})

	t.Run("NoPassword", func(t *testing.T) {
		cluster := v1beta1.NewPostgresCluster()
		cluster.Name = "pg5"
		cluster.Spec.Users = []v1beta1.PostgresUserSpec{
			{Name: "uppercase", Options: "SUPERUSER PASSWORD ''"},
			{Name: "lowercase", Options: "password 'asdf'"},
		}

		recorder := events.NewRecorder(t, runtime.Scheme)
		reconciler := &Reconciler{Recorder: recorder}

		reconciler.validatePostgresUsers(cluster)
		assert.Equal(t, len(recorder.Events), 2)

		for i, event := range recorder.Events {
			assert.Equal(t, event.Regarding.Name, cluster.Name)
			assert.Equal(t, event.Reason, "InvalidUser")
			assert.Assert(t, cmp.Contains(event.Note, "cannot assign password"))
			assert.Assert(t, cmp.Contains(event.Note,
				fmt.Sprintf("spec.users[%d].options", i)))
		}
	})

	t.Run("Valid", func(t *testing.T) {
		cluster := v1beta1.NewPostgresCluster()
		cluster.Spec.Users = []v1beta1.PostgresUserSpec{
			{Name: "normal", Options: "CREATEDB valid until '2006-01-02'"},
			{Name: "very-full", Options: "NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOLOGIN NOREPLICATION NOBYPASSRLS CONNECTION LIMIT 5"},
		}

		reconciler := &Reconciler{}
		assert.Assert(t, reconciler.Recorder == nil,
			"expected the following to not use a Recorder at all")

		reconciler.validatePostgresUsers(cluster)
	})
}
