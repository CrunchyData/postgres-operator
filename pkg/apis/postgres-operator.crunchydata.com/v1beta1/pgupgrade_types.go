// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
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

	// The name of the Postgres cluster to upgrade.
	// ---
	// +kubebuilder:validation:MinLength=1
	// +required
	PostgresClusterName string `json:"postgresClusterName"`

	// The image name to use for major PostgreSQL upgrades.
	// +optional
	Image *string `json:"image,omitempty"`

	// ImagePullPolicy is used to determine when Kubernetes will attempt to
	// pull (download) container images.
	// More info: https://kubernetes.io/docs/concepts/containers/images/#image-pull-policy
	// ---
	// Kubernetes assumes the evaluation cost of an enum value is very large.
	// TODO(k8s-1.29): Drop MaxLength after Kubernetes 1.29; https://issue.k8s.io/119511
	// +kubebuilder:validation:MaxLength=15
	// +kubebuilder:validation:Type=string
	//
	// +kubebuilder:validation:Enum={Always,Never,IfNotPresent}
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// TODO(benjaminjb) Check the behavior: does updating ImagePullSecrets cause
	// all running PGUpgrade pods to restart?

	// The image pull secrets used to pull from a private registry.
	// Changing this value causes all running PGUpgrade pods to restart.
	// https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Resource requirements for the PGUpgrade container.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitzero"`

	// Scheduling constraints of the PGUpgrade pod.
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// TODO(benjaminjb) Check the behavior: does updating PriorityClassName cause
	// PGUpgrade to restart?

	// Priority class name for the PGUpgrade pod. Changing this
	// value causes PGUpgrade pod to restart.
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption
	// +optional
	PriorityClassName *string `json:"priorityClassName,omitempty"`

	// Tolerations of the PGUpgrade pod.
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	PGUpgradeSettings `json:",inline"`
}

// Arguments and settings for the pg_upgrade tool.
// See: https://www.postgresql.org/docs/current/pgupgrade.html
// ---
// +kubebuilder:validation:XValidation:rule=`self.fromPostgresVersion < self.toPostgresVersion`
// +kubebuilder:validation:XValidation:rule=`!has(self.transferMethod) || (self.toPostgresVersion < 12 ? self.transferMethod in ["Copy","Link"] : true)`,message="Only Copy or Link before PostgreSQL 12"
// +kubebuilder:validation:XValidation:rule=`!has(self.transferMethod) || (self.toPostgresVersion < 17 ? self.transferMethod in ["Clone","Copy","Link"] : true)`,message="Only Clone, Copy, or Link before PostgreSQL 17"
type PGUpgradeSettings struct {

	// The major version of PostgreSQL before the upgrade.
	// ---
	// +kubebuilder:validation:Minimum=11
	// +kubebuilder:validation:Maximum=17
	// +required
	FromPostgresVersion int32 `json:"fromPostgresVersion"`

	// The number of simultaneous processes pg_upgrade should use.
	// More info: https://www.postgresql.org/docs/current/pgupgrade.html
	// ---
	// +kubebuilder:validation:Minimum=0
	// +optional
	Jobs int32 `json:"jobs,omitempty"`

	// The major version of PostgreSQL to be upgraded to.
	// ---
	// +kubebuilder:validation:Minimum=11
	// +kubebuilder:validation:Maximum=17
	// +required
	ToPostgresVersion int32 `json:"toPostgresVersion"`

	// The method pg_upgrade should use to transfer files to the new cluster.
	// More info: https://www.postgresql.org/docs/current/pgupgrade.html
	// ---
	// Different versions of the tool have different methods.
	// - Copy and Link forever:  https://git.postgresql.org/gitweb/?p=postgresql.git;f=src/bin/pg_upgrade/pg_upgrade.h;hb=REL_10_0#l232
	// - Clone since 12:         https://git.postgresql.org/gitweb/?p=postgresql.git;f=src/bin/pg_upgrade/pg_upgrade.h;hb=REL_12_0#l232
	// - CopyFileRange since 17: https://git.postgresql.org/gitweb/?p=postgresql.git;f=src/bin/pg_upgrade/pg_upgrade.h;hb=REL_17_0#l251
	//
	// Kubernetes assumes the evaluation cost of an enum value is very large.
	// TODO(k8s-1.29): Drop MaxLength after Kubernetes 1.29; https://issue.k8s.io/119511
	// +kubebuilder:validation:MaxLength=15
	//
	// +kubebuilder:validation:Enum={Clone,Copy,CopyFileRange,Link}
	// +optional
	TransferMethod string `json:"transferMethod,omitempty"`
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
//+kubebuilder:storageversion
//+versionName=v1beta1

// PGUpgrade is the Schema for the pgupgrades API
type PGUpgrade struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +optional
	Spec PGUpgradeSpec `json:"spec,omitzero"`
	// +optional
	Status PGUpgradeStatus `json:"status,omitzero"`
}

//+kubebuilder:object:root=true

// PGUpgradeList contains a list of PGUpgrade
type PGUpgradeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []PGUpgrade `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PGUpgrade{}, &PGUpgradeList{})
}
