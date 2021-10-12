//go:build envtest
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

	"go.opentelemetry.io/otel"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/crunchydata/postgres-operator/internal/naming"
)

func TestManageControllerRefs(t *testing.T) {

	// setup the test environment and ensure a clean teardown
	tEnv, tClient, cfg := setupTestEnv(t, ControllerName)
	r := &Reconciler{}
	ctx, cancel := setupManager(t, cfg, func(mgr manager.Manager) {
		r = &Reconciler{
			Client:   mgr.GetClient(),
			Recorder: mgr.GetEventRecorderFor(ControllerName),
			Tracer:   otel.Tracer(ControllerName),
			Owner:    ControllerName,
		}
	})
	t.Cleanup(func() {
		teardownManager(cancel, t)
		teardownTestEnv(t, tEnv)
	})

	clusterName := "hippo"
	namespace := "postgres-operator-test-" + rand.String(6)

	ns := &corev1.Namespace{}
	ns.Name = namespace
	assert.NilError(t, tClient.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, tClient.Delete(ctx, ns)) })

	cluster := testCluster()
	cluster.Namespace = ns.Name

	// create the test PostgresCluster
	if err := tClient.Create(ctx, cluster); err != nil {
		t.Fatal(err)
	}

	// create a base StatefulSet that can be used by the various tests below
	objBase := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"label1": "val1"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"label1": "val1"},
				},
			},
		},
	}

	t.Run("adopt Object", func(t *testing.T) {

		obj := objBase.DeepCopy()
		obj.Name = "adpot"
		obj.Labels = map[string]string{naming.LabelCluster: clusterName}

		if err := r.Client.Create(ctx, obj); err != nil {
			t.Error(err)
		}

		if err := r.manageControllerRefs(ctx, obj); err != nil {
			t.Error(err)
		}

		if err := tClient.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
			t.Error(err)
		}

		var foundControllerOwnerRef bool
		for _, ref := range obj.GetOwnerReferences() {
			if *ref.Controller && *ref.BlockOwnerDeletion &&
				ref.UID == cluster.GetUID() &&
				ref.Name == clusterName && ref.Kind == "PostgresCluster" {
				foundControllerOwnerRef = true
				break
			}
		}

		if !foundControllerOwnerRef {
			t.Error("unable to find expected controller ref")
		}
	})

	t.Run("release Object", func(t *testing.T) {

		isTrue := true
		obj := objBase.DeepCopy()
		obj.Name = "release"
		obj.OwnerReferences = append(obj.OwnerReferences, metav1.OwnerReference{
			APIVersion:         "group/version",
			Kind:               "PostgresCluster",
			Name:               clusterName,
			UID:                cluster.GetUID(),
			Controller:         &isTrue,
			BlockOwnerDeletion: &isTrue,
		})

		if err := r.Client.Create(ctx, obj); err != nil {
			t.Error(err)
		}

		if err := r.manageControllerRefs(ctx, obj); err != nil {
			t.Error(err)
		}

		if err := tClient.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
			t.Error(err)
		}

		if len(obj.GetOwnerReferences()) != 0 {
			t.Error("expected orphaned Object but found controller ref")
		}
	})

	t.Run("ignore Object: no matching labels or owner refs", func(t *testing.T) {

		obj := objBase.DeepCopy()
		obj.Name = "ignore-no-labels-refs"
		obj.Labels = map[string]string{"ignore-label": "ignore-value"}

		if err := r.Client.Create(ctx, obj); err != nil {
			t.Error(err)
		}

		if err := r.manageControllerRefs(ctx, obj); err != nil {
			t.Error(err)
		}

		if err := tClient.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
			t.Error(err)
		}

		if len(obj.GetOwnerReferences()) != 0 {
			t.Error("expected orphaned Object but found controller ref")
		}
	})

	t.Run("ignore Object: PostgresCluster does not exist", func(t *testing.T) {

		obj := objBase.DeepCopy()
		obj.Name = "ignore-no-postgrescluster"
		obj.Labels = map[string]string{naming.LabelCluster: "noexist"}

		if err := r.Client.Create(ctx, obj); err != nil {
			t.Error(err)
		}

		if err := r.manageControllerRefs(ctx, obj); err != nil {
			t.Error(err)
		}

		if err := tClient.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
			t.Error(err)
		}

		if len(obj.GetOwnerReferences()) != 0 {
			t.Error("expected orphaned Object but found controller ref")
		}
	})
}
