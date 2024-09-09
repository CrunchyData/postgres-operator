// Copyright 2023 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package registration

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NOTE: This type can go away following https://go.dev/issue/47487.

type RegistrationFunc func(record.EventRecorder, client.Object, *[]metav1.Condition) bool

func (fn RegistrationFunc) Required(rec record.EventRecorder, obj client.Object, conds *[]metav1.Condition) bool {
	return fn(rec, obj, conds)
}

var _ Registration = RegistrationFunc(nil)
