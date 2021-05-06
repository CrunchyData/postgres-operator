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
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
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
	testCronSchedule := "*/15 * * * *"

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
					Global: map[string]string{"repo2-test": "config",
						"repo3-test": "config", "repo4-test": "config"},
					Repos: []v1beta1.PGBackRestRepo{{
						Name: "repo1",
						BackupSchedules: &v1beta1.PGBackRestBackupSchedules{
							Full: &testCronSchedule,
						},
						Volume: &v1beta1.RepoPVC{
							VolumeClaimSpec: v1.PersistentVolumeClaimSpec{
								AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
								Resources: v1.ResourceRequirements{
									Requests: map[v1.ResourceName]resource.Quantity{
										v1.ResourceStorage: resource.MustParse("1Gi"),
									},
								},
							},
						},
					}, {
						Name: "repo2",
						Azure: &v1beta1.RepoAzure{
							Container: "container",
						},
					}, {
						Name: "repo3",
						GCS: &v1beta1.RepoGCS{
							Bucket: "bucket",
						},
					}, {
						Name: "repo4",
						S3: &v1beta1.RepoS3{
							Bucket:   "bucket",
							Endpoint: "endpoint",
							Region:   "region",
						},
					}},
					RepoHost: &v1beta1.PGBackRestRepoHost{
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
		t.Errorf("unable to reconcile pgBackRest: %v", err)
	}

	// repo is the first defined repo
	repo := postgresCluster.Spec.Archive.PGBackRest.Repos[0]

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
			if r.Volume == nil {
				continue
			}
			var foundRepoVol bool
			for _, v := range repoVols.Items {
				if v.GetName() ==
					naming.PGBackRestRepoVolume(postgresCluster, r.Name).Name {
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

	t.Run("verify stanza creation", func(t *testing.T) {

		clusterCopy := postgresCluster.DeepCopy()

		stanzaCreateFail := func(namespace, pod, container string, stdin io.Reader, stdout,
			stderr io.Writer, command ...string) error {
			return errors.New("fake stanza create failed")
		}

		stanzaCreateSuccess := func(namespace, pod, container string, stdin io.Reader, stdout,
			stderr io.Writer, command ...string) error {
			return nil
		}

		// first add a fake dedicated repo pod to the env
		repoHost := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fake-repo-host",
				Namespace: namespace,
				Labels:    naming.PGBackRestDedicatedLabels(clusterName),
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{{Name: "test", Image: "test"}},
			},
		}
		assert.NilError(t, r.Client.Create(ctx, repoHost))

		assert.NilError(t, wait.Poll(time.Second/2, time.Second*3, func() (bool, error) {
			if err := r.Client.Get(ctx,
				client.ObjectKeyFromObject(repoHost), &corev1.Pod{}); err != nil {
				return false, nil
			}
			return true, nil
		}))

		// first verify a stanza create success
		r.PodExec = stanzaCreateSuccess
		meta.SetStatusCondition(&clusterCopy.Status.Conditions, metav1.Condition{
			ObservedGeneration: clusterCopy.GetGeneration(),
			Type:               ConditionRepoHostReady,
			Status:             metav1.ConditionTrue,
			Reason:             "RepoHostReady",
			Message:            "pgBackRest dedicated repository host is ready",
		})
		clusterCopy.Status.Patroni = &v1beta1.PatroniStatus{
			SystemIdentifier: "6952526174828511264",
		}

		result, err := r.reconcilePGBackRest(ctx, clusterCopy, instanceNames)
		assert.NilError(t, err)
		assert.Assert(t, result == (reconcile.Result{}))

		events := &corev1.EventList{}
		err = wait.Poll(time.Second/2, time.Second*2, func() (bool, error) {
			if err := tClient.List(ctx, events, &client.MatchingFields{
				"involvedObject.kind":      "PostgresCluster",
				"involvedObject.name":      clusterName,
				"involvedObject.namespace": namespace,
				"involvedObject.uid":       string(clusterUID),
				"reason":                   "StanzasCreated",
			}); err != nil {
				return false, err
			}
			if len(events.Items) != 1 {
				return false, nil
			}
			return true, nil
		})
		assert.NilError(t, err)

		// status should indicate stanzas were created
		for _, r := range clusterCopy.Status.PGBackRest.Repos {
			assert.Assert(t, r.StanzaCreated)
		}

		// now verify failure event
		clusterCopy = postgresCluster.DeepCopy()
		r.PodExec = stanzaCreateFail
		meta.SetStatusCondition(&clusterCopy.Status.Conditions, metav1.Condition{
			ObservedGeneration: clusterCopy.GetGeneration(),
			Type:               ConditionRepoHostReady,
			Status:             metav1.ConditionTrue,
			Reason:             "RepoHostReady",
			Message:            "pgBackRest dedicated repository host is ready",
		})
		clusterCopy.Status.Patroni = &v1beta1.PatroniStatus{
			SystemIdentifier: "6952526174828511264",
		}

		result, err = r.reconcilePGBackRest(ctx, clusterCopy, instanceNames)
		assert.NilError(t, err)
		assert.Assert(t, result != (reconcile.Result{}))

		events = &corev1.EventList{}
		err = wait.Poll(time.Second/2, time.Second*2, func() (bool, error) {
			if err := tClient.List(ctx, events, &client.MatchingFields{
				"involvedObject.kind":      "PostgresCluster",
				"involvedObject.name":      clusterName,
				"involvedObject.namespace": namespace,
				"involvedObject.uid":       string(clusterUID),
				"reason":                   "UnableToCreateStanzas",
			}); err != nil {
				return false, err
			}
			if len(events.Items) != 1 {
				return false, nil
			}
			return true, nil
		})
		assert.NilError(t, err)

		// status should indicate stanaza were not created
		for _, r := range clusterCopy.Status.PGBackRest.Repos {
			assert.Assert(t, !r.StanzaCreated)
		}
	})

	t.Run("verify pgbackrest schedule cronjob", func(t *testing.T) {
		requeue := r.reconcilePGBackRestCronJob(context.Background(), postgresCluster)
		assert.Assert(t, !requeue)

		returnedCronJob := &batchv1beta1.CronJob{}
		if err := tClient.Get(ctx, types.NamespacedName{
			Name:      postgresCluster.Name + "-pgbackrest-repo1-full",
			Namespace: postgresCluster.GetNamespace(),
		}, returnedCronJob); err != nil {
			assert.NilError(t, err)
		}

		// check returned cronjob matches set spec
		assert.Equal(t, returnedCronJob.Name, "hippocluster-pgbackrest-repo1-full")
		assert.Equal(t, returnedCronJob.Spec.Schedule, testCronSchedule)
		assert.Equal(t, returnedCronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Name,
			"pgbackrest")

	})

	t.Run("verify pgbackrest schedule found", func(t *testing.T) {

		assert.Assert(t, backupScheduleFound(repo, "full"))

		testrepo := v1beta1.PGBackRestRepo{
			Name: "repo1",
			BackupSchedules: &v1beta1.PGBackRestBackupSchedules{
				Full:         &testCronSchedule,
				Differential: &testCronSchedule,
				Incremental:  &testCronSchedule,
			}}

		assert.Assert(t, backupScheduleFound(testrepo, "full"))
		assert.Assert(t, backupScheduleFound(testrepo, "diff"))
		assert.Assert(t, backupScheduleFound(testrepo, "incr"))

	})

	t.Run("verify pgbackrest schedule not found", func(t *testing.T) {

		assert.Assert(t, !backupScheduleFound(repo, "notabackuptype"))

		noscheduletestrepo := v1beta1.PGBackRestRepo{Name: "repo1"}
		assert.Assert(t, !backupScheduleFound(noscheduletestrepo, "full"))

	})

}
