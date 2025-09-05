// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"testing"

	"github.com/go-logr/logr/funcr"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/events"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestStoreDesiredRequest(t *testing.T) {
	ctx := context.Background()

	setupLogCapture := func(ctx context.Context) (context.Context, *[]string) {
		calls := []string{}
		testlog := funcr.NewJSON(func(object string) {
			calls = append(calls, object)
		}, funcr.Options{
			Verbosity: 1,
		})
		return logging.NewContext(ctx, testlog), &calls
	}

	cluster := v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhino",
			Namespace: "test-namespace",
		},
		Spec: v1beta1.PostgresClusterSpec{
			InstanceSets: []v1beta1.PostgresInstanceSetSpec{{
				Name:     "red",
				Replicas: initialize.Int32(1),
				DataVolumeClaimSpec: v1beta1.VolumeClaimSpecWithAutoGrow{
					VolumeClaimSpec: v1beta1.VolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Limits: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceStorage: resource.MustParse("1Gi"),
							}}}},
				WALVolumeClaimSpec: &v1beta1.VolumeClaimSpecWithAutoGrow{
					VolumeClaimSpec: v1beta1.VolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Limits: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceStorage: resource.MustParse("1Gi"),
							}}}},
			}, {
				Name:               "blue",
				Replicas:           initialize.Int32(1),
				WALVolumeClaimSpec: &v1beta1.VolumeClaimSpecWithAutoGrow{},
			}, {
				Name:     "green",
				Replicas: initialize.Int32(1),
			}},
			Backups: v1beta1.Backups{
				PGBackRest: v1beta1.PGBackRestArchive{
					Repos: []v1beta1.PGBackRestRepo{{
						Name: "repo1",
						Volume: &v1beta1.RepoPVC{
							VolumeClaimSpec: v1beta1.VolumeClaimSpecWithAutoGrow{
								VolumeClaimSpec: v1beta1.VolumeClaimSpec{
									AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
									Resources: corev1.VolumeResourceRequirements{
										Limits: map[corev1.ResourceName]resource.Quantity{
											corev1.ResourceStorage: resource.MustParse("1Gi"),
										}}}},
						},
					}, {
						Name:   "repo2",
						Volume: &v1beta1.RepoPVC{},
					}, {
						Name: "repo3",
					}}},
			},
		},
	}

	testCases := []struct {
		tcName               string
		Voltype              string
		host                 string
		desiredRequest       string
		desiredRequestBackup string
		expectedValue        string
		expectedNumLogs      int
		expectedLog          string
		expectedNumEvents    int
		expectedEvent        string
	}{{
		tcName:  "PGData-BadRequestNoBackup",
		Voltype: "pgData", host: "red",
		desiredRequest: "woot", desiredRequestBackup: "", expectedValue: "",
		expectedNumEvents: 0, expectedNumLogs: 1,
		expectedLog: "Unable to parse pgData volume request from status (woot) for rhino/red",
	}, {
		tcName:  "PGData-BadRequestWithBackup",
		Voltype: "pgData", host: "red",
		desiredRequest: "foo", desiredRequestBackup: "1Gi", expectedValue: "1Gi",
		expectedNumEvents: 0, expectedNumLogs: 1,
		expectedLog: "Unable to parse pgData volume request from status (foo) for rhino/red",
	}, {
		tcName:  "PGData-NoLimitNoEvent",
		Voltype: "pgData", host: "blue",
		desiredRequest: "1Gi", desiredRequestBackup: "", expectedValue: "1Gi",
		expectedNumEvents: 0, expectedNumLogs: 0,
	}, {
		tcName:  "PGData-BadBackupRequest",
		Voltype: "pgData", host: "red",
		desiredRequest: "2Gi", desiredRequestBackup: "bar", expectedValue: "2Gi",
		expectedNumEvents: 1, expectedEvent: "pgData volume expansion to 2Gi requested for rhino/red.",
		expectedNumLogs: 1, expectedLog: "Unable to parse pgData volume request from status backup (bar) for rhino/red",
	}, {
		tcName:  "PGData-ValueUpdateWithEvent",
		Voltype: "pgData", host: "red",
		desiredRequest: "1Gi", desiredRequestBackup: "", expectedValue: "1Gi",
		expectedNumEvents: 1, expectedEvent: "pgData volume expansion to 1Gi requested for rhino/red.",
		expectedNumLogs: 0,
	}, {
		tcName:  "PGWAL-BadRequestNoBackup",
		Voltype: "pgWAL", host: "red",
		desiredRequest: "woot", desiredRequestBackup: "", expectedValue: "",
		expectedNumEvents: 0, expectedNumLogs: 1,
		expectedLog: "Unable to parse pgWAL volume request from status (woot) for rhino/red",
	}, {
		tcName:  "PGWAL-BadRequestWithBackup",
		Voltype: "pgWAL", host: "red",
		desiredRequest: "foo", desiredRequestBackup: "1Gi", expectedValue: "1Gi",
		expectedNumEvents: 0, expectedNumLogs: 1,
		expectedLog: "Unable to parse pgWAL volume request from status (foo) for rhino/red",
	}, {
		tcName:  "PGWAL-NoLimitNoEvent",
		Voltype: "pgWAL", host: "blue",
		desiredRequest: "1Gi", desiredRequestBackup: "", expectedValue: "1Gi",
		expectedNumEvents: 0, expectedNumLogs: 0,
	}, {
		tcName:  "PGWAL-NoVolumeDefined",
		Voltype: "pgWAL", host: "green",
		desiredRequest: "1Gi", desiredRequestBackup: "", expectedValue: "",
		expectedNumEvents: 0, expectedNumLogs: 0,
	}, {
		tcName:  "PGWAL-BadBackupRequest",
		Voltype: "pgWAL", host: "red",
		desiredRequest: "2Gi", desiredRequestBackup: "bar", expectedValue: "2Gi",
		expectedNumEvents: 1, expectedEvent: "pgWAL volume expansion to 2Gi requested for rhino/red.",
		expectedNumLogs: 1, expectedLog: "Unable to parse pgWAL volume request from status backup (bar) for rhino/red",
	}, {
		tcName:  "PGWAL-ValueUpdateWithEvent",
		Voltype: "pgWAL", host: "red",
		desiredRequest: "1Gi", desiredRequestBackup: "", expectedValue: "1Gi",
		expectedNumEvents: 1, expectedEvent: "pgWAL volume expansion to 1Gi requested for rhino/red.",
		expectedNumLogs: 0,
	}, {
		tcName:  "Repo-BadRequestNoBackup",
		Voltype: "repo1", host: "repo-host",
		desiredRequest: "woot", desiredRequestBackup: "", expectedValue: "",
		expectedNumEvents: 0, expectedNumLogs: 1,
		expectedLog: "Unable to parse repo1 volume request from status (woot) for rhino/repo-host",
	}, {
		tcName:  "Repo-BadRequestWithBackup",
		Voltype: "repo1", host: "repo-host",
		desiredRequest: "foo", desiredRequestBackup: "1Gi", expectedValue: "1Gi",
		expectedNumEvents: 0, expectedNumLogs: 1,
		expectedLog: "Unable to parse repo1 volume request from status (foo) for rhino/repo-host",
	}, {
		tcName:  "Repo-NoLimitNoEvent",
		Voltype: "repo2", host: "repo-host",
		desiredRequest: "1Gi", desiredRequestBackup: "", expectedValue: "1Gi",
		expectedNumEvents: 0, expectedNumLogs: 0,
	}, {
		tcName:  "Repo-NoRepoDefined",
		Voltype: "repo3", host: "repo-host",
		desiredRequest: "1Gi", desiredRequestBackup: "", expectedValue: "",
		expectedNumEvents: 0, expectedNumLogs: 0,
	}, {
		tcName:  "Repo-BadBackupRequest",
		Voltype: "repo1", host: "repo-host",
		desiredRequest: "2Gi", desiredRequestBackup: "bar", expectedValue: "2Gi",
		expectedNumEvents: 1, expectedEvent: "repo1 volume expansion to 2Gi requested for rhino/repo-host.",
		expectedNumLogs: 1, expectedLog: "Unable to parse repo1 volume request from status backup (bar) for rhino/repo-host",
	}, {
		tcName:  "Repo-ValueUpdateWithEvent",
		Voltype: "repo1", host: "repo-host",
		desiredRequest: "1Gi", desiredRequestBackup: "", expectedValue: "1Gi",
		expectedNumEvents: 1, expectedEvent: "repo1 volume expansion to 1Gi requested for rhino/repo-host.",
		expectedNumLogs: 0,
	}}

	for _, tc := range testCases {
		t.Run(tc.tcName, func(t *testing.T) {
			recorder := events.NewRecorder(t, runtime.Scheme)
			reconciler := &Reconciler{Recorder: recorder}
			ctx, logs := setupLogCapture(ctx)

			value := reconciler.storeDesiredRequest(
				ctx,
				&cluster,
				tc.Voltype,
				tc.host,
				tc.desiredRequest,
				tc.desiredRequestBackup,
			)
			assert.Equal(t, value, tc.expectedValue)
			assert.Equal(t, len(recorder.Events), tc.expectedNumEvents)
			if tc.expectedNumEvents == 1 {
				assert.Equal(t, recorder.Events[0].Regarding.Name, cluster.Name)
				assert.Equal(t, recorder.Events[0].Reason, "VolumeAutoGrow")
				assert.Equal(t, recorder.Events[0].Note, tc.expectedEvent)
			}
			assert.Equal(t, len(*logs), tc.expectedNumLogs)
			if tc.expectedNumLogs == 1 {
				assert.Assert(t, cmp.Contains((*logs)[0], tc.expectedLog))
			}
		})

	}
}

