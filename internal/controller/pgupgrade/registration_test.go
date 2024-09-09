// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

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
		recorder := events.NewRecorder(t, runtime.Scheme)
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
