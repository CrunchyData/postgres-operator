/*
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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

// RestrictedPodSecurityContext returns a v1.PodSecurityContext with safe defaults.
// See https://docs.k8s.io/concepts/security/pod-security-standards/
func RestrictedPodSecurityContext() *corev1.PodSecurityContext {
	return &corev1.PodSecurityContext{
		// Fail to start a container if its image runs as UID 0 (root).
		RunAsNonRoot: Bool(true),
	}
}

// RestrictedSecurityContext returns a v1.SecurityContext with safe defaults.
// See https://docs.k8s.io/concepts/security/pod-security-standards/
func RestrictedSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		// Prevent any container processes from gaining privileges.
		AllowPrivilegeEscalation: Bool(false),

		// Processes in privileged containers are essentially root on the host.
		Privileged: Bool(false),

		// Limit filesystem changes to volumes that are mounted read-write.
		ReadOnlyRootFilesystem: Bool(true),

		// Fail to start the container if its image runs as UID 0 (root).
		RunAsNonRoot: Bool(true),
	}
}
