// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

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

		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
}
