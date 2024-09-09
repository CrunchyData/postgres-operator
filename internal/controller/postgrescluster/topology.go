// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// defaultTopologySpreadConstraints returns constraints that prefer to schedule
// pods on different nodes and in different zones.
func defaultTopologySpreadConstraints(selector metav1.LabelSelector) []corev1.TopologySpreadConstraint {
	return []corev1.TopologySpreadConstraint{
		{
			TopologyKey:       corev1.LabelHostname,
			WhenUnsatisfiable: corev1.ScheduleAnyway,
			LabelSelector:     &selector, MaxSkew: 1,
		},
		{
			TopologyKey:       corev1.LabelTopologyZone,
			WhenUnsatisfiable: corev1.ScheduleAnyway,
			LabelSelector:     &selector, MaxSkew: 1,
		},
	}
}
