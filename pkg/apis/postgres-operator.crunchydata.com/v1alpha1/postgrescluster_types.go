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

package v1alpha1

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PostgresClusterSpec defines the desired state of PostgresCluster
type PostgresClusterSpec struct {

	// PostgreSQL archive configuration
	// +kubebuilder:validation:Required
	Archive Archive `json:"archive"`

	// +listType=map
	// +listMapKey=name
	InstanceSets []PostgresInstanceSetSpec `json:"instances"`

	// +optional
	Patroni *PatroniSpec `json:"patroni,omitempty"`

	// The port on which PostgreSQL should listen.
	// +optional
	// +kubebuilder:default=5432
	Port *int32 `json:"port,omitempty"`
}

func (s *PostgresClusterSpec) Default() {
	for i := range s.InstanceSets {
		s.InstanceSets[i].Default(i)
	}

	if s.Patroni == nil {
		s.Patroni = new(PatroniSpec)
	}
	s.Patroni.Default()

	if s.Port == nil {
		s.Port = new(int32)
		*s.Port = 5432
	}
}

// Archive defines a PostgreSQL archive configuration
type Archive struct {

	// pgBackRest archive configuration
	// +kubebuilder:validation:Required
	PGBackRest PGBackRestArchive `json:"pgbackrest"`
}

// PGBackRestArchive defines a pgBackRest archive configuration
type PGBackRestArchive struct {

	// Projected volumes containing custom pgBackRest configuration
	Configuration []corev1.VolumeProjection `json:"configuration,omitempty"`

	// The image name to use for the pgBackRest image
	// +kubebuilder:validation:Required
	Image string `json:"image,omitempty"`

	// Defines a pgBackRest repository host
	// +kubebuilder:validation:Required
	RepoHost RepoHost `json:"repoHost"`

	// A volume for use with a pgBackRest repository host
	// +kubebuilder:validation:Required
	RepoVolume VolumeSpec `json:"repoVolume"`
}

// PGBackRestStatus defines the status of pgBackRest within a PostgresCluster
type PGBackRestStatus struct {

	// Status information for the pgBackRest repository host
	// +optional
	RepoHost *RepoHostStatus `json:"repoHost,omitempty"`
}

// RepoHost represents a pgBackRest dedicated repository host
type RepoHost struct {

	// Resource requirements for a pgBackRest repository host
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// RepoHostStatus defines the status of pgBackRest repository host
type RepoHostStatus struct {
	metav1.TypeMeta `json:",inline"`

	// The name of the pgBackRest repository host
	// +optional
	Name string `json:"name"`

	// Whether or not the pgBackRest repository host is ready for use
	// +optional
	Ready bool `json:"ready"`
}

// PostgresClusterStatus defines the observed state of PostgresCluster
type PostgresClusterStatus struct {

	// +optional
	Patroni *PatroniStatus `json:"patroni,omitempty"`

	// Status information for pgBackRest
	// +optional
	PGBackRest *PGBackRestStatus `json:"pgbackrest,omitempty"`

	// observedGeneration represents the .metadata.generation on which the status was based.
	// +optional
	// +kubebuilder:validation:Minimum=0
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// conditions represent the observations of postgrescluster's current state.
	// Known .status.conditions.type are: TODO
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type PostgresInstanceSetSpec struct {
	// +optional
	// +kubebuilder:default=""
	Name string `json:"name"`

	// +optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=0
	Replicas *int32 `json:"replicas,omitempty"`

	// +optional
	Resources v1.ResourceRequirements `json:"resources,omitempty"`
}

func (s *PostgresInstanceSetSpec) Default(i int) {
	if s.Name == "" {
		s.Name = fmt.Sprintf("%02d", i)
	}
	if s.Replicas == nil {
		s.Replicas = new(int32)
		*s.Replicas = 1
	}
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PostgresCluster is the Schema for the postgresclusters API
type PostgresCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgresClusterSpec   `json:"spec,omitempty"`
	Status PostgresClusterStatus `json:"status,omitempty"`
}

// Default implements "sigs.k8s.io/controller-runtime/pkg/webhook.Defaulter" so
// a webhook can be registered for the type.
// - https://book.kubebuilder.io/reference/webhook-overview.html
func (c *PostgresCluster) Default() {
	if len(c.APIVersion) == 0 {
		c.APIVersion = GroupVersion.String()
	}
	if len(c.Kind) == 0 {
		c.Kind = "PostgresCluster"
	}
	c.Spec.Default()
}

// +kubebuilder:object:root=true

// PostgresClusterList contains a list of PostgresCluster
type PostgresClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgresCluster `json:"items"`
}

// VolumeSpec defines the volume types available for use in a PostgresCluster
// Please note that this type could benefit from the "union/oneOf" semantics described in the
// following proposal, being that "only one of the given fields can be set" for this type:
// https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/1027-api-unions
type VolumeSpec struct {

	//  Defines an EmptyDir volume
	EmptyDir *corev1.EmptyDirVolumeSource `json:"emptyDir,omitempty"`

	// Defines a PersistentVolumeClaim used create and/or bind a volume
	VolumeClaimTemplate *corev1.PersistentVolumeClaimSpec `json:"volumeClaimTemplate,omitempty"`
}

func init() {
	SchemeBuilder.Register(&PostgresCluster{}, &PostgresClusterList{})
}
