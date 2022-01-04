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

package postgrescluster

// Note: The behavior for an empty selector differs between the
// policy/v1beta1 and policy/v1 APIs for PodDisruptionBudgets. For
// policy/v1beta1 an empty selector matches zero pods, while for
// policy/v1 an empty selector matches every pod in the namespace.
// https://kubernetes.io/docs/tasks/run-application/configure-pdb/#specifying-a-poddisruptionbudget
import (
	"github.com/pkg/errors"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// generatePodDisruptionBudget takes parameters required to fill out a PDB and
// returns the PDB
func (r *Reconciler) generatePodDisruptionBudget(
	cluster *v1beta1.PostgresCluster,
	meta metav1.ObjectMeta,
	minAvailable *intstr.IntOrString,
	selector metav1.LabelSelector,
) (*policyv1beta1.PodDisruptionBudget, error) {
	pdb := &policyv1beta1.PodDisruptionBudget{
		ObjectMeta: meta,
		Spec: policyv1beta1.PodDisruptionBudgetSpec{
			MinAvailable: minAvailable,
			Selector:     &selector,
		},
	}
	pdb.SetGroupVersionKind(policyv1beta1.SchemeGroupVersion.WithKind("PodDisruptionBudget"))
	err := errors.WithStack(r.setControllerReference(cluster, pdb))
	return pdb, err
}

// getMinAvailable contains logic to either parse a user provided IntOrString
// value or determine a default minimum available based on replicas. In both
// cases it returns the minAvailable as an int32 that should be set on a
// PodDisruptionBudget
func getMinAvailable(
	minAvailable *intstr.IntOrString,
	replicas int32,
) *intstr.IntOrString {
	// TODO: Webhook Validation for minAvailable in the spec
	// - MinAvailable should be less than replicas
	// - MinAvailable as a string value should be a percentage string <= 100%
	if minAvailable != nil {
		return minAvailable
	}

	// If the user does not provide 'minAvailable', we will set a default
	// based on the number of replicas.
	var expect int32

	// We default to '1' if they have more than one replica defined.
	if replicas > 1 {
		expect = 1
	}

	// If more than one replica is not defined, we will default to '0'
	return initialize.IntOrStringInt32(expect)
}
