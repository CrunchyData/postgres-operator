// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
)

// +structType=atomic
type OptionalConfigMapKeyRef struct {
	ConfigMapKeyRef `json:",inline"`

	// Whether or not the ConfigMap or its data must be defined. Defaults to false.
	// +optional
	Optional *bool `json:"optional,omitempty"`
}

// AsProjection returns a copy of this as a [corev1.ConfigMapProjection].
func (in *OptionalConfigMapKeyRef) AsProjection(path string) corev1.ConfigMapProjection {
	out := in.ConfigMapKeyRef.AsProjection(path)
	if in.Optional != nil {
		v := *in.Optional
		out.Optional = &v
	}
	return out
}

// +structType=atomic
type ConfigMapKeyRef struct {
	// Name of the ConfigMap.
	// ---
	// https://pkg.go.dev/k8s.io/kubernetes/pkg/apis/core/validation#ValidateConfigMapName
	// +required
	Name DNS1123Subdomain `json:"name"`

	// Name of the data field within the ConfigMap.
	// ---
	// https://github.com/kubernetes/kubernetes/blob/v1.32.0/pkg/apis/core/validation/validation.go#L2849
	// https://pkg.go.dev/k8s.io/apimachinery/pkg/util/validation#IsConfigMapKey
	// +required
	Key ConfigDataKey `json:"key"`
}

// AsProjection returns a copy of this as a [corev1.ConfigMapProjection].
func (in *ConfigMapKeyRef) AsProjection(path string) corev1.ConfigMapProjection {
	var out corev1.ConfigMapProjection
	out.Name = in.Name
	out.Items = []corev1.KeyToPath{{Key: in.Key, Path: path}}
	return out
}

// +structType=atomic
type OptionalSecretKeyRef struct {
	SecretKeyRef `json:",inline"`

	// Whether or not the Secret or its data must be defined. Defaults to false.
	// +optional
	Optional *bool `json:"optional,omitempty"`
}

// AsProjection returns a copy of this as a [corev1.SecretProjection].
func (in *OptionalSecretKeyRef) AsProjection(path string) corev1.SecretProjection {
	out := in.SecretKeyRef.AsProjection(path)
	if in.Optional != nil {
		v := *in.Optional
		out.Optional = &v
	}
	return out
}

// +structType=atomic
type SecretKeyRef struct {
	// Name of the Secret.
	// ---
	// https://pkg.go.dev/k8s.io/kubernetes/pkg/apis/core/validation#ValidateSecretName
	// +required
	Name DNS1123Subdomain `json:"name"`

	// Name of the data field within the Secret.
	// ---
	// https://releases.k8s.io/v1.32.0/pkg/apis/core/validation/validation.go#L2867
	// https://pkg.go.dev/k8s.io/apimachinery/pkg/util/validation#IsConfigMapKey
	// +required
	Key ConfigDataKey `json:"key"`
}

// AsProjection returns a copy of this as a [corev1.SecretProjection].
func (in *SecretKeyRef) AsProjection(path string) corev1.SecretProjection {
	var out corev1.SecretProjection
	out.Name = in.Name
	out.Items = []corev1.KeyToPath{{Key: in.Key, Path: path}}
	return out
}
