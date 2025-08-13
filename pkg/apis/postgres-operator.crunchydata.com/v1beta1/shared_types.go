// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kube-openapi/pkg/validation/strfmt"
)

// ---
// https://pkg.go.dev/k8s.io/apimachinery/pkg/util/validation#IsConfigMapKey
//
// +kubebuilder:validation:MinLength=1
// +kubebuilder:validation:MaxLength=253
// +kubebuilder:validation:Pattern=`^[-._a-zA-Z0-9]+$`
// +kubebuilder:validation:XValidation:rule=`self != "." && !self.startsWith("..")`,message=`cannot be "." or start with ".."`
type ConfigDataKey = string

// ---
// https://docs.k8s.io/concepts/overview/working-with-objects/names#dns-subdomain-names
// https://pkg.go.dev/k8s.io/apimachinery/pkg/util/validation#IsDNS1123Subdomain
// https://pkg.go.dev/k8s.io/apiserver/pkg/cel/library#Format
//
// +kubebuilder:validation:MinLength=1
// +kubebuilder:validation:MaxLength=253
// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?([.][a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
type DNS1123Subdomain = string

// ---
// https://docs.k8s.io/concepts/overview/working-with-objects/names#dns-label-names
// https://pkg.go.dev/k8s.io/apimachinery/pkg/util/validation#IsDNS1123Label
// https://pkg.go.dev/k8s.io/apiserver/pkg/cel/library#Format
//
// +kubebuilder:validation:MinLength=1
// +kubebuilder:validation:MaxLength=63
// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
type DNS1123Label = string

// ---
// Duration represents a string accepted by the Kubernetes API in the "duration"
// [format]. This format extends the "duration" [defined by OpenAPI] by allowing
// some whitespace and more units:
//
//   - nanoseconds: ns, nano, nanos
//   - microseconds: us, µs, micro, micros
//   - milliseconds: ms, milli, millis
//   - seconds: s, sec, secs
//   - minutes: m, min, mins
//   - hours: h, hr, hour, hours
//   - days: d, day, days
//   - weeks: w, wk, week, weeks
//
// An empty amount is represented as "0" with no unit.
// One day is always 24 hours and one week is always 7 days (168 hours).
//
// +kubebuilder:validation:Format=duration
// +kubebuilder:validation:MinLength=1
// +kubebuilder:validation:Type=string
//
// During CEL validation, a value of this type is a "google.protobuf.Duration".
// It is safe to pass the value to `duration()` but not necessary.
//
// - https://docs.k8s.io/reference/using-api/cel/#type-system-integration
// - https://github.com/google/cel-spec/blob/-/doc/langdef.md#types-and-conversions
//
// NOTE: When using this type, reject fractional numbers using a Pattern to
// avoid an upstream bug: https://github.com/kubernetes/kube-openapi/issues/523
//
// [defined by OpenAPI]: https://spec.openapis.org/registry/format/duration.html
// [format]: https://spec.openapis.org/oas/latest.html#data-type-format
type Duration struct {
	parsed metav1.Duration
	string
}

// NewDuration creates a duration from the Kubernetes "duration" format in s.
func NewDuration(s string) (*Duration, error) {
	td, err := strfmt.ParseDuration(s)

	// The unkeyed fields here helpfully raise warnings from the compiler
	// if [metav1.Duration] changes shape in the future.
	type unkeyed metav1.Duration
	umd := unkeyed{td}

	return &Duration{metav1.Duration(umd), s}, err
}

// AsDuration returns a copy of d as a [metav1.Duration].
func (d *Duration) AsDuration() metav1.Duration {
	return d.parsed
}

// MarshalJSON implements [json.Marshaler].
func (d Duration) MarshalJSON() ([]byte, error) {
	if d.parsed.Duration == 0 {
		return json.Marshal("0")
	}

	return json.Marshal(d.string)
}

// UnmarshalJSON implements [json.Unmarshaler].
func (d *Duration) UnmarshalJSON(data []byte) error {
	var next *Duration
	var str string

	err := json.Unmarshal(data, &str)
	if err == nil {
		next, err = NewDuration(str)
	}
	if err == nil {
		*d = *next
	}
	return err
}

// ---
// NOTE(validation): Every PVC must have at least one accessMode. NOTE(KEP-5073)
// TODO(k8s-1.28): fieldPath=`.accessModes`,reason="FieldValueRequired"
// - https://releases.k8s.io/v1.25.0/pkg/apis/core/validation/validation.go#L2098-L2100
// - https://releases.k8s.io/v1.32.0/pkg/apis/core/validation/validation.go#L2303-L2305
// +kubebuilder:validation:XValidation:rule=`0 < size(self.accessModes)`,message=`missing accessModes`
//
// NOTE(validation): Every PVC must have a positive storage request. NOTE(KEP-5073)
// TODO(k8s-1.28): fieldPath=`.resources.requests.storage`,reason="FieldValueRequired"
// TODO(k8s-1.29): `&& 0 < quantity(self.resources.requests.storage).sign()`
// - https://releases.k8s.io/v1.25.0/pkg/apis/core/validation/validation.go#L2126-L2133
// - https://releases.k8s.io/v1.32.0/pkg/apis/core/validation/validation.go#L2329-L2336
// +kubebuilder:validation:XValidation:rule=`has(self.resources.requests.storage)`,message=`missing storage request`
//
// +structType=atomic
type VolumeClaimSpec corev1.PersistentVolumeClaimSpec

// DeepCopyInto copies the receiver into out. Both must be non-nil.
func (spec *VolumeClaimSpec) DeepCopyInto(out *VolumeClaimSpec) {
	(*corev1.PersistentVolumeClaimSpec)(spec).DeepCopyInto((*corev1.PersistentVolumeClaimSpec)(out))
}

