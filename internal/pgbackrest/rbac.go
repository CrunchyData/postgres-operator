// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgbackrest

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// +kubebuilder:rbac:groups="",resources="pods",verbs={list}
// +kubebuilder:rbac:groups="",resources="pods/exec",verbs={create}

// Permissions returns the RBAC rules pgBackRest needs for a cluster.
func Permissions(cluster *v1beta1.PostgresCluster) []rbacv1.PolicyRule {

	rules := make([]rbacv1.PolicyRule, 0, 2)

	rules = append(rules, rbacv1.PolicyRule{
		APIGroups: []string{corev1.SchemeGroupVersion.Group},
		Resources: []string{"pods"},
		Verbs:     []string{"list"},
	})

	rules = append(rules, rbacv1.PolicyRule{
		APIGroups: []string{corev1.SchemeGroupVersion.Group},
		Resources: []string{"pods/exec"},
		Verbs:     []string{"create"},
	})

	return rules
}

// +kubebuilder:rbac:groups="",resources="pods",verbs={patch}

// RepoHostPermissions returns the RBAC rules the pgBackRest repo host needs.
func RepoHostPermissions(cluster *v1beta1.PostgresCluster) []rbacv1.PolicyRule {

	rules := make([]rbacv1.PolicyRule, 0, 1)

	rules = append(rules, rbacv1.PolicyRule{
		APIGroups: []string{corev1.SchemeGroupVersion.Group},
		Resources: []string{"pods"},
		Verbs:     []string{"patch"},
	})

	return rules
}
