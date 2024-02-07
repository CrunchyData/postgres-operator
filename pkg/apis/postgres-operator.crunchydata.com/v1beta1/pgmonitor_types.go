/*
 Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
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

import corev1 "k8s.io/api/core/v1"

// PGMonitorSpec defines the desired state of the pgMonitor tool suite
type PGMonitorSpec struct {
	// +optional
	Exporter *ExporterSpec `json:"exporter,omitempty"`
}

type ExporterSpec struct {

	// Projected volumes containing custom PostgreSQL Exporter configuration.  Currently supports
	// the customization of PostgreSQL Exporter queries. If a "queries.yml" file is detected in
	// any volume projected using this field, it will be loaded using the "extend.query-path" flag:
	// https://github.com/prometheus-community/postgres_exporter#flags
	// Changing the values of field causes PostgreSQL and the exporter to restart.
	// +optional
	Configuration []corev1.VolumeProjection `json:"configuration,omitempty"`

	// Projected secret containing custom TLS certificates to encrypt output from the exporter
	// web server
	// +optional
	CustomTLSSecret *corev1.SecretProjection `json:"customTLSSecret,omitempty"`

	// The image name to use for crunchy-postgres-exporter containers. The image may
	// also be set using the RELATED_IMAGE_PGEXPORTER environment variable.
	// +optional
	Image string `json:"image,omitempty"`

	// Changing this value causes PostgreSQL and the exporter to restart.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}
