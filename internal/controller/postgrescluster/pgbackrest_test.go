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
	"fmt"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
)

func TestReconcilePGBackRest(t *testing.T) {

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

	clusterName := "hippocluster"
	clusterUID := types.UID("hippouid")
	namespace := "hippo"
	pgBackRestImage := "testimage"

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
			UID:       clusterUID,
		},
		Spec: v1alpha1.PostgresClusterSpec{
			Archive: v1alpha1.Archive{
				PGBackRest: v1alpha1.PGBackRestArchive{
					Image: pgBackRestImage,
				},
			},
		},
	}

	result, err := r.reconcilePGBackRest(ctx, postgresCluster)
	if err != nil || result != (reconcile.Result{}) {
		t.Error(fmt.Errorf("unable to reconcile pgBackRest: %v", err))
	}

	// test that the repo was created properly
	t.Run("verify pgbackrest repo StatefulSet", func(t *testing.T) {

		// get the pgBackRest repo deployment using the labels we expect it to have
		repos := &appsv1.StatefulSetList{}
		if err := tClient.List(ctx, repos, client.InNamespace(namespace),
			client.MatchingLabels{
				naming.LabelCluster:        clusterName,
				naming.LabelPGBackRest:     "",
				naming.LabelPGBackRestRepo: "",
			}); err != nil {
			t.Fatal(err)
		}

		repo := appsv1.StatefulSet{}
		// verify that we found a repo deployment as expected
		if len(repos.Items) == 0 {
			t.Errorf("Did not find a repo deployment")
		} else if len(repos.Items) > 1 {
			t.Errorf("Too many repo deployments found")
		} else {
			repo = repos.Items[0]
		}

		// verify proper number of replicas
		if *repo.Spec.Replicas != 1 {
			t.Errorf("%v replicas found for repo deployment, expected %v",
				repo.Spec.Replicas, 1)
		}

		// verify proper ownership
		var foundOwnershipRef bool
		for _, r := range repo.GetOwnerReferences() {
			if r.Kind == "PostgresCluster" && r.Name == clusterName &&
				r.UID == types.UID(clusterUID) {

				foundOwnershipRef = true
				break
			}
		}

		if !foundOwnershipRef {
			t.Errorf("did not find expected ownership references")
		}

		// verify proper matching labels
		expectedLabels := map[string]string{
			naming.LabelCluster:        clusterName,
			naming.LabelPGBackRest:     "",
			naming.LabelPGBackRestRepo: "",
		}
		expectedLabelsSelector, err := metav1.LabelSelectorAsSelector(
			metav1.SetAsLabelSelector(expectedLabels))
		if err != nil {
			t.Error(err)
		}
		if !expectedLabelsSelector.Matches(labels.Set(repo.GetLabels())) {
			t.Errorf("repo host is missing an expected label: found=%v, expected=%v",
				repo.GetLabels(), expectedLabels)
		}

		// verify that the repohost container exists and contains the proper env vars
		var repoHostContExists bool
		repoContainer := corev1.Container{}
		for _, c := range repo.Spec.Template.Spec.Containers {
			if c.Name == PGBackRestRepoContainerName {
				repoHostContExists = true
				repoContainer = c
				break
			}
		}
		// now verify the proper env within the container
		if repoHostContExists {
			var foundModeEnvVar bool
			for _, envVar := range repoContainer.Env {
				if envVar.Name == "MODE" && envVar.Value == "pgbackrest-repo" {
					foundModeEnvVar = true
					break
				}
			}
			if !foundModeEnvVar {
				t.Error("repo host is missing the proper MODE environment variable")
			}
		} else {
			t.Errorf("repo host is missing a container with name %s",
				PGBackRestRepoContainerName)
		}

		repoHostStatus := postgresCluster.Status.PGBackRest.RepoHost
		if repoHostStatus != nil {
			if repoHostStatus.APIVersion != "apps/v1" || repoHostStatus.Kind != "StatefulSet" {
				t.Errorf("invalid version/kind for repo host status")
			}
			if repoHostStatus.Name == "" {
				t.Errorf("invalid repo host name in repo host status")
			}
		} else {
			t.Errorf("repo host status is missing")
		}

		var foundConditionRepoHostsReady bool
		for _, c := range postgresCluster.Status.Conditions {
			if c.Type == "PGBackRestRepoHostsReady" {
				foundConditionRepoHostsReady = true
				break
			}
		}
		if !foundConditionRepoHostsReady {
			t.Errorf("status condition PGBackRestRepoHostsReady is missing")
		}

		events := &corev1.EventList{}
		if err := wait.Poll(time.Second/2, time.Second*2, func() (bool, error) {
			if err := tClient.List(ctx, events, &client.MatchingFields{
				"involvedObject.kind":      "PostgresCluster",
				"involvedObject.name":      clusterName,
				"involvedObject.namespace": namespace,
				"involvedObject.uid":       string(clusterUID),
				"reason":                   "RepoHostCreated",
			}); err != nil {
				return false, err
			}
			if len(events.Items) != 1 {
				return false, nil
			}
			return true, nil
		}); err != nil {
			t.Error(err)
		}
	})
}
