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

package pgbackrest

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// +kubebuilder:rbac:namespace=pgbackrest,groups="",resources=pods,verbs=list
// +kubebuilder:rbac:namespace=pgbackrest,groups="",resources=pods/exec,verbs=create

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
