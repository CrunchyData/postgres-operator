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
	// Log records are exported in small batches. Set this field to change their size and frequency.
	// ---
	// +optional
	Batches *OpenTelemetryLogsBatchSpec `json:"batches,omitempty"`

	// Exporters allows users to specify which exporters they want to use in
	// the logs pipeline.
	// +optional
	Exporters []string `json:"exporters,omitempty"`

	// How long to retain log files locally. An RFC 3339 duration or a number
	// and unit: `12 hr`, `3d`, `4 weeks`, etc.
	// ---
	// Kubernetes ensures the value is in the "duration" format, but go ahead
	// and loosely validate the format to show some acceptable units.
	// NOTE: This rejects fractional numbers: https://github.com/kubernetes/kube-openapi/issues/523
	// +kubebuilder:validation:Pattern=`^(PT)?( *[0-9]+ *(?i:(h|hr|d|w|wk)|(hour|day|week)s?))+$`
	//
	// `controller-gen` needs to know "Type=string" to allow a "Pattern".
	// +kubebuilder:validation:Type=string
	//
	// Set a max length to keep rule costs low.
	// +kubebuilder:validation:MaxLength=20
	// +kubebuilder:validation:XValidation:rule=`duration("1h") <= self && self <= duration("8760h")`,message="must be at least one hour"
	//
	// +optional
	RetentionPeriod *Duration `json:"retentionPeriod,omitempty"`
}

// ---
// Configuration for the OpenTelemetry Batch Processor
// https://pkg.go.dev/go.opentelemetry.io/collector/processor/batchprocessor#section-readme
//
// The batch processor stops batching when *either* of these is zero, but that is confusing.
// Make the user set both so it is evident there is *no* motivation to create any batch.
// +kubebuilder:validation:XValidation:rule=`(has(self.minRecords) && self.minRecords == 0) == (has(self.maxDelay) && self.maxDelay == duration('0'))`,message=`to disable batching, both minRecords and maxDelay must be zero`
//
// +kubebuilder:validation:XValidation:rule=`!has(self.maxRecords) || self.minRecords <= self.maxRecords`,message=`minRecords cannot be larger than maxRecords`
// +structType=atomic
type OpenTelemetryLogsBatchSpec struct {
	// Maximum time to wait before exporting a log record. Higher numbers
	// allow more records to be deduplicated and compressed before export.
	// ---
	// Kubernetes ensures the value is in the "duration" format, but go ahead
	// and loosely validate the format to show some acceptable units.
	// NOTE: This rejects fractional numbers: https://github.com/kubernetes/kube-openapi/issues/523
	// +kubebuilder:validation:Pattern=`^((PT)?( *[0-9]+ *(?i:(ms|s|m)|(milli|sec|min)s?))+|0)$`
	//
	// `controller-gen` needs to know "Type=string" to allow a "Pattern".
	// +kubebuilder:validation:Type=string
	//
	// Set a max length to keep rule costs low.
	// +kubebuilder:validation:MaxLength=20
	// +kubebuilder:validation:XValidation:rule=`duration("0") <= self && self <= duration("5m")`
	//
	// +default="200ms"
	// +optional
	MaxDelay *Duration `json:"maxDelay,omitempty"`

	// Maximum number of records to include in an exported batch. When present,
	// batches this size are sent without any further delay.
	// ---
	// +kubebuilder:validation:Minimum=1
	// +optional
	MaxRecords *int32 `json:"maxRecords,omitempty"`

	// Number of records to wait for before exporting a batch. Higher numbers
	// allow more records to be deduplicated and compressed before export.
	// ---
	// +kubebuilder:validation:Minimum=0
	// +default=8192
	// +optional
	MinRecords *int32 `json:"minRecords,omitempty"`
}

func (s *OpenTelemetryLogsBatchSpec) Default() {
	if s.MaxDelay == nil {
		s.MaxDelay, _ = NewDuration("200ms")
	}
	if s.MinRecords == nil {
		s.MinRecords = new(int32)
		*s.MinRecords = 8192
	}
}
