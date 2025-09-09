// Copyright 2017 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/internal/initialize"
)

// PodSecurityContext returns a v1.PodSecurityContext for cluster that can write
// to PersistentVolumes.
// This func sets the supplmental groups and fsGgroup if present.
// fsGroup should not be present in OpenShift environments
func PodSecurityContext(fsgroup int64, supplementalGroups []int64) *corev1.PodSecurityContext {
	psc := initialize.PodSecurityContext()

	// Use the specified supplementary groups except for root. The CRD has
	// similar validation, but we should never emit a PodSpec with that group.
	// - https://docs.k8s.io/concepts/security/pod-security-standards/
	for i := range supplementalGroups {
		if gid := supplementalGroups[i]; gid > 0 {
			psc.SupplementalGroups = append(psc.SupplementalGroups, gid)
		}
	}

	// OpenShift assigns a filesystem group based on a SecurityContextConstraint.
	// Otherwise, set a filesystem group so PostgreSQL can write to files
	// regardless of the UID or GID of a container.
	// - https://cloud.redhat.com/blog/a-guide-to-openshift-and-uids
	// - https://docs.k8s.io/tasks/configure-pod-container/security-context/
	// - https://docs.openshift.com/container-platform/4.8/authentication/managing-security-context-constraints.html
	if fsgroup > 0 {
		psc.FSGroup = initialize.Int64(fsgroup)
	}

	return psc
}
