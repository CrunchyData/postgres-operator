/*
 Copyright 2021 - 2023 Crunchy Data Solutions, Inc.
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

package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
)

func TestExtractSecretContract(t *testing.T) {
	// We expect ExtractSecret to populate GVK, Namespace, and Name.

	t.Run("GVK", func(t *testing.T) {
		empty := &corev1.Secret{}

		extracted, err := corev1apply.ExtractSecret(empty, "")
		assert.NilError(t, err)

		if assert.Check(t, extracted.APIVersion != nil) {
			assert.Equal(t, *extracted.APIVersion, "v1")
		}
		if assert.Check(t, extracted.Kind != nil) {
			assert.Equal(t, *extracted.Kind, "Secret")
		}
	})

	t.Run("Name", func(t *testing.T) {
		named := &corev1.Secret{}
		named.Namespace, named.Name = "ns1", "s2"

		extracted, err := corev1apply.ExtractSecret(named, "")
		assert.NilError(t, err)

		if assert.Check(t, extracted.Namespace != nil) {
			assert.Equal(t, *extracted.Namespace, "ns1")
		}
		if assert.Check(t, extracted.Name != nil) {
			assert.Equal(t, *extracted.Name, "s2")
		}
	})

	t.Run("ResourceVersion", func(t *testing.T) {
		versioned := &corev1.Secret{}
		versioned.ResourceVersion = "asdf"

		extracted, err := corev1apply.ExtractSecret(versioned, "")
		assert.NilError(t, err)

		// ResourceVersion is not copied from the original.
		assert.Assert(t, extracted.ResourceVersion == nil)
	})
}

func TestInstallationReconcile(t *testing.T) {
	// Scenario:
	//   When there is no Secret and no Installation in memory,
	//   Then Reconcile should register with the API.
	//
	t.Run("FreshStart", func(t *testing.T) {
		var reconciler *InstallationReconciler
		var secret *corev1.Secret

		beforeEach := func() {
			reconciler = new(InstallationReconciler)
			secret = new(corev1.Secret)
			self.Installation = Installation{}
		}

		t.Run("ItRegisters", func(t *testing.T) {
			beforeEach()

			// API double; spy on requests.
			var requests []http.Request
			{
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests = append(requests, *r)
					_ = json.NewEncoder(w).Encode(map[string]any{
						"id": "abc", "auth_object": map[string]any{"secret": "xyz"},
					})
				}))
				t.Cleanup(server.Close)

				reconciler.NewClient = func() *Client {
					c := NewClient(server.URL, "")
					c.Backoff.Steps = 1
					assert.Equal(t, c.BaseURL.String(), server.URL)
					return c
				}
			}

			// Kubernetes double; spy on SSA patches.
			var applies []string
			{
				reconciler.Writer = runtime.ClientPatch(func(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
					assert.Equal(t, string(patch.Type()), "application/apply-patch+yaml")

					data, err := patch.Data(obj)
					applies = append(applies, string(data))
					return err
				})
			}

			ctx := context.Background()
			err := reconciler.reconcile(ctx, secret)
			assert.NilError(t, err)

			// It calls the API.
			assert.Equal(t, len(requests), 1)
			assert.Equal(t, requests[0].Method, "POST")
			assert.Equal(t, requests[0].URL.Path, "/vendor/operator/installations")

			// It stores the result in memory.
			assert.Equal(t, self.ID, "abc")
			assert.Equal(t, self.AuthObject.Secret, "xyz")

			// It stores the result in Kubernetes.
			assert.Equal(t, len(applies), 1)
			assert.Assert(t, cmp.Contains(applies[0], `"kind":"Secret"`))

			var decoded corev1.Secret
			assert.NilError(t, yaml.Unmarshal([]byte(applies[0]), &decoded))
			assert.Assert(t, cmp.Contains(string(decoded.Data["bridge-token"]), `"id":"abc"`))
			assert.Assert(t, cmp.Contains(string(decoded.Data["bridge-token"]), `"secret":"xyz"`))
		})

		t.Run("KubernetesError", func(t *testing.T) {
			beforeEach()

			// API double; successful.
			{
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_ = json.NewEncoder(w).Encode(map[string]any{
						"id": "123", "auth_object": map[string]any{"secret": "456"},
					})
				}))
				t.Cleanup(server.Close)

				reconciler.NewClient = func() *Client {
					c := NewClient(server.URL, "")
					c.Backoff.Steps = 1
					assert.Equal(t, c.BaseURL.String(), server.URL)
					return c
				}
			}

			// Kubernetes double; failure.
			expected := errors.New("boom")
			{
				reconciler.Writer = runtime.ClientPatch(func(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
					return expected
				})
			}

			ctx := context.Background()
			err := reconciler.reconcile(ctx, secret)
			assert.Equal(t, err, expected, "expected a Kubernetes error")

			// It stores the API result in memory.
			assert.Equal(t, self.ID, "123")
			assert.Equal(t, self.AuthObject.Secret, "456")
		})
	})

	// Scenario:
	//   When there is no Secret but an Installation exists in memory,
	//   Then Reconcile should store it in Kubernetes.
	//
	t.Run("LostSecret", func(t *testing.T) {
		var reconciler *InstallationReconciler
		var secret *corev1.Secret

		beforeEach := func(token []byte) {
			reconciler = new(InstallationReconciler)
			secret = new(corev1.Secret)
			secret.Data = map[string][]byte{
				KeyBridgeToken: token,
			}
			self.Installation = Installation{ID: "asdf"}
		}

		for _, tt := range []struct {
			Name  string
			Token []byte
		}{
			{Name: "NoToken", Token: nil},
			{Name: "BadToken", Token: []byte(`asdf`)},
		} {
			t.Run(tt.Name, func(t *testing.T) {
				beforeEach(tt.Token)

				// Kubernetes double; spy on SSA patches.
				var applies []string
				{
					reconciler.Writer = runtime.ClientPatch(func(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
						assert.Equal(t, string(patch.Type()), "application/apply-patch+yaml")

						data, err := patch.Data(obj)
						applies = append(applies, string(data))
						return err
					})
				}

				ctx := context.Background()
				err := reconciler.reconcile(ctx, secret)
				assert.NilError(t, err)

				assert.Equal(t, self.ID, "asdf", "expected no change to memory")

				// It stores the memory in Kubernetes.
				assert.Equal(t, len(applies), 1)
				assert.Assert(t, cmp.Contains(applies[0], `"kind":"Secret"`))

				var decoded corev1.Secret
				assert.NilError(t, yaml.Unmarshal([]byte(applies[0]), &decoded))
				assert.Assert(t, cmp.Contains(string(decoded.Data["bridge-token"]), `"id":"asdf"`))
			})
		}

		t.Run("KubernetesError", func(t *testing.T) {
			beforeEach(nil)

			// Kubernetes double; failure.
			expected := errors.New("boom")
			{
				reconciler.Writer = runtime.ClientPatch(func(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
					return expected
				})
			}

			ctx := context.Background()
			err := reconciler.reconcile(ctx, secret)
			assert.Equal(t, err, expected, "expected a Kubernetes error")
			assert.Equal(t, self.ID, "asdf", "expected no change to memory")
		})
	})

	// Scenario:
	//   When there is a Secret but no Installation in memory,
	//   Then Reconcile should store it in memory.
	//
	t.Run("Restart", func(t *testing.T) {
		var reconciler *InstallationReconciler
		var secret *corev1.Secret

		beforeEach := func() {
			reconciler = new(InstallationReconciler)
			secret = new(corev1.Secret)
			secret.Data = map[string][]byte{KeyBridgeToken: []byte(`{"id":"xyz"}`)}
			self.Installation = Installation{}
		}

		t.Run("ItLoads", func(t *testing.T) {
			beforeEach()

			ctx := context.Background()
			err := reconciler.reconcile(ctx, secret)
			assert.NilError(t, err)

			assert.Equal(t, self.ID, "xyz")
		})
	})
}
