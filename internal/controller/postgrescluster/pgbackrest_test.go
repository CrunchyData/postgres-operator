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
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/pgbackrest"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
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
	pgBackRestImage := "testimage"

	ns := &v1.Namespace{}
	ns.GenerateName = "postgres-operator-test-"
	assert.NilError(t, tClient.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, tClient.Delete(ctx, ns)) })
	namespace := ns.Name

	// create a PostgresCluster to test with
	postgresCluster := &v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: namespace,
			UID:       clusterUID,
		},
		Spec: v1beta1.PostgresClusterSpec{
			Archive: v1beta1.Archive{
				PGBackRest: v1beta1.PGBackRestArchive{
					Repos: []v1beta1.RepoVolume{{
						Name: "repo1",
						VolumeClaimSpec: v1.PersistentVolumeClaimSpec{
							AccessModes: []v1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
							Resources: v1.ResourceRequirements{
								Requests: map[v1.ResourceName]resource.Quantity{
									v1.ResourceStorage: resource.MustParse("1Gi"),
								},
							},
						},
					}, {
						Name: "repo2",
						VolumeClaimSpec: v1.PersistentVolumeClaimSpec{
							AccessModes: []v1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
							Resources: v1.ResourceRequirements{
								Requests: map[v1.ResourceName]resource.Quantity{
									v1.ResourceStorage: resource.MustParse("2Gi"),
								},
							},
						},
					}},
					RepoHost: &v1beta1.RepoHost{
						Dedicated: &v1beta1.DedicatedRepo{
							Resources: &corev1.ResourceRequirements{},
						},
						Image: pgBackRestImage,
					},
				},
			},
		},
	}

	instanceNames := []string{"instance1", "instance2", "instance3"}
	result, err := r.reconcilePGBackRest(ctx, postgresCluster, instanceNames)
	if err != nil || result != (reconcile.Result{}) {
		t.Error(fmt.Errorf("unable to reconcile pgBackRest: %v", err))
	}

	// test that the repo was created properly
	t.Run("verify pgbackrest dedicated repo StatefulSet", func(t *testing.T) {

		// get the pgBackRest repo sts using the labels we expect it to have
		dedicatedRepos := &appsv1.StatefulSetList{}
		if err := tClient.List(ctx, dedicatedRepos, client.InNamespace(namespace),
			client.MatchingLabels{
				naming.LabelCluster:             clusterName,
				naming.LabelPGBackRest:          "",
				naming.LabelPGBackRestRepoHost:  "",
				naming.LabelPGBackRestDedicated: "",
			}); err != nil {
			t.Fatal(err)
		}

		repo := appsv1.StatefulSet{}
		// verify that we found a repo sts as expected
		if len(dedicatedRepos.Items) == 0 {
			t.Fatal("Did not find a dedicated repo sts")
		} else if len(dedicatedRepos.Items) > 1 {
			t.Fatal("Too many dedicated repo sts's found")
		} else {
			repo = dedicatedRepos.Items[0]
		}

		// verify proper number of replicas
		if *repo.Spec.Replicas != 1 {
			t.Errorf("%v replicas found for dedicated repo sts, expected %v",
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
			naming.LabelCluster:             clusterName,
			naming.LabelPGBackRest:          "",
			naming.LabelPGBackRestRepoHost:  "",
			naming.LabelPGBackRestDedicated: "",
		}
		expectedLabelsSelector, err := metav1.LabelSelectorAsSelector(
			metav1.SetAsLabelSelector(expectedLabels))
		if err != nil {
			t.Error(err)
		}
		if !expectedLabelsSelector.Matches(labels.Set(repo.GetLabels())) {
			t.Errorf("dedicated repo host is missing an expected label: found=%v, expected=%v",
				repo.GetLabels(), expectedLabels)
		}

		// verify that the repohost container exists and contains the proper env vars
		var repoHostContExists bool
		for _, c := range repo.Spec.Template.Spec.Containers {
			if c.Name == naming.PGBackRestRepoContainerName {
				repoHostContExists = true
			}
		}
		// now verify the proper env within the container
		if !repoHostContExists {
			t.Errorf("dedicated repo host is missing a container with name %s",
				naming.PGBackRestRepoContainerName)
		}

		repoHostStatus := postgresCluster.Status.PGBackRest.RepoHost
		if repoHostStatus != nil {
			if repoHostStatus.APIVersion != "apps/v1" || repoHostStatus.Kind != "StatefulSet" {
				t.Errorf("invalid version/kind for dedicated repo host status")
			}
		} else {
			t.Errorf("dedicated repo host status is missing")
		}

		var foundConditionRepoHostsReady bool
		for _, c := range postgresCluster.Status.Conditions {
			if c.Type == "PGBackRestRepoHostReady" {
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

	t.Run("verify pgbackrest repo volumes", func(t *testing.T) {

		// get the pgBackRest repo sts using the labels we expect it to have
		repoVols := &v1.PersistentVolumeClaimList{}
		if err := tClient.List(ctx, repoVols, client.InNamespace(namespace),
			client.MatchingLabels{
				naming.LabelCluster:              clusterName,
				naming.LabelPGBackRest:           "",
				naming.LabelPGBackRestRepoVolume: "",
			}); err != nil {
			t.Fatal(err)
		}
		assert.Assert(t, len(repoVols.Items) > 0)

		for _, r := range postgresCluster.Spec.Archive.PGBackRest.Repos {
			var foundRepoVol bool
			for _, v := range repoVols.Items {
				if v.GetName() == naming.PGBackRestRepoVolume(postgresCluster, r.Name).Name {
					foundRepoVol = true
					break
				}
			}
			assert.Assert(t, foundRepoVol)
		}
	})

	t.Run("verify pgbackrest configuration", func(t *testing.T) {

		config := &v1.ConfigMap{}
		if err := tClient.Get(ctx, types.NamespacedName{
			Name:      naming.PGBackRestConfig(postgresCluster).Name,
			Namespace: postgresCluster.GetNamespace(),
		}, config); err != nil {
			assert.NilError(t, err)
		}
		assert.Assert(t, len(config.Data) > 0)

		for _, n := range instanceNames {
			var instanceConfFound, dedicatedRepoConfFound bool
			for k, v := range config.Data {
				if v != "" {
					if k == n+".conf" {
						instanceConfFound = true
					} else if k == pgbackrest.CMRepoKey {
						dedicatedRepoConfFound = true
					}
				}
			}
			assert.Check(t, instanceConfFound)
			assert.Check(t, dedicatedRepoConfFound)
		}

		sshConfig := &v1.ConfigMap{}
		if err := tClient.Get(ctx, types.NamespacedName{
			Name:      naming.PGBackRestSSHConfig(postgresCluster).Name,
			Namespace: postgresCluster.GetNamespace(),
		}, sshConfig); err != nil {
			assert.NilError(t, err)
		}
		assert.Assert(t, len(sshConfig.Data) > 0)

		var foundSSHConfig, foundSSHDConfig bool
		for k, v := range sshConfig.Data {
			if v != "" {
				if k == "ssh_config" {
					foundSSHConfig = true
				} else if k == "sshd_config" {
					foundSSHDConfig = true
				}
			}
		}
		assert.Check(t, foundSSHConfig)
		assert.Check(t, foundSSHDConfig)

		sshSecret := &v1.Secret{}
		if err := tClient.Get(ctx, types.NamespacedName{
			Name:      naming.PGBackRestSSHSecret(postgresCluster).Name,
			Namespace: postgresCluster.GetNamespace(),
		}, sshSecret); err != nil {
			assert.NilError(t, err)
		}
		assert.Assert(t, len(sshSecret.Data) > 0)

		var foundPubKey, foundPrivKey, foundKnownHosts bool
		for k, v := range sshSecret.Data {
			if len(v) > 0 {
				if k == "id_ecdsa.pub" {
					foundPubKey = true
				} else if k == "id_ecdsa" {
					foundPrivKey = true
				} else if k == "ssh_known_hosts" {
					foundKnownHosts = true
				}
			}
		}
		assert.Check(t, foundPubKey)
		assert.Check(t, foundPrivKey)
		assert.Check(t, foundKnownHosts)
	})
}
