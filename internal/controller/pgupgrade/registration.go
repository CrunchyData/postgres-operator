// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgupgrade

import (
	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/crunchydata/postgres-operator/internal/registration"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func (r *PGUpgradeReconciler) UpgradeAuthorized(upgrade *v1beta1.PGUpgrade) bool {
	// Allow an upgrade in progress to complete, when the registration requirement is introduced.
	// But don't allow new upgrades to be started until a valid token is applied.
	progressing := meta.FindStatusCondition(upgrade.Status.Conditions, ConditionPGUpgradeProgressing) != nil
	required := r.Registration.Required(r.Recorder, upgrade, &upgrade.Status.Conditions)

	// If a valid token has not been applied, warn the user.
	if required && !progressing {
		registration.SetRequiredWarning(r.Recorder, upgrade, &upgrade.Status.Conditions)
		return false
	}

	return true
}