// AsPersistentVolumeClaimSpec returns a copy of spec as a [corev1.PersistentVolumeClaimSpec].
func (spec *VolumeClaimSpec) AsPersistentVolumeClaimSpec() corev1.PersistentVolumeClaimSpec {
	var out corev1.PersistentVolumeClaimSpec
	spec.DeepCopyInto((*VolumeClaimSpec)(&out))
	return out
}

// ---
// SchemalessObject is a map compatible with JSON object.
//
// Use with the following markers:
//   - kubebuilder:pruning:PreserveUnknownFields
//   - kubebuilder:validation:Schemaless
//   - kubebuilder:validation:Type=object
//
// NOTE: PreserveUnknownFields allows arbitrary values within fields of this
// type but also prevents any validation rules from reaching inside; its CEL
// type is "object" or "message" with zero fields:
// https://kubernetes.io/docs/reference/using-api/cel/#type-system-integration
type SchemalessObject map[string]any

// DeepCopy creates a new SchemalessObject by copying the receiver.
func (in SchemalessObject) DeepCopy() SchemalessObject {
	return runtime.DeepCopyJSON(in)
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
	// ---
	// Kubernetes assumes the evaluation cost of an enum value is very large.
	// TODO(k8s-1.29): Drop MaxLength after Kubernetes 1.29; https://issue.k8s.io/119511
	// +kubebuilder:validation:MaxLength=15
	//
	// +optional
	// +kubebuilder:default=ClusterIP
	// +kubebuilder:validation:Enum={ClusterIP,NodePort,LoadBalancer}
	Type string `json:"type"`

	// More info: https://kubernetes.io/docs/reference/kubernetes-api/service-resources/service-v1/
	// ---
	// +optional
	// +kubebuilder:validation:Enum=SingleStack;PreferDualStack;RequireDualStack
	IPFamilyPolicy *corev1.IPFamilyPolicy `json:"ipFamilyPolicy,omitempty"`

	// +optional
	// +kubebuilder:validation:items:Enum={IPv4,IPv6}
	IPFamilies []corev1.IPFamily `json:"ipFamilies,omitempty"`

	// More info: https://kubernetes.io/docs/concepts/services-networking/service/#traffic-policies
	// ---
	// Kubernetes assumes the evaluation cost of an enum value is very large.
	// TODO(k8s-1.29): Drop MaxLength after Kubernetes 1.29; https://issue.k8s.io/119511
	// +kubebuilder:validation:MaxLength=10
	// +kubebuilder:validation:Type=string
	//
	// +optional
	// +kubebuilder:validation:Enum={Cluster,Local}
	InternalTrafficPolicy *corev1.ServiceInternalTrafficPolicy `json:"internalTrafficPolicy,omitempty"`

	// More info: https://kubernetes.io/docs/concepts/services-networking/service/#traffic-policies
	// ---
	// Kubernetes assumes the evaluation cost of an enum value is very large.
	// TODO(k8s-1.29): Drop MaxLength after Kubernetes 1.29; https://issue.k8s.io/119511
	// +kubebuilder:validation:MaxLength=10
	// +kubebuilder:validation:Type=string
	//
	// +optional
	// +kubebuilder:validation:Enum={Cluster,Local}
	ExternalTrafficPolicy *corev1.ServiceExternalTrafficPolicy `json:"externalTrafficPolicy,omitempty"`
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

// ---
// Only one applier should be managing each volume definition.
// https://docs.k8s.io/reference/using-api/server-side-apply#merge-strategy
// +structType=atomic
type AdditionalVolume struct {
	// Name of an existing PersistentVolumeClaim.
	// ---
	// https://pkg.go.dev/k8s.io/kubernetes/pkg/apis/core/validation#ValidatePersistentVolumeClaim
	// https://pkg.go.dev/k8s.io/kubernetes/pkg/apis/core/validation#ValidatePersistentVolumeName
	//
	// +required
	ClaimName DNS1123Subdomain `json:"claimName"`

	// The names of containers in which to mount this volume.
	// The default mounts the volume in *all* containers. An empty list does not mount the volume to any containers.
	// ---
	// These are matched against [corev1.Container.Name] in a PodSpec, which is a [DNS1123Label].
	// https://pkg.go.dev/k8s.io/kubernetes/pkg/apis/core/validation#ValidatePodSpec
	//
	// Container names are unique within a Pod, so this list can be, too.
	// +listType=set
	//
	// +kubebuilder:validation:MaxItems=10
	// +optional
	Containers []DNS1123Label `json:"containers"`

	// The name of the directory in which to mount this volume.
	// Volumes are mounted in containers at `/volumes/{name}`.
	// ---
	// This also goes into the [corev1.Volume.Name] field, which is a [DNS1123Label].
	// https://pkg.go.dev/k8s.io/kubernetes/pkg/apis/core/validation#ValidatePodSpec
	// https://pkg.go.dev/k8s.io/kubernetes/pkg/apis/core/validation#ValidateVolumes
	//
	// We prepend "volumes-" to avoid collisions with other [corev1.PodSpec.Volumes],
	// so the maximum is 8 less than the inherited 63.
	// +kubebuilder:validation:MaxLength=55
	//
	// +required
	Name DNS1123Label `json:"name"`

	// When true, mount the volume read-only, otherwise read-write. Defaults to false.
	// ---
	// [corev1.VolumeMount.ReadOnly]
	//
	// +optional
	ReadOnly bool `json:"readOnly,omitempty"`
}
