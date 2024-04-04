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
	"testing"

	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/registration"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/events"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestUpgradeAuthorized(t *testing.T) {
	t.Run("UpgradeAlreadyInProgress", func(t *testing.T) {
		reconciler := new(PGUpgradeReconciler)
		upgrade := new(v1beta1.PGUpgrade)

		for _, required := range []bool{false, true} {
			reconciler.Registration = registration.RegistrationFunc(
				func(record.EventRecorder, client.Object, *[]metav1.Condition) bool {
					return required
				})

			meta.SetStatusCondition(&upgrade.Status.Conditions, metav1.Condition{
				Type:   ConditionPGUpgradeProgressing,
				Status: metav1.ConditionTrue,
			})

			result := reconciler.UpgradeAuthorized(upgrade)
			assert.Assert(t, result, "expected signal to proceed")

			progressing := meta.FindStatusCondition(upgrade.Status.Conditions, ConditionPGUpgradeProgressing)
			assert.Equal(t, progressing.Status, metav1.ConditionTrue)
		}
	})

	t.Run("RegistrationRequired", func(t *testing.T) {
		scheme, err := runtime.CreatePostgresOperatorScheme()
		assert.NilError(t, err)

		recorder := events.NewRecorder(t, scheme)
		upgrade := new(v1beta1.PGUpgrade)
		upgrade.Name = "some-upgrade"

		reconciler := PGUpgradeReconciler{
			Recorder: recorder,
			Registration: registration.RegistrationFunc(
				func(record.EventRecorder, client.Object, *[]metav1.Condition) bool {
					return true
				}),
		}

		meta.RemoveStatusCondition(&upgrade.Status.Conditions, ConditionPGUpgradeProgressing)

		result := reconciler.UpgradeAuthorized(upgrade)
		assert.Assert(t, !result, "expected signal to not proceed")

		condition := meta.FindStatusCondition(upgrade.Status.Conditions, v1beta1.Registered)
		if assert.Check(t, condition != nil) {
			assert.Equal(t, condition.Status, metav1.ConditionFalse)
		}

		if assert.Check(t, len(recorder.Events) > 0) {
			assert.Equal(t, recorder.Events[0].Type, "Warning")
			assert.Equal(t, recorder.Events[0].Regarding.Kind, "PGUpgrade")
			assert.Equal(t, recorder.Events[0].Regarding.Name, "some-upgrade")
			assert.Assert(t, cmp.Contains(recorder.Events[0].Note, "requires"))
		}
	})

	t.Run("RegistrationCompleted", func(t *testing.T) {
		reconciler := new(PGUpgradeReconciler)
		upgrade := new(v1beta1.PGUpgrade)

		called := false
		reconciler.Registration = registration.RegistrationFunc(
			func(record.EventRecorder, client.Object, *[]metav1.Condition) bool {
				called = true
				return false
			})

		meta.RemoveStatusCondition(&upgrade.Status.Conditions, ConditionPGUpgradeProgressing)

		result := reconciler.UpgradeAuthorized(upgrade)
		assert.Assert(t, result, "expected signal to proceed")
		assert.Assert(t, called, "expected registration package to clear conditions")
	})
}
