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
	"k8s.io/apimachinery/pkg/runtime"
)

type PatroniSpec struct {
	// TODO(cbandy): Find a better way to have a map[string]interface{} here.
	// See: https://github.com/kubernetes-sigs/controller-tools/pull/528
	// TODO(cbandy): Describe this field.

	// +optional
	// +kubebuilder:validation:XPreserveUnknownFields
	DynamicConfiguration runtime.RawExtension `json:"dynamicConfiguration,omitempty"`

	// TODO(cbandy): Remove this completely.

	// DEPRECATED. A manual switch to enable dynamic configuration.
	// +optional
	EDC *bool `json:"edc,omitempty"`

	// TODO(cbandy): Describe the downtime involved with changing.

	// The port on which Patroni should listen.
	// +optional
	// +kubebuilder:default=8008
	Port *int32 `json:"port,omitempty"`

	// TODO(cbandy): Add UseConfigMaps bool, default false.
	// TODO(cbandy): Allow other DCS: etcd, raft, etc?
	// N.B. changing this will cause downtime.
	// - https://patroni.readthedocs.io/en/latest/kubernetes.html
}

func (s *PatroniSpec) Default() {
	if s.Port == nil {
		s.Port = new(int32)
		*s.Port = 8008
	}
}
