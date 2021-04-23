// +build envtest

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

package postgrescluster

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	CrunchyPostgresHAImage = "gcr.io/crunchy-dev-test/crunchy-postgres-ha:centos8-12.6-multi.dev2"
	CrunchyPGBackRestImage = "gcr.io/crunchy-dev-test/crunchy-pgbackrest:centos8-12.6-multi.dev2"
)

// setupTestEnv configures and starts an EnvTest instance of etcd and the Kubernetes API server
// for test usage, as well as creates a new client instance.
func setupTestEnv(t *testing.T,
	controllerName string) (*envtest.Environment, client.Client, *rest.Config) {

	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
	}
	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Test environment started")

	pgoScheme, err := runtime.CreatePostgresOperatorScheme()
	if err != nil {
		t.Fatal(err)
	}
	client, err := client.New(cfg, client.Options{Scheme: pgoScheme})
	if err != nil {
		t.Fatal(err)
	}

	return testEnv, client, cfg
}

// setupManager creates the runtime manager used during controller testing
func setupManager(t *testing.T, cfg *rest.Config,
	contollerSetup func(mgr manager.Manager)) (context.Context, context.CancelFunc) {

	mgr, err := runtime.CreateRuntimeManager("", cfg, true)
	if err != nil {
		t.Fatal(err)
	}

	contollerSetup(mgr)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := mgr.Start(ctx); err != nil {
			t.Error(err)
		}
	}()
	t.Log("Manager started")

	return ctx, cancel
}

// teardownTestEnv stops the test environment when the tests
// have completed
func teardownTestEnv(t *testing.T, testEnv *envtest.Environment) {
	if err := testEnv.Stop(); err != nil {
		t.Error(err)
	}
	t.Log("Test environment stopped")
}

// teardownManager stops the runtimem manager when the tests
// have completed
func teardownManager(cancel context.CancelFunc, t *testing.T) {
	cancel()
	t.Log("Manager stopped")
}
