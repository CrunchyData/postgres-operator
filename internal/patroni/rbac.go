// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package patroni

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// "list", "patch", and "watch" are required. Include "get" for good measure.
// +kubebuilder:rbac:namespace=patroni,groups="",resources="pods",verbs={get}
// +kubebuilder:rbac:namespace=patroni,groups="",resources="pods",verbs={list,watch}
// +kubebuilder:rbac:namespace=patroni,groups="",resources="pods",verbs={patch}

// TODO(cbandy): Separate these so that one can choose ConfigMap over Endpoints.

// When using Endpoints for DCS, "create", "list", "patch", and "watch" are
// required. Include "get" for good measure. The `patronictl scaffold` and
// `patronictl remove` commands require "deletecollection".
// +kubebuilder:rbac:namespace=patroni,groups="",resources="endpoints",verbs={get}
// +kubebuilder:rbac:namespace=patroni,groups="",resources="endpoints",verbs={create,deletecollection}
// +kubebuilder:rbac:namespace=patroni,groups="",resources="endpoints",verbs={list,watch}
// +kubebuilder:rbac:namespace=patroni,groups="",resources="endpoints",verbs={patch}
// +kubebuilder:rbac:namespace=patroni,groups="",resources="services",verbs={create}

// The OpenShift RestrictedEndpointsAdmission plugin requires special
// authorization to create Endpoints that contain Pod IPs.
// - https://github.com/openshift/origin/pull/9383
// +kubebuilder:rbac:namespace=patroni,groups="",resources="endpoints/restricted",verbs={create}

// Permissions returns the RBAC rules Patroni needs for cluster.
func Permissions(cluster *v1beta1.PostgresCluster) []rbacv1.PolicyRule {
	// TODO(cbandy): This must change when using ConfigMaps for DCS.

	rules := make([]rbacv1.PolicyRule, 0, 4)

	rules = append(rules, rbacv1.PolicyRule{
		APIGroups: []string{corev1.SchemeGroupVersion.Group},
		Resources: []string{"endpoints"},
		Verbs:     []string{"create", "deletecollection", "get", "list", "patch", "watch"},
	})

	if cluster.Spec.OpenShift != nil && *cluster.Spec.OpenShift {
		rules = append(rules, rbacv1.PolicyRule{
			APIGroups: []string{corev1.SchemeGroupVersion.Group},
			Resources: []string{"endpoints/restricted"},
			Verbs:     []string{"create"},
		})
	}

	rules = append(rules, rbacv1.PolicyRule{
		APIGroups: []string{corev1.SchemeGroupVersion.Group},
		Resources: []string{"pods"},
		Verbs:     []string{"get", "list", "patch", "watch"},
	})

	// When using Endpoints for DCS, Patroni tries to create the "{scope}-config" service.
	// NOTE(cbandy): The PostgresCluster controller already creates this Service;
	// it might be possible to eliminate this permission if it also created the
	// Endpoints.
	rules = append(rules, rbacv1.PolicyRule{
		APIGroups: []string{corev1.SchemeGroupVersion.Group},
		Resources: []string{"services"},
		Verbs:     []string{"create"},
	})

	return rules
}
