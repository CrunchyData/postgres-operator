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
	// ---
	// +optional
	Image string `json:"image,omitempty"`

	// Resources holds the resource requirements for the collector container.
	// ---
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitzero"`

	// Config is the place for users to configure exporters and provide files.
	// ---
	// +optional
	Config *InstrumentationConfigSpec `json:"config,omitempty"`

	// Logs is the place for users to configure the log collection.
	// ---
	// +optional
	Logs *InstrumentationLogsSpec `json:"logs,omitempty"`

	// Metrics is the place for users to configure metrics collection.
	// ---
	// +optional
	Metrics *InstrumentationMetricsSpec `json:"metrics,omitempty"`
}

// InstrumentationConfigSpec allows users to configure their own exporters,
// add files, etc.
type InstrumentationConfigSpec struct {
	// Resource detectors add identifying attributes to logs and metrics. These run in the order they are defined.
	// More info: https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/-/processor/resourcedetectionprocessor#readme
	// ---
	// +kubebuilder:validation:MaxItems=10
	// +kubebuilder:validation:MinItems=1
	// +listMapKey=name
	// +listType=map
	// +optional
	Detectors []OpenTelemetryResourceDetector `json:"detectors,omitempty"`

	// Exporters allows users to configure OpenTelemetry exporters that exist
	// in the collector image.
	// ---
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Type=object
	// +optional
	Exporters SchemalessObject `json:"exporters,omitempty"`

	// Files allows the user to mount projected volumes into the collector
	// Pod so that files can be referenced by the collector as needed.
	// ---
	// +kubebuilder:validation:MinItems=1
	// +listType=atomic
	// +optional
	Files []corev1.VolumeProjection `json:"files,omitempty"`

	// EnvironmentVariables allows the user to add environment variables to the
	// collector container.
	// ---
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:items:XValidation:rule=`self.name != 'K8S_POD_NAMESPACE' && self.name != 'K8S_POD_NAME' && self.name != 'PGPASSWORD'`,message="Cannot overwrite environment variables set by operator"
	// +listType=atomic
	// +optional
	EnvironmentVariables []corev1.EnvVar `json:"environmentVariables,omitempty"`
}

// InstrumentationLogsSpec defines the configuration for collecting logs via
// OpenTelemetry.
type InstrumentationLogsSpec struct {
	// Log records are exported in small batches. Set this field to change their size and frequency.
	// ---
	// +optional
	Batches *OpenTelemetryLogsBatchSpec `json:"batches,omitempty"`

	// The names of exporters that should send logs.
	// ---
	// +kubebuilder:validation:MinItems=1
	// +listType=set
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

type InstrumentationMetricsSpec struct {
	// Where users can turn off built-in metrics and also provide their own
	// custom queries.
	// ---
	// +optional
	CustomQueries *InstrumentationCustomQueriesSpec `json:"customQueries,omitempty"`

	// The names of exporters that should send metrics.
	// ---
	// +kubebuilder:validation:MinItems=1
	// +listType=set
	// +optional
	Exporters []string `json:"exporters,omitempty"`

	// User defined databases to target for default per-db metrics
	// ---
	// +optional
	PerDBMetricTargets []string `json:"perDBMetricTargets,omitempty"`
}

type InstrumentationCustomQueriesSpec struct {
	// User defined queries and metrics.
	// ---
	// +optional
	Add []InstrumentationCustomQueries `json:"add,omitempty"`

	// A list of built-in queries that should be removed. If all queries for a
	// given SQL statement are removed, the SQL statement will no longer be run.
	// ---
	// +optional
	Remove []string `json:"remove,omitempty"`
}

type InstrumentationCustomQueries struct {
	// The name of this batch of queries, which will be used in naming the OTel
	// SqlQuery receiver.
	// ---
	// OTel restricts component names from having whitespace, control characters,
	// or symbols.
	// https://github.com/open-telemetry/opentelemetry-collector/blob/main/component/identifiable.go#L23-L26
	// +kubebuilder:validation:Pattern=`^[^\pZ\pC\pS]+$`
	//
	// Set a max length to keep rule costs low.
	// +kubebuilder:validation:MaxLength=20
	//
	// +required
	Name string `json:"name"`

	// A ConfigMap holding the yaml file that contains the queries.
	// ---
	// +required
	Queries ConfigMapKeyRef `json:"queries"`

	// How often the queries should be run.
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
	// +kubebuilder:validation:XValidation:rule=`duration("0") <= self && self <= duration("60m")`
	//
	// +default="5s"
	// +optional
	CollectionInterval *Duration `json:"collectionInterval,omitempty"`

	// The databases to target with added custom queries.
	// Default behavior is to target `postgres`.
	// ---
	// +optional
	Databases []string `json:"databases,omitempty"`
}

// ---
// Configuration for the OpenTelemetry Batch Processor
// https://pkg.go.dev/go.opentelemetry.io/collector/processor/batchprocessor#section-readme
// ---
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

// ---
// +structType=atomic
type OpenTelemetryResourceDetector struct {
	// Name of the resource detector to enable: `aks`, `eks`, `gcp`, etc.
	// ---
	// +kubebuilder:validation:MaxLength=20
	// +kubebuilder:validation:MinLength=1
	// +required
	Name string `json:"name"`

	// Attributes to use from this detector. Detectors usually add every attribute
	// they know automatically. Names omitted here behave according to detector defaults.
	// ---
	// +kubebuilder:validation:MaxProperties=30
	// +kubebuilder:validation:MinProperties=1
	// +mapType=atomic
	// +optional
	Attributes map[string]bool `json:"attributes,omitempty"`
}
