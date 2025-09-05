// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestAddAdditionalVolumesAndMounts(t *testing.T) {
	t.Parallel()

	podSpec := corev1.PodSpec{
		InitContainers: []corev1.Container{
			{Name: "startup"},
			{Name: "config"},
		},
		Containers: []corev1.Container{
			{Name: "database"},
			{Name: "other"},
		},
	}

	testCases := []struct {
		tcName                 string
		additionalVolumes      []v1beta1.AdditionalVolume
		expectedContainers     string
		expectedInitContainers string
		expectedVolumes        string
		expectedMissing        []string
	}{{
		tcName: "all containers",
		additionalVolumes: []v1beta1.AdditionalVolume{{
			ClaimName: "required",
			Name:      "required",
		}},
		expectedContainers: `- name: database
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required
- name: other
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required`,
		expectedInitContainers: `- name: startup
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required
- name: config
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required`,
		expectedVolumes: `- name: volumes-required
  persistentVolumeClaim:
    claimName: required`,
		expectedMissing: []string{},
	}, {
		tcName: "no containers",
		additionalVolumes: []v1beta1.AdditionalVolume{{
			Containers: []string{},
			ClaimName:  "required",
			Name:       "required",
		}},
		expectedContainers: `- name: database
  resources: {}
- name: other
  resources: {}`,
		expectedInitContainers: `- name: startup
  resources: {}
- name: config
  resources: {}`,
		expectedVolumes: `- name: volumes-required
  persistentVolumeClaim:
    claimName: required`,
		expectedMissing: []string{},
	}, {
		tcName: "multiple volumes",
		additionalVolumes: []v1beta1.AdditionalVolume{{
			ClaimName: "required",
			Name:      "required",
		}, {
			ClaimName: "also",
			Name:      "other",
		}},
		expectedContainers: `- name: database
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required
  - mountPath: /volumes/other
    name: volumes-other
- name: other
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required
  - mountPath: /volumes/other
    name: volumes-other`,
		expectedInitContainers: `- name: startup
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required
  - mountPath: /volumes/other
    name: volumes-other
- name: config
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required
  - mountPath: /volumes/other
    name: volumes-other`,
		expectedVolumes: `- name: volumes-required
  persistentVolumeClaim:
    claimName: required
- name: volumes-other
  persistentVolumeClaim:
    claimName: also`,
		expectedMissing: []string{},
	}, {
		tcName: "database and startup containers only",
		additionalVolumes: []v1beta1.AdditionalVolume{{
			Containers: []string{"database", "startup"},
			ClaimName:  "required",
			Name:       "required",
		}},
		expectedContainers: `- name: database
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required
- name: other
  resources: {}`,
		expectedInitContainers: `- name: startup
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required
- name: config
  resources: {}`,
		expectedVolumes: `- name: volumes-required
  persistentVolumeClaim:
    claimName: required`,
		expectedMissing: []string{},
	}, {
		tcName: "container is missing",
		additionalVolumes: []v1beta1.AdditionalVolume{{
			Containers: []string{"database", "startup", "missing", "container"},
			ClaimName:  "required",
			Name:       "required",
		}},
		expectedContainers: `- name: database
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required
- name: other
  resources: {}`,
		expectedInitContainers: `- name: startup
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required
- name: config
  resources: {}`,
		expectedVolumes: `- name: volumes-required
  persistentVolumeClaim:
    claimName: required`,
		expectedMissing: []string{"missing", "container"},
	}, {
		tcName: "readonly",
		additionalVolumes: []v1beta1.AdditionalVolume{{
			Containers: []string{"database"},
			ClaimName:  "required",
			Name:       "required",
			ReadOnly:   true,
		}},
		expectedContainers: `- name: database
  resources: {}
  volumeMounts:
  - mountPath: /volumes/required
    name: volumes-required
    readOnly: true
- name: other
  resources: {}`,
		expectedInitContainers: `- name: startup
  resources: {}
- name: config
  resources: {}`,
		expectedVolumes: `- name: volumes-required
  persistentVolumeClaim:
    claimName: required
    readOnly: true`,
		expectedMissing: []string{},
	}}

	for _, tc := range testCases {
		t.Run(tc.tcName, func(t *testing.T) {
			sink := podSpec.DeepCopy()
			missingContainers := AddAdditionalVolumesAndMounts(sink, tc.additionalVolumes)

			assert.Assert(t, cmp.MarshalMatches(sink.Containers, tc.expectedContainers))
			assert.Assert(t, cmp.MarshalMatches(sink.InitContainers, tc.expectedInitContainers))
			assert.Assert(t, cmp.MarshalMatches(sink.Volumes, tc.expectedVolumes))

			slices.Sort(missingContainers)
			slices.Sort(tc.expectedMissing)
			assert.DeepEqual(t, missingContainers, tc.expectedMissing)
		})
	}
}

func TestAddVolumeAndMountsToPod(t *testing.T) {
	pod := &corev1.PodSpec{
		Containers: []corev1.Container{
			{Name: "database"},
			{Name: "other"},
			{Name: "pgbackrest"},
		},
		InitContainers: []corev1.Container{
			{Name: "initializer"},
			{Name: "another"},
		},
	}

	alwaysExpect := func(t testing.TB, result *corev1.PodSpec) {
		// Only Containers, InitContainers, and Volumes fields have changed.
		assert.DeepEqual(t, *pod, *result, cmpopts.IgnoreFields(*pod, "Containers", "InitContainers", "Volumes"))

		// Volume is mounted to all containers
		assert.Assert(t, cmp.MarshalMatches(result.Containers, `
- name: database
  resources: {}
  volumeMounts:
  - mountPath: /volumes/volume-name
    name: volume-name
- name: other
  resources: {}
  volumeMounts:
  - mountPath: /volumes/volume-name
    name: volume-name
- name: pgbackrest
  resources: {}
  volumeMounts:
  - mountPath: /volumes/volume-name
    name: volume-name
		`))

		// Volume is mounted to all init containers
		assert.Assert(t, cmp.MarshalMatches(result.InitContainers, `
- name: initializer
  resources: {}
  volumeMounts:
  - mountPath: /volumes/volume-name
    name: volume-name
- name: another
  resources: {}
  volumeMounts:
  - mountPath: /volumes/volume-name
    name: volume-name
		`))
	}

	out := pod.DeepCopy()
	AddCloudLogVolumeToPod(out, "volume-name")
	alwaysExpect(t, out)
}
