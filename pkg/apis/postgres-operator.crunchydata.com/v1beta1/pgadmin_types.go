/*
 Copyright 2021 - 2023 Crunchy Data Solutions, Inc.
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

// PGAdminConfiguration represents pgAdmin configuration files.
type PGAdminConfiguration struct {
	// Files allows the user to mount projected volumes into the pgAdmin
	// container so that files can be referenced by pgAdmin as needed.
	Files []corev1.VolumeProjection `json:"files,omitempty"`

	// A Secret containing the value for the LDAP_BIND_PASSWORD setting.
	// More info: https://www.pgadmin.org/docs/pgadmin4/latest/ldap.html
	// +optional
	LDAPBindPassword *corev1.SecretKeySelector `json:"ldapBindPassword,omitempty"`

	// Settings for the pgAdmin server process. Keys should be uppercase and
	// values must be constants.
	// More info: https://www.pgadmin.org/docs/pgadmin4/latest/config_py.html
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Type=object
	Settings SchemalessObject `json:"settings,omitempty"`
}

// PGAdminPodSpec defines the desired state of a pgAdmin deployment.
type PGAdminPodSpec struct {
	// +optional
	Metadata *Metadata `json:"metadata,omitempty"`

	// Scheduling constraints of a pgAdmin pod. Changing this value causes
	// pgAdmin to restart.
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// Configuration settings for the pgAdmin process. Changes to any of these
	// values will be loaded without validation. Be careful, as
	// you may put pgAdmin into an unusable state.
	// +optional
	Config PGAdminConfiguration `json:"config,omitempty"`

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

	// Priority class name for the pgAdmin pod. Changing this value causes pgAdmin
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
