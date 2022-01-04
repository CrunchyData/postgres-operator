/*
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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
