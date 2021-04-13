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

// PGBouncerPodSpec defines the desired state of a PgBouncer connection pooler.
type PGBouncerPodSpec struct {

	// Scheduling constraints of a PgBouncer pod.
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// TODO(cbandy): custom PgBouncer configuration
	// +optional
	//Configuration []corev1.VolumeProjection `json:"configuration,omitempty"`

	// A secret projection containing a certificate and key with which to encrypt
	// connections to PgBouncer. The "tls.crt", "tls.key", and "ca.crt" paths must
	// be PEM-encoded certificates and keys.
	// More info: https://kubernetes.io/docs/concepts/configuration/secret/#projection-of-secret-keys-to-specific-paths
	// +optional
	CustomTLSSecret *corev1.SecretProjection `json:"customTLSSecret,omitempty"`

	// TODO(cbandy): Make this optional. Derive the name from somewhere else?

	// Name of a container image that can run PgBouncer.
	// More info: https://kubernetes.io/docs/concepts/containers/images
	Image string `json:"image"`

	// Port on which PgBouncer should listen for client connections.
	// +optional
	// +kubebuilder:default=5432
	// +kubebuilder:validation:Minimum=1024
	Port *int32 `json:"port,omitempty"`

	// Compute resources of a PgBouncer container.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Tolerations of a PgBouncer pod.
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

func (s *PGBouncerPodSpec) Default() {
	if s.Port == nil {
		s.Port = new(int32)
		*s.Port = 5432
	}
}

type PGBouncerPodStatus struct {

	// Identifies the revision of PgBouncer assets that have been installed into
	// PostgreSQL.
	PostgreSQLRevision string `json:"postgresRevision,omitempty"`

	// Total number of ready pods.
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// Total number of non-terminated pods.
	Replicas int32 `json:"replicas,omitempty"`
}