func TestLimitIsSet(t *testing.T) {

	cluster := v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhino",
			Namespace: "test-namespace",
		},
		Spec: v1beta1.PostgresClusterSpec{
			InstanceSets: []v1beta1.PostgresInstanceSetSpec{{
				Name:     "red",
				Replicas: initialize.Int32(1),
				DataVolumeClaimSpec: v1beta1.VolumeClaimSpecWithAutoGrow{
					VolumeClaimSpec: v1beta1.VolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Limits: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceStorage: resource.MustParse("1Gi"),
							}}}},
				WALVolumeClaimSpec: &v1beta1.VolumeClaimSpecWithAutoGrow{
					VolumeClaimSpec: v1beta1.VolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Limits: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceStorage: resource.MustParse("2Gi"),
							}}}},
			}, {
				Name:     "blue",
				Replicas: initialize.Int32(1),
			}, {
				Name:               "green",
				Replicas:           initialize.Int32(1),
				WALVolumeClaimSpec: &v1beta1.VolumeClaimSpecWithAutoGrow{},
			}},
			Backups: v1beta1.Backups{
				PGBackRest: v1beta1.PGBackRestArchive{
					Repos: []v1beta1.PGBackRestRepo{{
						Name: "repo1",
						Volume: &v1beta1.RepoPVC{
							VolumeClaimSpec: v1beta1.VolumeClaimSpecWithAutoGrow{
								VolumeClaimSpec: v1beta1.VolumeClaimSpec{
									AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
									Resources: corev1.VolumeResourceRequirements{
										Limits: map[corev1.ResourceName]resource.Quantity{
											corev1.ResourceStorage: resource.MustParse("1Gi"),
										},
									},
								},
							}}},
						{
							Name:   "repo2",
							Volume: &v1beta1.RepoPVC{},
						}, {
							Name: "repo3",
						},
					}}},
		}}

	testCases := []struct {
		tcName       string
		Voltype      string
		instanceName string
		expected     *bool
	}{{
		tcName:       "PGDATA Limit is set for defined volume",
		Voltype:      "pgData",
		instanceName: "red",
		expected:     initialize.Pointer(true),
	}, {
		tcName:       "PGDATA Limit is not set for defined volume",
		Voltype:      "pgData",
		instanceName: "blue",
		expected:     initialize.Pointer(false),
	}, {
		tcName:       "PGDATA Check volume for non-existent instance",
		Voltype:      "pgData",
		instanceName: "orange",
		expected:     nil,
	}, {
		tcName:       "PGWAL Limit is set for defined volume",
		Voltype:      "pgWAL",
		instanceName: "red",
		expected:     initialize.Pointer(true),
	}, {
		tcName:       "PGWAL WAL volume defined but limit is not set",
		Voltype:      "pgWAL",
		instanceName: "green",
		expected:     initialize.Pointer(false),
	}, {
		tcName:       "PGWAL Instance has no pg_wal volume defined",
		Voltype:      "pgWAL",
		instanceName: "blue",
		expected:     nil,
	}, {
		tcName:       "PGWAL Check volume for non-existent instance",
		Voltype:      "pgWAL",
		instanceName: "orange",
		expected:     nil,
	}, {
		tcName:       "REPO Limit set for defined volume",
		Voltype:      "repo1",
		instanceName: "",
		expected:     initialize.Pointer(true),
	}, {
		tcName:       "REPO Limit is not set for defined volume",
		Voltype:      "repo2",
		instanceName: "",
		expected:     initialize.Pointer(false),
	}, {
		tcName:       "REPO volume not defined for repo",
		Voltype:      "repo3",
		instanceName: "",
		expected:     nil,
	}, {
		tcName:       "REPO Check volume for non-existent repo",
		Voltype:      "repo4",
		instanceName: "",
		expected:     nil,
	}}

	for _, tc := range testCases {
		t.Run(tc.tcName, func(t *testing.T) {

			limitSet := limitIsSet(&cluster, tc.Voltype, tc.instanceName)
			if tc.expected == nil {
				assert.Assert(t, limitSet == nil)
			} else {
				assert.Assert(t, limitSet != nil)
				assert.Check(t, *limitSet == *tc.expected)
			}
		})
	}
}

