/*
 Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
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

package require

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Namespace creates a random namespace that is deleted by t.Cleanup. It calls
// t.Fatal when creation fails. The caller may delete the namespace at any time.
func Namespace(t testing.TB, cc client.Client) *corev1.Namespace {
	t.Helper()

	ns := &corev1.Namespace{}
	ns.GenerateName = "postgres-operator-test-"
	ns.Labels = map[string]string{"postgres-operator-test": t.Name()}

	ctx := context.Background()
	assert.NilError(t, cc.Create(ctx, ns))

	t.Cleanup(func() {
		assert.Check(t, client.IgnoreNotFound(cc.Delete(ctx, ns)))
	})

	return ns
}
