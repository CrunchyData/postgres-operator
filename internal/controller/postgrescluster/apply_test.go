// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/crunchydata/postgres-operator/internal/testing/require"
)

func TestServerSideApply(t *testing.T) {
	ctx := context.Background()
	cfg, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	ns := setupNamespace(t, cc)

	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	assert.NilError(t, err)

	server, err := dc.ServerVersion()
	assert.NilError(t, err)

	serverVersion, err := version.ParseGeneric(server.GitVersion)
	assert.NilError(t, err)

	t.Run("ObjectMeta", func(t *testing.T) {
		reconciler := Reconciler{Client: cc, Owner: client.FieldOwner(t.Name())}
		constructor := func() *corev1.ConfigMap {
			var cm corev1.ConfigMap
			cm.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))
			cm.Namespace, cm.Name = ns.Name, "object-meta"
			cm.Data = map[string]string{"key": "value"}
			return &cm
		}

		// Create the object.
		before := constructor()
		assert.NilError(t, cc.Patch(ctx, before, client.Apply, reconciler.Owner))
		assert.Assert(t, before.GetResourceVersion() != "")

		// Allow the Kubernetes API clock to advance.
		time.Sleep(time.Second)

		// client.Apply changes the ResourceVersion inadvertently.
		after := constructor()
		assert.NilError(t, cc.Patch(ctx, after, client.Apply, reconciler.Owner))
		assert.Assert(t, after.GetResourceVersion() != "")

		switch {
		case serverVersion.LessThan(version.MustParseGeneric("1.25.15")):
		case serverVersion.AtLeast(version.MustParseGeneric("1.26")) && serverVersion.LessThan(version.MustParseGeneric("1.26.10")):
		case serverVersion.AtLeast(version.MustParseGeneric("1.27")) && serverVersion.LessThan(version.MustParseGeneric("1.27.7")):

			assert.Assert(t, after.GetResourceVersion() != before.GetResourceVersion(),
				"expected https://issue.k8s.io/116861")

		default:
			assert.Assert(t, after.GetResourceVersion() == before.GetResourceVersion())
		}

		// Our apply method generates the correct apply-patch.
		again := constructor()
		assert.NilError(t, reconciler.apply(ctx, again))
		assert.Assert(t, again.GetResourceVersion() != "")
		assert.Assert(t, again.GetResourceVersion() == after.GetResourceVersion(),
			"expected to correctly no-op")
	})

	t.Run("ControllerReference", func(t *testing.T) {
		reconciler := Reconciler{Client: cc, Owner: client.FieldOwner(t.Name())}

		// Setup two possible controllers.
		controller1 := new(corev1.ConfigMap)
		controller1.Namespace, controller1.Name = ns.Name, "controller1"
		assert.NilError(t, cc.Create(ctx, controller1))

		controller2 := new(corev1.ConfigMap)
		controller2.Namespace, controller2.Name = ns.Name, "controller2"
		assert.NilError(t, cc.Create(ctx, controller2))

		// Create an object that is controlled.
		controlled := new(corev1.ConfigMap)
		controlled.Namespace, controlled.Name = ns.Name, "controlled"
		assert.NilError(t,
			controllerutil.SetControllerReference(controller1, controlled, cc.Scheme()))
		assert.NilError(t, cc.Create(ctx, controlled))

		original := metav1.GetControllerOfNoCopy(controlled)
		assert.Assert(t, original != nil)

		// Try to change the controller using client.Apply.
		applied := new(corev1.ConfigMap)
		applied.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))
		applied.Namespace, applied.Name = controlled.Namespace, controlled.Name
		assert.NilError(t,
			controllerutil.SetControllerReference(controller2, applied, cc.Scheme()))

		err1 := cc.Patch(ctx, applied, client.Apply, client.ForceOwnership, reconciler.Owner)

		// Patch not accepted; the ownerReferences field is invalid.
		assert.Assert(t, apierrors.IsInvalid(err1), "got %#v", err1)
		assert.ErrorContains(t, err1, "one reference")

		var status *apierrors.StatusError
		assert.Assert(t, errors.As(err1, &status))
		assert.Assert(t, status.ErrStatus.Details != nil)
		assert.Assert(t, len(status.ErrStatus.Details.Causes) != 0)
		assert.Equal(t, status.ErrStatus.Details.Causes[0].Field, "metadata.ownerReferences")

		// Try to change the controller using our apply method.
		err2 := reconciler.apply(ctx, applied)

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

	t.Run("StatefulSetStatus", func(t *testing.T) {
		constructor := func(name string) *appsv1.StatefulSet {
			var sts appsv1.StatefulSet
			sts.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("StatefulSet"))
			sts.Namespace, sts.Name = ns.Name, name
			sts.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: map[string]string{"select": name},
			}
			sts.Spec.Template.Labels = map[string]string{"select": name}
			return &sts
		}

		reconciler := Reconciler{Client: cc, Owner: client.FieldOwner(t.Name())}
		upstream := constructor("status-upstream")

		// The structs defined in "k8s.io/api/apps/v1" marshal empty status fields.
		switch {
		case serverVersion.LessThan(version.MustParseGeneric("1.22")):
			assert.ErrorContains(t,
				cc.Patch(ctx, upstream, client.Apply, client.ForceOwnership, reconciler.Owner),
				"field not declared in schema",
				"expected https://issue.k8s.io/109210")

		default:
			assert.NilError(t,
				cc.Patch(ctx, upstream, client.Apply, client.ForceOwnership, reconciler.Owner))
		}

		// Our apply method generates the correct apply-patch.
		again := constructor("status-local")
		assert.NilError(t, reconciler.apply(ctx, again))
	})

	t.Run("ServiceSelector", func(t *testing.T) {
		constructor := func(name string) *corev1.Service {
			var service corev1.Service
			service.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Service"))
			service.Namespace, service.Name = ns.Name, name
			service.Spec.Ports = []corev1.ServicePort{{
				Port: 9999, Protocol: corev1.ProtocolTCP,
			}}
			return &service
		}

		t.Run("wrong-keys", func(t *testing.T) {
			reconciler := Reconciler{Client: cc, Owner: client.FieldOwner(t.Name())}

			intent := constructor("some-selector")
			intent.Spec.Selector = map[string]string{"k1": "v1"}

			// Create the Service.
			before := intent.DeepCopy()
			assert.NilError(t,
				cc.Patch(ctx, before, client.Apply, client.ForceOwnership, reconciler.Owner))

			// Something external mucks it up.
			assert.NilError(t,
				cc.Patch(ctx, before,
					client.RawPatch(client.Merge.Type(), []byte(`{"spec":{"selector":{"bad":"v2"}}}`)),
					client.FieldOwner("wrong")))

			// client.Apply cannot correct it in old versions of Kubernetes.
			after := intent.DeepCopy()
			assert.NilError(t,
				cc.Patch(ctx, after, client.Apply, client.ForceOwnership, reconciler.Owner))

			switch {
			case serverVersion.LessThan(version.MustParseGeneric("1.22")):

				assert.Assert(t, len(after.Spec.Selector) != len(intent.Spec.Selector),
					"expected https://issue.k8s.io/97970, got %v", after.Spec.Selector)

			default:
				assert.DeepEqual(t, after.Spec.Selector, intent.Spec.Selector)
			}

			// Our apply method corrects it.
			again := intent.DeepCopy()
			assert.NilError(t, reconciler.apply(ctx, again))
			assert.DeepEqual(t, again.Spec.Selector, intent.Spec.Selector)

			var count int
			var managed *metav1.ManagedFieldsEntry
			for i := range again.ManagedFields {
				if again.ManagedFields[i].Manager == t.Name() {
					count++
					managed = &again.ManagedFields[i]
				}
			}

			assert.Equal(t, count, 1, "expected manager once in %v", again.ManagedFields)
			assert.Equal(t, managed.Operation, metav1.ManagedFieldsOperationApply)

			assert.Assert(t, managed.FieldsV1 != nil)
			assert.Assert(t, strings.Contains(string(managed.FieldsV1.Raw), `"f:selector":{`),
				"expected f:selector in %s", managed.FieldsV1.Raw)
		})

		for _, tt := range []struct {
			name     string
			selector map[string]string
		}{
			{"zero", nil},
			{"empty", make(map[string]string)},
		} {
			t.Run(tt.name, func(t *testing.T) {
				reconciler := Reconciler{Client: cc, Owner: client.FieldOwner(t.Name())}

				intent := constructor(tt.name + "-selector")
				intent.Spec.Selector = tt.selector

				// Create the Service.
				before := intent.DeepCopy()
				assert.NilError(t,
					cc.Patch(ctx, before, client.Apply, client.ForceOwnership, reconciler.Owner))

				// Something external mucks it up.
				assert.NilError(t,
					cc.Patch(ctx, before,
						client.RawPatch(client.Merge.Type(), []byte(`{"spec":{"selector":{"bad":"v2"}}}`)),
						client.FieldOwner("wrong")))

				// client.Apply cannot correct it.
				after := intent.DeepCopy()
				assert.NilError(t,
					cc.Patch(ctx, after, client.Apply, client.ForceOwnership, reconciler.Owner))

				assert.Assert(t, len(after.Spec.Selector) != len(intent.Spec.Selector),
					"got %v", after.Spec.Selector)

				// Our apply method corrects it.
				again := intent.DeepCopy()
				assert.NilError(t, reconciler.apply(ctx, again))
				assert.Assert(t,
					equality.Semantic.DeepEqual(again.Spec.Selector, intent.Spec.Selector),
					"\n--- again.Spec.Selector\n+++ intent.Spec.Selector\n%v",
					cmp.Diff(again.Spec.Selector, intent.Spec.Selector))

				var count int
				var managed *metav1.ManagedFieldsEntry
				for i := range again.ManagedFields {
					if again.ManagedFields[i].Manager == t.Name() {
						count++
						managed = &again.ManagedFields[i]
					}
				}

				assert.Equal(t, count, 1, "expected manager once in %v", again.ManagedFields)
				assert.Equal(t, managed.Operation, metav1.ManagedFieldsOperationApply)

				// The selector field is forgotten, however.
				assert.Assert(t, managed.FieldsV1 != nil)
				assert.Assert(t, !strings.Contains(string(managed.FieldsV1.Raw), `"f:selector":{`),
					"expected f:selector to be missing from %s", managed.FieldsV1.Raw)
			})
		}
	})

	t.Run("ServiceType", func(t *testing.T) {
		constructor := func(name string) *corev1.Service {
			var service corev1.Service
			service.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Service"))
			service.Namespace, service.Name = ns.Name, name
			service.Spec.Ports = []corev1.ServicePort{
				{Name: "one", Port: 9999, Protocol: corev1.ProtocolTCP},
				{Name: "two", Port: 1234, Protocol: corev1.ProtocolTCP},
			}
			return &service
		}

		reconciler := Reconciler{Client: cc, Owner: client.FieldOwner(t.Name())}

		// Start as NodePort.
		intent := constructor("node-port")
		intent.Spec.Type = corev1.ServiceTypeNodePort

		// Create the Service.
		before := intent.DeepCopy()
		assert.NilError(t,
			cc.Patch(ctx, before, client.Apply, client.ForceOwnership, reconciler.Owner))

		// Change to ClusterIP.
		intent.Spec.Type = corev1.ServiceTypeClusterIP

		// client.Apply cannot change it in old versions of Kubernetes.
		after := intent.DeepCopy()
		err := cc.Patch(ctx, after, client.Apply, client.ForceOwnership, reconciler.Owner)

		switch {
		case serverVersion.LessThan(version.MustParseGeneric("1.20")):

			assert.ErrorContains(t, err, "nodePort: Forbidden",
				"expected https://issue.k8s.io/33766")

		default:
			assert.NilError(t, err)
			assert.Equal(t, after.Spec.Type, intent.Spec.Type)
			assert.Equal(t, after.Spec.ClusterIP, before.Spec.ClusterIP,
				"expected to keep the same ClusterIP")
		}

		// Our apply method changes it.
		again := intent.DeepCopy()
		assert.NilError(t, reconciler.apply(ctx, again))
		assert.Equal(t, again.Spec.Type, intent.Spec.Type)
		assert.Equal(t, again.Spec.ClusterIP, before.Spec.ClusterIP,
			"expected to keep the same ClusterIP")
	})
}
