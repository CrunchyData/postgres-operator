package upgradecheck

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

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wojas/genericr"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/version"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

type fakeClientWithError struct {
	crclient.Client
	errorType string
}

func (f *fakeClientWithError) Get(ctx context.Context, key types.NamespacedName, obj crclient.Object) error {
	switch f.errorType {
	case "get error":
		return fmt.Errorf("get error")
	default:
		return f.Client.Get(ctx, key, obj)
	}
}

// TODO: PatchType is not supported currently by fake
// - https://github.com/kubernetes/client-go/issues/970
// Once that gets fixed, we can test without envtest
func (f *fakeClientWithError) Patch(ctx context.Context, obj crclient.Object,
	patch crclient.Patch, opts ...crclient.PatchOption) error {
	switch {
	case f.errorType == "patch error":
		return fmt.Errorf("patch error")
	default:
		return f.Client.Patch(ctx, obj, patch, opts...)
	}
}

func (f *fakeClientWithError) List(ctx context.Context, objList crclient.ObjectList,
	opts ...crclient.ListOption) error {
	switch f.errorType {
	case "list error":
		return fmt.Errorf("list error")
	default:
		return f.Client.List(ctx, objList, opts...)
	}
}

func setupDeploymentID(t *testing.T) string {
	t.Helper()
	deploymentID = string(uuid.NewUUID())
	return deploymentID
}

func setupFakeClientWithPGOScheme(t *testing.T, includeCluster bool) crclient.Client {
	t.Helper()
	pgoScheme, err := runtime.CreatePostgresOperatorScheme()
	if err != nil {
		t.Fatal(err)
	}
	if includeCluster {
		pc := &v1beta1.PostgresClusterList{
			Items: []v1beta1.PostgresCluster{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "hippo",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "elephant",
					},
				},
			},
		}
		return fake.NewClientBuilder().WithScheme(pgoScheme).WithLists(pc).Build()
	}
	return fake.NewClientBuilder().WithScheme(pgoScheme).Build()
}

func setupVersionServer(t *testing.T, works bool) (version.Info, *httptest.Server) {
	t.Helper()
	expect := version.Info{
		Major:     "1",
		Minor:     "22",
		GitCommit: "v1.22.2",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,
		req *http.Request) {
		if works {
			output, _ := json.Marshal(expect)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// We don't need to check the error output from this
			_, _ = w.Write(output)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)

	return expect, server
}

func setupLogCapture(ctx context.Context) (context.Context, *[]string) {
	calls := []string{}
	testlog := genericr.New(func(input genericr.Entry) {
		calls = append(calls, input.Message)
	})
	return logging.NewContext(ctx, testlog), &calls
}

// setupNamespace creates a namespace that will be deleted by t.Cleanup.
// For upgradechecking, this namespace is set to `postgres-operator`,
// which sometimes is created by other parts of the testing apparatus,
// cf., the createnamespace call in `make check-envtest-existing`.
// When creation fails, it calls t.Fatal. The caller may delete the namespace
// at any time.
func setupNamespace(t testing.TB, cc crclient.Client) {
	t.Helper()
	ns := &corev1.Namespace{}
	ns.Name = "postgres-operator"
	ns.Labels = map[string]string{"postgres-operator-test": t.Name()}

	ctx := context.Background()
	exists := &corev1.Namespace{}
	assert.NilError(t, crclient.IgnoreNotFound(
		cc.Get(ctx, crclient.ObjectKeyFromObject(ns), exists)))
	if exists.Name != "" {
		return
	}
	assert.NilError(t, cc.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, crclient.IgnoreNotFound(cc.Delete(ctx, ns))) })
}
