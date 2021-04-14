// +build envtest

package postgrescluster

/*
 Copyright 2021 Crunchy Data Solutions, Inc.
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

import (
	"testing"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/util"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
	"go.opentelemetry.io/otel"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func TestPatroniAuthSecret(t *testing.T) {
	// setup the test environment and ensure a clean teardown
	tEnv, tClient, cfg := setupTestEnv(t, ControllerName)

	testScheme := runtime.NewScheme()
	scheme.AddToScheme(testScheme)
	v1beta1.AddToScheme(testScheme)

	// set up a non-cached client
	newClient, err := client.New(cfg, client.Options{Scheme: testScheme})
	assert.NilError(t, err)

	r := &Reconciler{}
	ctx, cancel := setupManager(t, cfg, func(mgr manager.Manager) {
		r = &Reconciler{
			Client:   newClient,
			Recorder: mgr.GetEventRecorderFor(ControllerName),
			Tracer:   otel.Tracer(ControllerName),
			Owner:    ControllerName,
		}
	})
	t.Cleanup(func() {
		teardownManager(cancel, t)
		teardownTestEnv(t, tEnv)
	})

	// test postgrescluster values
	var (
		clusterName        = "hippocluster"
		namespace          = "postgres-operator-test-" + rand.String(6)
		clusterUID         = types.UID("hippouid")
		maintainedPassword string
	)

	ns := &corev1.Namespace{}
	ns.Name = namespace
	assert.NilError(t, tClient.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, tClient.Delete(ctx, ns)) })

	// create a PostgresCluster to test with
	postgresCluster := &v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: namespace,
			UID:       clusterUID,
		},
	}

	t.Run("reconcile", func(t *testing.T) {
		_, err := r.reconcilePatroniAuthSecret(ctx, postgresCluster)
		assert.NilError(t, err)
	})

	t.Run("validate", func(t *testing.T) {

		patroniAuthSecret := &corev1.Secret{ObjectMeta: naming.PatroniAuthSecret(postgresCluster)}
		patroniAuthSecret.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))
		err = r.Client.Get(ctx, client.ObjectKeyFromObject(patroniAuthSecret), patroniAuthSecret)
		assert.NilError(t, err)

		t.Run("yaml", func(t *testing.T) {
			// Check that the patroni.yaml key has been created in the secret
			// Contents of the yaml are tested TODO (jmckulk): where?
			_, ok := patroniAuthSecret.Data["patroni.yaml"]
			assert.Assert(t, ok)
		})

		t.Run("password", func(t *testing.T) {
			// Password is stored as a string field in the secret
			password, ok := patroniAuthSecret.Data["password"]
			assert.Assert(t, ok)
			assert.Equal(t, len(password), util.DefaultGeneratedPasswordLength)

			// Keeps password to ensure that it is unchanged after second reconcile
			maintainedPassword = string(password)
		})
	})

	t.Run("maintained after reconcile", func(t *testing.T) {
		_, err := r.reconcilePatroniAuthSecret(ctx, postgresCluster)
		assert.NilError(t, err)

		updatedPatroniAuthSecret := &corev1.Secret{ObjectMeta: naming.PatroniAuthSecret(postgresCluster)}
		updatedPatroniAuthSecret.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))
		err = r.Client.Get(ctx, client.ObjectKeyFromObject(updatedPatroniAuthSecret), updatedPatroniAuthSecret)
		assert.NilError(t, err)

		t.Run("password", func(t *testing.T) {
			password, ok := updatedPatroniAuthSecret.Data["password"]
			assert.Assert(t, ok)
			assert.Equal(t, maintainedPassword, string(password))
		})

	})
}
