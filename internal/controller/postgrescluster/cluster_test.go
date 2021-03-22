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
	"fmt"
	"net/url"
	"testing"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/util"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
	"go.opentelemetry.io/otel"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func TestReconcilePGUserSecret(t *testing.T) {

	// setup the test environment and ensure a clean teardown
	tEnv, tClient, cfg := setupTestEnv(t, ControllerName)

	testScheme := runtime.NewScheme()
	scheme.AddToScheme(testScheme)
	v1alpha1.AddToScheme(testScheme)

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
		postgresPort             = int32(5432)
		clusterName              = "hippocluster"
		namespace                = "postgres-operator-test-" + rand.String(6)
		clusterUID               = types.UID("hippouid")
		returnedConnectionString string
	)

	ns := &corev1.Namespace{}
	ns.Name = namespace
	assert.NilError(t, tClient.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, tClient.Delete(ctx, ns)) })

	// create a PostgresCluster to test with
	postgresCluster := &v1alpha1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: namespace,
			UID:       clusterUID,
		},
		Spec: v1alpha1.PostgresClusterSpec{
			Port: &postgresPort,
		},
	}

	t.Run("create postgres user secret", func(t *testing.T) {

		_, err := r.reconcilePGUserSecret(ctx, postgresCluster)
		assert.NilError(t, err)
	})

	t.Run("validate postgres user secret", func(t *testing.T) {

		pgUserSecret := &v1.Secret{ObjectMeta: naming.PostgresUserSecret(postgresCluster)}
		pgUserSecret.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Secret"))
		err = r.Client.Get(ctx, client.ObjectKeyFromObject(pgUserSecret), pgUserSecret)
		assert.NilError(t, err)

		databasename, ok := pgUserSecret.Data["dbname"]
		assert.Assert(t, ok)
		assert.Equal(t, string(databasename), postgresCluster.Name)

		host, ok := pgUserSecret.Data["host"]
		assert.Assert(t, ok)
		assert.Equal(t, string(host), fmt.Sprintf("%s.%s.svc", "hippocluster-primary", namespace))

		port, ok := pgUserSecret.Data["port"]
		assert.Assert(t, ok)
		assert.Equal(t, string(port), fmt.Sprintf("%d", *postgresCluster.Spec.Port))

		username, ok := pgUserSecret.Data["user"]
		assert.Assert(t, ok)
		assert.Equal(t, string(username), postgresCluster.Name)

		password, ok := pgUserSecret.Data["password"]
		assert.Assert(t, ok)
		assert.Equal(t, len(password), util.DefaultGeneratedPasswordLength)

		secretConnectionString1, ok := pgUserSecret.Data["uri"]

		assert.Assert(t, ok)

		returnedConnectionString = string(secretConnectionString1)

		testConnectionString := (&url.URL{
			Scheme: "postgresql",
			Host:   fmt.Sprintf("%s.%s.svc:%d", "hippocluster-primary", namespace, *postgresCluster.Spec.Port),
			User:   url.UserPassword(string(pgUserSecret.Data["user"]), string(pgUserSecret.Data["password"])),
			Path:   string(pgUserSecret.Data["dbname"]),
		}).String()

		assert.Equal(t, returnedConnectionString, testConnectionString)
	})

	t.Run("validate postgres user password is not changed after another reconcile", func(t *testing.T) {

		pgUserSecret2, err := r.reconcilePGUserSecret(ctx, postgresCluster)
		assert.NilError(t, err)

		returnedConnectionString2, ok := pgUserSecret2.Data["uri"]

		assert.Assert(t, ok)
		assert.Equal(t, string(returnedConnectionString2), returnedConnectionString)

	})

}
