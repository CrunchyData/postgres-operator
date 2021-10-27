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

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
)

// PGAdminPodSpec defines the desired state of a pgAdmin deployment.
type PGAdminPodSpec struct {
	// +optional
	Metadata *Metadata `json:"metadata,omitempty"`

	// Scheduling constraints of a pgAdmin pod. Changing this value causes
	// pgAdmin to restart.
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// Defines a PersistentVolumeClaim for pgAdmin data.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes
	// +kubebuilder:validation:Required
	DataVolumeClaimSpec corev1.PersistentVolumeClaimSpec `json:"dataVolumeClaimSpec"`

	// Name of a container image that can run pgAdmin 4. Changing this value causes
	// pgAdmin to restart. The image may also be set using the RELATED_IMAGE_PGADMIN
	// environment variable.
	// More info: https://kubernetes.io/docs/concepts/containers/images
	// +optional
	Image string `json:"image,omitempty"`

	// Port on which pgAdmin should listen for client connections. Changing
	// this value causes pgAdmin to restart.
	// +optional
	// +kubebuilder:default=5050
	// +kubebuilder:validation:Minimum=1024
	Port *int32 `json:"port,omitempty"`

	// Priority class name for the pgAdmin pod. Changing this value causes PostgreSQL
	// to restart.
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/
	// +optional
	PriorityClassName *string `json:"priorityClassName,omitempty"`

	// Number of desired pgAdmin pods.
	// +optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1
	Replicas *int32 `json:"replicas,omitempty"`

	// Compute resources of a pgAdmin container. Changing this value causes
	// pgAdmin to restart.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Specification of the service that exposes pgAdmin.
	// +optional
	Service *ServiceSpec `json:"service,omitempty"`

	// Tolerations of a pgAdmin pod. Changing this value causes pgAdmin to restart.
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Topology spread constraints of a pgAdmin pod. Changing this value causes
	// pgAdmin to restart.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-topology-spread-constraints/
	// +optional
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
}

// Default sets the port and replica count for pgAdmin if not set
func (s *PGAdminPodSpec) Default() {
	if s.Port == nil {
		s.Port = new(int32)
		*s.Port = 5050
	}

	if s.Replicas == nil {
		s.Replicas = new(int32)
		*s.Replicas = 1
	}
}

// PGAdminPodStatus represents the observed state of a pgAdmin deployment.
type PGAdminPodStatus struct {

	// Hash that indicates which users have been installed into pgAdmin.
	UsersRevision string `json:"usersRevision,omitempty"`
}
