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
	"fmt"
	"io"
	"strconv"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/pgbackrest"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

var testCronSchedule string = "*/15 * * * *"

func fakePostgresCluster(clusterName, namespace, clusterUID string,
	includeDedicatedRepo bool) *v1beta1.PostgresCluster {
	postgresCluster := &v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: namespace,
			UID:       types.UID(clusterUID),
		},
		Spec: v1beta1.PostgresClusterSpec{
			PostgresVersion: 13,
			InstanceSets:    []v1beta1.PostgresInstanceSetSpec{},
			Archive: v1beta1.Archive{
				PGBackRest: v1beta1.PGBackRestArchive{
					Image: "test.com/crunchy-pgbackrest:test",
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
				},
			},
		},
	}

	if includeDedicatedRepo {
		postgresCluster.Spec.Archive.PGBackRest.RepoHost = &v1beta1.PGBackRestRepoHost{
			Dedicated: &v1beta1.DedicatedRepo{
				Resources: &corev1.ResourceRequirements{},
			},
		}
	}

	return postgresCluster
}

func TestReconcilePGBackRest(t *testing.T) {

	// setup the test environment and ensure a clean teardown
	tEnv, tClient, cfg := setupTestEnv(t, ControllerName)
	t.Cleanup(func() { teardownTestEnv(t, tEnv) })
	r := &Reconciler{}
	ctx, cancel := setupManager(t, cfg, func(mgr manager.Manager) {
		r = &Reconciler{
			Client:   mgr.GetClient(),
			Recorder: mgr.GetEventRecorderFor(ControllerName),
			Tracer:   otel.Tracer(ControllerName),
			Owner:    ControllerName,
		}
	})
	t.Cleanup(func() { teardownManager(cancel, t) })

	clusterName := "hippocluster"
	clusterUID := "hippouid"

	ns := &v1.Namespace{}
	ns.GenerateName = "postgres-operator-test-"
	assert.NilError(t, tClient.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, tClient.Delete(ctx, ns)) })
	namespace := ns.Name

	// create a PostgresCluster to test with
	postgresCluster := fakePostgresCluster(clusterName, ns.GetName(), clusterUID, true)

	instances := &observedInstances{
		forCluster: []*Instance{{Name: "instance1"}, {Name: "instance2"}, {Name: "instance3"}},
	}
	result, err := r.reconcilePGBackRest(ctx, postgresCluster, instances)
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

		for _, instance := range instances.forCluster {
			var instanceConfFound, dedicatedRepoConfFound bool
			for k, v := range config.Data {
				if v != "" {
					if k == instance.Name+".conf" {
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

func TestReconcilePGBackRestRBAC(t *testing.T) {

	// setup the test environment and ensure a clean teardown
	tEnv, tClient, cfg := setupTestEnv(t, ControllerName)
	t.Cleanup(func() { teardownTestEnv(t, tEnv) })
	r := &Reconciler{}
	ctx, cancel := setupManager(t, cfg, func(mgr manager.Manager) {
		r = &Reconciler{
			Client:   mgr.GetClient(),
			Recorder: mgr.GetEventRecorderFor(ControllerName),
			Tracer:   otel.Tracer(ControllerName),
			Owner:    ControllerName,
		}
	})
	t.Cleanup(func() { teardownManager(cancel, t) })

	clusterName := "hippocluster"
	clusterUID := "hippouid"

	ns := &v1.Namespace{}
	ns.GenerateName = "postgres-operator-test-"
	assert.NilError(t, tClient.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, tClient.Delete(ctx, ns)) })

	// create a PostgresCluster to test with
	postgresCluster := fakePostgresCluster(clusterName, ns.GetName(), clusterUID, true)
	postgresCluster.Status.PGBackRest = &v1beta1.PGBackRestStatus{
		Repos: []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: false}},
	}

	serviceAccount, err := r.reconcilePGBackRestRBAC(ctx, postgresCluster)
	assert.NilError(t, err)
	assert.Assert(t, serviceAccount != nil)

	// first verify the service account has been created
	sa := &corev1.ServiceAccount{}
	err = tClient.Get(ctx, types.NamespacedName{
		Name:      naming.PGBackRestRBAC(postgresCluster).Name,
		Namespace: postgresCluster.GetNamespace(),
	}, sa)
	assert.NilError(t, err)

	role := &rbacv1.Role{}
	err = tClient.Get(ctx, types.NamespacedName{
		Name:      naming.PGBackRestRBAC(postgresCluster).Name,
		Namespace: postgresCluster.GetNamespace(),
	}, role)
	assert.NilError(t, err)
	assert.Assert(t, len(role.Rules) > 0)

	roleBinding := &rbacv1.RoleBinding{}
	err = tClient.Get(ctx, types.NamespacedName{
		Name:      naming.PGBackRestRBAC(postgresCluster).Name,
		Namespace: postgresCluster.GetNamespace(),
	}, roleBinding)
	assert.NilError(t, err)
	assert.Assert(t, roleBinding.RoleRef.Name == role.GetName())

	var foundSubject bool
	for _, subject := range roleBinding.Subjects {
		if subject.Name == sa.GetName() {
			foundSubject = true
		}
	}
	assert.Assert(t, foundSubject)
}