func TestSetVolumeSize(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cluster := v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "elephant",
			Namespace: "test-namespace",
		},
		Spec: v1beta1.PostgresClusterSpec{
			InstanceSets: []v1beta1.PostgresInstanceSetSpec{{
				Name:     "some-instance",
				Replicas: initialize.Int32(1),
			}},
		},
	}

	setupLogCapture := func(ctx context.Context) (context.Context, *[]string) {
		calls := []string{}
		testlog := funcr.NewJSON(func(object string) {
			calls = append(calls, object)
		}, funcr.Options{
			Verbosity: 1,
		})
		return logging.NewContext(ctx, testlog), &calls
	}

	// helper functions
	pvcSpec := func(request, limit string) *corev1.PersistentVolumeClaimSpec {
		return &corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceStorage: resource.MustParse(request),
				},
				Limits: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceStorage: resource.MustParse(limit),
				}}}
	}

	desiredStatus := func(request string) v1beta1.PostgresClusterStatus {
		desiredMap := make(map[string]string)
		desiredMap["elephant-some-instance-wxyz-0"] = request
		return v1beta1.PostgresClusterStatus{
			InstanceSets: []v1beta1.PostgresInstanceSetStatus{{
				Name:                "some-instance",
				DesiredPGDataVolume: desiredMap,
			}}}
	}

	t.Run("RequestAboveLimit", func(t *testing.T) {
		recorder := events.NewRecorder(t, runtime.Scheme)
		reconciler := &Reconciler{Recorder: recorder}
		ctx, logs := setupLogCapture(ctx)

		spec := pvcSpec("4Gi", "3Gi")

		reconciler.setVolumeSize(ctx, &cluster, spec, "pgData", "some-instance")

		assert.Assert(t, cmp.MarshalMatches(spec, `
accessModes:
- ReadWriteOnce
resources:
  limits:
    storage: 3Gi
  requests:
    storage: 3Gi
`))
		assert.Equal(t, len(*logs), 0)
		assert.Equal(t, len(recorder.Events), 1)
		assert.Equal(t, recorder.Events[0].Regarding.Name, cluster.Name)
		assert.Equal(t, recorder.Events[0].Reason, "VolumeRequestOverLimit")
		assert.Equal(t, recorder.Events[0].Note, "pgData volume request (4Gi) for elephant/some-instance is greater than set limit (3Gi). Limit value will be used.")
	})

	t.Run("NoFeatureGate", func(t *testing.T) {
		recorder := events.NewRecorder(t, runtime.Scheme)
		reconciler := &Reconciler{Recorder: recorder}
		ctx, logs := setupLogCapture(ctx)

		spec := pvcSpec("1Gi", "3Gi")

		desiredMap := make(map[string]string)
		desiredMap["elephant-some-instance-wxyz-0"] = "2Gi"
		cluster.Status = v1beta1.PostgresClusterStatus{
			InstanceSets: []v1beta1.PostgresInstanceSetStatus{{
				Name:                "some-instance",
				DesiredPGDataVolume: desiredMap,
			}},
		}

		reconciler.setVolumeSize(ctx, &cluster, spec, "pgData", "some-instance")

		assert.Assert(t, cmp.MarshalMatches(spec, `
accessModes:
- ReadWriteOnce
resources:
  limits:
    storage: 3Gi
  requests:
    storage: 1Gi
	`))

		assert.Equal(t, len(recorder.Events), 0)
		assert.Equal(t, len(*logs), 0)

		// clear status for other tests
		cluster.Status = v1beta1.PostgresClusterStatus{}
	})

	t.Run("FeatureEnabled", func(t *testing.T) {
		gate := feature.NewGate()
		assert.NilError(t, gate.SetFromMap(map[string]bool{
			feature.AutoGrowVolumes: true,
		}))
		ctx := feature.NewContext(ctx, gate)

		t.Run("StatusNoLimit", func(t *testing.T) {
			recorder := events.NewRecorder(t, runtime.Scheme)
			reconciler := &Reconciler{Recorder: recorder}
			ctx, logs := setupLogCapture(ctx)

			spec := &corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceStorage: resource.MustParse("1Gi"),
					}}}
			cluster.Status = desiredStatus("2Gi")

			reconciler.setVolumeSize(ctx, &cluster, spec, "pgData", "some-instance")

			assert.Assert(t, cmp.MarshalMatches(spec, `
accessModes:
- ReadWriteOnce
resources:
  requests:
    storage: 1Gi
`))
			assert.Equal(t, len(recorder.Events), 0)
			assert.Equal(t, len(*logs), 0)

			// clear status for other tests
			cluster.Status = v1beta1.PostgresClusterStatus{}
		})

		t.Run("LimitNoStatus", func(t *testing.T) {
			recorder := events.NewRecorder(t, runtime.Scheme)
			reconciler := &Reconciler{Recorder: recorder}
			ctx, logs := setupLogCapture(ctx)

			spec := pvcSpec("1Gi", "2Gi")

			reconciler.setVolumeSize(ctx, &cluster, spec, "pgData", "some-instance")

			assert.Assert(t, cmp.MarshalMatches(spec, `
accessModes:
- ReadWriteOnce
resources:
  limits:
    storage: 2Gi
  requests:
    storage: 1Gi
`))
			assert.Equal(t, len(recorder.Events), 0)
			assert.Equal(t, len(*logs), 0)
		})

		t.Run("BadStatusWithLimit", func(t *testing.T) {
			recorder := events.NewRecorder(t, runtime.Scheme)
			reconciler := &Reconciler{Recorder: recorder}
			ctx, logs := setupLogCapture(ctx)

			spec := pvcSpec("1Gi", "3Gi")
			cluster.Status = desiredStatus("NotAValidValue")

			reconciler.setVolumeSize(ctx, &cluster, spec, "pgData", "some-instance")

			assert.Assert(t, cmp.MarshalMatches(spec, `
accessModes:
- ReadWriteOnce
resources:
  limits:
    storage: 3Gi
  requests:
    storage: 1Gi
`))

			assert.Equal(t, len(recorder.Events), 0)
			assert.Equal(t, len(*logs), 1)
			assert.Assert(t, cmp.Contains((*logs)[0],
				"For elephant/some-instance: Unable to parse pgData volume request: NotAValidValue"))
		})

		t.Run("StatusWithLimit", func(t *testing.T) {
			recorder := events.NewRecorder(t, runtime.Scheme)
			reconciler := &Reconciler{Recorder: recorder}
			ctx, logs := setupLogCapture(ctx)

			spec := pvcSpec("1Gi", "3Gi")
			cluster.Status = desiredStatus("2Gi")

			reconciler.setVolumeSize(ctx, &cluster, spec, "pgData", "some-instance")

			assert.Assert(t, cmp.MarshalMatches(spec, `
accessModes:
- ReadWriteOnce
resources:
  limits:
    storage: 3Gi
  requests:
    storage: 2Gi
`))
			assert.Equal(t, len(recorder.Events), 0)
			assert.Equal(t, len(*logs), 0)
		})

		t.Run("StatusWithLimitGrowToLimit", func(t *testing.T) {
			recorder := events.NewRecorder(t, runtime.Scheme)
			reconciler := &Reconciler{Recorder: recorder}
			ctx, logs := setupLogCapture(ctx)

			spec := pvcSpec("1Gi", "2Gi")
			cluster.Status = desiredStatus("2Gi")

			reconciler.setVolumeSize(ctx, &cluster, spec, "pgData", "some-instance")

			assert.Assert(t, cmp.MarshalMatches(spec, `
accessModes:
- ReadWriteOnce
resources:
  limits:
    storage: 2Gi
  requests:
    storage: 2Gi
`))

			assert.Equal(t, len(*logs), 0)
			assert.Equal(t, len(recorder.Events), 1)
			assert.Equal(t, recorder.Events[0].Regarding.Name, cluster.Name)
			assert.Equal(t, recorder.Events[0].Reason, "VolumeLimitReached")
			assert.Equal(t, recorder.Events[0].Note, "pgData volume(s) for elephant/some-instance are at size limit (2Gi).")
		})

		t.Run("DesiredStatusOverLimit", func(t *testing.T) {
			recorder := events.NewRecorder(t, runtime.Scheme)
			reconciler := &Reconciler{Recorder: recorder}
			ctx, logs := setupLogCapture(ctx)

			spec := pvcSpec("4Gi", "5Gi")
			cluster.Status = desiredStatus("10Gi")

			reconciler.setVolumeSize(ctx, &cluster, spec, "pgData", "some-instance")

			assert.Assert(t, cmp.MarshalMatches(spec, `
accessModes:
- ReadWriteOnce
resources:
  limits:
    storage: 5Gi
  requests:
    storage: 5Gi
`))

			assert.Equal(t, len(*logs), 0)
			assert.Equal(t, len(recorder.Events), 2)
			var found1, found2 bool
			for _, event := range recorder.Events {
				if event.Reason == "VolumeLimitReached" {
					found1 = true
					assert.Equal(t, event.Regarding.Name, cluster.Name)
					assert.Equal(t, event.Note, "pgData volume(s) for elephant/some-instance are at size limit (5Gi).")
				}
				if event.Reason == "DesiredVolumeAboveLimit" {
					found2 = true
					assert.Equal(t, event.Regarding.Name, cluster.Name)
					assert.Equal(t, event.Note,
						"The desired size (10Gi) for the elephant/some-instance pgData volume(s) is greater than the size limit (5Gi).")
				}
			}
			assert.Assert(t, found1 && found2)
		})

		// NB: The code in 'setVolumeSize' is identical no matter the volume type.
		// Since the different statuses are pulled out by the embedded 'getDesiredVolumeSize'
		// function, we can just try a couple scenarios to validate the behavior
		// for the repo and pg_wal volumes.
		t.Run("StatusWithLimitGrowToLimit-RepoHost", func(t *testing.T) {
			recorder := events.NewRecorder(t, runtime.Scheme)
			reconciler := &Reconciler{Recorder: recorder}
			ctx, logs := setupLogCapture(ctx)

			spec := pvcSpec("1Gi", "2Gi")

			cluster.Status = v1beta1.PostgresClusterStatus{
				PGBackRest: &v1beta1.PGBackRestStatus{
					Repos: []v1beta1.RepoStatus{{
						Name:              "repo1",
						DesiredRepoVolume: "2Gi",
					}}}}

			reconciler.setVolumeSize(ctx, &cluster, spec, "repo1", "repo-host")

			assert.Assert(t, cmp.MarshalMatches(spec, `
accessModes:
- ReadWriteOnce
resources:
  limits:
    storage: 2Gi
  requests:
    storage: 2Gi
`))

			assert.Equal(t, len(*logs), 0)
			assert.Equal(t, len(recorder.Events), 1)
			assert.Equal(t, recorder.Events[0].Regarding.Name, cluster.Name)
			assert.Equal(t, recorder.Events[0].Reason, "VolumeLimitReached")
			assert.Equal(t, recorder.Events[0].Note, "repo1 volume(s) for elephant/repo-host are at size limit (2Gi).")
		})

		t.Run("StatusWithLimitGrowToLimit-pgWAL", func(t *testing.T) {
			recorder := events.NewRecorder(t, runtime.Scheme)
			reconciler := &Reconciler{Recorder: recorder}
			ctx, logs := setupLogCapture(ctx)

			spec := pvcSpec("2Gi", "3Gi")

			cluster.Status = v1beta1.PostgresClusterStatus{
				InstanceSets: []v1beta1.PostgresInstanceSetStatus{{
					Name:               "another-instance",
					DesiredPGWALVolume: map[string]string{"elephant-another-instance-abcd-0": "3Gi"},
				}}}

			reconciler.setVolumeSize(ctx, &cluster, spec, "pgWAL", "another-instance")

			assert.Assert(t, cmp.MarshalMatches(spec, `
accessModes:
- ReadWriteOnce
resources:
  limits:
    storage: 3Gi
  requests:
    storage: 3Gi
`))

			assert.Equal(t, len(*logs), 0)
			assert.Equal(t, len(recorder.Events), 1)
			assert.Equal(t, recorder.Events[0].Regarding.Name, cluster.Name)
			assert.Equal(t, recorder.Events[0].Reason, "VolumeLimitReached")
			assert.Equal(t, recorder.Events[0].Note, "pgWAL volume(s) for elephant/another-instance are at size limit (3Gi).")
		})

	})
}

