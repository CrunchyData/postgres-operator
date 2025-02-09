// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import corev1 "k8s.io/api/core/v1"

// InstrumentationSpec defines the configuration for collecting logs and metrics
// via OpenTelemetry.
type InstrumentationSpec struct {
	// Image name to use for collector containers. When omitted, the value
	// comes from an operator environment variable.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	Image string `json:"image,omitempty"`

	// Resources holds the resource requirements for the collector container.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Config is the place for users to configure exporters and provide files.
	// +optional
	Config *InstrumentationConfigSpec `json:"config,omitempty"`

	// Logs is the place for users to configure the log collection.
	// +optional
	Logs *InstrumentationLogsSpec `json:"logs,omitempty"`
}

// InstrumentationConfigSpec allows users to configure their own exporters,
// add files, etc.
type InstrumentationConfigSpec struct {
	// Exporters allows users to configure OpenTelemetry exporters that exist
	// in the collector image.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Type=object
	// +optional
	Exporters SchemalessObject `json:"exporters,omitempty"`

	// Files allows the user to mount projected volumes into the collector
	// Pod so that files can be referenced by the collector as needed.
	// +optional
	Files []corev1.VolumeProjection `json:"files,omitempty"`
}

// InstrumentationLogsSpec defines the configuration for collecting logs via
// OpenTelemetry.
type InstrumentationLogsSpec struct {
	// Exporters allows users to specify which exporters they want to use in
	// the logs pipeline.
	// +optional
	Exporters []string `json:"exporters,omitempty"`
}