func TestReconcileStanzaCreate(t *testing.T) {

	// setup the test environment and ensure a clean teardown
	tEnv, tClient, cfg := setupTestEnv(t, ControllerName)
	t.Cleanup(func() { teardownTestEnv(t, tEnv) })
	r := &Reconciler{}
	ctx, cancel := setupManager(t, cfg, func(mgr manager.Manager) {
		r = &Reconciler{
			Client:   mgr.GetClient(),
			Recorder: mgr.GetEventRecorderFor(ControllerName),
			Tracer:   otel.Tracer(ControllerName),
			Owner:    ControllerName,
		}
	})
	t.Cleanup(func() { teardownManager(cancel, t) })

	clusterName := "hippocluster"
	clusterUID := "hippouid"

	ns := &v1.Namespace{}
	ns.GenerateName = "postgres-operator-test-"
	assert.NilError(t, tClient.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, tClient.Delete(ctx, ns)) })
	namespace := ns.Name

	// create a PostgresCluster to test with
	postgresCluster := fakePostgresCluster(clusterName, ns.GetName(), clusterUID, true)
	postgresCluster.Status.PGBackRest = &v1beta1.PGBackRestStatus{
		Repos: []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: false}},
	}

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

	err := wait.Poll(time.Second/2, time.Second*3, func() (bool, error) {
		if err := r.Client.Get(ctx,
			client.ObjectKeyFromObject(repoHost), &corev1.Pod{}); err != nil {
			return false, nil
		}
		return true, nil
	})
	assert.NilError(t, err)

	// now verify a stanza create success
	r.PodExec = stanzaCreateSuccess
	meta.SetStatusCondition(&postgresCluster.Status.Conditions, metav1.Condition{
		ObservedGeneration: postgresCluster.GetGeneration(),
		Type:               ConditionRepoHostReady,
		Status:             metav1.ConditionTrue,
		Reason:             "RepoHostReady",
		Message:            "pgBackRest dedicated repository host is ready",
	})
	postgresCluster.Status.Patroni = &v1beta1.PatroniStatus{
		SystemIdentifier: "6952526174828511264",
	}

	configHashMistmatch, err := r.reconcileStanzaCreate(ctx, postgresCluster, "abcde12345")
	assert.NilError(t, err)
	assert.Assert(t, !configHashMistmatch)

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
	for _, r := range postgresCluster.Status.PGBackRest.Repos {
		assert.Assert(t, r.StanzaCreated)
	}

	// now verify failure event
	postgresCluster = fakePostgresCluster(clusterName, ns.GetName(), clusterUID, true)
	postgresCluster.Status.PGBackRest = &v1beta1.PGBackRestStatus{
		Repos: []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: false}},
	}
	r.PodExec = stanzaCreateFail
	meta.SetStatusCondition(&postgresCluster.Status.Conditions, metav1.Condition{
		ObservedGeneration: postgresCluster.GetGeneration(),
		Type:               ConditionRepoHostReady,
		Status:             metav1.ConditionTrue,
		Reason:             "RepoHostReady",
		Message:            "pgBackRest dedicated repository host is ready",
	})
	postgresCluster.Status.Patroni = &v1beta1.PatroniStatus{
		SystemIdentifier: "6952526174828511264",
	}

	configHashMismatch, err := r.reconcileStanzaCreate(ctx, postgresCluster, "abcde12345")
	assert.Error(t, err, "fake stanza create failed: ")
	assert.Assert(t, !configHashMismatch)

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
	for _, r := range postgresCluster.Status.PGBackRest.Repos {
		assert.Assert(t, !r.StanzaCreated)
	}
}

func TestGetPGBackRestExecSelector(t *testing.T) {

	testCases := []struct {
		cluster           *v1beta1.PostgresCluster
		desc              string
		expectedSelector  string
		expectedContainer string
	}{{
		desc: "dedicated repo host enabled",
		cluster: &v1beta1.PostgresCluster{
			ObjectMeta: metav1.ObjectMeta{Name: "hippo"},
			Spec: v1beta1.PostgresClusterSpec{
				Archive: v1beta1.Archive{
					PGBackRest: v1beta1.PGBackRestArchive{
						RepoHost: &v1beta1.PGBackRestRepoHost{
							Dedicated: &v1beta1.DedicatedRepo{},
						},
					},
				},
			},
		},
		expectedSelector: "postgres-operator.crunchydata.com/cluster=hippo," +
			"postgres-operator.crunchydata.com/pgbackrest=," +
			"postgres-operator.crunchydata.com/pgbackrest-dedicated=," +
			"postgres-operator.crunchydata.com/pgbackrest-host=",
		expectedContainer: "pgbackrest",
	}, {
		desc: "repo host enabled",
		cluster: &v1beta1.PostgresCluster{
			ObjectMeta: metav1.ObjectMeta{Name: "hippo"},
			Spec: v1beta1.PostgresClusterSpec{
				Archive: v1beta1.Archive{
					PGBackRest: v1beta1.PGBackRestArchive{
						RepoHost: &v1beta1.PGBackRestRepoHost{},
					},
				},
			},
		},
		expectedSelector: "postgres-operator.crunchydata.com/cluster=hippo," +
			"postgres-operator.crunchydata.com/instance," +
			"postgres-operator.crunchydata.com/role=master",
		expectedContainer: "pgbackrest",
	}, {
		desc: "no repo host enabled",
		cluster: &v1beta1.PostgresCluster{
			ObjectMeta: metav1.ObjectMeta{Name: "hippo"},
			Spec: v1beta1.PostgresClusterSpec{
				Archive: v1beta1.Archive{
					PGBackRest: v1beta1.PGBackRestArchive{},
				},
			},
		},
		expectedSelector: "postgres-operator.crunchydata.com/cluster=hippo," +
			"postgres-operator.crunchydata.com/instance," +
			"postgres-operator.crunchydata.com/role=master",
		expectedContainer: "database",
	}}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			selector, container, err := getPGBackRestExecSelector(tc.cluster)
			assert.NilError(t, err)
			assert.Assert(t, selector.String() == tc.expectedSelector)
			assert.Assert(t, container == tc.expectedContainer)
		})
	}
}

