// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgbackrest

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// +kubebuilder:rbac:namespace=pgbackrest,groups="",resources="pods",verbs={list}
// +kubebuilder:rbac:namespace=pgbackrest,groups="",resources="pods/exec",verbs={create}

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
