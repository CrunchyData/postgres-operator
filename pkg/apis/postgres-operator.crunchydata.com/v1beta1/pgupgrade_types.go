// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PGUpgradeSpec defines the desired state of PGUpgrade
type PGUpgradeSpec struct {

	// +optional
	Metadata *Metadata `json:"metadata,omitempty"`

	// The name of the cluster to be updated
	// +required
	// +kubebuilder:validation:MinLength=1
	PostgresClusterName string `json:"postgresClusterName"`

	// The image name to use for major PostgreSQL upgrades.
	// +optional
	Image *string `json:"image,omitempty"`

	// ImagePullPolicy is used to determine when Kubernetes will attempt to
	// pull (download) container images.
	// More info: https://kubernetes.io/docs/concepts/containers/images/#image-pull-policy
	// +kubebuilder:validation:Enum={Always,Never,IfNotPresent}
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// TODO(benjaminjb) Check the behavior: does updating ImagePullSecrets cause
	// all running PGUpgrade pods to restart?

	// The image pull secrets used to pull from a private registry.
	// Changing this value causes all running PGUpgrade pods to restart.
	// https://k8s.io/docs/tasks/configure-pod-container/pull-image-private-registry/
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// TODO(benjaminjb): define webhook validation to make sure
	// `fromPostgresVersion` is below `toPostgresVersion`
	// or leverage other validation rules, such as the Common Expression Language
	// rules currently in alpha as of Kubernetes 1.23
	// - https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#validation-rules

	// The major version of PostgreSQL before the upgrade.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=10
	// +kubebuilder:validation:Maximum=17
	FromPostgresVersion int `json:"fromPostgresVersion"`

	// TODO(benjaminjb): define webhook validation to make sure
	// `fromPostgresVersion` is below `toPostgresVersion`
	// or leverage other validation rules, such as the Common Expression Language
	// rules currently in alpha as of Kubernetes 1.23

	// The major version of PostgreSQL to be upgraded to.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=10
	// +kubebuilder:validation:Maximum=17
	ToPostgresVersion int `json:"toPostgresVersion"`

	// The image name to use for PostgreSQL containers after upgrade.
	// When omitted, the value comes from an operator environment variable.
	// +optional
	ToPostgresImage string `json:"toPostgresImage,omitempty"`

	// Resource requirements for the PGUpgrade container.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Scheduling constraints of the PGUpgrade pod.
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// TODO(benjaminjb) Check the behavior: does updating PriorityClassName cause
	// PGUpgrade to restart?

	// Priority class name for the PGUpgrade pod. Changing this
	// value causes PGUpgrade pod to restart.
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/
	// +optional
	PriorityClassName *string `json:"priorityClassName,omitempty"`

	// Tolerations of the PGUpgrade pod.
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

// PGUpgradeStatus defines the observed state of PGUpgrade
type PGUpgradeStatus struct {
	// conditions represent the observations of PGUpgrade's current state.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// observedGeneration represents the .metadata.generation on which the status was based.
	// +optional
	// +kubebuilder:validation:Minimum=0
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PGUpgrade is the Schema for the pgupgrades API
type PGUpgrade struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PGUpgradeSpec   `json:"spec,omitempty"`
	Status PGUpgradeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PGUpgradeList contains a list of PGUpgrade
type PGUpgradeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PGUpgrade `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PGUpgrade{}, &PGUpgradeList{})
}
