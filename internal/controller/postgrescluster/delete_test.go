//go:build envtest
// +build envtest

/*
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package postgrescluster

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestReconcilerHandleDeleteNamespace(t *testing.T) {
	if !strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
		t.Skip("requires a running garbage collection controller")
	}

	ctx := context.Background()
	env, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 2)

	ns := setupNamespace(t, cc)

	var mm struct {
		manager.Manager
		Context context.Context
		Error   chan error
		Stop    context.CancelFunc
	}

	var err error
	mm.Context, mm.Stop = context.WithCancel(context.Background())
	mm.Error = make(chan error, 1)
	mm.Manager, err = manager.New(env.Config, manager.Options{
		Namespace: ns.Name,
		Scheme:    cc.Scheme(),

		HealthProbeBindAddress: "0", // disable
		MetricsBindAddress:     "0", // disable
	})
	assert.NilError(t, err)

	reconciler := Reconciler{
		Client:   mm.GetClient(),
		Owner:    client.FieldOwner(t.Name()),
		Recorder: new(record.FakeRecorder),
		Tracer:   otel.Tracer(t.Name()),
	}
	assert.NilError(t, reconciler.SetupWithManager(mm.Manager))

	go func() { mm.Error <- mm.Start(mm.Context) }()
	t.Cleanup(func() { mm.Stop(); assert.Check(t, <-mm.Error) })

	cluster := &v1beta1.PostgresCluster{}
	assert.NilError(t, yaml.Unmarshal([]byte(`{
		spec: {
			postgresVersion: 13,
			instances: [
				{
					replicas: 2,
					dataVolumeClaimSpec: {
						accessModes: [ReadWriteOnce],
						resources: { requests: { storage: 1Gi } },
					},
				},
			],
			backups: { 
				pgbackrest: {
					repos: [{
						name: repo1,
						volume: {
							volumeClaimSpec: {
								accessModes: [ReadWriteOnce],
								resources: { requests: { storage: 1Gi } },
							},
						},
					}],
				},
			},
		},
	}`), cluster))

	cluster.Namespace = ns.Name
	cluster.Name = strings.ToLower("DeleteNamespace")
	cluster.Spec.Image = CrunchyPostgresHAImage
	cluster.Spec.Backups.PGBackRest.Image = CrunchyPGBackRestImage

	assert.NilError(t, cc.Create(ctx, cluster))

	t.Cleanup(func() {
		// Remove finalizers, if any, so the namespace can terminate.
		assert.Check(t, client.IgnoreNotFound(
			cc.Patch(ctx, cluster, client.RawPatch(
				client.Merge.Type(), []byte(`{"metadata":{"finalizers":[]}}`)))))
	})

	// Wait until instances are healthy.
	if ready := int32(0); !assert.Check(t,
		wait.Poll(time.Second, Scale(time.Minute), func() (bool, error) {
			assert.NilError(t, cc.Get(ctx, client.ObjectKeyFromObject(cluster), cluster))

			ready = 0
			for _, set := range cluster.Status.InstanceSets {
				ready += set.ReadyReplicas
			}
			return ready >= 2, nil
		}), "expected 2 instances to be ready, got: %v", ready,
	) {
		t.FailNow()
	}

	// Delete the namespace.
	assert.NilError(t, cc.Delete(ctx, ns))

	assert.NilError(t, wait.PollImmediate(time.Second, Scale(time.Minute), func() (bool, error) {
		err := cc.Get(ctx, client.ObjectKeyFromObject(cluster), cluster)
		return apierrors.IsNotFound(err), client.IgnoreNotFound(err)
	}), "expected cluster to be deleted, got:\n%+v", *cluster)

	// Kubernetes will continue to remove things after the PostgresCluster is gone.
	// In some cases, a Pod might get stuck in a deleted-and-creating state.
	// Conditions in the Namespace status indicate what is going on.
	var namespace corev1.Namespace
	assert.NilError(t, wait.PollImmediate(time.Second, Scale(3*time.Minute), func() (bool, error) {
		err := cc.Get(ctx, client.ObjectKeyFromObject(ns), &namespace)
		return apierrors.IsNotFound(err), client.IgnoreNotFound(err)
	}), "expected namespace to be deleted, got status:\n%+v", namespace.Status)
}
