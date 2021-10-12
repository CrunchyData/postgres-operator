//go:build envtest
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

package pgbackrest

import (
	"path/filepath"
	"testing"

	corev1 "k8s.io/api/core/v1"

	// Google Kubernetes Engine / Google Cloud Platform authentication provider
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
)

// Testing namespace and postgrescluster name
const (
	testclustername = "testcluster"
	testFieldOwner  = "pgbackrestConfigTestFieldOwner"
)

// getCMData returns the 'Data' content from the specified configmap
func getCMData(cm corev1.ConfigMap, key string) string {

	return cm.Data[key]
}

// simpleMarshalContains takes in a YAML object and checks whether
// it includes the expected string
func simpleMarshalContains(actual interface{}, expected string) bool {
	b, err := yaml.Marshal(actual)

	if err != nil {
		return false
	}

	if string(b) == expected {
		return true
	}
	return false
}

// setupTestEnv configures and starts an EnvTest instance of etcd and the Kubernetes API server
// for test usage, as well as creates a new client instance.
func setupTestEnv(t *testing.T) (*envtest.Environment, client.Client) {

	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
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

	return testEnv, client
}

// teardownTestEnv stops the test environment when the tests
// have completed
func teardownTestEnv(t *testing.T, testEnv *envtest.Environment) {
	if err := testEnv.Stop(); err != nil {
		t.Error(err)
	}
	t.Log("Test environment stopped")
}
