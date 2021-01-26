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
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

	t.Run("ControllerReference", func(t *testing.T) {
		reconciler := Reconciler{Client: cc, Owner: client.FieldOwner(t.Name())}

		// Setup two possible controllers.
		controller1 := new(v1.ConfigMap)
		controller1.Namespace, controller1.Name = ns.Name, "controller1"
		assert.NilError(t, cc.Create(ctx, controller1))

		controller2 := new(v1.ConfigMap)
		controller2.Namespace, controller2.Name = ns.Name, "controller2"
		assert.NilError(t, cc.Create(ctx, controller2))

		// Create an object that is controlled.
		controlled := new(v1.ConfigMap)
		controlled.Namespace, controlled.Name = ns.Name, "controlled"
		assert.NilError(t,
			controllerutil.SetControllerReference(controller1, controlled, cc.Scheme()))
		assert.NilError(t, cc.Create(ctx, controlled))

		original := metav1.GetControllerOfNoCopy(controlled)
		assert.Assert(t, original != nil)

		// Try to change the controller using client.Apply.
		applied := new(v1.ConfigMap)
		applied.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("ConfigMap"))
		applied.Namespace, applied.Name = controlled.Namespace, controlled.Name
		assert.NilError(t,
			controllerutil.SetControllerReference(controller2, applied, cc.Scheme()))

		err1 := cc.Patch(ctx, applied, client.Apply, client.ForceOwnership, reconciler.Owner)

		// Patch not accepted; the ownerReferences field is invalid.
		assert.Assert(t, kerrors.IsInvalid(err1), "got %#v", err1)
		assert.ErrorContains(t, err1, "one reference")

		var status *kerrors.StatusError
		assert.Assert(t, errors.As(err1, &status))
		assert.Assert(t, status.ErrStatus.Details != nil)
		assert.Assert(t, len(status.ErrStatus.Details.Causes) != 0)
		assert.Equal(t, status.ErrStatus.Details.Causes[0].Field, "metadata.ownerReferences")

		// Try to change the controller using our apply method.
		err2 := reconciler.apply(ctx, applied, client.ForceOwnership)

		// Same result; patch not accepted.
		assert.DeepEqual(t, err1, err2,
			// Message fields contain GoStrings of metav1.OwnerReference, 🤦
			// so ignore pointer addresses therein.
			cmp.FilterPath(func(p cmp.Path) bool {
				return p.Last().String() == ".Message"
			}, cmp.Transformer("", func(s string) string {
				return regexp.MustCompile(`\(0x[^)]+\)`).ReplaceAllString(s, "()")
			})),
		)
	})
}
