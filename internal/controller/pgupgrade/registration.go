// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
