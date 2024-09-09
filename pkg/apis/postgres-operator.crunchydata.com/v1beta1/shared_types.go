// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// SchemalessObject is a map compatible with JSON object.
//
// Use with the following markers:
// - kubebuilder:pruning:PreserveUnknownFields
// - kubebuilder:validation:Schemaless
// - kubebuilder:validation:Type=object
type SchemalessObject map[string]any

// DeepCopy creates a new SchemalessObject by copying the receiver.
func (in *SchemalessObject) DeepCopy() *SchemalessObject {
	if in == nil {
		return nil
	}
	out := new(SchemalessObject)
	*out = runtime.DeepCopyJSON(*in)
	return out
}

type ServiceSpec struct {
	// +optional
	Metadata *Metadata `json:"metadata,omitempty"`

	// The port on which this service is exposed when type is NodePort or
	// LoadBalancer. Value must be in-range and not in use or the operation will
	// fail. If unspecified, a port will be allocated if this Service requires one.
	// - https://kubernetes.io/docs/concepts/services-networking/service/#type-nodeport
	// +optional
	NodePort *int32 `json:"nodePort,omitempty"`

	// More info: https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types
	//
	// +optional
	// +kubebuilder:default=ClusterIP
	// +kubebuilder:validation:Enum={ClusterIP,NodePort,LoadBalancer}
	Type string `json:"type"`

	// More info: https://kubernetes.io/docs/concepts/services-networking/service/#traffic-policies
	//
	// +optional
	// +kubebuilder:validation:Enum={Cluster,Local}
	InternalTrafficPolicy *corev1.ServiceInternalTrafficPolicyType `json:"internalTrafficPolicy,omitempty"`

	// More info: https://kubernetes.io/docs/concepts/services-networking/service/#traffic-policies
	//
	// +optional
	// +kubebuilder:validation:Enum={Cluster,Local}
	ExternalTrafficPolicy *corev1.ServiceExternalTrafficPolicyType `json:"externalTrafficPolicy,omitempty"`
}

// Sidecar defines the configuration of a sidecar container
type Sidecar struct {
	// Resource requirements for a sidecar container
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// Metadata contains metadata for custom resources
type Metadata struct {
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// GetLabelsOrNil gets labels from a Metadata pointer, if Metadata
// hasn't been set return nil
func (meta *Metadata) GetLabelsOrNil() map[string]string {
	if meta == nil {
		return nil
	}
	return meta.Labels
}

// GetAnnotationsOrNil gets annotations from a Metadata pointer, if Metadata
// hasn't been set return nil
func (meta *Metadata) GetAnnotationsOrNil() map[string]string {
	if meta == nil {
		return nil
	}
	return meta.Annotations
}
