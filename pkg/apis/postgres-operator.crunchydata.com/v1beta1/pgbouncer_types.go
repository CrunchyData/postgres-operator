// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// PGBouncerConfiguration represents PgBouncer configuration files.
type PGBouncerConfiguration struct {

	// Files to mount under "/etc/pgbouncer". When specified, settings in the
	// "pgbouncer.ini" file are loaded before all others. From there, other
	// files may be included by absolute path. Changing these references causes
	// PgBouncer to restart, but changes to the file contents are automatically
	// reloaded.
	// More info: https://www.pgbouncer.org/config.html#include-directive
	// +optional
	Files []corev1.VolumeProjection `json:"files,omitempty"`

	// NOTE(cbandy): map[string]string fields are not presented in the OpenShift
	// web console: https://github.com/openshift/console/issues/9538

	// Settings that apply to the entire PgBouncer process.
	// More info: https://www.pgbouncer.org/config.html
	// +optional
	Global map[string]string `json:"global,omitempty"`

	// PgBouncer database definitions. The key is the database requested by a
	// client while the value is a libpq-styled connection string. The special
	// key "*" acts as a fallback. When this field is empty, PgBouncer is
	// configured with a single "*" entry that connects to the primary
	// PostgreSQL instance.
	// More info: https://www.pgbouncer.org/config.html#section-databases
	// +optional
	Databases map[string]string `json:"databases,omitempty"`

	// Connection settings specific to particular users.
	// More info: https://www.pgbouncer.org/config.html#section-users
	// +optional
	Users map[string]string `json:"users,omitempty"`
}

// PGBouncerPodSpec defines the desired state of a PgBouncer connection pooler.
type PGBouncerPodSpec struct {
	// +optional
	Metadata *Metadata `json:"metadata,omitempty"`

	// Scheduling constraints of a PgBouncer pod. Changing this value causes
	// PgBouncer to restart.
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// Configuration settings for the PgBouncer process. Changes to any of these
	// values will be automatically reloaded without validation. Be careful, as
	// you may put PgBouncer into an unusable state.
	// More info: https://www.pgbouncer.org/usage.html#reload
	// +optional
	Config PGBouncerConfiguration `json:"config,omitempty"`

	// Custom sidecars for a PgBouncer pod. Changing this value causes
	// PgBouncer to restart.
	// +optional
	Containers []corev1.Container `json:"containers,omitempty"`

	// A secret projection containing a certificate and key with which to encrypt
	// connections to PgBouncer. The "tls.crt", "tls.key", and "ca.crt" paths must
	// be PEM-encoded certificates and keys. Changing this value causes PgBouncer
	// to restart.
	// More info: https://kubernetes.io/docs/concepts/configuration/secret/#projection-of-secret-keys-to-specific-paths
	// +optional
	CustomTLSSecret *corev1.SecretProjection `json:"customTLSSecret,omitempty"`

	// Name of a container image that can run PgBouncer 1.15 or newer. Changing
	// this value causes PgBouncer to restart. The image may also be set using
	// the RELATED_IMAGE_PGBOUNCER environment variable.
	// More info: https://kubernetes.io/docs/concepts/containers/images
	// +optional
	Image string `json:"image,omitempty"`

	// Port on which PgBouncer should listen for client connections. Changing
	// this value causes PgBouncer to restart.
	// +optional
	// +kubebuilder:default=5432
	// +kubebuilder:validation:Minimum=1024
	Port *int32 `json:"port,omitempty"`

	// Priority class name for the pgBouncer pod. Changing this value causes
	// PostgreSQL to restart.
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/
	// +optional
	PriorityClassName *string `json:"priorityClassName,omitempty"`

	// Number of desired PgBouncer pods.
	// +optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=0
	Replicas *int32 `json:"replicas,omitempty"`

	// Minimum number of pods that should be available at a time.
	// Defaults to one when the replicas field is greater than one.
	// +optional
	MinAvailable *intstr.IntOrString `json:"minAvailable,omitempty"`

	// Compute resources of a PgBouncer container. Changing this value causes
	// PgBouncer to restart.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Specification of the service that exposes PgBouncer.
	// +optional
	Service *ServiceSpec `json:"service,omitempty"`

	// Configuration for pgBouncer sidecar containers
	// +optional
	Sidecars *PGBouncerSidecars `json:"sidecars,omitempty"`

	// Tolerations of a PgBouncer pod. Changing this value causes PgBouncer to
	// restart.
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Topology spread constraints of a PgBouncer pod. Changing this value causes
	// PgBouncer to restart.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-topology-spread-constraints/
	// +optional
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
}

// PGBouncerSidecars defines the configuration for pgBouncer sidecar containers
type PGBouncerSidecars struct {
	// Defines the configuration for the pgBouncer config sidecar container
	// +optional
	PGBouncerConfig *Sidecar `json:"pgbouncerConfig,omitempty"`
}

// Default returns the default port for PgBouncer (5432) if a port is not
// explicitly set
func (s *PGBouncerPodSpec) Default() {
	if s.Port == nil {
		s.Port = new(int32)
		*s.Port = 5432
	}

	if s.Replicas == nil {
		s.Replicas = new(int32)
		*s.Replicas = 1
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