func TestReconcileReplicaCreateBackup(t *testing.T) {

	// setup the test environment and ensure a clean teardown
	tEnv, tClient, cfg := setupTestEnv(t, ControllerName)
	t.Cleanup(func() { teardownTestEnv(t, tEnv) })
	r := &Reconciler{}
	ctx, cancel := setupManager(t, cfg, func(mgr manager.Manager) {
		r = &Reconciler{
			Client:   mgr.GetClient(),
			Recorder: mgr.GetEventRecorderFor(ControllerName),
			Tracer:   otel.Tracer(ControllerName),
			Owner:    ControllerName,
		}
	})
	t.Cleanup(func() { teardownManager(cancel, t) })

	clusterName := "hippocluster"
	clusterUID := "hippouid"

	ns := &v1.Namespace{}
	ns.GenerateName = "postgres-operator-test-"
	assert.NilError(t, tClient.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, tClient.Delete(ctx, ns)) })
	namespace := ns.Name

	// create a PostgresCluster to test with
	postgresCluster := fakePostgresCluster(clusterName, ns.GetName(), clusterUID, true)
	// set status for the "replica create" repo, e.g. the repo ad index 0
	postgresCluster.Status.PGBackRest = &v1beta1.PGBackRestStatus{
		Repos: []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: false}},
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

	err := wait.Poll(time.Second/2, time.Second*3, func() (bool, error) {
		if err := r.Client.Get(ctx,
			client.ObjectKeyFromObject(repoHost), &corev1.Pod{}); err != nil {
			return false, nil
		}
		return true, nil
	})
	assert.NilError(t, err)

	meta.SetStatusCondition(&postgresCluster.Status.Conditions, metav1.Condition{
		ObservedGeneration: postgresCluster.GetGeneration(),
		Type:               ConditionRepoHostReady,
		Status:             metav1.ConditionTrue,
		Reason:             "RepoHostReady",
		Message:            "pgBackRest dedicated repository host is ready",
	})
	meta.SetStatusCondition(&postgresCluster.Status.Conditions, metav1.Condition{
		ObservedGeneration: postgresCluster.GetGeneration(),
		Type:               ConditionReplicaRepoReady,
		Status:             metav1.ConditionTrue,
		Reason:             "StanzaCreated",
		Message:            "pgBackRest replica create repo is ready for backups",
	})
	postgresCluster.Status.Patroni = &v1beta1.PatroniStatus{
		SystemIdentifier: "6952526174828511264",
	}

	replicaCreateRepo := postgresCluster.Spec.Archive.PGBackRest.Repos[0].Name
	configHash := "abcde12345"

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "hippo-sa"},
	}

	err = r.reconcileReplicaCreateBackup(ctx, postgresCluster, []*batchv1.Job{}, sa,
		configHash, replicaCreateRepo)
	assert.NilError(t, err)

	// now find the expected job
	jobs := &batchv1.JobList{}
	err = tClient.List(ctx, jobs, &client.ListOptions{
		LabelSelector: naming.PGBackRestBackupJobSelector(clusterName, replicaCreateRepo,
			naming.BackupReplicaCreate),
	})
	assert.NilError(t, err)
	assert.Assert(t, len(jobs.Items) == 1)
	backupJob := jobs.Items[0]

	var foundOwnershipRef bool
	// verify ownership refs
	for _, ref := range backupJob.ObjectMeta.GetOwnerReferences() {
		if ref.Name == clusterName {
			foundOwnershipRef = true
			break
		}
	}
	assert.Assert(t, foundOwnershipRef)

	var foundConfigAnnotation, foundHashAnnotation bool
	// verify annotations
	for k, v := range backupJob.GetAnnotations() {
		if k == naming.PGBackRestCurrentConfig && v == pgbackrest.CMRepoKey {
			foundConfigAnnotation = true
		}
		if k == naming.PGBackRestConfigHash && v == configHash {
			foundHashAnnotation = true
		}
	}
	assert.Assert(t, foundConfigAnnotation)
	assert.Assert(t, foundHashAnnotation)

	// verify container & env vars
	assert.Assert(t, len(backupJob.Spec.Template.Spec.Containers) == 1)
	assert.Assert(t,
		backupJob.Spec.Template.Spec.Containers[0].Name == naming.PGBackRestRepoContainerName)
	container := backupJob.Spec.Template.Spec.Containers[0]
	for _, env := range container.Env {
		switch env.Name {
		case "COMMAND":
			assert.Assert(t, env.Value == "backup")
		case "COMMAND_OPTS":
			assert.Assert(t, env.Value == "--stanza=db --repo=1")
		case "COMPARE_HASH":
			assert.Assert(t, env.Value == "true")
		case "CONTAINER":
			assert.Assert(t, env.Value == naming.PGBackRestRepoContainerName)
		case "NAMESPACE":
			assert.Assert(t, env.Value == namespace)
		case "SELECTOR":
			assert.Assert(t, env.Value == "postgres-operator.crunchydata.com/cluster=hippocluster,"+
				"postgres-operator.crunchydata.com/pgbackrest=,"+
				"postgres-operator.crunchydata.com/pgbackrest-dedicated=,"+
				"postgres-operator.crunchydata.com/pgbackrest-host=")
		}
	}
	// verify mounted configuration is present
	assert.Assert(t, len(container.VolumeMounts) == 1)

	// verify volume for configuration is present
	assert.Assert(t, len(backupJob.Spec.Template.Spec.Volumes) == 1)

	// now set the job to complete
	backupJob.Status.Conditions = append(backupJob.Status.Conditions,
		batchv1.JobCondition{Type: batchv1.JobComplete, Status: corev1.ConditionTrue})

	// call reconcile function again
	err = r.reconcileReplicaCreateBackup(ctx, postgresCluster, []*batchv1.Job{&backupJob}, sa,
		configHash, replicaCreateRepo)
	assert.NilError(t, err)

	// verify the proper conditions have been set
	var foundCompletedCondition bool
	condition := meta.FindStatusCondition(postgresCluster.Status.Conditions, ConditionReplicaCreate)
	if condition != nil && (condition.Status == metav1.ConditionTrue) {
		foundCompletedCondition = true
	}
	assert.Assert(t, foundCompletedCondition)

	// verify the status has been updated properly
	var replicaCreateRepoStatus *v1beta1.RepoStatus
	for i, r := range postgresCluster.Status.PGBackRest.Repos {
		if r.Name == replicaCreateRepo {
			replicaCreateRepoStatus = &postgresCluster.Status.PGBackRest.Repos[i]
			break
		}
	}
	if assert.Check(t, replicaCreateRepoStatus != nil) {
		assert.Assert(t, replicaCreateRepoStatus.ReplicaCreateBackupComplete)
	}
}

