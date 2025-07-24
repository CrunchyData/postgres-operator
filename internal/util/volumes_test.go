// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
)

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

	volume := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "volume-name",
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
	AddVolumeAndMountsToPod(out, volume)
	alwaysExpect(t, out)
}
