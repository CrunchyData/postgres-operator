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
	"testing"
	"time"

	"gotest.tools/v3/assert"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestServerSideApply(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	env := &envtest.Environment{}
	config, err := env.Start()
	assert.NilError(t, err)
	t.Cleanup(func() { assert.Check(t, env.Stop()) })

	cc, err := client.New(config, client.Options{})
	assert.NilError(t, err)

	ns := &v1.Namespace{}
	ns.GenerateName = "postgres-operator-test-"
	assert.NilError(t, cc.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, ns)) })

	for _, tt := range []struct {
		name        string
		constructor func() client.Object
	}{
		{"ConfigMap", func() client.Object {
			var cm v1.ConfigMap
			cm.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("ConfigMap"))
			cm.Namespace, cm.Name = ns.Name, "cm1"
			cm.Data = map[string]string{"key": "value"}
			return &cm
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			reconciler := Reconciler{Client: cc, Owner: client.FieldOwner(t.Name())}

			// Create the object.
			before := tt.constructor()
			assert.NilError(t, cc.Patch(ctx, before, client.Apply, reconciler.Owner))
			assert.Assert(t, before.GetResourceVersion() != "")

			// Allow the Kubernetes API clock to advance.
			time.Sleep(time.Second)

			// client.Apply changes the ResourceVersion inadvertently.
			after := tt.constructor()
			assert.NilError(t, cc.Patch(ctx, after, client.Apply, reconciler.Owner))
			assert.Assert(t, after.GetResourceVersion() != "")
			assert.Assert(t, after.GetResourceVersion() != before.GetResourceVersion(),
				"expected https://github.com/kubernetes-sigs/controller-runtime/issues/1356")

			// Our apply method generates the correct apply-patch.
			again := tt.constructor()
			assert.NilError(t, reconciler.apply(ctx, again))
			assert.Assert(t, again.GetResourceVersion() != "")
			assert.Assert(t, again.GetResourceVersion() == after.GetResourceVersion(),
				"expected to correctly no-op")
		})
	}
}