func TestReconcileManualBackup(t *testing.T) {

	// setup the test environment and ensure a clean teardown
	tEnv, tClient, cfg := setupTestEnv(t, ControllerName)
	t.Cleanup(func() { teardownTestEnv(t, tEnv) })
	r := &Reconciler{}
	ctx, cancel := setupManager(t, cfg, func(mgr manager.Manager) {
		r = &Reconciler{
			Client:   mgr.GetClient(),
			Recorder: mgr.GetEventRecorderFor(ControllerName),
			Tracer:   otel.Tracer(ControllerName),
			Owner:    ControllerName,
		}
	})
	t.Cleanup(func() { teardownManager(cancel, t) })

	ns := &v1.Namespace{}
	ns.GenerateName = "postgres-operator-test-"
	assert.NilError(t, tClient.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, tClient.Delete(ctx, ns)) })

	defaultBackupId := "default-backup-id"
	backupId := metav1.Now().OpenAPISchemaFormat()

	fakeJob := func(clusterName, repoName string) *batchv1.Job {
		return &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "manual-backup-" + rand.String(4),
				Namespace:   ns.GetName(),
				Annotations: map[string]string{naming.PGBackRestBackup: defaultBackupId},
				Labels: naming.PGBackRestBackupJobLabels(clusterName, repoName,
					naming.BackupManual),
			},
		}
	}

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "hippo-sa"},
	}

	instances := &observedInstances{
		forCluster: []*Instance{{
			Name: "instance1",
			Pods: []*v1.Pod{{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{naming.LabelRole: naming.RolePatroniLeader},
				},
			}},
		}},
	}

	testCases := []struct {
		// a description of the test
		testDesc string
		// whether or not the test only applies to configs with dedicated repo hosts
		dedicatedOnly bool
		// whether or not to mock a current job in the env before reonciling (this job is not
		// actully created, but rather just passed into the reconcile function under test)
		createCurrentJob bool
		// conditions to apply to the job if created (these are always set to "true")
		jobConditions []batchv1.JobConditionType
		// conditions to apply to the mock postgrescluster
		clusterConditions map[string]metav1.ConditionStatus
		// the status to apply to the mock postgrescluster
		status *v1beta1.PostgresClusterStatus
		// the ID used to populate the "backup" annotation for the test (can be empty)
		backupId string
		// the manual backup field to define in the postgrescluster spec for the test
		manual *v1beta1.PGBackRestManualBackup
		// whether or not the test should expect a Job to be reconciled
		expectReconcile bool
		// whether or not the test should expect a current job in the env to be deleted
		expectCurrentJobDeletion bool
		// the reason associated with the expected event for the test (can be empty if
		// no event is expected)
		expectedEventReason string
	}{{
		testDesc:         "cluster not bootstrapped should not reconcile",
		createCurrentJob: false,
		clusterConditions: map[string]metav1.ConditionStatus{
			ConditionRepoHostReady: metav1.ConditionTrue,
			ConditionReplicaCreate: metav1.ConditionTrue,
		},
		status: &v1beta1.PostgresClusterStatus{
			PGBackRest: &v1beta1.PGBackRestStatus{
				Repos: []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: true}}},
		},
		backupId:                 backupId,
		manual:                   &v1beta1.PGBackRestManualBackup{RepoName: "repo1"},
		expectCurrentJobDeletion: false,
		expectReconcile:          false,
	}, {
		testDesc:          "no conditions should not reconcile",
		createCurrentJob:  false,
		clusterConditions: map[string]metav1.ConditionStatus{},
		status: &v1beta1.PostgresClusterStatus{
			Patroni: &v1beta1.PatroniStatus{SystemIdentifier: "12345abcde"},
			PGBackRest: &v1beta1.PGBackRestStatus{
				Repos: []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: true}}},
		},
		backupId:                 backupId,
		manual:                   &v1beta1.PGBackRestManualBackup{RepoName: "repo1"},
		expectCurrentJobDeletion: false,
		expectReconcile:          false,
	}, {
		testDesc:         "no repo host ready condition should not reconcile",
		dedicatedOnly:    true,
		createCurrentJob: false,
		clusterConditions: map[string]metav1.ConditionStatus{
			ConditionReplicaCreate: metav1.ConditionTrue,
		},
		status: &v1beta1.PostgresClusterStatus{
			Patroni: &v1beta1.PatroniStatus{SystemIdentifier: "12345abcde"},
			PGBackRest: &v1beta1.PGBackRestStatus{
				Repos: []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: true}}},
		},
		backupId:                 backupId,
		manual:                   &v1beta1.PGBackRestManualBackup{RepoName: "repo1"},
		expectCurrentJobDeletion: false,
		expectReconcile:          false,
	}, {
		testDesc:         "no replica create condition should not reconcile",
		createCurrentJob: false,
		clusterConditions: map[string]metav1.ConditionStatus{
			ConditionRepoHostReady: metav1.ConditionTrue,
		},
		status: &v1beta1.PostgresClusterStatus{
			Patroni: &v1beta1.PatroniStatus{SystemIdentifier: "12345abcde"},
			PGBackRest: &v1beta1.PGBackRestStatus{
				Repos: []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: true}}},
		},
		backupId:                 backupId,
		manual:                   &v1beta1.PGBackRestManualBackup{RepoName: "repo1"},
		expectCurrentJobDeletion: false,
		expectReconcile:          false,
	}, {
		testDesc:         "false repo host ready condition should not reconcile",
		dedicatedOnly:    true,
		createCurrentJob: false,
		clusterConditions: map[string]metav1.ConditionStatus{
			ConditionRepoHostReady: metav1.ConditionFalse,
			ConditionReplicaCreate: metav1.ConditionTrue,
		},
		status: &v1beta1.PostgresClusterStatus{
			Patroni: &v1beta1.PatroniStatus{SystemIdentifier: "12345abcde"},
			PGBackRest: &v1beta1.PGBackRestStatus{
				Repos: []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: true}}},
		},
		backupId:                 backupId,
		manual:                   &v1beta1.PGBackRestManualBackup{RepoName: "repo1"},
		expectCurrentJobDeletion: false,
		expectReconcile:          false,
	}, {
		testDesc:         "false replica create condition should not reconcile",
		createCurrentJob: false,
		clusterConditions: map[string]metav1.ConditionStatus{
			ConditionRepoHostReady: metav1.ConditionTrue,
			ConditionReplicaCreate: metav1.ConditionFalse,
		},
		status: &v1beta1.PostgresClusterStatus{
			Patroni: &v1beta1.PatroniStatus{SystemIdentifier: "12345abcde"},
			PGBackRest: &v1beta1.PGBackRestStatus{
				Repos: []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: true}}},
		},
		backupId:                 backupId,
		manual:                   &v1beta1.PGBackRestManualBackup{RepoName: "repo1"},
		expectCurrentJobDeletion: false,
		expectReconcile:          false,
	}, {
		testDesc:         "no manual backup defined should not reconcile",
		createCurrentJob: false,
		clusterConditions: map[string]metav1.ConditionStatus{
			ConditionRepoHostReady: metav1.ConditionTrue,
			ConditionReplicaCreate: metav1.ConditionTrue,
		},
		status: &v1beta1.PostgresClusterStatus{
			Patroni: &v1beta1.PatroniStatus{SystemIdentifier: "12345abcde"},
			PGBackRest: &v1beta1.PGBackRestStatus{
				Repos: []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: true}}},
		},
		backupId:                 backupId,
		manual:                   nil,
		expectCurrentJobDeletion: false,
		expectReconcile:          false,
	}, {
		testDesc:         "manual backup already complete should not reconcile",
		createCurrentJob: false,
		clusterConditions: map[string]metav1.ConditionStatus{
			ConditionRepoHostReady: metav1.ConditionTrue,
			ConditionReplicaCreate: metav1.ConditionTrue,
		},
		status: &v1beta1.PostgresClusterStatus{
			Patroni: &v1beta1.PatroniStatus{SystemIdentifier: "12345abcde"},
			PGBackRest: &v1beta1.PGBackRestStatus{
				ManualBackup: &v1beta1.PGBackRestManualBackupStatus{
					ID: backupId, Finished: true},
				Repos: []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: true}}},
		},
		backupId:                 backupId,
		manual:                   nil,
		expectCurrentJobDeletion: false,
		expectReconcile:          false,
	}, {
		testDesc:         "empty backup annotation should not reconcile",
		createCurrentJob: false,
		clusterConditions: map[string]metav1.ConditionStatus{
			ConditionRepoHostReady: metav1.ConditionTrue,
			ConditionReplicaCreate: metav1.ConditionTrue,
		},
		status: &v1beta1.PostgresClusterStatus{
			Patroni: &v1beta1.PatroniStatus{SystemIdentifier: "12345abcde"},
			PGBackRest: &v1beta1.PGBackRestStatus{
				Repos: []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: true}}},
		},
		backupId:                 "",
		manual:                   &v1beta1.PGBackRestManualBackup{RepoName: "repo1"},
		expectCurrentJobDeletion: false,
		expectReconcile:          false,
	}, {
		testDesc:         "missing repo status should not reconcile",
		createCurrentJob: false,
		clusterConditions: map[string]metav1.ConditionStatus{
			ConditionRepoHostReady: metav1.ConditionTrue,
			ConditionReplicaCreate: metav1.ConditionTrue,
		},
		status: &v1beta1.PostgresClusterStatus{
			Patroni: &v1beta1.PatroniStatus{SystemIdentifier: "12345abcde"},
			PGBackRest: &v1beta1.PGBackRestStatus{
				Repos: []v1beta1.RepoStatus{}},
		},
		backupId:                 backupId,
		manual:                   &v1beta1.PGBackRestManualBackup{RepoName: "repo1"},
		expectCurrentJobDeletion: false,
		expectReconcile:          false,
		expectedEventReason:      "InvalidBackupRepo",
	}, {
		testDesc:         "reconcile job when no current job exists",
		createCurrentJob: false,
		clusterConditions: map[string]metav1.ConditionStatus{
			ConditionRepoHostReady: metav1.ConditionTrue,
			ConditionReplicaCreate: metav1.ConditionTrue,
		},
		status: &v1beta1.PostgresClusterStatus{
			Patroni: &v1beta1.PatroniStatus{SystemIdentifier: "12345abcde"},
			PGBackRest: &v1beta1.PGBackRestStatus{
				Repos: []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: true}}},
		},
		backupId:                 backupId,
		manual:                   &v1beta1.PGBackRestManualBackup{RepoName: "repo1"},
		expectCurrentJobDeletion: false,
		expectReconcile:          true,
	}, {
		testDesc:         "reconcile job when current job exists for id and is in progress",
		createCurrentJob: true,
		clusterConditions: map[string]metav1.ConditionStatus{
			ConditionRepoHostReady: metav1.ConditionTrue,
			ConditionReplicaCreate: metav1.ConditionTrue,
		},
		status: &v1beta1.PostgresClusterStatus{
			Patroni: &v1beta1.PatroniStatus{SystemIdentifier: "12345abcde"},
			PGBackRest: &v1beta1.PGBackRestStatus{
				Repos: []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: true}}},
		},
		backupId:                 defaultBackupId,
		manual:                   &v1beta1.PGBackRestManualBackup{RepoName: "repo1"},
		expectCurrentJobDeletion: false,
		expectReconcile:          true,
	}, {
		testDesc:         "reconcile new job when in-progess job exists for another id",
		createCurrentJob: true,
		clusterConditions: map[string]metav1.ConditionStatus{
			ConditionRepoHostReady: metav1.ConditionTrue,
			ConditionReplicaCreate: metav1.ConditionTrue,
		},
		status: &v1beta1.PostgresClusterStatus{
			Patroni: &v1beta1.PatroniStatus{SystemIdentifier: "12345abcde"},
			PGBackRest: &v1beta1.PGBackRestStatus{
				Repos: []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: true}}},
		},
		backupId:                 backupId,
		manual:                   &v1beta1.PGBackRestManualBackup{RepoName: "repo1"},
		expectCurrentJobDeletion: false,
		expectReconcile:          true,
	}, {
		testDesc:         "delete current job since job is complete and new backup id",
		createCurrentJob: true,
		jobConditions:    []batchv1.JobConditionType{batchv1.JobComplete},
		clusterConditions: map[string]metav1.ConditionStatus{
			ConditionRepoHostReady: metav1.ConditionTrue,
			ConditionReplicaCreate: metav1.ConditionTrue,
		},
		status: &v1beta1.PostgresClusterStatus{
			Patroni: &v1beta1.PatroniStatus{SystemIdentifier: "12345abcde"},
			PGBackRest: &v1beta1.PGBackRestStatus{
				Repos: []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: true}}},
		},
		backupId:                 backupId,
		manual:                   &v1beta1.PGBackRestManualBackup{RepoName: "repo1"},
		expectCurrentJobDeletion: true,
		expectReconcile:          false,
	}, {
		testDesc:         "delete current job since job is failed and new backup id",
		createCurrentJob: true,
		jobConditions:    []batchv1.JobConditionType{batchv1.JobFailed},
		clusterConditions: map[string]metav1.ConditionStatus{
			ConditionRepoHostReady: metav1.ConditionTrue,
			ConditionReplicaCreate: metav1.ConditionTrue,
		},
		status: &v1beta1.PostgresClusterStatus{
			Patroni: &v1beta1.PatroniStatus{SystemIdentifier: "12345abcde"},
			PGBackRest: &v1beta1.PGBackRestStatus{
				Repos: []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: true}}},
		},
		backupId:                 backupId,
		manual:                   &v1beta1.PGBackRestManualBackup{RepoName: "repo1"},
		expectCurrentJobDeletion: true,
		expectReconcile:          false,
	}}

	for _, dedicated := range []bool{true, false} {
		for i, tc := range testCases {
			var clusterName string
			if !dedicated {
				tc.testDesc = "no repo " + tc.testDesc
				clusterName = "manual-backup-no-repo-" + strconv.Itoa(i)
			} else {
				clusterName = "manual-backup-" + strconv.Itoa(i)
			}
			t.Run(tc.testDesc, func(t *testing.T) {

				if tc.dedicatedOnly && !dedicated {
					t.Skip()
				}

				ctx := context.Background()

				postgresCluster := fakePostgresCluster(clusterName, ns.GetName(), "", dedicated)
				postgresCluster.Spec.Archive.PGBackRest.Manual = tc.manual
				postgresCluster.Status = *tc.status
				postgresCluster.Annotations = map[string]string{naming.PGBackRestBackup: tc.backupId}
				for condition, status := range tc.clusterConditions {
					meta.SetStatusCondition(&postgresCluster.Status.Conditions, metav1.Condition{
						Type: condition, Reason: "testing", Status: status})
				}
				assert.NilError(t, tClient.Create(ctx, postgresCluster))
				assert.NilError(t, tClient.Status().Update(ctx, postgresCluster))

				currentJobs := []*batchv1.Job{}
				if tc.createCurrentJob {
					job := fakeJob(postgresCluster.GetName(), tc.manual.RepoName)
					job.Status.Conditions = []batchv1.JobCondition{}
					for _, c := range tc.jobConditions {
						job.Status.Conditions = append(job.Status.Conditions,
							batchv1.JobCondition{Type: c, Status: corev1.ConditionTrue})
					}
					currentJobs = append(currentJobs, job)
				}

				err := r.reconcileManualBackup(ctx, postgresCluster, currentJobs, sa, instances)

				if tc.expectReconcile {

					// verify expected behavior when a reconcile is expected

					assert.NilError(t, err)

					jobs := &batchv1.JobList{}
					err := tClient.List(ctx, jobs, &client.ListOptions{
						LabelSelector: naming.PGBackRestBackupJobSelector(clusterName,
							tc.manual.RepoName, naming.BackupManual),
					})
					assert.NilError(t, err)
					assert.Assert(t, len(jobs.Items) == 1)

					var foundOwnershipRef bool
					for _, r := range jobs.Items[0].GetOwnerReferences() {
						if r.Kind == "PostgresCluster" && r.Name == clusterName &&
							r.UID == postgresCluster.GetUID() {
							foundOwnershipRef = true
							break
						}
					}
					assert.Assert(t, foundOwnershipRef)

					// verify status is populated with the proper ID
					assert.Assert(t, postgresCluster.Status.PGBackRest.ManualBackup != nil)
					assert.Assert(t, postgresCluster.Status.PGBackRest.ManualBackup.ID != "")

					return
				} else {

					// verify expected results when a reconcile is not expected

					// if a deletion is expected, then an error is expected.  otherwise an error is
					// not expected.
					if tc.expectCurrentJobDeletion {
						assert.Assert(t, kerr.IsNotFound(err))
						assert.ErrorContains(t, err,
							fmt.Sprintf(`"%s" not found`, currentJobs[0].GetName()))
					} else {
						assert.NilError(t, err)
					}

					jobs := &batchv1.JobList{}
					// just use a pgbackrest selector to check for the existence of any job since
					// we might not have a repo name for tests within a manual backup defined
					err := tClient.List(ctx, jobs, &client.ListOptions{
						LabelSelector: naming.PGBackRestSelector(clusterName),
					})
					assert.NilError(t, err)
					assert.Assert(t, len(jobs.Items) == 0)

					// if an event is expected, the check for it
					if tc.expectedEventReason != "" {
						events := &corev1.EventList{}
						err = wait.Poll(time.Second/2, time.Second*2, func() (bool, error) {
							if err := tClient.List(ctx, events, &client.MatchingFields{
								"involvedObject.kind":      "PostgresCluster",
								"involvedObject.name":      clusterName,
								"involvedObject.namespace": ns.GetName(),
								"involvedObject.uid":       string(postgresCluster.GetUID()),
								"reason":                   tc.expectedEventReason,
							}); err != nil {
								return false, err
							}
							if len(events.Items) != 1 {
								return false, nil
							}
							return true, nil
						})
						assert.NilError(t, err)
					}
					return
				}
			})
		}
	}
}

