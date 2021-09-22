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
	"os"
	"strconv"
	"strings"
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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crunchydata/postgres-operator/internal/initialize"
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
			Port:            initialize.Int32(5432),
			Shutdown:        initialize.Bool(false),
			PostgresVersion: 13,
			ImagePullSecrets: []v1.LocalObjectReference{{
				Name: "myImagePullSecret"},
			},
			Image: "example.com/crunchy-postgres-ha:test",
			InstanceSets: []v1beta1.PostgresInstanceSetSpec{{
				Name: "instance1",
				DataVolumeClaimSpec: v1.PersistentVolumeClaimSpec{
					AccessModes: []v1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
				},
			}},
			Backups: v1beta1.Backups{
				PGBackRest: v1beta1.PGBackRestArchive{
					Image: "example.com/crunchy-pgbackrest:test",
					Jobs: &v1beta1.BackupJobs{
						PriorityClassName: initialize.String("some-priority-class"),
					},
					Global: map[string]string{"repo2-test": "config",
						"repo3-test": "config", "repo4-test": "config"},
					Repos: []v1beta1.PGBackRestRepo{{
						Name: "repo1",
						S3: &v1beta1.RepoS3{
							Bucket:   "bucket",
							Endpoint: "endpoint",
							Region:   "region",
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
		postgresCluster.Spec.Backups.PGBackRest.Repos[0] = v1beta1.PGBackRestRepo{
			Name: "repo1",
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
		}
		postgresCluster.Spec.Backups.PGBackRest.RepoHost = &v1beta1.PGBackRestRepoHost{
			PriorityClassName: initialize.String("some-priority-class"),
			Resources:         corev1.ResourceRequirements{},
			Affinity:          &corev1.Affinity{},
			Tolerations: []v1.Toleration{
				{Key: "woot"},
			},
			TopologySpreadConstraints: []v1.TopologySpreadConstraint{
				{MaxSkew: int32(1)},
			},
		}
	}
	// always add schedule info to the first repo
	postgresCluster.Spec.Backups.PGBackRest.Repos[0].BackupSchedules = &v1beta1.PGBackRestBackupSchedules{
		Full:         &testCronSchedule,
		Differential: &testCronSchedule,
		Incremental:  &testCronSchedule,
	}

	return postgresCluster
}

func TestReconcilePGBackRest(t *testing.T) {
	// Garbage collector cleans up test resources before the test completes
	if strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
		t.Skip("USE_EXISTING_CLUSTER: Test fails due to garbage collection")
	}

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

	// create a service account to test with
	serviceAccount, err := r.reconcilePGBackRestRBAC(ctx, postgresCluster)
	assert.NilError(t, err)
	assert.Assert(t, serviceAccount != nil)

	// create the 'observed' instances and set the leader
	instances := &observedInstances{
		forCluster: []*Instance{{Name: "instance1",
			Pods: []*v1.Pod{{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{naming.LabelRole: naming.RolePatroniLeader},
				},
				Spec: v1.PodSpec{},
			}},
		}, {Name: "instance2"}, {Name: "instance3"}},
	}

	// set status
	postgresCluster.Status = v1beta1.PostgresClusterStatus{
		Patroni: &v1beta1.PatroniStatus{SystemIdentifier: "12345abcde"},
		PGBackRest: &v1beta1.PGBackRestStatus{
			RepoHost: &v1beta1.RepoHostStatus{Ready: true},
			Repos:    []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: true}}},
	}

	// set conditions
	clusterConditions := map[string]metav1.ConditionStatus{
		ConditionRepoHostReady: metav1.ConditionTrue,
		ConditionReplicaCreate: metav1.ConditionTrue,
	}
	for condition, status := range clusterConditions {
		meta.SetStatusCondition(&postgresCluster.Status.Conditions, metav1.Condition{
			Type: condition, Reason: "testing", Status: status})
	}

	result, err := r.reconcilePGBackRest(ctx, postgresCluster, instances)
	if err != nil || result != (reconcile.Result{}) {
		t.Errorf("unable to reconcile pgBackRest: %v", err)
	}

	// repo is the first defined repo
	repo := postgresCluster.Spec.Backups.PGBackRest.Repos[0]

	// test that the repo was created properly
	t.Run("verify pgbackrest dedicated repo StatefulSet", func(t *testing.T) {

		// get the pgBackRest repo sts using the labels we expect it to have
		dedicatedRepos := &appsv1.StatefulSetList{}
		if err := tClient.List(ctx, dedicatedRepos, client.InNamespace(namespace),
			client.MatchingLabels{
				naming.LabelCluster:             clusterName,
				naming.LabelPGBackRest:          "",
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

		// Ensure Affinity Spec has been added to dedicated repo
		if repo.Spec.Template.Spec.Affinity == nil {
			t.Error("dedicated repo host is missing affinity spec")
		}

		// Ensure Tolerations have been added to dedicated repo
		if repo.Spec.Template.Spec.Tolerations == nil {
			t.Error("dedicated repo host is missing tolerations")
		}

		// Ensure TopologySpreadConstraints have been added to dedicated repo
		if repo.Spec.Template.Spec.TopologySpreadConstraints == nil {
			t.Error("dedicated repo host is missing topology spread constraints")
		}

		// Ensure pod priority class has been added to dedicated repo
		if repo.Spec.Template.Spec.PriorityClassName != "some-priority-class" {
			t.Error("dedicated repo host priority class not set correctly")
		}

		// Ensure imagePullSecret has been added to the dedicated repo
		if repo.Spec.Template.Spec.ImagePullSecrets == nil {
			t.Error("image pull secret is missing tolerations")
		}

		if repo.Spec.Template.Spec.ImagePullSecrets != nil {
			if repo.Spec.Template.Spec.ImagePullSecrets[0].Name !=
				"myImagePullSecret" {
				t.Error("image pull secret name is not set correctly")
			}
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
		if err := wait.Poll(time.Second/2, Scale(time.Second*2), func() (bool, error) {
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

		for _, r := range postgresCluster.Spec.Backups.PGBackRest.Repos {
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

		var instanceConfFound, dedicatedRepoConfFound bool
		for k, v := range config.Data {
			if v != "" {
				if k == pgbackrest.CMInstanceKey {
					instanceConfFound = true
				} else if k == pgbackrest.CMRepoKey {
					dedicatedRepoConfFound = true
				}
			}
		}
		assert.Check(t, instanceConfFound)
		assert.Check(t, dedicatedRepoConfFound)

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

		// set status
		postgresCluster.Status = v1beta1.PostgresClusterStatus{
			Patroni: &v1beta1.PatroniStatus{SystemIdentifier: "12345abcde"},
			PGBackRest: &v1beta1.PGBackRestStatus{
				Repos: []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: true}}},
		}

		// set conditions
		clusterConditions := map[string]metav1.ConditionStatus{
			ConditionRepoHostReady: metav1.ConditionTrue,
			ConditionReplicaCreate: metav1.ConditionTrue,
		}

		for condition, status := range clusterConditions {
			meta.SetStatusCondition(&postgresCluster.Status.Conditions, metav1.Condition{
				Type: condition, Reason: "testing", Status: status})
		}

		requeue := r.reconcileScheduledBackups(context.Background(),
			postgresCluster, serviceAccount)
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
		assert.Assert(t, returnedCronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].SecurityContext != &corev1.SecurityContext{})

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

	t.Run("pgbackrest schedule suspended status", func(t *testing.T) {

		returnedCronJob := &batchv1beta1.CronJob{}
		if err := tClient.Get(ctx, types.NamespacedName{
			Name:      postgresCluster.Name + "-pgbackrest-repo1-full",
			Namespace: postgresCluster.GetNamespace(),
		}, returnedCronJob); err != nil {
			assert.NilError(t, err)
		}

		t.Run("pgbackrest schedule suspended false", func(t *testing.T) {
			assert.Assert(t, !*returnedCronJob.Spec.Suspend)
		})

		t.Run("shutdown", func(t *testing.T) {
			*postgresCluster.Spec.Shutdown = true
			postgresCluster.Spec.Standby = nil

			requeue := r.reconcileScheduledBackups(ctx,
				postgresCluster, serviceAccount)
			assert.Assert(t, !requeue)

			assert.NilError(t, tClient.Get(ctx, types.NamespacedName{
				Name:      postgresCluster.Name + "-pgbackrest-repo1-full",
				Namespace: postgresCluster.GetNamespace(),
			}, returnedCronJob))

			assert.Assert(t, *returnedCronJob.Spec.Suspend)
		})

		t.Run("standby", func(t *testing.T) {
			*postgresCluster.Spec.Shutdown = false
			postgresCluster.Spec.Standby = &v1beta1.PostgresStandbySpec{
				Enabled: true,
			}

			requeue := r.reconcileScheduledBackups(ctx,
				postgresCluster, serviceAccount)
			assert.Assert(t, !requeue)

			assert.NilError(t, tClient.Get(ctx, types.NamespacedName{
				Name:      postgresCluster.Name + "-pgbackrest-repo1-full",
				Namespace: postgresCluster.GetNamespace(),
			}, returnedCronJob))

			assert.Assert(t, *returnedCronJob.Spec.Suspend)
		})
	})
}

func TestReconcilePGBackRestRBAC(t *testing.T) {
	// Garbage collector cleans up test resources before the test completes
	if strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
		t.Skip("USE_EXISTING_CLUSTER: Test fails due to garbage collection")
	}

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

	instances := newObservedInstances(postgresCluster, nil, []corev1.Pod{{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{"status": `"role":"master"`},
			Labels: map[string]string{
				naming.LabelCluster:  postgresCluster.GetName(),
				naming.LabelInstance: "",
				naming.LabelRole:     naming.RolePatroniLeader,
			},
		},
	}})

	stanzaCreateFail := func(namespace, pod, container string, stdin io.Reader, stdout,
		stderr io.Writer, command ...string) error {
		return errors.New("fake stanza create failed")
	}

	stanzaCreateSuccess := func(namespace, pod, container string, stdin io.Reader, stdout,
		stderr io.Writer, command ...string) error {
		return nil
	}

	// now verify a stanza create success
	r.PodExec = stanzaCreateSuccess
	meta.SetStatusCondition(&postgresCluster.Status.Conditions, metav1.Condition{
		ObservedGeneration: postgresCluster.GetGeneration(),
		Type:               ConditionRepoHostReady,
		Status:             metav1.ConditionTrue,
		Reason:             "RepoHostReady",
		Message:            "pgBackRest dedicated repository host is ready",
	})

	configHashMistmatch, err := r.reconcileStanzaCreate(ctx, postgresCluster, instances, "abcde12345")
	assert.NilError(t, err)
	assert.Assert(t, !configHashMistmatch)

	events := &corev1.EventList{}
	err = wait.Poll(time.Second/2, Scale(time.Second*2), func() (bool, error) {
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

	configHashMismatch, err := r.reconcileStanzaCreate(ctx, postgresCluster, instances, "abcde12345")
	assert.Error(t, err, "fake stanza create failed: ")
	assert.Assert(t, !configHashMismatch)

	events = &corev1.EventList{}
	err = wait.Poll(time.Second/2, Scale(time.Second*2), func() (bool, error) {
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

	// status should indicate stanza were not created
	for _, r := range postgresCluster.Status.PGBackRest.Repos {
		assert.Assert(t, !r.StanzaCreated)
	}
}

func TestGetPGBackRestExecSelector(t *testing.T) {

	testCases := []struct {
		cluster           *v1beta1.PostgresCluster
		repoName          string
		desc              string
		expectedSelector  string
		expectedContainer string
	}{{
		desc: "volume repo defined dedicated repo host enabled",
		cluster: &v1beta1.PostgresCluster{
			ObjectMeta: metav1.ObjectMeta{Name: "hippo"},
			Spec: v1beta1.PostgresClusterSpec{
				Backups: v1beta1.Backups{
					PGBackRest: v1beta1.PGBackRestArchive{
						Repos: []v1beta1.PGBackRestRepo{{
							Name:   "repo1",
							Volume: &v1beta1.RepoPVC{},
						}},
					},
				},
			},
		},
		repoName: "repo1",
		expectedSelector: "postgres-operator.crunchydata.com/cluster=hippo," +
			"postgres-operator.crunchydata.com/pgbackrest=," +
			"postgres-operator.crunchydata.com/pgbackrest-dedicated=",
		expectedContainer: "pgbackrest",
	}, {
		desc: "cloud repo defined no repo host enabled",
		cluster: &v1beta1.PostgresCluster{
			ObjectMeta: metav1.ObjectMeta{Name: "hippo"},
			Spec: v1beta1.PostgresClusterSpec{
				Backups: v1beta1.Backups{
					PGBackRest: v1beta1.PGBackRestArchive{
						Repos: []v1beta1.PGBackRestRepo{{
							Name: "repo1",
							S3:   &v1beta1.RepoS3{},
						}},
					},
				},
			},
		},
		repoName: "repo1",
		expectedSelector: "postgres-operator.crunchydata.com/cluster=hippo," +
			"postgres-operator.crunchydata.com/instance," +
			"postgres-operator.crunchydata.com/role=master",
		expectedContainer: "database",
	}}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			selector, container, err := getPGBackRestExecSelector(tc.cluster, tc.repoName)
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
	instances := newObservedInstances(postgresCluster, nil, []corev1.Pod{{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{"status": `"role":"master"`},
			Labels: map[string]string{
				naming.LabelCluster:  postgresCluster.GetName(),
				naming.LabelInstance: "",
				naming.LabelRole:     naming.RolePatroniLeader,
			},
		},
	}})

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

	replicaCreateRepo := postgresCluster.Spec.Backups.PGBackRest.Repos[0].Name
	configHash := "abcde12345"

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "hippo-sa"},
	}

	err := r.reconcileReplicaCreateBackup(ctx, postgresCluster, instances,
		[]*batchv1.Job{}, sa, configHash, replicaCreateRepo)
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
				"postgres-operator.crunchydata.com/pgbackrest-dedicated=")
		}
	}
	// verify mounted configuration is present
	assert.Assert(t, len(container.VolumeMounts) == 1)

	// verify volume for configuration is present
	assert.Assert(t, len(backupJob.Spec.Template.Spec.Volumes) == 1)

	// verify the image pull secret
	assert.Assert(t, backupJob.Spec.Template.Spec.ImagePullSecrets != nil)
	assert.Equal(t, backupJob.Spec.Template.Spec.ImagePullSecrets[0].Name,
		"myImagePullSecret")

	// verify the priority class
	assert.Equal(t, backupJob.Spec.Template.Spec.PriorityClassName, "some-priority-class")

	// now set the job to complete
	backupJob.Status.Conditions = append(backupJob.Status.Conditions,
		batchv1.JobCondition{Type: batchv1.JobComplete, Status: corev1.ConditionTrue})

	// call reconcile function again
	err = r.reconcileReplicaCreateBackup(ctx, postgresCluster, instances,
		[]*batchv1.Job{&backupJob}, sa, configHash, replicaCreateRepo)
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
		// whether or not the primary instance should be read-only
		standby bool
		// whether or not to mock a current job in the env before reconciling (this job is not
		// actually created, but rather just passed into the reconcile function under test)
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
		testDesc:         "read-only cluster should not reconcile",
		createCurrentJob: false,
		clusterConditions: map[string]metav1.ConditionStatus{
			ConditionRepoHostReady: metav1.ConditionTrue,
			ConditionReplicaCreate: metav1.ConditionTrue,
		},
		standby: true,
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
			PGBackRest: &v1beta1.PGBackRestStatus{
				ManualBackup: &v1beta1.PGBackRestJobStatus{
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
			PGBackRest: &v1beta1.PGBackRestStatus{
				Repos: []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: true}}},
		},
		backupId:                 defaultBackupId,
		manual:                   &v1beta1.PGBackRestManualBackup{RepoName: "repo1"},
		expectCurrentJobDeletion: false,
		expectReconcile:          true,
	}, {
		testDesc:         "reconcile new job when in-progress job exists for another id",
		createCurrentJob: true,
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
				postgresCluster.Spec.Backups.PGBackRest.Manual = tc.manual
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

				if tc.standby {
					instances.forCluster[0].Pods[0].Annotations = map[string]string{}
				} else {
					instances.forCluster[0].Pods[0].Annotations = map[string]string{
						"status": `"role":"master"`,
					}
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

					// verify image pull secret
					assert.Assert(t, len(jobs.Items[0].Spec.Template.Spec.ImagePullSecrets) > 0)
					assert.Equal(t, jobs.Items[0].Spec.Template.Spec.ImagePullSecrets[0].Name, "myImagePullSecret")

					// verify the priority class
					assert.Equal(t, jobs.Items[0].Spec.Template.Spec.PriorityClassName, "some-priority-class")

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
						err = wait.Poll(time.Second/2, Scale(time.Second*2), func() (bool, error) {
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
	// Garbage collector cleans up test resources before the test completes
	if strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
		t.Skip("USE_EXISTING_CLUSTER: Test fails due to garbage collection")
	}

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
				Backups: v1beta1.Backups{
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
				Backups: v1beta1.Backups{
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
				Backups: v1beta1.Backups{
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
				Backups: v1beta1.Backups{
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
				Backups: v1beta1.Backups{
					PGBackRest: v1beta1.PGBackRestArchive{
						Repos: []v1beta1.PGBackRestRepo{{Volume: &v1beta1.RepoPVC{}}},
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
				Backups: v1beta1.Backups{
					PGBackRest: v1beta1.PGBackRestArchive{},
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
				Backups: v1beta1.Backups{
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
					Labels:    naming.PGBackRestDedicatedLabels("keep-ssh-cm"),
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
				Backups: v1beta1.Backups{
					PGBackRest: v1beta1.PGBackRestArchive{
						Repos: []v1beta1.PGBackRestRepo{{Volume: &v1beta1.RepoPVC{}}},
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
					Labels:    naming.PGBackRestDedicatedLabels("delete-ssh-cm"),
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
				Backups: v1beta1.Backups{
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
					Labels:    naming.PGBackRestDedicatedLabels("keep-ssh-secret"),
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
				Backups: v1beta1.Backups{
					PGBackRest: v1beta1.PGBackRestArchive{
						Repos: []v1beta1.PGBackRestRepo{{Volume: &v1beta1.RepoPVC{}}},
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
					Labels:    naming.PGBackRestDedicatedLabels("delete-ssh-secret"),
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
				Backups: v1beta1.Backups{
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

func TestReconcilePostgresClusterDataSource(t *testing.T) {

	// setup the test environment and ensure a clean teardown
	tEnv, tClient, cfg := setupTestEnv(t, ControllerName)
	t.Cleanup(func() { teardownTestEnv(t, tEnv) })
	r := &Reconciler{}
	ctx, cancel := setupManager(t, cfg, func(mgr manager.Manager) {
		r = &Reconciler{
			Client:   tClient,
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
	namespace := ns.Name

	type testResult struct {
		jobCount, pvcCount                                      int
		invalidSourceRepo, invalidSourceCluster, invalidOptions bool
		expectedClusterCondition                                *metav1.Condition
	}

	for _, dedicated := range []bool{true, false} {
		testCases := []struct {
			desc                string
			dataSource          *v1beta1.DataSource
			clusterBootstrapped bool
			sourceClusterName   string
			sourceClusterRepos  []v1beta1.PGBackRestRepo
			result              testResult
		}{{
			desc: "initial reconcile",
			dataSource: &v1beta1.DataSource{PostgresCluster: &v1beta1.PostgresClusterDataSource{
				ClusterName: "init-source", RepoName: "repo1",
			}},
			clusterBootstrapped: false,
			sourceClusterName:   "init-source",
			sourceClusterRepos:  []v1beta1.PGBackRestRepo{{Name: "repo1"}},
			result: testResult{
				jobCount: 1, pvcCount: 1,
				invalidSourceRepo: false, invalidSourceCluster: false, invalidOptions: false,
				expectedClusterCondition: nil,
			},
		}, {
			desc: "invalid source cluster",
			dataSource: &v1beta1.DataSource{PostgresCluster: &v1beta1.PostgresClusterDataSource{
				ClusterName: "the-wrong-source", RepoName: "repo1",
			}},
			clusterBootstrapped: false,
			sourceClusterName:   "the-right-source",
			sourceClusterRepos:  []v1beta1.PGBackRestRepo{{Name: "repo1"}},
			result: testResult{
				jobCount: 0, pvcCount: 0,
				invalidSourceRepo: false, invalidSourceCluster: true, invalidOptions: false,
				expectedClusterCondition: nil,
			},
		}, {
			desc: "invalid source repo",
			dataSource: &v1beta1.DataSource{PostgresCluster: &v1beta1.PostgresClusterDataSource{
				ClusterName: "invalid-repo", RepoName: "repo2",
			}},
			clusterBootstrapped: false,
			sourceClusterName:   "invalid-repo",
			sourceClusterRepos:  []v1beta1.PGBackRestRepo{{Name: "repo1"}},
			result: testResult{
				jobCount: 0, pvcCount: 0,
				invalidSourceRepo: true, invalidSourceCluster: false, invalidOptions: false,
				expectedClusterCondition: nil,
			},
		}, {
			desc: "invalid option: repo",
			dataSource: &v1beta1.DataSource{PostgresCluster: &v1beta1.PostgresClusterDataSource{
				ClusterName: "invalid-repo-option", RepoName: "repo1",
				Options: []string{"--repo"},
			}},
			clusterBootstrapped: false,
			sourceClusterName:   "invalid-repo-option",
			sourceClusterRepos:  []v1beta1.PGBackRestRepo{{Name: "repo1"}},
			result: testResult{
				jobCount: 0, pvcCount: 1,
				invalidSourceRepo: false, invalidSourceCluster: false, invalidOptions: true,
				expectedClusterCondition: nil,
			},
		}, {
			desc: "invalid option: stanza",
			dataSource: &v1beta1.DataSource{PostgresCluster: &v1beta1.PostgresClusterDataSource{
				ClusterName: "invalid-stanza-option", RepoName: "repo1",
				Options: []string{"--stanza"},
			}},
			clusterBootstrapped: false,
			sourceClusterName:   "invalid-stanza-option",
			sourceClusterRepos:  []v1beta1.PGBackRestRepo{{Name: "repo1"}},
			result: testResult{
				jobCount: 0, pvcCount: 1,
				invalidSourceRepo: false, invalidSourceCluster: false, invalidOptions: true,
				expectedClusterCondition: nil,
			},
		}, {
			desc: "invalid option: pg1-path",
			dataSource: &v1beta1.DataSource{PostgresCluster: &v1beta1.PostgresClusterDataSource{
				ClusterName: "invalid-pgpath-option", RepoName: "repo1",
				Options: []string{"--pg1-path"},
			}},
			clusterBootstrapped: false,
			sourceClusterName:   "invalid-pgpath-option",
			sourceClusterRepos:  []v1beta1.PGBackRestRepo{{Name: "repo1"}},
			result: testResult{
				jobCount: 0, pvcCount: 1,
				invalidSourceRepo: false, invalidSourceCluster: false, invalidOptions: true,
				expectedClusterCondition: nil,
			},
		}, {
			desc: "cluster bootstrapped init condition missing",
			dataSource: &v1beta1.DataSource{PostgresCluster: &v1beta1.PostgresClusterDataSource{
				ClusterName: "bootstrapped-init-missing", RepoName: "repo1",
			}},
			clusterBootstrapped: true,
			sourceClusterName:   "init-cond-missing",
			sourceClusterRepos:  []v1beta1.PGBackRestRepo{{Name: "repo1"}},
			result: testResult{
				jobCount: 0, pvcCount: 0,
				invalidSourceRepo: false, invalidSourceCluster: false, invalidOptions: false,
				expectedClusterCondition: &metav1.Condition{
					Type:    ConditionPostgresDataInitialized,
					Status:  metav1.ConditionTrue,
					Reason:  "ClusterAlreadyBootstrapped",
					Message: "The cluster is already bootstrapped",
				},
			},
		}, {
			desc: "data source config change deletes job",
			dataSource: &v1beta1.DataSource{PostgresCluster: &v1beta1.PostgresClusterDataSource{
				ClusterName: "invalid-hash", RepoName: "repo1",
			}},
			clusterBootstrapped: true,
			sourceClusterName:   "invalid-hash",
			sourceClusterRepos:  []v1beta1.PGBackRestRepo{{Name: "repo1"}},
			result: testResult{
				jobCount: 0, pvcCount: 0,
				invalidSourceRepo: false, invalidSourceCluster: false, invalidOptions: false,
				expectedClusterCondition: nil,
			},
		}}

		for i, tc := range testCases {
			if !dedicated {
				tc.desc += "-no-repo"
			}
			t.Run(tc.desc, func(t *testing.T) {

				clusterName := "hippocluster-" + strconv.Itoa(i)
				if !dedicated {
					clusterName = clusterName + "-no-repo"
				}
				clusterUID := "hippouid" + strconv.Itoa(i)

				cluster := fakePostgresCluster(clusterName, namespace, clusterUID, dedicated)
				cluster.Spec.DataSource = tc.dataSource
				assert.NilError(t, tClient.Create(ctx, cluster))
				if tc.clusterBootstrapped {
					cluster.Status.Patroni = &v1beta1.PatroniStatus{
						SystemIdentifier: "123456789",
					}
				}
				cluster.Status.StartupInstance = "testinstance"
				cluster.Status.StartupInstanceSet = "instance1"
				assert.NilError(t, tClient.Status().Update(ctx, cluster))
				if !dedicated {
					tc.sourceClusterName = tc.sourceClusterName + "-no-repo"
				}
				sourceCluster := fakePostgresCluster(tc.sourceClusterName, namespace,
					"source"+clusterUID, dedicated)
				sourceCluster.Spec.Backups.PGBackRest.Repos = tc.sourceClusterRepos
				assert.NilError(t, tClient.Create(ctx, sourceCluster))

				sourceClusterPrimary := &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "primary-" + tc.sourceClusterName,
						Namespace: namespace,
						Labels: map[string]string{
							naming.LabelCluster:     tc.sourceClusterName,
							naming.LabelInstanceSet: "test",
							naming.LabelInstance:    "test-abcd",
							naming.LabelRole:        naming.RolePatroniLeader,
						},
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{{
							Name:    "test",
							Image:   "test",
							Command: []string{"test"},
						}},
					},
				}
				assert.NilError(t, tClient.Create(ctx, sourceClusterPrimary))

				var pgclusterDataSource *v1beta1.PostgresClusterDataSource
				if tc.dataSource != nil {
					pgclusterDataSource = tc.dataSource.PostgresCluster
				}
				err := r.reconcilePostgresClusterDataSource(ctx, cluster, pgclusterDataSource,
					"testhash", nil)
				assert.NilError(t, err)

				restoreJobs := &batchv1.JobList{}
				assert.NilError(t, tClient.List(ctx, restoreJobs, &client.ListOptions{
					LabelSelector: naming.PGBackRestRestoreJobSelector(clusterName),
				}))
				assert.Assert(t, tc.result.jobCount == len(restoreJobs.Items))
				if len(restoreJobs.Items) == 1 {
					assert.Assert(t, restoreJobs.Items[0].Labels[naming.LabelStartupInstance] != "")
					assert.Assert(t, restoreJobs.Items[0].Annotations[naming.PGBackRestConfigHash] != "")
				}

				dataPVCs := &v1.PersistentVolumeClaimList{}
				selector, err := naming.AsSelector(naming.Cluster(cluster.Name))
				assert.NilError(t, err)
				dataRoleReq, err := labels.NewRequirement(naming.LabelRole, selection.Equals,
					[]string{naming.RolePostgresData})
				assert.NilError(t, err)
				selector.Add(*dataRoleReq)
				assert.NilError(t, tClient.List(ctx, dataPVCs, &client.ListOptions{
					LabelSelector: selector,
				}))

				assert.Assert(t, tc.result.pvcCount == len(dataPVCs.Items))

				if tc.result.expectedClusterCondition != nil {
					condition := meta.FindStatusCondition(cluster.Status.Conditions,
						tc.result.expectedClusterCondition.Type)
					if assert.Check(t, condition != nil) {
						assert.Equal(t, tc.result.expectedClusterCondition.Status, condition.Status)
						assert.Equal(t, tc.result.expectedClusterCondition.Reason, condition.Reason)
						assert.Equal(t, tc.result.expectedClusterCondition.Message, condition.Message)
					}
				}

				if tc.result.invalidSourceCluster || tc.result.invalidSourceRepo ||
					tc.result.invalidOptions {
					events := &corev1.EventList{}
					if err := wait.Poll(time.Second/2, Scale(time.Second*2), func() (bool, error) {
						if err := tClient.List(ctx, events, &client.MatchingFields{
							"involvedObject.kind":      "PostgresCluster",
							"involvedObject.name":      clusterName,
							"involvedObject.namespace": namespace,
							"reason":                   "InvalidDataSource",
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
				}
			})
		}
	}
}

func TestGenerateBackupJobIntent(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		_, err := generateBackupJobSpecIntent(
			&v1beta1.PostgresCluster{},
			"", "", "", "", "",
			nil, nil,
		)
		assert.NilError(t, err)
	})

	cluster := &v1beta1.PostgresCluster{
		Spec: v1beta1.PostgresClusterSpec{
			Backups: v1beta1.Backups{PGBackRest: v1beta1.PGBackRestArchive{}},
		},
	}

	t.Run("Resources not defined in jobs", func(t *testing.T) {
		job, err := generateBackupJobSpecIntent(
			cluster,
			"", "", "", "", "",
			nil, nil,
		)
		assert.NilError(t, err)
		assert.DeepEqual(t, job.Template.Spec.Containers[0].Resources, corev1.ResourceRequirements{})
	})

	cluster.Spec.Backups.PGBackRest.Jobs = &v1beta1.BackupJobs{
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("1m"),
			},
		},
	}

	t.Run("Resources defined", func(t *testing.T) {
		job, err := generateBackupJobSpecIntent(
			cluster,
			"", "", "", "", "",
			nil, nil,
		)
		assert.NilError(t, err)
		assert.DeepEqual(t, job.Template.Spec.Containers[0].Resources,
			corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("1m"),
				}},
		)
	})
}

func TestGenerateRestoreJobIntent(t *testing.T) {
	env, cc, _ := setupTestEnv(t, ControllerName)
	t.Cleanup(func() { teardownTestEnv(t, env) })

	r := Reconciler{
		Client: cc,
	}

	t.Run("empty", func(t *testing.T) {
		err := r.generateRestoreJobIntent(&v1beta1.PostgresCluster{}, "", "",
			[]string{}, []corev1.VolumeMount{}, []corev1.Volume{},
			&v1beta1.PostgresClusterDataSource{}, &batchv1.Job{})
		assert.NilError(t, err)
	})

	configHash := "hash"
	instanceName := "name"
	cmd := []string{"cmd", "blah"}
	volumeMounts := []corev1.VolumeMount{{
		Name: "mount",
	}}
	volumes := []corev1.Volume{{
		Name: "volume",
	}}
	dataSource := &v1beta1.PostgresClusterDataSource{
		// ClusterName/Namespace, Repo, and Options are tested in
		// TestReconcilePostgresClusterDataSource
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("1Gi"),
			},
		},
		Affinity: &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{{
						MatchExpressions: []corev1.NodeSelectorRequirement{{
							Key:      "key",
							Operator: "Exist",
						}},
					}},
				},
			},
		},
		Tolerations: []corev1.Toleration{{
			Key:      "key",
			Operator: "Exist",
		}},
		PriorityClassName: initialize.String("some-priority-class"),
	}
	cluster := &v1beta1.PostgresCluster{
		Spec: v1beta1.PostgresClusterSpec{
			Metadata: &v1beta1.Metadata{
				Labels:      map[string]string{"Global": "test"},
				Annotations: map[string]string{"Global": "test"},
			},
			Backups: v1beta1.Backups{PGBackRest: v1beta1.PGBackRestArchive{
				Metadata: &v1beta1.Metadata{
					Labels:      map[string]string{"Backrest": "test"},
					Annotations: map[string]string{"Backrest": "test"},
				},
			}},
			Image:            "image",
			ImagePullSecrets: []corev1.LocalObjectReference{{Name: "Secret"}},
		},
	}

	for _, openshift := range []bool{true, false} {
		cluster.Spec.OpenShift = initialize.Bool(openshift)

		job := &batchv1.Job{}
		err := r.generateRestoreJobIntent(cluster, configHash, instanceName,
			cmd, volumeMounts, volumes, dataSource, job)
		assert.NilError(t, err, job)

		t.Run(fmt.Sprintf("openshift-%v", openshift), func(t *testing.T) {
			t.Run("ObjectMeta", func(t *testing.T) {
				t.Run("Name", func(t *testing.T) {
					assert.Equal(t, job.ObjectMeta.Name,
						naming.PGBackRestRestoreJob(cluster).Name)
				})
				t.Run("Namespace", func(t *testing.T) {
					assert.Equal(t, job.ObjectMeta.Namespace,
						naming.PGBackRestRestoreJob(cluster).Namespace)
				})
				t.Run("Annotations", func(t *testing.T) {
					// configHash is defined as an annotation on the job
					annotations := labels.Set(job.GetAnnotations())
					assert.Assert(t, annotations.Has("Global"))
					assert.Assert(t, annotations.Has("Backrest"))
					assert.Equal(t, annotations.Get(naming.PGBackRestConfigHash), configHash)
				})
				t.Run("Labels", func(t *testing.T) {
					// instanceName is defined as a label on the job
					label := labels.Set(job.GetLabels())
					assert.Equal(t, label.Get("Global"), "test")
					assert.Equal(t, label.Get("Backrest"), "test")
					assert.Equal(t, label.Get(naming.LabelStartupInstance), instanceName)
				})
			})
			t.Run("Spec", func(t *testing.T) {
				t.Run("Template", func(t *testing.T) {
					t.Run("ObjectMeta", func(t *testing.T) {
						t.Run("Annotations", func(t *testing.T) {
							annotations := labels.Set(job.Spec.Template.GetAnnotations())
							assert.Assert(t, annotations.Has("Global"))
							assert.Assert(t, annotations.Has("Backrest"))
							assert.Equal(t, annotations.Get(naming.PGBackRestConfigHash), configHash)
						})
						t.Run("Labels", func(t *testing.T) {
							label := labels.Set(job.Spec.Template.GetLabels())
							assert.Equal(t, label.Get("Global"), "test")
							assert.Equal(t, label.Get("Backrest"), "test")
							assert.Equal(t, label.Get(naming.LabelStartupInstance), instanceName)
						})
					})
					t.Run("Spec", func(t *testing.T) {
						t.Run("Containers", func(t *testing.T) {
							assert.Assert(t, len(job.Spec.Template.Spec.Containers) == 1)
							t.Run("Command", func(t *testing.T) {
								assert.DeepEqual(t, job.Spec.Template.Spec.Containers[0].Command,
									[]string{"cmd", "blah"})
							})
							t.Run("Image", func(t *testing.T) {
								assert.Equal(t, job.Spec.Template.Spec.Containers[0].Image,
									"image")
							})
							t.Run("Name", func(t *testing.T) {
								assert.Equal(t, job.Spec.Template.Spec.Containers[0].Name,
									naming.PGBackRestRestoreContainerName)
							})
							t.Run("VolumeMount", func(t *testing.T) {
								assert.DeepEqual(t, job.Spec.Template.Spec.Containers[0].VolumeMounts,
									[]corev1.VolumeMount{{
										Name: "mount",
									}})
							})
							t.Run("Env", func(t *testing.T) {
								assert.DeepEqual(t, job.Spec.Template.Spec.Containers[0].Env,
									[]corev1.EnvVar{{Name: "PGHOST", Value: "/tmp"}})
							})
							t.Run("SecurityContext", func(t *testing.T) {
								assert.DeepEqual(t, job.Spec.Template.Spec.Containers[0].SecurityContext,
									initialize.RestrictedSecurityContext())
							})
							t.Run("Resources", func(t *testing.T) {
								assert.DeepEqual(t, job.Spec.Template.Spec.Containers[0].Resources,
									dataSource.Resources)
							})
						})
						t.Run("RestartPolicy", func(t *testing.T) {
							assert.Equal(t, job.Spec.Template.Spec.RestartPolicy,
								corev1.RestartPolicyNever)
						})
						t.Run("Volumes", func(t *testing.T) {
							assert.DeepEqual(t, job.Spec.Template.Spec.Volumes,
								[]corev1.Volume{{
									Name: "volume",
								}})
						})
						t.Run("Affinity", func(t *testing.T) {
							assert.DeepEqual(t, job.Spec.Template.Spec.Affinity,
								dataSource.Affinity)
						})
						t.Run("Tolerations", func(t *testing.T) {
							assert.DeepEqual(t, job.Spec.Template.Spec.Tolerations,
								dataSource.Tolerations)
						})
						t.Run("Pod Priority Class", func(t *testing.T) {
							assert.DeepEqual(t, job.Spec.Template.Spec.PriorityClassName,
								"some-priority-class")
						})
						t.Run("ImagePullSecret", func(t *testing.T) {
							assert.DeepEqual(t, job.Spec.Template.Spec.ImagePullSecrets,
								[]corev1.LocalObjectReference{{
									Name: "Secret",
								}})
						})
						t.Run("PodSecurityContext", func(t *testing.T) {
							assert.Assert(t, job.Spec.Template.Spec.SecurityContext != nil)
						})
					})
				})
			})
		})
	}
}

func TestObserveRestoreEnv(t *testing.T) {

	// setup the test environment and ensure a clean teardown
	tEnv, tClient, cfg := setupTestEnv(t, ControllerName)
	t.Cleanup(func() { teardownTestEnv(t, tEnv) })
	r := &Reconciler{}
	ctx, cancel := setupManager(t, cfg, func(mgr manager.Manager) {
		r = &Reconciler{
			Client:   tClient,
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
	namespace := ns.Name

	generateJob := func(clusterName string, completed, failed *bool) *batchv1.Job {

		cluster := &v1beta1.PostgresCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: namespace,
			},
		}
		meta := naming.PGBackRestRestoreJob(cluster)
		labels := naming.PGBackRestRestoreJobLabels(cluster.Name)
		meta.Labels = labels
		meta.Annotations = map[string]string{naming.PGBackRestConfigHash: "testhash"}

		restoreJob := &batchv1.Job{
			ObjectMeta: meta,
			Spec: batchv1.JobSpec{
				Template: v1.PodTemplateSpec{
					ObjectMeta: meta,
					Spec: v1.PodSpec{
						Containers: []v1.Container{{
							Image: "test",
							Name:  naming.PGBackRestRestoreContainerName,
						}},
						RestartPolicy: v1.RestartPolicyNever,
					},
				},
			},
		}

		if completed != nil {
			if *completed {
				restoreJob.Status.Conditions = append(restoreJob.Status.Conditions, batchv1.JobCondition{
					Type:    batchv1.JobComplete,
					Status:  corev1.ConditionTrue,
					Reason:  "test",
					Message: "test",
				})
			} else {
				restoreJob.Status.Conditions = append(restoreJob.Status.Conditions, batchv1.JobCondition{
					Type:    batchv1.JobComplete,
					Status:  corev1.ConditionFalse,
					Reason:  "test",
					Message: "test",
				})
			}
		} else if failed != nil {
			if *failed {
				restoreJob.Status.Conditions = append(restoreJob.Status.Conditions, batchv1.JobCondition{
					Type:    batchv1.JobFailed,
					Status:  corev1.ConditionTrue,
					Reason:  "test",
					Message: "test",
				})
			} else {
				restoreJob.Status.Conditions = append(restoreJob.Status.Conditions, batchv1.JobCondition{
					Type:    batchv1.JobFailed,
					Status:  corev1.ConditionFalse,
					Reason:  "test",
					Message: "test",
				})
			}
		}

		return restoreJob
	}

	type testResult struct {
		foundRestoreJob          bool
		endpointCount            int
		expectedClusterCondition *metav1.Condition
	}

	for _, dedicated := range []bool{true, false} {
		testCases := []struct {
			desc            string
			createResources func(t *testing.T, cluster *v1beta1.PostgresCluster)
			result          testResult
		}{{
			desc: "restore job and all patroni endpoints exist",
			createResources: func(t *testing.T, cluster *v1beta1.PostgresCluster) {
				fakeLeaderEP := &v1.Endpoints{}
				fakeLeaderEP.ObjectMeta = naming.PatroniLeaderEndpoints(cluster)
				fakeLeaderEP.ObjectMeta.Namespace = namespace
				assert.NilError(t, r.Client.Create(ctx, fakeLeaderEP))
				fakeDCSEP := &v1.Endpoints{}
				fakeDCSEP.ObjectMeta = naming.PatroniDistributedConfiguration(cluster)
				fakeDCSEP.ObjectMeta.Namespace = namespace
				assert.NilError(t, r.Client.Create(ctx, fakeDCSEP))
				fakeFailoverEP := &v1.Endpoints{}
				fakeFailoverEP.ObjectMeta = naming.PatroniTrigger(cluster)
				fakeFailoverEP.ObjectMeta.Namespace = namespace
				assert.NilError(t, r.Client.Create(ctx, fakeFailoverEP))

				job := generateJob(cluster.Name, initialize.Bool(false), initialize.Bool(false))
				assert.NilError(t, r.Client.Create(ctx, job))
			},
			result: testResult{
				foundRestoreJob:          true,
				endpointCount:            3,
				expectedClusterCondition: nil,
			},
		}, {
			desc: "patroni endpoints only exist",
			createResources: func(t *testing.T, cluster *v1beta1.PostgresCluster) {
				fakeLeaderEP := &v1.Endpoints{}
				fakeLeaderEP.ObjectMeta = naming.PatroniLeaderEndpoints(cluster)
				fakeLeaderEP.ObjectMeta.Namespace = namespace
				assert.NilError(t, r.Client.Create(ctx, fakeLeaderEP))
				fakeDCSEP := &v1.Endpoints{}
				fakeDCSEP.ObjectMeta = naming.PatroniDistributedConfiguration(cluster)
				fakeDCSEP.ObjectMeta.Namespace = namespace
				assert.NilError(t, r.Client.Create(ctx, fakeDCSEP))
				fakeFailoverEP := &v1.Endpoints{}
				fakeFailoverEP.ObjectMeta = naming.PatroniTrigger(cluster)
				fakeFailoverEP.ObjectMeta.Namespace = namespace
				assert.NilError(t, r.Client.Create(ctx, fakeFailoverEP))
			},
			result: testResult{
				foundRestoreJob:          false,
				endpointCount:            3,
				expectedClusterCondition: nil,
			},
		}, {
			desc: "restore job only exists",
			createResources: func(t *testing.T, cluster *v1beta1.PostgresCluster) {
				job := generateJob(cluster.Name, initialize.Bool(false), initialize.Bool(false))
				assert.NilError(t, r.Client.Create(ctx, job))
			},
			result: testResult{
				foundRestoreJob:          true,
				endpointCount:            0,
				expectedClusterCondition: nil,
			},
		}, {
			desc: "restore job completed data init condition true",
			createResources: func(t *testing.T, cluster *v1beta1.PostgresCluster) {
				if strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
					t.Skip("requires mocking of Job conditions")
				}
				job := generateJob(cluster.Name, initialize.Bool(true), nil)
				assert.NilError(t, r.Client.Create(ctx, job.DeepCopy()))
				assert.NilError(t, r.Client.Status().Update(ctx, job))
			},
			result: testResult{
				foundRestoreJob: true,
				endpointCount:   0,
				expectedClusterCondition: &metav1.Condition{
					Type:    ConditionPostgresDataInitialized,
					Status:  metav1.ConditionTrue,
					Reason:  "PGBackRestRestoreComplete",
					Message: "pgBackRest restore completed successfully",
				},
			},
		}, {
			desc: "restore job failed data init condition false",
			createResources: func(t *testing.T, cluster *v1beta1.PostgresCluster) {
				if strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
					t.Skip("requires mocking of Job conditions")
				}
				job := generateJob(cluster.Name, nil, initialize.Bool(true))
				assert.NilError(t, r.Client.Create(ctx, job.DeepCopy()))
				assert.NilError(t, r.Client.Status().Update(ctx, job))
			},
			result: testResult{
				foundRestoreJob: true,
				endpointCount:   0,
				expectedClusterCondition: &metav1.Condition{
					Type:    ConditionPostgresDataInitialized,
					Status:  metav1.ConditionFalse,
					Reason:  "PGBackRestRestoreFailed",
					Message: "pgBackRest restore failed",
				},
			},
		}}

		for i, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {

				clusterName := "observe-restore-env" + strconv.Itoa(i)
				if !dedicated {
					clusterName = clusterName + "-no-repo"
				}
				clusterUID := clusterName
				cluster := fakePostgresCluster(clusterName, namespace, clusterUID, dedicated)
				tc.createResources(t, cluster)

				endpoints, job, err := r.observeRestoreEnv(ctx, cluster)
				assert.NilError(t, err)

				assert.Assert(t, tc.result.foundRestoreJob == (job != nil))
				assert.Assert(t, tc.result.endpointCount == len(endpoints))

				if tc.result.expectedClusterCondition != nil {
					condition := meta.FindStatusCondition(cluster.Status.Conditions,
						tc.result.expectedClusterCondition.Type)
					if assert.Check(t, condition != nil) {
						assert.Equal(t, tc.result.expectedClusterCondition.Status, condition.Status)
						assert.Equal(t, tc.result.expectedClusterCondition.Reason, condition.Reason)
						assert.Equal(t, tc.result.expectedClusterCondition.Message, condition.Message)
					}
				}
			})
		}
	}
}

func TestPrepareForRestore(t *testing.T) {

	// setup the test environment and ensure a clean teardown
	tEnv, tClient, cfg := setupTestEnv(t, ControllerName)
	t.Cleanup(func() { teardownTestEnv(t, tEnv) })
	r := &Reconciler{}
	ctx, cancel := setupManager(t, cfg, func(mgr manager.Manager) {
		r = &Reconciler{
			Client:   tClient,
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
	namespace := ns.Name

	generateJob := func(clusterName string) *batchv1.Job {

		cluster := &v1beta1.PostgresCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: namespace,
			},
		}
		meta := naming.PGBackRestRestoreJob(cluster)
		labels := naming.PGBackRestRestoreJobLabels(cluster.Name)
		meta.Labels = labels
		meta.Annotations = map[string]string{naming.PGBackRestConfigHash: "testhash"}

		restoreJob := &batchv1.Job{
			ObjectMeta: meta,
			Spec: batchv1.JobSpec{
				Template: v1.PodTemplateSpec{
					ObjectMeta: meta,
					Spec: v1.PodSpec{
						Containers: []v1.Container{{
							Image: "test",
							Name:  naming.PGBackRestRestoreContainerName,
						}},
						RestartPolicy: v1.RestartPolicyNever,
					},
				},
			},
		}

		return restoreJob
	}

	type testResult struct {
		restoreJobExists         bool
		endpointCount            int
		expectedClusterCondition *metav1.Condition
	}
	const primaryInstanceName = "primary-instance"
	const primaryInstanceSetName = "primary-instance-set"

	for _, dedicated := range []bool{true, false} {
		testCases := []struct {
			desc            string
			createResources func(t *testing.T, cluster *v1beta1.PostgresCluster) (*batchv1.Job, []corev1.Endpoints)
			fakeObserved    *observedInstances
			result          testResult
		}{{
			desc: "remove restore jobs",
			createResources: func(t *testing.T,
				cluster *v1beta1.PostgresCluster) (*batchv1.Job, []corev1.Endpoints) {
				job := generateJob(cluster.Name)
				assert.NilError(t, r.Client.Create(ctx, job))
				return job, nil
			},
			result: testResult{
				restoreJobExists: false,
				endpointCount:    0,
				expectedClusterCondition: &metav1.Condition{
					Type:    ConditionPGBackRestRestoreProgressing,
					Status:  metav1.ConditionTrue,
					Reason:  "RestoreInPlaceRequested",
					Message: "Preparing cluster to restore in-place: removing restore job",
				},
			},
		}, {
			desc: "remove patroni endpoints",
			createResources: func(t *testing.T,
				cluster *v1beta1.PostgresCluster) (*batchv1.Job, []corev1.Endpoints) {
				fakeLeaderEP := v1.Endpoints{}
				fakeLeaderEP.ObjectMeta = naming.PatroniLeaderEndpoints(cluster)
				fakeLeaderEP.ObjectMeta.Namespace = namespace
				assert.NilError(t, r.Client.Create(ctx, &fakeLeaderEP))
				fakeDCSEP := v1.Endpoints{}
				fakeDCSEP.ObjectMeta = naming.PatroniDistributedConfiguration(cluster)
				fakeDCSEP.ObjectMeta.Namespace = namespace
				assert.NilError(t, r.Client.Create(ctx, &fakeDCSEP))
				fakeFailoverEP := v1.Endpoints{}
				fakeFailoverEP.ObjectMeta = naming.PatroniTrigger(cluster)
				fakeFailoverEP.ObjectMeta.Namespace = namespace
				assert.NilError(t, r.Client.Create(ctx, &fakeFailoverEP))
				return nil, []corev1.Endpoints{fakeLeaderEP, fakeDCSEP, fakeFailoverEP}
			},
			result: testResult{
				restoreJobExists: false,
				endpointCount:    0,
				expectedClusterCondition: &metav1.Condition{
					Type:    ConditionPGBackRestRestoreProgressing,
					Status:  metav1.ConditionTrue,
					Reason:  "RestoreInPlaceRequested",
					Message: "Preparing cluster to restore in-place: removing DCS",
				},
			},
		}, {
			desc: "cluster fully prepared",
			createResources: func(t *testing.T,
				cluster *v1beta1.PostgresCluster) (*batchv1.Job, []corev1.Endpoints) {
				return nil, []corev1.Endpoints{}
			},
			result: testResult{
				restoreJobExists: false,
				endpointCount:    0,
				expectedClusterCondition: &metav1.Condition{
					Type:    ConditionPGBackRestRestoreProgressing,
					Status:  metav1.ConditionTrue,
					Reason:  ReasonReadyForRestore,
					Message: "Restoring cluster in-place",
				},
			},
		}, {
			desc: "primary as startup instance",
			fakeObserved: &observedInstances{forCluster: []*Instance{{
				Name: primaryInstanceName,
				Spec: &v1beta1.PostgresInstanceSetSpec{Name: primaryInstanceSetName},
				Pods: []*v1.Pod{{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{naming.LabelRole: naming.RolePatroniLeader},
					},
				}}},
			}},
			createResources: func(t *testing.T,
				cluster *v1beta1.PostgresCluster) (*batchv1.Job, []corev1.Endpoints) {
				return nil, []corev1.Endpoints{}
			},
			result: testResult{
				restoreJobExists: false,
				endpointCount:    0,
				expectedClusterCondition: &metav1.Condition{
					Type:    ConditionPGBackRestRestoreProgressing,
					Status:  metav1.ConditionTrue,
					Reason:  ReasonReadyForRestore,
					Message: "Restoring cluster in-place",
				},
			},
		}}

		for i, tc := range testCases {
			name := tc.desc
			if !dedicated {
				name = tc.desc + "-no-repo"
			}
			t.Run(name, func(t *testing.T) {

				clusterName := "prepare-for-restore-" + strconv.Itoa(i)
				if !dedicated {
					clusterName = clusterName + "-no-repo"
				}
				clusterUID := clusterName
				cluster := fakePostgresCluster(clusterName, namespace, clusterUID, dedicated)
				cluster.Status.Patroni = &v1beta1.PatroniStatus{SystemIdentifier: "abcde12345"}
				cluster.Status.Proxy.PGBouncer.PostgreSQLRevision = "abcde12345"
				cluster.Status.Monitoring.ExporterConfiguration = "abcde12345"
				meta.SetStatusCondition(&cluster.Status.Conditions, metav1.Condition{
					ObservedGeneration: cluster.GetGeneration(),
					Type:               ConditionPostgresDataInitialized,
					Status:             metav1.ConditionTrue,
					Reason:             "PGBackRestRestoreComplete",
					Message:            "pgBackRest restore completed successfully",
				})

				job, endpoints := tc.createResources(t, cluster)
				restoreID := "test-restore-id"

				fakeObserved := &observedInstances{forCluster: []*Instance{}}
				if tc.fakeObserved != nil {
					fakeObserved = tc.fakeObserved
				}
				assert.NilError(t, r.prepareForRestore(ctx, cluster, fakeObserved, endpoints,
					job, restoreID))

				var primaryInstance *Instance
				for i, instance := range fakeObserved.forCluster {
					isPrimary, _ := instance.IsPrimary()
					if isPrimary {
						primaryInstance = fakeObserved.forCluster[i]
					}
				}

				if primaryInstance != nil {
					assert.Assert(t, cluster.Status.StartupInstance == primaryInstanceName)
				} else {
					assert.Equal(t, cluster.Status.StartupInstance,
						naming.GenerateStartupInstance(cluster, &cluster.Spec.InstanceSets[0]).Name)
				}

				leaderEP, dcsEP, failoverEP := v1.Endpoints{}, v1.Endpoints{}, v1.Endpoints{}
				currentEndpoints := []v1.Endpoints{}
				if err := r.Client.Get(ctx, naming.AsObjectKey(naming.PatroniLeaderEndpoints(cluster)),
					&leaderEP); err != nil {
					assert.NilError(t, client.IgnoreNotFound(err))
				} else {
					currentEndpoints = append(currentEndpoints, leaderEP)
				}
				if err := r.Client.Get(ctx, naming.AsObjectKey(naming.PatroniDistributedConfiguration(cluster)),
					&dcsEP); err != nil {
					assert.NilError(t, client.IgnoreNotFound(err))
				} else {
					currentEndpoints = append(currentEndpoints, dcsEP)
				}
				if err := r.Client.Get(ctx, naming.AsObjectKey(naming.PatroniTrigger(cluster)),
					&failoverEP); err != nil {
					assert.NilError(t, client.IgnoreNotFound(err))
				} else {
					currentEndpoints = append(currentEndpoints, failoverEP)
				}

				restoreJobs := &batchv1.JobList{}
				assert.NilError(t, r.Client.List(ctx, restoreJobs, &client.ListOptions{
					LabelSelector: naming.PGBackRestRestoreJobSelector(cluster.GetName()),
				}))

				assert.Assert(t, tc.result.endpointCount == len(currentEndpoints))
				assert.Assert(t, tc.result.restoreJobExists == (len(restoreJobs.Items) == 1))

				if tc.result.expectedClusterCondition != nil {
					condition := meta.FindStatusCondition(cluster.Status.Conditions,
						tc.result.expectedClusterCondition.Type)
					if assert.Check(t, condition != nil) {
						assert.Equal(t, tc.result.expectedClusterCondition.Status, condition.Status)
						assert.Equal(t, tc.result.expectedClusterCondition.Reason, condition.Reason)
						assert.Equal(t, tc.result.expectedClusterCondition.Message, condition.Message)
					}
					if tc.result.expectedClusterCondition.Reason == ReasonReadyForRestore {
						assert.Assert(t, cluster.Status.Patroni == nil)
						assert.Assert(t, cluster.Status.Proxy.PGBouncer.PostgreSQLRevision == "")
						assert.Assert(t, cluster.Status.Monitoring.ExporterConfiguration == "")
						assert.Assert(t, meta.FindStatusCondition(cluster.Status.Conditions,
							ConditionPostgresDataInitialized) == nil)
					}
				}
			})
		}
	}
}

func TestReconcileScheduledBackups(t *testing.T) {
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

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "hippo-sa"},
	}

	testCases := []struct {
		// a description of the test
		testDesc string
		// whether or not the test only applies to configs with dedicated repo hosts
		dedicatedOnly bool
		// conditions to apply to the mock postgrescluster
		clusterConditions map[string]metav1.ConditionStatus
		// the status to apply to the mock postgrescluster
		status *v1beta1.PostgresClusterStatus
		// whether or not the test should expect a Job to be reconciled
		expectReconcile bool
		// whether or not the test should expect a Job to be requeued
		expectRequeue bool
		// the reason associated with the expected event for the test (can be empty if
		// no event is expected)
		expectedEventReason string
		// the observed instances
		instances *observedInstances
	}{
		{
			testDesc: "should reconcile, no requeue",
			clusterConditions: map[string]metav1.ConditionStatus{
				ConditionRepoHostReady: metav1.ConditionTrue,
				ConditionReplicaCreate: metav1.ConditionTrue,
			},
			status: &v1beta1.PostgresClusterStatus{
				Patroni: &v1beta1.PatroniStatus{SystemIdentifier: "12345abcde"},
				PGBackRest: &v1beta1.PGBackRestStatus{
					Repos: []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: true}}},
			},
			expectReconcile: true,
			expectRequeue:   false,
		}, {
			testDesc: "cluster not bootstrapped, should not reconcile",
			status: &v1beta1.PostgresClusterStatus{
				PGBackRest: &v1beta1.PGBackRestStatus{
					Repos: []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: true}}},
			},
			expectReconcile: false,
			expectRequeue:   false,
		}, {
			testDesc:      "no repo host ready condition, should not reconcile",
			dedicatedOnly: true,
			status: &v1beta1.PostgresClusterStatus{
				Patroni: &v1beta1.PatroniStatus{SystemIdentifier: "12345abcde"},
				PGBackRest: &v1beta1.PGBackRestStatus{
					Repos: []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: true}}},
			},
			expectReconcile: false,
			expectRequeue:   false,
		}, {
			testDesc: "no replica create condition, should not reconcile",
			status: &v1beta1.PostgresClusterStatus{
				Patroni: &v1beta1.PatroniStatus{SystemIdentifier: "12345abcde"},
				PGBackRest: &v1beta1.PGBackRestStatus{
					Repos: []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: true}}},
			},
			expectReconcile: false,
			expectRequeue:   false,
		}, {
			testDesc:      "false repo host ready condition, should not reconcile",
			dedicatedOnly: true,
			status: &v1beta1.PostgresClusterStatus{
				Patroni: &v1beta1.PatroniStatus{SystemIdentifier: "12345abcde"},
				PGBackRest: &v1beta1.PGBackRestStatus{
					Repos: []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: true}}},
			},
			expectReconcile: false,
			expectRequeue:   false,
		}, {
			testDesc: "false replica create condition, should not reconcile",
			status: &v1beta1.PostgresClusterStatus{
				Patroni: &v1beta1.PatroniStatus{SystemIdentifier: "12345abcde"},
				PGBackRest: &v1beta1.PGBackRestStatus{
					Repos: []v1beta1.RepoStatus{{Name: "repo1", StanzaCreated: true}}},
			},
			expectReconcile: false,
			expectRequeue:   false,
		}, {
			testDesc: "missing repo status, should not reconcile",
			clusterConditions: map[string]metav1.ConditionStatus{
				ConditionRepoHostReady: metav1.ConditionTrue,
				ConditionReplicaCreate: metav1.ConditionTrue,
			},
			status: &v1beta1.PostgresClusterStatus{
				Patroni: &v1beta1.PatroniStatus{SystemIdentifier: "12345abcde"},
				PGBackRest: &v1beta1.PGBackRestStatus{
					Repos: []v1beta1.RepoStatus{}},
			},
			expectReconcile:     false,
			expectRequeue:       false,
			expectedEventReason: "InvalidBackupRepo",
		}}

	for _, dedicated := range []bool{true, false} {
		for i, tc := range testCases {

			var clusterName string
			if !dedicated {
				tc.testDesc = "no repo " + tc.testDesc
				clusterName = "scheduled-backup-no-repo-" + strconv.Itoa(i)
			} else {
				clusterName = "scheduled-backup-" + strconv.Itoa(i)
			}

			t.Run(tc.testDesc, func(t *testing.T) {

				if tc.dedicatedOnly && !dedicated {
					t.Skip()
				}

				ctx := context.Background()

				postgresCluster := fakePostgresCluster(clusterName, ns.GetName(), "", dedicated)
				postgresCluster.Status = *tc.status
				for condition, status := range tc.clusterConditions {
					meta.SetStatusCondition(&postgresCluster.Status.Conditions, metav1.Condition{
						Type: condition, Reason: "testing", Status: status})
				}
				assert.NilError(t, tClient.Create(ctx, postgresCluster))
				assert.NilError(t, tClient.Status().Update(ctx, postgresCluster))

				var requeue bool
				if tc.instances != nil {
					requeue = r.reconcileScheduledBackups(ctx, postgresCluster, sa)
				} else {
					requeue = r.reconcileScheduledBackups(ctx, postgresCluster, sa)
				}

				if !tc.expectReconcile && !tc.expectRequeue {
					// expect no reconcile, no requeue
					assert.Assert(t, !requeue)

					// if an event is expected, the check for it
					if tc.expectedEventReason != "" {
						events := &corev1.EventList{}
						err := wait.Poll(time.Second/2, Scale(time.Second*2), func() (bool, error) {
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
				} else if !tc.expectReconcile && tc.expectRequeue {
					// expect requeue, no reconcile
					assert.Assert(t, requeue)
					return
				} else {
					// expect reconcile, no requeue
					assert.Assert(t, !requeue)

					// check for all three defined backup types
					backupTypes := []string{"full", "diff", "incr"}

					for _, backupType := range backupTypes {

						returnedCronJob := &batchv1beta1.CronJob{}
						if err := tClient.Get(ctx, types.NamespacedName{
							Name:      postgresCluster.Name + "-pgbackrest-repo1-" + backupType,
							Namespace: postgresCluster.GetNamespace(),
						}, returnedCronJob); err != nil {
							assert.NilError(t, err)
						}

						// check returned cronjob matches set spec
						assert.Equal(t, returnedCronJob.Name, clusterName+"-pgbackrest-repo1-"+backupType)
						assert.Equal(t, returnedCronJob.Spec.Schedule, testCronSchedule)
						assert.Equal(t, returnedCronJob.Spec.JobTemplate.Spec.Template.Spec.PriorityClassName, "some-priority-class")
						assert.Equal(t, returnedCronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Name,
							"pgbackrest")
						assert.Assert(t, returnedCronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].SecurityContext != &corev1.SecurityContext{})

						// verify the image pull secret
						if returnedCronJob.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets == nil {
							t.Error("image pull secret is missing tolerations")
						}

						if returnedCronJob.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets != nil {
							if returnedCronJob.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets[0].Name !=
								"myImagePullSecret" {
								t.Error("image pull secret name is not set correctly")
							}
						}
					}
					return
				}
			})
		}
	}
}

func TestSetScheduledJobStatus(t *testing.T) {

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

	t.Run("set scheduled backup status", func(t *testing.T) {
		// create a PostgresCluster to test with
		postgresCluster := fakePostgresCluster(clusterName, ns.GetName(), clusterUID, true)

		testJob := &batchv1.Job{
			TypeMeta: metav1.TypeMeta{
				Kind: "Job",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   "TestJob",
				Labels: map[string]string{"postgres-operator.crunchydata.com/pgbackrest-cronjob": "full"},
			},
			Status: batchv1.JobStatus{
				Active:    1,
				Succeeded: 2,
				Failed:    3,
			},
		}

		// convert the runtime.Object to an unstructured object
		unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(testJob)
		assert.NilError(t, err)
		unstructuredJob := &unstructured.Unstructured{
			Object: unstructuredObj,
		}

		// add it to an unstructured list
		uList := &unstructured.UnstructuredList{}
		uList.Items = append(uList.Items, *unstructuredJob)

		// set the status
		r.setScheduledJobStatus(ctx, postgresCluster, uList.Items)

		assert.Assert(t, len(postgresCluster.Status.PGBackRest.ScheduledBackups) > 0)
		assert.Equal(t, postgresCluster.Status.PGBackRest.ScheduledBackups[0].Active, int32(1))
		assert.Equal(t, postgresCluster.Status.PGBackRest.ScheduledBackups[0].Succeeded, int32(2))
		assert.Equal(t, postgresCluster.Status.PGBackRest.ScheduledBackups[0].Failed, int32(3))
	})

	t.Run("fail to set scheduled backup status due to missing label", func(t *testing.T) {
		// create a PostgresCluster to test with
		postgresCluster := fakePostgresCluster(clusterName, ns.GetName(), clusterUID, true)

		testJob := &batchv1.Job{
			TypeMeta: metav1.TypeMeta{
				Kind: "Job",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "TestJob",
			},
			Status: batchv1.JobStatus{
				Active:    1,
				Succeeded: 2,
				Failed:    3,
			},
		}

		// convert the runtime.Object to an unstructured object
		unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(testJob)
		assert.NilError(t, err)
		unstructuredJob := &unstructured.Unstructured{
			Object: unstructuredObj,
		}

		// add it to an unstructured list
		uList := &unstructured.UnstructuredList{}
		uList.Items = append(uList.Items, *unstructuredJob)

		// set the status
		r.setScheduledJobStatus(ctx, postgresCluster, uList.Items)
		assert.Assert(t, len(postgresCluster.Status.PGBackRest.ScheduledBackups) == 0)
	})
}
