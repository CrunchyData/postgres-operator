// Copyright 2023 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package registration

import (
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

type Registration interface {
	// Required returns true when registration is required but the token is missing or invalid.
	Required(record.EventRecorder, client.Object, *[]metav1.Condition) bool
}

var URL = os.Getenv("REGISTRATION_URL")

func SetAdvanceWarning(recorder record.EventRecorder, object client.Object, conditions *[]metav1.Condition) {
	recorder.Eventf(object, corev1.EventTypeWarning, "Register Soon",
		"Crunchy Postgres for Kubernetes requires registration for upgrades."+
			" Register now to be ready for your next upgrade. See %s for details.", URL)

	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:   v1beta1.Registered,
		Status: metav1.ConditionFalse,
		Reason: "TokenRequired",
		Message: fmt.Sprintf(
			"Crunchy Postgres for Kubernetes requires registration for upgrades."+
				" Register now to be ready for your next upgrade. See %s for details.", URL),
		ObservedGeneration: object.GetGeneration(),
	})
}

func SetRequiredWarning(recorder record.EventRecorder, object client.Object, conditions *[]metav1.Condition) {
	recorder.Eventf(object, corev1.EventTypeWarning, "Registration Required",
		"Crunchy Postgres for Kubernetes requires registration for upgrades."+
			" Register now to be ready for your next upgrade. See %s for details.", URL)

	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:   v1beta1.Registered,
		Status: metav1.ConditionFalse,
		Reason: "TokenRequired",
		Message: fmt.Sprintf(
			"Crunchy Postgres for Kubernetes requires registration for upgrades."+
				" Upgrade suspended. See %s for details.", URL),
		ObservedGeneration: object.GetGeneration(),
	})
}

func emitFailedWarning(recorder record.EventRecorder, object client.Object) {
	recorder.Eventf(object, corev1.EventTypeWarning, "Token Authentication Failed",
		"See %s for details.", URL)
}

func emitVerifiedEvent(recorder record.EventRecorder, object client.Object) {
	recorder.Event(object, corev1.EventTypeNormal, "Token Verified",
		"Thank you for registering your installation of Crunchy Postgres for Kubernetes.")
}