func TestGetPGBackRestResources(t *testing.T) {

	// setup the test environment and ensure a clean teardown
	tEnv, tClient, cfg := setupTestEnv(t, ControllerName)
	t.Cleanup(func() { teardownTestEnv(t, tEnv) })
	r := &Reconciler{}
	ctx, cancel := setupManager(t, cfg, func(mgr manager.Manager) {
		r = &Reconciler{
			Client:   mgr.GetClient(),
			Recorder: mgr.GetEventRecorderFor(ControllerName),
			Tracer:   otel.Tracer(ControllerName),
			Owner:    ControllerName,
		}
	})
	t.Cleanup(func() { teardownManager(cancel, t) })

	clusterName := "hippocluster"
	clusterUID := "hippouid"

	ns := &v1.Namespace{}
	ns.GenerateName = "postgres-operator-test-"
	assert.NilError(t, tClient.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, tClient.Delete(ctx, ns)) })
	namespace := ns.Name

	type testResult struct {
		jobCount, hostCount, pvcCount      int
		sshConfigPresent, sshSecretPresent bool
	}

	testCases := []struct {
		desc            string
		createResources []client.Object
		cluster         *v1beta1.PostgresCluster
		result          testResult
	}{{
		desc: "repo still defined keep job",
		createResources: []client.Object{
			&batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "keep-job",
					Namespace: namespace,
					Labels: naming.PGBackRestBackupJobLabels(clusterName, "repo1",
						naming.BackupReplicaCreate),
				},
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers:    []v1.Container{{Name: "test", Image: "test"}},
							RestartPolicy: v1.RestartPolicyNever,
						},
					},
				},
			},
		},
		cluster: &v1beta1.PostgresCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: namespace,
				UID:       types.UID(clusterUID),
			},
			Spec: v1beta1.PostgresClusterSpec{
				Archive: v1beta1.Archive{
					PGBackRest: v1beta1.PGBackRestArchive{
						Repos: []v1beta1.PGBackRestRepo{{Name: "repo1"}},
					},
				},
			},
		},
		result: testResult{
			jobCount: 1, pvcCount: 0, hostCount: 0,
			sshConfigPresent: false, sshSecretPresent: false,
		},
	}, {
		desc: "repo no longer exists delete job",
		createResources: []client.Object{
			&batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "delete-job",
					Namespace: namespace,
					Labels: naming.PGBackRestBackupJobLabels(clusterName, "repo1",
						naming.BackupReplicaCreate),
				},
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers:    []v1.Container{{Name: "test", Image: "test"}},
							RestartPolicy: v1.RestartPolicyNever,
						},
					},
				},
			},
		},
		cluster: &v1beta1.PostgresCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: namespace,
				UID:       types.UID(clusterUID),
			},
			Spec: v1beta1.PostgresClusterSpec{
				Archive: v1beta1.Archive{
					PGBackRest: v1beta1.PGBackRestArchive{
						Repos: []v1beta1.PGBackRestRepo{{Name: "repo4"}},
					},
				},
			},
		},
		result: testResult{
			jobCount: 0, pvcCount: 0, hostCount: 0,
			sshConfigPresent: false, sshSecretPresent: false,
		},
	}, {
		desc: "repo still defined keep pvc",
		createResources: []client.Object{
			&corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "keep-pvc",
					Namespace: namespace,
					Labels:    naming.PGBackRestRepoVolumeLabels(clusterName, "repo1"),
				},
				Spec: v1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
				},
			},
		},
		cluster: &v1beta1.PostgresCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: namespace,
				UID:       types.UID(clusterUID),
			},
			Spec: v1beta1.PostgresClusterSpec{
				Archive: v1beta1.Archive{
					PGBackRest: v1beta1.PGBackRestArchive{
						Repos: []v1beta1.PGBackRestRepo{{
							Name:   "repo1",
							Volume: &v1beta1.RepoPVC{},
						}},
					},
				},
			},
		},
		result: testResult{
			jobCount: 0, pvcCount: 1, hostCount: 0,
			sshConfigPresent: false, sshSecretPresent: false,
		},
	}, {
		desc: "repo no longer exists delete pvc",
		createResources: []client.Object{
			&corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "delete-pvc",
					Namespace: namespace,
					Labels:    naming.PGBackRestRepoVolumeLabels(clusterName, "repo1"),
				},
				Spec: v1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
				},
			},
		},
		cluster: &v1beta1.PostgresCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: namespace,
				UID:       types.UID(clusterUID),
			},
			Spec: v1beta1.PostgresClusterSpec{
				Archive: v1beta1.Archive{
					PGBackRest: v1beta1.PGBackRestArchive{
						Repos: []v1beta1.PGBackRestRepo{{
							Name:   "repo4",
							Volume: &v1beta1.RepoPVC{},
						}},
					},
				},
			},
		},
		result: testResult{
			jobCount: 0, pvcCount: 0, hostCount: 0,
			sshConfigPresent: false, sshSecretPresent: false,
		},
	}, {
		desc: "dedicated repo host defined keep dedicated sts",
		createResources: []client.Object{
			&appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "keep-dedicated",
					Namespace: namespace,
					Labels:    naming.PGBackRestDedicatedLabels(clusterName),
				},
				Spec: appsv1.StatefulSetSpec{
					Selector: metav1.SetAsLabelSelector(
						naming.PGBackRestDedicatedLabels(clusterName)),
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: naming.PGBackRestDedicatedLabels(clusterName),
						},
						Spec: v1.PodSpec{},
					},
				},
			},
		},
		cluster: &v1beta1.PostgresCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: namespace,
				UID:       types.UID(clusterUID),
			},
			Spec: v1beta1.PostgresClusterSpec{
				Archive: v1beta1.Archive{
					PGBackRest: v1beta1.PGBackRestArchive{
						RepoHost: &v1beta1.PGBackRestRepoHost{
							Dedicated: &v1beta1.DedicatedRepo{},
						},
					},
				},
			},
		},
		result: testResult{
			jobCount: 0, pvcCount: 0, hostCount: 1,
			sshConfigPresent: false, sshSecretPresent: false,
		},
	}, {
		desc: "no dedicated repo host defined delete dedicated sts",
		createResources: []client.Object{
			&appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "delete-dedicated",
					Namespace: namespace,
					Labels:    naming.PGBackRestDedicatedLabels(clusterName),
				},
				Spec: appsv1.StatefulSetSpec{
					Selector: metav1.SetAsLabelSelector(
						naming.PGBackRestDedicatedLabels(clusterName)),
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: naming.PGBackRestDedicatedLabels(clusterName),
						},
						Spec: v1.PodSpec{},
					},
				},
			},
		},
		cluster: &v1beta1.PostgresCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: namespace,
				UID:       types.UID(clusterUID),
			},
			Spec: v1beta1.PostgresClusterSpec{
				Archive: v1beta1.Archive{
					PGBackRest: v1beta1.PGBackRestArchive{
						RepoHost: &v1beta1.PGBackRestRepoHost{},
					},
				},
			},
		},
		result: testResult{
			jobCount: 0, pvcCount: 0, hostCount: 0,
			sshConfigPresent: false, sshSecretPresent: false,
		},
	}, {
		desc: "no repo host defined delete dedicated sts",
		createResources: []client.Object{
			&appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "delete-dedicated-no-repo-host",
					Namespace: namespace,
					Labels:    naming.PGBackRestDedicatedLabels(clusterName),
				},
				Spec: appsv1.StatefulSetSpec{
					Selector: metav1.SetAsLabelSelector(
						naming.PGBackRestDedicatedLabels(clusterName)),
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: naming.PGBackRestDedicatedLabels(clusterName),
						},
						Spec: v1.PodSpec{},
					},
				},
			},
		},
		cluster: &v1beta1.PostgresCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: namespace,
				UID:       types.UID(clusterUID),
			},
			Spec: v1beta1.PostgresClusterSpec{
				Archive: v1beta1.Archive{
					PGBackRest: v1beta1.PGBackRestArchive{},
				},
			},
		},
		result: testResult{
			jobCount: 0, pvcCount: 0, hostCount: 0,
			sshConfigPresent: false, sshSecretPresent: false,
		},
	}, {
		desc: "dedicated repo host defined keep ssh configmap",
		createResources: []client.Object{
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					// cleanup logic is sensitive the name of this resource
					Name:      "keep-ssh-cm-ssh-config",
					Namespace: namespace,
					Labels:    naming.PGBackRestRepoHostLabels("keep-ssh-cm"),
				},
				Data: map[string]string{},
			},
		},
		cluster: &v1beta1.PostgresCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "keep-ssh-cm",
				Namespace: namespace,
				UID:       types.UID(clusterUID),
			},
			Spec: v1beta1.PostgresClusterSpec{
				Archive: v1beta1.Archive{
					PGBackRest: v1beta1.PGBackRestArchive{
						RepoHost: &v1beta1.PGBackRestRepoHost{
							Dedicated: &v1beta1.DedicatedRepo{},
						},
					},
				},
			},
		},
		result: testResult{
			jobCount: 0, pvcCount: 0, hostCount: 0,
			sshConfigPresent: true, sshSecretPresent: false,
		},
	}, {
		desc: "repo host defined keep ssh configmap",
		createResources: []client.Object{
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					// cleanup logic is sensitive the name of this resource
					Name:      "keep-ssh-cm-repo-host-ssh-config",
					Namespace: namespace,
					Labels:    naming.PGBackRestRepoHostLabels("keep-ssh-cm-repo-host"),
				},
				Data: map[string]string{},
			},
		},
		cluster: &v1beta1.PostgresCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "keep-ssh-cm-repo-host",
				Namespace: namespace,
				UID:       types.UID(clusterUID),
			},
			Spec: v1beta1.PostgresClusterSpec{
				Archive: v1beta1.Archive{
					PGBackRest: v1beta1.PGBackRestArchive{
						RepoHost: &v1beta1.PGBackRestRepoHost{},
					},
				},
			},
		},
		result: testResult{
			jobCount: 0, pvcCount: 0, hostCount: 0,
			sshConfigPresent: true, sshSecretPresent: false,
		},
	}, {
		desc: "no repo host defined keep delete configmap",
		createResources: []client.Object{
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					// cleanup logic is sensitive the name of this resource
					Name:      "delete-ssh-cm-ssh-config",
					Namespace: namespace,
					Labels:    naming.PGBackRestRepoHostLabels("delete-ssh-cm"),
				},
				Data: map[string]string{},
			},
		},
		cluster: &v1beta1.PostgresCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "delete-ssh-cm",
				Namespace: namespace,
				UID:       types.UID(clusterUID),
			},
			Spec: v1beta1.PostgresClusterSpec{
				Archive: v1beta1.Archive{
					PGBackRest: v1beta1.PGBackRestArchive{},
				},
			},
		},
		result: testResult{
			jobCount: 0, pvcCount: 0, hostCount: 0,
			sshConfigPresent: false, sshSecretPresent: false,
		},
	}, {
		desc: "dedicated repo host defined keep ssh secret",
		createResources: []client.Object{
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					// cleanup logic is sensitive the name of this resource
					Name:      "keep-ssh-secret-ssh",
					Namespace: namespace,
					Labels:    naming.PGBackRestRepoHostLabels("keep-ssh-secret"),
				},
				Data: map[string][]byte{},
			},
		},
		cluster: &v1beta1.PostgresCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "keep-ssh-secret",
				Namespace: namespace,
				UID:       types.UID(clusterUID),
			},
			Spec: v1beta1.PostgresClusterSpec{
				Archive: v1beta1.Archive{
					PGBackRest: v1beta1.PGBackRestArchive{
						RepoHost: &v1beta1.PGBackRestRepoHost{
							Dedicated: &v1beta1.DedicatedRepo{},
						},
					},
				},
			},
		},
		result: testResult{
			jobCount: 0, pvcCount: 0, hostCount: 0,
			sshConfigPresent: false, sshSecretPresent: true,
		},
	}, {
		desc: "repo host defined keep ssh secret",
		createResources: []client.Object{
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					// cleanup logic is sensitive the name of this resource
					Name:      "keep-ssh-secret-repo-host-ssh",
					Namespace: namespace,
					Labels:    naming.PGBackRestRepoHostLabels("keep-ssh-secret-repo-host"),
				},
				Data: map[string][]byte{},
			},
		},
		cluster: &v1beta1.PostgresCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "keep-ssh-secret-repo-host",
				Namespace: namespace,
				UID:       types.UID(clusterUID),
			},
			Spec: v1beta1.PostgresClusterSpec{
				Archive: v1beta1.Archive{
					PGBackRest: v1beta1.PGBackRestArchive{
						RepoHost: &v1beta1.PGBackRestRepoHost{},
					},
				},
			},
		},
		result: testResult{
			jobCount: 0, pvcCount: 0, hostCount: 0,
			sshConfigPresent: false, sshSecretPresent: true,
		},
	}, {
		desc: "no repo host defined keep delete secret",
		createResources: []client.Object{
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					// cleanup logic is sensitive the name of this resource
					Name:      "delete-ssh-secret-ssh-secret",
					Namespace: namespace,
					Labels:    naming.PGBackRestRepoHostLabels("delete-ssh-secret"),
				},
				Data: map[string][]byte{},
			},
		},
		cluster: &v1beta1.PostgresCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "delete-ssh-secret",
				Namespace: namespace,
				UID:       types.UID(clusterUID),
			},
			Spec: v1beta1.PostgresClusterSpec{
				Archive: v1beta1.Archive{
					PGBackRest: v1beta1.PGBackRestArchive{},
				},
			},
		},
		result: testResult{
			jobCount: 0, pvcCount: 0, hostCount: 0,
			sshConfigPresent: false, sshSecretPresent: false,
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			for _, resource := range tc.createResources {

				err := controllerutil.SetControllerReference(tc.cluster, resource,
					tClient.Scheme())
				assert.NilError(t, err)
				assert.NilError(t, tClient.Create(ctx, resource))

				resources, err := r.getPGBackRestResources(ctx, tc.cluster)
				assert.NilError(t, err)

				assert.Assert(t, tc.result.jobCount == len(resources.replicaCreateBackupJobs))
				assert.Assert(t, tc.result.hostCount == len(resources.hosts))
				assert.Assert(t, tc.result.pvcCount == len(resources.pvcs))
				assert.Assert(t, tc.result.sshConfigPresent == (resources.sshConfig != nil))
				assert.Assert(t, tc.result.sshSecretPresent == (resources.sshSecret != nil))
			}
		})
	}
}
