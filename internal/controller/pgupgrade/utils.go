// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgupgrade

import (
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// The owner reference created by controllerutil.SetControllerReference blocks
// deletion. The OwnerReferencesPermissionEnforcement plugin requires that the
// creator of such a reference have either "delete" permission on the owner or
// "update" permission on the owner's "finalizers" subresource.
// - https://docs.k8s.io/reference/access-authn-authz/admission-controllers/
// +kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="pgupgrades/finalizers",verbs={update}

// setControllerReference sets owner as a Controller OwnerReference on controlled.
// It panics if another controller is already set.
func (r *PGUpgradeReconciler) setControllerReference(
	owner *v1beta1.PGUpgrade, controlled client.Object,
) {
	if metav1.GetControllerOf(controlled) != nil {
		panic(controllerutil.SetControllerReference(owner, controlled, r.Client.Scheme()))
	}

	controlled.SetOwnerReferences(append(
		controlled.GetOwnerReferences(),
		metav1.OwnerReference{
			APIVersion:         v1beta1.GroupVersion.String(),
			Kind:               "PGUpgrade",
			Name:               owner.GetName(),
			UID:                owner.GetUID(),
			BlockOwnerDeletion: initialize.Pointer(true),
			Controller:         initialize.Pointer(true),
		},
	))
}

// Merge takes sets of labels and merges them. The last set
// provided will win in case of conflicts.
func Merge(sets ...map[string]string) labels.Set {
	merged := labels.Set{}
	for _, set := range sets {
		merged = labels.Merge(merged, set)
	}
	return merged
}

// defaultFromEnv reads the environment variable key when value is empty.
func defaultFromEnv(value, key string) string {
	if value == "" {
		return os.Getenv(key)
	}
	return value
}
