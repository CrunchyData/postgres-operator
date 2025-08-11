// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"testing"

	"github.com/go-logr/logr/funcr"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/naming"
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
				DataVolumeClaimSpec: v1beta1.VolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.VolumeResourceRequirements{
						Limits: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						}}},
			}, {
				Name:     "blue",
				Replicas: initialize.Int32(1),
			}}}}

	t.Run("BadRequestNoBackup", func(t *testing.T) {
		recorder := events.NewRecorder(t, runtime.Scheme)
		reconciler := &Reconciler{Recorder: recorder}
		ctx, logs := setupLogCapture(ctx)

		value := reconciler.storeDesiredRequest(ctx, &cluster, "pgData", "red", "woot", "")

		assert.Equal(t, value, "")
		assert.Equal(t, len(recorder.Events), 0)
		assert.Equal(t, len(*logs), 1)
		assert.Assert(t, cmp.Contains((*logs)[0], "Unable to parse pgData volume request from status"))
	})

	t.Run("BadRequestWithBackup", func(t *testing.T) {
		recorder := events.NewRecorder(t, runtime.Scheme)
		reconciler := &Reconciler{Recorder: recorder}
		ctx, logs := setupLogCapture(ctx)

		value := reconciler.storeDesiredRequest(ctx, &cluster, "pgData", "red", "foo", "1Gi")

		assert.Equal(t, value, "1Gi")
		assert.Equal(t, len(recorder.Events), 0)
		assert.Equal(t, len(*logs), 1)
		assert.Assert(t, cmp.Contains((*logs)[0], "Unable to parse pgData volume request from status (foo) for rhino/red"))
	})

	t.Run("NoLimitNoEvent", func(t *testing.T) {
		recorder := events.NewRecorder(t, runtime.Scheme)
		reconciler := &Reconciler{Recorder: recorder}
		ctx, logs := setupLogCapture(ctx)

		value := reconciler.storeDesiredRequest(ctx, &cluster, "pgData", "blue", "1Gi", "")

		assert.Equal(t, value, "1Gi")
		assert.Equal(t, len(*logs), 0)
		assert.Equal(t, len(recorder.Events), 0)
	})

	t.Run("BadBackupRequest", func(t *testing.T) {
		recorder := events.NewRecorder(t, runtime.Scheme)
		reconciler := &Reconciler{Recorder: recorder}
		ctx, logs := setupLogCapture(ctx)

		value := reconciler.storeDesiredRequest(ctx, &cluster, "pgData", "red", "2Gi", "bar")

		assert.Equal(t, value, "2Gi")
		assert.Equal(t, len(*logs), 1)
		assert.Assert(t, cmp.Contains((*logs)[0], "Unable to parse pgData volume request from status backup (bar) for rhino/red"))
		assert.Equal(t, len(recorder.Events), 1)
		assert.Equal(t, recorder.Events[0].Regarding.Name, cluster.Name)
		assert.Equal(t, recorder.Events[0].Reason, "VolumeAutoGrow")
		assert.Equal(t, recorder.Events[0].Note, "pgData volume expansion to 2Gi requested for rhino/red.")
	})

	t.Run("ValueUpdateWithEvent", func(t *testing.T) {
		recorder := events.NewRecorder(t, runtime.Scheme)
		reconciler := &Reconciler{Recorder: recorder}
		ctx, logs := setupLogCapture(ctx)

		value := reconciler.storeDesiredRequest(ctx, &cluster, "pgData", "red", "1Gi", "")

		assert.Equal(t, value, "1Gi")
		assert.Equal(t, len(*logs), 0)
		assert.Equal(t, len(recorder.Events), 1)
		assert.Equal(t, recorder.Events[0].Regarding.Name, cluster.Name)
		assert.Equal(t, recorder.Events[0].Reason, "VolumeAutoGrow")
		assert.Equal(t, recorder.Events[0].Note, "pgData volume expansion to 1Gi requested for rhino/red.")
	})

	t.Run("NoLimitNoEvent", func(t *testing.T) {
		recorder := events.NewRecorder(t, runtime.Scheme)
		reconciler := &Reconciler{Recorder: recorder}
		ctx, logs := setupLogCapture(ctx)

		value := reconciler.storeDesiredRequest(ctx, &cluster, "pgData", "blue", "1Gi", "")

		assert.Equal(t, value, "1Gi")
		assert.Equal(t, len(*logs), 0)
		assert.Equal(t, len(recorder.Events), 0)
	})
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
				DataVolumeClaimSpec: v1beta1.VolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.VolumeResourceRequirements{
						Limits: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						}}},
			}, {
				Name:     "blue",
				Replicas: initialize.Int32(1),
			}}}}

	testCases := []struct {
		tcName       string
		Voltype      string
		instanceName string
		expected     bool
	}{{
		tcName:       "Limit is set for instance PGDATA volume",
		Voltype:      "pgData",
		instanceName: "red",
		expected:     true,
	}, {
		tcName:       "Limit is not set for instance PGDATA volume",
		Voltype:      "pgData",
		instanceName: "blue",
		expected:     false,
	}, {
		tcName:       "Check PGDATA volume for non-existent instance",
		Voltype:      "pgData",
		instanceName: "orange",
		expected:     false,
	}}

	for _, tc := range testCases {
		t.Run(tc.tcName, func(t *testing.T) {

			limitSet := limitIsSet(&cluster, tc.Voltype, tc.instanceName)
			assert.Check(t, limitSet == tc.expected)
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

	instance := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "elephant-some-instance-wxyz-0",
			Namespace: cluster.Namespace,
		}}

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
	instanceSetSpec := func(request, limit string) *v1beta1.PostgresInstanceSetSpec {
		return &v1beta1.PostgresInstanceSetSpec{
			Name: "some-instance",
			DataVolumeClaimSpec: v1beta1.VolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceStorage: resource.MustParse(request),
					},
					Limits: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceStorage: resource.MustParse(limit),
					}}}}
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

		pvc := &corev1.PersistentVolumeClaim{ObjectMeta: naming.InstancePostgresDataVolume(instance)}
		spec := instanceSetSpec("4Gi", "3Gi")
		pvc.Spec = spec.DataVolumeClaimSpec.AsPersistentVolumeClaimSpec()

		reconciler.setVolumeSize(ctx, &cluster, pvc, "pgData", spec.Name)

		assert.Assert(t, cmp.MarshalMatches(pvc.Spec, `
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

		pvc := &corev1.PersistentVolumeClaim{ObjectMeta: naming.InstancePostgresDataVolume(instance)}
		spec := instanceSetSpec("1Gi", "3Gi")

		desiredMap := make(map[string]string)
		desiredMap["elephant-some-instance-wxyz-0"] = "2Gi"
		cluster.Status = v1beta1.PostgresClusterStatus{
			InstanceSets: []v1beta1.PostgresInstanceSetStatus{{
				Name:                "some-instance",
				DesiredPGDataVolume: desiredMap,
			}},
		}

		pvc.Spec = spec.DataVolumeClaimSpec.AsPersistentVolumeClaimSpec()

		reconciler.setVolumeSize(ctx, &cluster, pvc, "pgData", spec.Name)

		assert.Assert(t, cmp.MarshalMatches(pvc.Spec, `
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

			pvc := &corev1.PersistentVolumeClaim{ObjectMeta: naming.InstancePostgresDataVolume(instance)}
			spec := &v1beta1.PostgresInstanceSetSpec{
				Name: "some-instance",
				DataVolumeClaimSpec: v1beta1.VolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.VolumeResourceRequirements{
						Requests: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						}}}}
			cluster.Status = desiredStatus("2Gi")
			pvc.Spec = spec.DataVolumeClaimSpec.AsPersistentVolumeClaimSpec()

			reconciler.setVolumeSize(ctx, &cluster, pvc, "pgData", spec.Name)

			assert.Assert(t, cmp.MarshalMatches(pvc.Spec, `
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

			pvc := &corev1.PersistentVolumeClaim{ObjectMeta: naming.InstancePostgresDataVolume(instance)}
			spec := instanceSetSpec("1Gi", "2Gi")
			pvc.Spec = spec.DataVolumeClaimSpec.AsPersistentVolumeClaimSpec()

			reconciler.setVolumeSize(ctx, &cluster, pvc, "pgData", spec.Name)

			assert.Assert(t, cmp.MarshalMatches(pvc.Spec, `
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

			pvc := &corev1.PersistentVolumeClaim{ObjectMeta: naming.InstancePostgresDataVolume(instance)}
			spec := instanceSetSpec("1Gi", "3Gi")
			cluster.Status = desiredStatus("NotAValidValue")
			pvc.Spec = spec.DataVolumeClaimSpec.AsPersistentVolumeClaimSpec()

			reconciler.setVolumeSize(ctx, &cluster, pvc, "pgData", spec.Name)

			assert.Assert(t, cmp.MarshalMatches(pvc.Spec, `
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

			pvc := &corev1.PersistentVolumeClaim{ObjectMeta: naming.InstancePostgresDataVolume(instance)}
			spec := instanceSetSpec("1Gi", "3Gi")
			cluster.Status = desiredStatus("2Gi")
			pvc.Spec = spec.DataVolumeClaimSpec.AsPersistentVolumeClaimSpec()

			reconciler.setVolumeSize(ctx, &cluster, pvc, "pgData", spec.Name)

			assert.Assert(t, cmp.MarshalMatches(pvc.Spec, `
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

			pvc := &corev1.PersistentVolumeClaim{ObjectMeta: naming.InstancePostgresDataVolume(instance)}
			spec := instanceSetSpec("1Gi", "2Gi")
			cluster.Status = desiredStatus("2Gi")
			pvc.Spec = spec.DataVolumeClaimSpec.AsPersistentVolumeClaimSpec()

			reconciler.setVolumeSize(ctx, &cluster, pvc, "pgData", spec.Name)

			assert.Assert(t, cmp.MarshalMatches(pvc.Spec, `
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

			pvc := &corev1.PersistentVolumeClaim{ObjectMeta: naming.InstancePostgresDataVolume(instance)}
			spec := instanceSetSpec("4Gi", "5Gi")
			cluster.Status = desiredStatus("10Gi")
			pvc.Spec = spec.DataVolumeClaimSpec.AsPersistentVolumeClaimSpec()

			reconciler.setVolumeSize(ctx, &cluster, pvc, "pgData", spec.Name)

			assert.Assert(t, cmp.MarshalMatches(pvc.Spec, `
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

	})
}

func TestDetermineDesiredVolumeRequest(t *testing.T) {
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
			}}}
	}

	testCases := []struct {
		tcName         string
		sizeFromStatus string
		pvcRequestSize string
		volType        string
		instanceName   string
		expected       string
	}{{
		tcName:         "Larger size requested",
		sizeFromStatus: "3Gi",
		pvcRequestSize: "2Gi",
		volType:        "pgData",
		instanceName:   "some-instance",
		expected:       "3Gi",
	}, {
		tcName:         "PVC is desired size",
		sizeFromStatus: "2Gi",
		pvcRequestSize: "2Gi",
		volType:        "pgData",
		instanceName:   "some-instance",
		expected:       "2Gi",
	}, {
		tcName:         "Original larger than status request",
		sizeFromStatus: "1Gi",
		pvcRequestSize: "2Gi",
		volType:        "pgData",
		instanceName:   "some-instance",
		expected:       "2Gi",
	}, {
		tcName:         "Instance doesn't exist",
		sizeFromStatus: "2Gi",
		pvcRequestSize: "1Gi",
		volType:        "pgData",
		instanceName:   "not-an-instance",
		expected:       "1Gi",
	}, {
		tcName:         "Bad Value",
		sizeFromStatus: "batman",
		pvcRequestSize: "1Gi",
		volType:        "pgData",
		instanceName:   "some-instance",
		expected:       "1Gi",
	}}

	for _, tc := range testCases {
		t.Run(tc.tcName, func(t *testing.T) {

			cluster.Status = pgDataStatus(tc.sizeFromStatus)
			request, err := resource.ParseQuantity(tc.pvcRequestSize)
			assert.NilError(t, err)

			dpv, err := getDesiredVolumeSize(&cluster, tc.volType, tc.instanceName, &request)
			assert.Equal(t, request.String(), tc.expected)

			if tc.tcName != "Bad Value" {
				assert.NilError(t, err)
				assert.Assert(t, dpv == "")
			} else {
				assert.ErrorContains(t, err, "quantities must match the regular expression")
				assert.Assert(t, dpv == "batman")
			}
		})
	}

}
