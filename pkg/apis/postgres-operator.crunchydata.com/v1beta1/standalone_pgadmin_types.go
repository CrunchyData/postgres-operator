// Copyright 2023 Crunchy Data Solutions, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PGAdminConfiguration represents pgAdmin configuration files.
type StandalonePGAdminConfiguration struct {
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

// PGAdminSpec defines the desired state of PGAdmin
type PGAdminSpec struct {

	// +optional
	Metadata *Metadata `json:"metadata,omitempty"`

	// Configuration settings for the pgAdmin process. Changes to any of these
	// values will be loaded without validation. Be careful, as
	// you may put pgAdmin into an unusable state.
	// +optional
	Config StandalonePGAdminConfiguration `json:"config,omitempty"`

	// Defines a PersistentVolumeClaim for pgAdmin data.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes
	// +kubebuilder:validation:Required
	DataVolumeClaimSpec corev1.PersistentVolumeClaimSpec `json:"dataVolumeClaimSpec"`

	// The image name to use for standalone pgAdmin instance.
	// +optional
	Image *string `json:"image,omitempty"`

	// ImagePullPolicy is used to determine when Kubernetes will attempt to
	// pull (download) container images.
	// More info: https://kubernetes.io/docs/concepts/containers/images/#image-pull-policy
	// +kubebuilder:validation:Enum={Always,Never,IfNotPresent}
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// The image pull secrets used to pull from a private registry.
	// Changing this value causes all running PGAdmin pods to restart.
	// https://k8s.io/docs/tasks/configure-pod-container/pull-image-private-registry/
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Resource requirements for the PGAdmin container.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Scheduling constraints of the PGAdmin pod.
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// Priority class name for the PGAdmin pod. Changing this
	// value causes PGAdmin pod to restart.
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/
	// +optional
	PriorityClassName *string `json:"priorityClassName,omitempty"`

	// Specification of the service that exposes pgAdmin.
	// +optional
	Service *ServiceSpec `json:"service,omitempty"`

	// Tolerations of the PGAdmin pod.
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

// PGAdminStatus defines the observed state of PGAdmin
type PGAdminStatus struct {

	// conditions represent the observations of pgadmin's current state.
	// Known .status.conditions.type are: "PersistentVolumeResizing",
	// "Progressing", "ProxyAvailable"
	// +optional
	// +listType=map
	// +listMapKey=type
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// observedGeneration represents the .metadata.generation on which the status was based.
	// +optional
	// +kubebuilder:validation:Minimum=0
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PGAdmin is the Schema for the pgadmins API
type PGAdmin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PGAdminSpec   `json:"spec,omitempty"`
	Status PGAdminStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PGAdminList contains a list of PGAdmin
type PGAdminList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PGAdmin `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PGAdmin{}, &PGAdminList{})
}
