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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
)

func TestManageSTSControllerRefs(t *testing.T) {

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
	namespace := "hippo"

	// create the test namespace
	if err := tClient.Create(ctx,
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}); err != nil {
		t.Error(err)
	}

	// create a PostgresCluster to test with
	postgresCluster := &v1alpha1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: namespace,
		},
		Spec: v1alpha1.PostgresClusterSpec{
			InstanceSets: []v1alpha1.PostgresInstanceSetSpec{{Name: "instance1"}},
		},
	}

	// create the test PostgresCluster
	if err := tClient.Create(ctx, postgresCluster); err != nil {
		t.Fatal(err)
	}

	// create a base StatefulSet that can be used by the various tests below
	stsBase := &appsv1.StatefulSet{
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

	t.Run("adopt StatefulSet", func(t *testing.T) {

		sts := stsBase.DeepCopy()
		sts.Name = "adpot"
		sts.Labels = map[string]string{naming.LabelCluster: clusterName}

		if err := r.Client.Create(ctx, sts); err != nil {
			t.Error(err)
		}

		if err := r.manageSTSControllerRefs(ctx, sts); err != nil {
			t.Error(err)
		}

		if err := tClient.Get(ctx, client.ObjectKeyFromObject(sts), sts); err != nil {
			t.Error(err)
		}

		var foundControllerOwnerRef bool
		for _, ref := range sts.GetOwnerReferences() {
			if *ref.Controller && *ref.BlockOwnerDeletion &&
				ref.UID == postgresCluster.GetUID() &&
				ref.Name == clusterName && ref.Kind == "PostgresCluster" {
				foundControllerOwnerRef = true
				break
			}
		}

		if !foundControllerOwnerRef {
			t.Error("unable to find expected controller ref")
		}
	})

	t.Run("release StatefulSet", func(t *testing.T) {

		isTrue := true
		sts := stsBase.DeepCopy()
		sts.Name = "release"
		sts.OwnerReferences = append(sts.OwnerReferences, metav1.OwnerReference{
			APIVersion:         "group/version",
			Kind:               "PostgresCluster",
			Name:               clusterName,
			UID:                postgresCluster.GetUID(),
			Controller:         &isTrue,
			BlockOwnerDeletion: &isTrue,
		})

		if err := r.Client.Create(ctx, sts); err != nil {
			t.Error(err)
		}

		if err := r.manageSTSControllerRefs(ctx, sts); err != nil {
			t.Error(err)
		}

		if err := tClient.Get(ctx, client.ObjectKeyFromObject(sts), sts); err != nil {
			t.Error(err)
		}

		if len(sts.GetOwnerReferences()) != 0 {
			t.Error("expected orphaned StatefulSet but found controller ref")
		}
	})

	t.Run("ignore StatefulSet: no matching labels or owner refs", func(t *testing.T) {

		sts := stsBase.DeepCopy()
		sts.Name = "ignore-no-labels-refs"
		sts.Labels = map[string]string{"ignore-label": "ignore-value"}

		if err := r.Client.Create(ctx, sts); err != nil {
			t.Error(err)
		}

		if err := r.manageSTSControllerRefs(ctx, sts); err != nil {
			t.Error(err)
		}

		if err := tClient.Get(ctx, client.ObjectKeyFromObject(sts), sts); err != nil {
			t.Error(err)
		}

		if len(sts.GetOwnerReferences()) != 0 {
			t.Error("expected orphaned StatefulSet but found controller ref")
		}
	})

	t.Run("ignore StatefulSet: PostgresCluster does not exist", func(t *testing.T) {

		sts := stsBase.DeepCopy()
		sts.Name = "ignore-no-postgrescluster"
		sts.Labels = map[string]string{naming.LabelCluster: "noexist"}

		if err := r.Client.Create(ctx, sts); err != nil {
			t.Error(err)
		}

		if err := r.manageSTSControllerRefs(ctx, sts); err != nil {
			t.Error(err)
		}

		if err := tClient.Get(ctx, client.ObjectKeyFromObject(sts), sts); err != nil {
			t.Error(err)
		}

		if len(sts.GetOwnerReferences()) != 0 {
			t.Error("expected orphaned StatefulSet but found controller ref")
		}
	})
}