func TestGetDesiredVolumeSize(t *testing.T) {
	t.Parallel()

	cluster := v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "elephant",
			Namespace: "test-namespace",
		},
		Spec: v1beta1.PostgresClusterSpec{
			InstanceSets: []v1beta1.PostgresInstanceSetSpec{{
				Name:     "some-instance",
				Replicas: initialize.Int32(1),
			}},
		},
	}

	pgDataStatus := func(request string) v1beta1.PostgresClusterStatus {
		desiredMap := make(map[string]string)
		desiredMap["elephant-some-instance-wxyz-0"] = request
		return v1beta1.PostgresClusterStatus{
			InstanceSets: []v1beta1.PostgresInstanceSetStatus{{
				Name:                "some-instance",
				DesiredPGDataVolume: desiredMap,
				DesiredPGWALVolume:  desiredMap,
			}, {
				Name:                "another-instance",
				DesiredPGDataVolume: desiredMap,
			}},
			PGBackRest: &v1beta1.PGBackRestStatus{
				Repos: []v1beta1.RepoStatus{{
					Name:              "repo1",
					DesiredRepoVolume: request,
				}}}}
	}

	testCases := []struct {
		tcName         string
		sizeFromStatus string
		pvcRequestSize string
		volType        string
		host           string
		expected       string
		expectedError  string
		expectedDPV    string
	}{{
		tcName:         "pgdata-Larger size requested",
		sizeFromStatus: "3Gi",
		pvcRequestSize: "2Gi",
		volType:        "pgData",
		host:           "some-instance",
		expected:       "3Gi",
	}, {
		tcName:         "pgdata-PVC is desired size",
		sizeFromStatus: "2Gi",
		pvcRequestSize: "2Gi",
		volType:        "pgData",
		host:           "some-instance",
		expected:       "2Gi",
	}, {
		tcName:         "pgdata-Original larger than status request",
		sizeFromStatus: "1Gi",
		pvcRequestSize: "2Gi",
		volType:        "pgData",
		host:           "some-instance",
		expected:       "2Gi",
	}, {
		tcName:         "pgdata-Instance doesn't exist",
		sizeFromStatus: "2Gi",
		pvcRequestSize: "1Gi",
		volType:        "pgData",
		host:           "not-an-instance",
		expected:       "1Gi",
	}, {
		tcName:         "pgdata-Bad Value",
		sizeFromStatus: "batman",
		pvcRequestSize: "1Gi",
		volType:        "pgData",
		host:           "some-instance",
		expected:       "1Gi",
		expectedError:  "quantities must match the regular expression",
		expectedDPV:    "batman",
	}, {
		tcName:         "pgwal-Larger size requested",
		sizeFromStatus: "3Gi",
		pvcRequestSize: "2Gi",
		volType:        "pgWAL",
		host:           "some-instance",
		expected:       "3Gi",
	}, {
		tcName:         "pgwal-PVC is desired size",
		sizeFromStatus: "2Gi",
		pvcRequestSize: "2Gi",
		volType:        "pgWAL",
		host:           "some-instance",
		expected:       "2Gi",
	}, {
		tcName:         "pgwal-Original larger than status request",
		sizeFromStatus: "1Gi",
		pvcRequestSize: "2Gi",
		volType:        "pgWAL",
		host:           "some-instance",
		expected:       "2Gi",
	}, {
		tcName:         "pgwal-Instance doesn't exist",
		sizeFromStatus: "2Gi",
		pvcRequestSize: "1Gi",
		volType:        "pgWAL",
		host:           "not-an-instance",
		expected:       "1Gi",
	}, {
		tcName:         "pgwal-Bad Value",
		sizeFromStatus: "batman",
		pvcRequestSize: "1Gi",
		volType:        "pgWAL",
		host:           "some-instance",
		expected:       "1Gi",
		expectedError:  "quantities must match the regular expression",
		expectedDPV:    "batman",
	}, {
		tcName:         "pgwal-No value set for instance",
		sizeFromStatus: "batman",
		pvcRequestSize: "5Gi",
		volType:        "pgWAL",
		host:           "another-instance",
		expected:       "5Gi",
	}, {
		tcName:         "repo1-Larger size requested",
		sizeFromStatus: "3Gi",
		pvcRequestSize: "2Gi",
		volType:        "repo1",
		host:           "repo-host",
		expected:       "3Gi",
	}, {
		tcName:         "repo1-PVC is desired size",
		sizeFromStatus: "2Gi",
		pvcRequestSize: "2Gi",
		volType:        "repo1",
		host:           "repo-host",
		expected:       "2Gi",
	}, {
		tcName:         "repo1-Original larger than status request",
		sizeFromStatus: "1Gi",
		pvcRequestSize: "2Gi",
		volType:        "repo1",
		host:           "repo-host",
		expected:       "2Gi",
	}, {
		tcName:         "repo1-repo doesn't exist",
		sizeFromStatus: "2Gi",
		pvcRequestSize: "1Gi",
		volType:        "repo2",
		host:           "repo-host",
		expected:       "1Gi",
	}, {
		tcName:         "repo1-Bad Value",
		sizeFromStatus: "robin",
		pvcRequestSize: "1Gi",
		volType:        "repo1",
		host:           "repo-host",
		expected:       "1Gi",
		expectedError:  "quantities must match the regular expression",
		expectedDPV:    "robin",
	}}

	for _, tc := range testCases {
		t.Run(tc.tcName, func(t *testing.T) {

			cluster.Status = pgDataStatus(tc.sizeFromStatus)
			request, err := resource.ParseQuantity(tc.pvcRequestSize)
			assert.NilError(t, err)

			dpv, err := getDesiredVolumeSize(&cluster, tc.volType, tc.host, &request)
			assert.Equal(t, request.String(), tc.expected)

			assert.Assert(t, dpv == tc.expectedDPV)

			if tc.expectedError == "" {
				assert.NilError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.expectedError)
			}
		})
	}

	// run this case separately since's it's handling a unique case
	t.Run("repo1-No repo status defined", func(t *testing.T) {

		// set status to nil
		cluster.Status.PGBackRest = nil
		request, err := resource.ParseQuantity("1Gi")
		assert.NilError(t, err)

		dpv, err := getDesiredVolumeSize(&cluster, "repo1", "repo-host", &request)
		assert.Equal(t, request.String(), "1Gi")

		assert.Assert(t, dpv == "")
		assert.ErrorContains(t, err, "PostgresCluster.Status.PGBackRest is nil")
	})

}
