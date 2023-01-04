/*
 Copyright 2021 - 2023 Crunchy Data Solutions, Inc.
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

package initialize

import (
	corev1 "k8s.io/api/core/v1"
)

// PodSecurityContext returns a v1.PodSecurityContext with some defaults.
func PodSecurityContext() *corev1.PodSecurityContext {
	onRootMismatch := corev1.FSGroupChangeOnRootMismatch
	return &corev1.PodSecurityContext{
		// If set to "OnRootMismatch", if the root of the volume already has
		// the correct permissions, the recursive permission change can be skipped
		FSGroupChangePolicy: &onRootMismatch,
	}
}

// RestrictedSecurityContext returns a v1.SecurityContext with safe defaults.
// See https://docs.k8s.io/concepts/security/pod-security-standards/
func RestrictedSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		// Prevent any container processes from gaining privileges.
		AllowPrivilegeEscalation: Bool(false),

		// Drop any capabilities granted by the container runtime.
		// This must be uppercase to pass Pod Security Admission.
		// - https://releases.k8s.io/v1.24.0/staging/src/k8s.io/pod-security-admission/policy/check_capabilities_restricted.go
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},

		// Processes in privileged containers are essentially root on the host.
		Privileged: Bool(false),

		// Limit filesystem changes to volumes that are mounted read-write.
		ReadOnlyRootFilesystem: Bool(true),

		// Fail to start the container if its image runs as UID 0 (root).
		RunAsNonRoot: Bool(true),
	}
}
